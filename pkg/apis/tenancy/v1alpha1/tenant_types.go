/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=tenants,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

type TenantSpec struct {
}

type TenantStatus struct {
	// Phase represents the current phase of Tenant.
	// E.g. Pending, Running, Terminating, Failed etc.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions defines current service state of Tenant.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (t *TenantStatus) IsPhase(p TenantPhase) bool {
	return t.Phase == string(p)
}

func (t *TenantStatus) SetPhase(p TenantPhase) {
	t.Phase = string(p)
}

func (t *Tenant) ClusterNamespaceInHost() string {
	return "tenant-" + t.Name
}

func (t *Tenant) GetConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *Tenant) SetConditions(conditions []metav1.Condition) {
	t.Status.Conditions = conditions
}
