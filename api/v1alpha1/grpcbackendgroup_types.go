package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type GrpcBackend struct { // nolint:revive
	Name string `json:"name"`
	// +kubebuilder:default:=1
	Weight              int64                `json:"weight,omitempty"`
	Service             *ServiceBackend      `json:"service,omitempty"`
	TLS                 *BackendTLS          `json:"tls,omitempty"`
	LoadBalancingConfig *LoadBalancingConfig `json:"loadBalancingConfig,omitempty"`

	// +kubebuilder:validation:Optional
	HealthChecks []*HealthCheck `json:"healthChecks,omitempty"`
}

// GrpcBackendGroupSpec defines the desired state of GrpcBackendGroup
type GrpcBackendGroupSpec struct { // nolint:revive
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	SessionAffinity *SessionAffinity `json:"sessionAffinity"`
	Backends        []*GrpcBackend   `json:"backends,omitempty"`
}

// GrpcBackendGroupStatus defines the observed state of GrpcBackendGroup
type GrpcBackendGroupStatus struct { // nolint:revive
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type GrpcHealthCheck struct {
	ServiceName string `json:"serviceName"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GrpcBackendGroup is the Schema for the grpcbackendgroups API
type GrpcBackendGroup struct { // nolint:revive
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GrpcBackendGroupSpec   `json:"spec,omitempty"`
	Status GrpcBackendGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GrpcBackendGroupList contains a list of GrpcBackendGroup
type GrpcBackendGroupList struct { // nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GrpcBackendGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GrpcBackendGroup{}, &GrpcBackendGroupList{})
}
