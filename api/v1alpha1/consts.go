// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	ConditionPropagated string = "Propagated" // Global propagation state

	ReasonAllReady          string = "AllReady"
	ReasonPartialReady      string = "PartialReady"
	ReasonNoneReady         string = "NoneReady"
	ReasonNoneDesired       string = "NoneDesired"
	ReasonDaemonSetUpdating string = "DaemonsetUpdating"
)

const (
	ConditionTokenSecret string = "TokenSecret"

	ReasonTokenSecretNotFound  string = "NotFound"
	ReasonTokenSecretFailed    string = "Failed"
	ReasonTokenSecretRetrieved string = "Retrieved"
)

const (
	ReasonCreateFailed  string = "CreateFailed"
	ReasonCreateSuccess string = "CreateSuccess"
	ReasonGetFailed     string = "GetFailed"
	ReasonUpdateSuccess string = "UpdateSuccess"
	ReasonUpdateFailed  string = "UpdateFailed"
	ReasonUptoDate      string = "ObjectUpToDate"
)

const (
	ConditionTokenDaemonSet string = "TokenDaemonset"
)

const (
	ConditionRoleForDaemonset string = "TokenDaemonsetRole"
)

const (
	ConditionServiceAccountForDaemonset string = "TokenDaemonsetServiceAccount"
)

const (
	ConditionServiceRoleBindingForDaemonset string = "TokenDaemonsetRoleBinding"
)
