// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nkpv1alpha1 "github.com/nutanix-cloud-native/vgpu-token-operator/api/v1alpha1"
)

//nolint:gochecknoinits //needed for test initialization
func init() {
	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	// use klog as the internal logger for this envtest environment.
	log.SetLogger(logger)
	// additionally force all of the controllers to use the Ginkgo logger.
	ctrl.SetLogger(logger)
	// add logger for ginkgo
	klog.SetOutput(GinkgoWriter)

	utilruntime.Must(scheme.AddToScheme(testScheme))
	utilruntime.Must(nkpv1alpha1.AddToScheme(testScheme))

	// Create the test environment.
	env = &envtest.Environment{
		CRDInstallOptions: envtest.CRDInstallOptions{
			ErrorIfPathMissing: true,
			Paths: []string{
				filepath.Join("..", "..", "charts", "vgpu-token-operator", "templates", "crd"),
			},
		},
	}
}

var (
	env        *envtest.Environment
	testScheme = scheme.Scheme
)

var cacheSyncBackoff = wait.Backoff{
	Duration: 100 * time.Millisecond,
	Factor:   1.5,
	Steps:    8,
	Jitter:   0.4,
}

// TestEnvironment encapsulates a Kubernetes local test environment.
type TestEnvironment struct {
	manager.Manager
	client.Client
	Config    *rest.Config
	WithWatch client.WithWatch

	//nolint:containedctx // test context used to make things easier
	doneMgr      context.Context
	directClient client.Client
	cancel       context.CancelFunc
}

// NewTestEnvironment creates a new environment spinning up a local api-server.
//
// This function should be called only once for each package you're running tests within,
// usually the environment is initialized in a suite_test.go file within a `BeforeSuite` ginkgo block.
func NewTestEnvironment() *TestEnvironment {
	if _, err := env.Start(); err != nil {
		fmt.Println("failed with err", err)
		panic(err)
	}

	options := manager.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
			CertDir:     env.WebhookInstallOptions.LocalServingCertDir,
		},
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			options.Cache = &client.CacheOptions{
				DisableFor:   []client.Object{&corev1.ConfigMap{}, &corev1.Secret{}},
				Unstructured: false,
			}
			return client.New(config, options)
		},
	}

	mgr, err := ctrl.NewManager(env.Config, options)
	if err != nil {
		klog.Fatalf("unable to create mgr: %+v", err)
	}

	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: scheme.Scheme,
		Mapper: mgr.GetRESTMapper(),
	})
	if err != nil {
		klog.Fatalf("Failed to start testenv manager: %v", err)
	}

	directClient, err := client.New(mgr.GetConfig(),
		client.Options{
			Scheme: mgr.GetScheme(),
			Mapper: mgr.GetRESTMapper(),
		})
	if err != nil {
		klog.Fatalf("Failed to create direct client: %v", err)
	}
	doneCtx, cancel := context.WithCancel(context.Background())
	return &TestEnvironment{
		Manager:      mgr,
		Client:       mgr.GetClient(),
		Config:       mgr.GetConfig(),
		doneMgr:      doneCtx,
		cancel:       cancel,
		WithWatch:    withWatch,
		directClient: directClient,
	}
}

func (t *TestEnvironment) StartManager() error {
	return t.Manager.Start(t.doneMgr)
}

// GetDirectClient returns a client that speaks directly to the API server, without using a cache. Tests should use this
// client to create objects and make assertions. They should NOT use the client returned by TestEnvironment.GetClient(),
// which uses a cache shared with the controllers under test. For more information, see
// https://github.com/kubernetes-sigs/controller-runtime/issues/343#issuecomment-469435686.
func (t *TestEnvironment) GetDirectClient() client.Client {
	return t.directClient
}

func (t *TestEnvironment) Stop() error {
	t.cancel()
	return env.Stop()
}

func (t *TestEnvironment) Cleanup(ctx context.Context, objs ...client.Object) error {
	errs := []error{}
	for _, o := range objs {
		err := t.Client.Delete(ctx, o)
		if apierrors.IsNotFound(err) {
			// If the object is not found, it must've been garbage collected
			// already. For example, if we delete namespace first and then
			// objects within it.
			continue
		}
		errs = append(errs, err)
	}
	return kerrors.NewAggregate(errs)
}

// CreateObj wraps around client.Create and creates the object.
func (t *TestEnvironment) CreateObj(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return t.Client.Create(ctx, obj, opts...)
}

// CreateAndWait creates the given object and waits for the cache to be updated accordingly.
//
// NOTE: Waiting for the cache to be updated helps in preventing test flakes due to the cache sync delays.
func (t *TestEnvironment) CreateAndWait(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if err := t.Client.Create(ctx, obj, opts...); err != nil {
		return err
	}

	// Makes sure the cache is updated with the new object
	objCopy := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(obj)
	if err := wait.ExponentialBackoff(
		cacheSyncBackoff,
		func() (done bool, err error) {
			if err := t.Get(ctx, key, objCopy); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err != nil {
		return errors.Wrapf(
			err,
			"object %s, %s is not being added to the testenv client cache",
			obj.GetObjectKind().GroupVersionKind().String(), key,
		)
	}
	return nil
}

func (t *TestEnvironment) CreateNamespace(ctx context.Context, generateName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", generateName),
			Labels: map[string]string{
				"testenv/original-name": generateName,
			},
		},
	}
	if err := t.Client.Create(ctx, ns); err != nil {
		return nil, err
	}
	return ns, nil
}
