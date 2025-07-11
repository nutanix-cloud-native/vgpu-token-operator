// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nkpv1alpha1 "github.com/nutanix-cloud-native/vgpu-token-operator/api/v1alpha1"
	"github.com/nutanix-cloud-native/vgpu-token-operator/pkg/generator"
	"github.com/nutanix-cloud-native/vgpu-token-operator/pkg/k8s/client"
)

//nolint:gosec // these are not hardcoded secrets
const tokenSecretRefIndex = ".spec.tokenSecret"

// VGPUTokenReconciler reconciles a VGPUToken object.
type VGPUTokenReconciler struct {
	ctrlclient.Client
	Scheme                   *runtime.Scheme
	VGPUTokenPropagatorImage string
}

// +kubebuilder:rbac:groups=vgpu-token.nutanix.com,resources=vgputokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vgpu-token.nutanix.com,resources=vgputokens/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vgpu-token.nutanix.com,resources=vgputokens/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *VGPUTokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile VGPUTokenReconciler")
	vgpuToken := &nkpv1alpha1.VGPUToken{}
	if err := r.Client.Get(ctx, req.NamespacedName, vgpuToken); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileNormal(ctx, vgpuToken)
}

func (r *VGPUTokenReconciler) reconcileNormal(
	ctx context.Context,
	vgpuToken *nkpv1alpha1.VGPUToken,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	// write status as the last thing we do
	defer func() {
		if updateErr := r.Status().Update(ctx, vgpuToken); updateErr != nil {
			log.Error(updateErr, "failed to update VGPUToken status")
		}
	}()
	var secretForDaemonSet corev1.Secret
	err := r.Client.Get(
		ctx,
		ctrlclient.ObjectKey{
			Name:      vgpuToken.Spec.TokenSecretRef.Name,
			Namespace: vgpuToken.GetNamespace(),
		},
		&secretForDaemonSet,
	)
	secretCond := metav1.Condition{
		Type:    nkpv1alpha1.ConditionTokenSecret,
		Status:  metav1.ConditionTrue,
		Message: "found secret",
		Reason:  nkpv1alpha1.ReasonTokenSecretRetrieved,
	}
	// sets the secret condition even in case of error
	defer func() {
		meta.SetStatusCondition(&vgpuToken.Status.Conditions, secretCond)
	}()
	if err != nil {
		log.Error(err, "failed to get secret")
		setCondition(
			&secretCond,
			metav1.ConditionFalse,
			err.Error(),
			nkpv1alpha1.ReasonTokenSecretFailed,
		)
		if apierrors.IsNotFound(err) {
			secretCond.Reason = nkpv1alpha1.ReasonTokenSecretNotFound
		}
		return ctrl.Result{}, err
	}

	if err := controllerutil.SetOwnerReference(vgpuToken, &secretForDaemonSet, r.Scheme); err != nil {
		errMsg := fmt.Sprintf("failed to set owner reference on secret %s", secretForDaemonSet.Name)
		log.Error(err, errMsg)
		setCondition(
			&secretCond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonCreateFailed,
			fmt.Sprintf("%s: %s", errMsg, err.Error()),
		)
	}

	err = r.reconcileServiceAccount(ctx, vgpuToken)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile service account %w", err)
	}

	err = r.reconcileRole(ctx, vgpuToken)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile role %w", err)
	}

	err = r.reconcileRoleBinding(ctx, vgpuToken)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile rolebinding %w", err)
	}

	ds, err := r.reconcileDaemonSet(ctx, vgpuToken, &secretForDaemonSet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile daemonset %w", err)
	}
	daemonSetCond := metav1.Condition{
		Type: nkpv1alpha1.ConditionPropagated,
	}
	// sets daemonset condition on status even if error occurs
	defer func() {
		meta.SetStatusCondition(&vgpuToken.Status.Conditions, daemonSetCond)
	}()
	if ds.Status.ObservedGeneration < ds.Generation {
		setCondition(
			&daemonSetCond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonDaemonSetUpdating,
			"DaemonSet is processing updates",
		)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	desired := ds.Status.DesiredNumberScheduled
	available := ds.Status.NumberAvailable
	var requeueAfter time.Duration
	switch {
	case desired == 0: // set none desired
		setCondition(
			&daemonSetCond,
			metav1.ConditionTrue,
			nkpv1alpha1.ReasonNoneDesired,
			"No replicas desired",
		)
	case available == 0: // set none ready
		setCondition(
			&daemonSetCond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonNoneReady,
			fmt.Sprintf("desired %d but none available", desired),
		)
	case available < desired: // set partial ready
		setCondition(
			&daemonSetCond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonPartialReady,
			fmt.Sprintf("desired %d but have %d available", desired, available),
		)
		requeueAfter = time.Minute * 2
	case desired == available: // set all ready
		setCondition(
			&daemonSetCond,
			metav1.ConditionTrue,
			nkpv1alpha1.ReasonAllReady,
			"All replicas ready",
		)
	}
	if requeueAfter > 0 {
		return ctrl.Result{
			RequeueAfter: requeueAfter,
		}, nil
	}
	return ctrl.Result{}, nil
}

func reconcileOwnedResource[T ctrlclient.Object](
	ctx context.Context,
	reconciler *VGPUTokenReconciler,
	token *nkpv1alpha1.VGPUToken,
	conditionType string,
	generateFunc func(string) T,
	newEmptyObj T,
) (T, error) {
	logger := logf.FromContext(ctx)
	namespace := token.Namespace
	desiredObj := generateFunc(namespace)

	resourceTypeName := desiredObj.GetObjectKind().GroupVersionKind().Kind
	logger.Info(fmt.Sprintf("Reconciling resource %s", resourceTypeName))

	gotObj := newEmptyObj
	gotObj.SetName(desiredObj.GetName())
	gotObj.SetNamespace(namespace)
	cond := metav1.Condition{Type: conditionType}
	defer func() {
		if cond.Status == metav1.ConditionTrue {
			cond.ObservedGeneration = token.Generation
		}
		meta.SetStatusCondition(&token.Status.Conditions, cond)
	}()
	getErr := reconciler.Get(ctx, ctrlclient.ObjectKeyFromObject(gotObj), gotObj)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			logger.Info("Resource not found, creating.")
			desiredObj := generateFunc(namespace)
			if err := controllerutil.SetOwnerReference(token, desiredObj, reconciler.Scheme); err != nil {
				errMsg := fmt.Sprintf("failed to set owner reference on %s", resourceTypeName)
				logger.Error(err, errMsg)
				setCondition(
					&cond,
					metav1.ConditionFalse,
					nkpv1alpha1.ReasonCreateFailed,
					fmt.Sprintf("%s: %s", errMsg, err.Error()),
				)
				return gotObj, fmt.Errorf("%s: %w", errMsg, err)
			}
			if createErr := reconciler.Create(ctx, desiredObj); createErr != nil {
				errMsg := fmt.Sprintf("failed to create %s", resourceTypeName)
				logger.Error(createErr, errMsg)
				setCondition(
					&cond,
					metav1.ConditionFalse,
					nkpv1alpha1.ReasonCreateFailed,
					fmt.Sprintf("%s: %s", errMsg, createErr.Error()),
				)
				return gotObj, fmt.Errorf("%s: %w", errMsg, createErr)
			}
			logger.Info("Resource created successfully.")
			setCondition(
				&cond,
				metav1.ConditionTrue,
				nkpv1alpha1.ReasonCreateSuccess,
				fmt.Sprintf("%s %s created successfully", resourceTypeName, desiredObj.GetName()),
			)
			return desiredObj, nil
		} else {
			errMsg := fmt.Sprintf("failed to get %s", resourceTypeName)
			logger.Error(getErr, errMsg)
			setCondition(
				&cond,
				metav1.ConditionFalse,
				nkpv1alpha1.ReasonGetFailed,
				fmt.Sprintf("%s: %s", errMsg, getErr.Error()),
			)
			return gotObj, fmt.Errorf("%s: %w", errMsg, getErr)
		}
	}
	logger.Info(fmt.Sprintf("Applying desired state to resource %s", resourceTypeName))
	if err := controllerutil.SetOwnerReference(token, desiredObj, reconciler.Scheme); err != nil {
		errMsg := fmt.Sprintf("failed to set owner reference for apply on %s", resourceTypeName)
		logger.Error(err, errMsg)
		setCondition(
			&cond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonUpdateFailed,
			fmt.Sprintf("%s: %s", errMsg, err.Error()),
		)
		return desiredObj, fmt.Errorf("%s: %w", errMsg, err)
	}
	if applyErr := client.ServerSideApply(
		ctx,
		reconciler.Client,
		desiredObj,
		client.ForceOwnership,
	); applyErr != nil {
		errMsg := fmt.Sprintf("failed to apply %s", resourceTypeName)
		logger.Error(applyErr, errMsg)
		setCondition(
			&cond,
			metav1.ConditionFalse,
			nkpv1alpha1.ReasonUpdateFailed,
			fmt.Sprintf("%s: %s", errMsg, applyErr.Error()),
		)
		return desiredObj, fmt.Errorf("%s: %w", errMsg, applyErr)
	}
	logger.Info(fmt.Sprintf("Resource applied successfully. %s", resourceTypeName))
	setCondition(
		&cond,
		metav1.ConditionTrue,
		nkpv1alpha1.ReasonUpdateSuccess,
		fmt.Sprintf("%s %s reconciled successfully", resourceTypeName, desiredObj.GetName()),
	)
	return desiredObj, nil
}

func setCondition(cond *metav1.Condition, status metav1.ConditionStatus, reason, message string) {
	cond.Status = status
	cond.Reason = reason
	cond.Message = message
	cond.LastTransitionTime = metav1.Now()
}

func (r *VGPUTokenReconciler) reconcileServiceAccount(
	ctx context.Context,
	vgpuToken *nkpv1alpha1.VGPUToken,
) error {
	_, err := reconcileOwnedResource(
		ctx,
		r,
		vgpuToken,
		nkpv1alpha1.ConditionServiceAccountForDaemonset,
		generator.GenerateServiceAccount,
		&corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ServiceAccount",
			},
		},
	)
	return err
}

func (r *VGPUTokenReconciler) reconcileRole(
	ctx context.Context,
	vgpuToken *nkpv1alpha1.VGPUToken,
) error {
	_, err := reconcileOwnedResource(
		ctx,
		r,
		vgpuToken,
		nkpv1alpha1.ConditionRoleForDaemonset,
		generator.GenerateRole,
		&rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "Role",
			},
		},
	)
	return err
}

func (r *VGPUTokenReconciler) reconcileRoleBinding(
	ctx context.Context,
	vgpuToken *nkpv1alpha1.VGPUToken,
) error {
	_, err := reconcileOwnedResource(
		ctx,
		r,
		vgpuToken,
		nkpv1alpha1.ConditionServiceRoleBindingForDaemonset,
		generator.GenerateRoleBinding,
		&rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "RoleBinding",
			},
		},
	)
	return err
}

func (r *VGPUTokenReconciler) reconcileDaemonSet(
	ctx context.Context,
	vgpuToken *nkpv1alpha1.VGPUToken,
	secretForDaemonSet *corev1.Secret,
) (*appsv1.DaemonSet, error) {
	return reconcileOwnedResource(
		ctx,
		r,
		vgpuToken,
		nkpv1alpha1.ConditionTokenDaemonSet,
		func(_ string) *appsv1.DaemonSet {
			ds := generator.GenerateDaemonSetForVGPUToken(
				vgpuToken,
				secretForDaemonSet,
				r.VGPUTokenPropagatorImage,
			)
			return &ds
		},
		&appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "DaemonSet",
			},
		},
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VGPUTokenReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&nkpv1alpha1.VGPUToken{},
		tokenSecretRefIndex,
		tokenSecretIndexer); err != nil {
		return fmt.Errorf("failed to add indexer for %s: %w", tokenSecretRefIndex, err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&nkpv1alpha1.VGPUToken{}).
		Named("vgputoken").
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.tokenSecretMapper),
		).
		Owns(&appsv1.DaemonSet{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}

func tokenSecretIndexer(obj ctrlclient.Object) []string {
	vgpuToken, ok := obj.(*nkpv1alpha1.VGPUToken)
	if !ok {
		return nil
	}
	refNamespace := vgpuToken.GetNamespace()
	return []string{fmt.Sprintf("%s/%s", refNamespace, vgpuToken.Spec.TokenSecretRef.Name)}
}

func (r *VGPUTokenReconciler) tokenSecretMapper(ctx context.Context, o ctrlclient.Object) []reconcile.Request {
	log := ctrl.Log.WithName("tokenSecretManager")
	secret, ok := o.(*corev1.Secret)
	if !ok {
		log.Error(fmt.Errorf("expected a Secret but got a %T", secret), "failed to get Secret")
		return nil
	}
	vgpuTokens := nkpv1alpha1.VGPUTokenList{}
	err := r.Client.List(ctx, &vgpuTokens, ctrlclient.MatchingFields{
		tokenSecretRefIndex: fmt.Sprintf("%s/%s", secret.GetNamespace(), secret.GetName()),
	})
	if err != nil {
		log.Error(err, "failed to list vgpuTokens")
		return nil
	}
	requests := []reconcile.Request{}
	for i := range vgpuTokens.Items {
		token := vgpuTokens.Items[i]
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: token.GetNamespace(),
				Name:      token.GetName(),
			},
		})
	}
	return requests
}
