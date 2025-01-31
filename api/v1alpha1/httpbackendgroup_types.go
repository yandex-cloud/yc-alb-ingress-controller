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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type HttpBackend struct { // nolint:revive
	Name string `json:"name"`
	// +kubebuilder:default:=1
	Weight              int64                 `json:"weight,omitempty"`
	UseHTTP2            bool                  `json:"useHttp2,omitempty"`
	Service             *ServiceBackend       `json:"service,omitempty"`
	StorageBucket       *StorageBucketBackend `json:"storageBucket,omitempty"`
	TLS                 *BackendTLS           `json:"tls,omitempty"`
	LoadBalancingConfig *LoadBalancingConfig  `json:"loadBalancingConfig,omitempty"`

	// +kubebuilder:validation:Optional
	HealthChecks []*HealthCheck `json:"healthChecks,omitempty"`
}

type LoadBalancingConfig struct {
	//+kubebuilder:default=RANDOM
	BalancerMode string `json:"balancerMode"`
	//+kubebuilder:default=0
	PanicThreshold int64 `json:"panicThreshold"`
	//+kubebuilder:default=0
	LocalityAwareRouting int64 `json:"localityAwareRouting"`
}

// ServiceBackendPort is the service port being referenced. See k8s.io/api/networking/v1/ServiceBackendPort
type ServiceBackendPort struct {
	// Name is the name of the port on the Service.
	// This is a mutually exclusive setting with "Number".
	Name string `json:"name,omitempty"`

	// Number is the numerical port number (e.g. 80) on the Service.
	// This is a mutually exclusive setting with "Name".
	Number int32 `json:"number,omitempty"`
}

type ServiceBackend struct {
	Name string             `json:"name"`
	Port ServiceBackendPort `json:"port"`
}

type StorageBucketBackend struct {
	Name string `json:"name"`
}

type BackendTLS struct {
	// +kubebuilder:validation:Optional
	Sni string `json:"sni"`
	// +kubebuilder:validation:Optional
	TrustedCa string `json:"trustedCa"`
}

type SessionAffinityCookie struct {
	Name string           `json:"name"`
	TTL  *metav1.Duration `json:"ttl,omitempty"`
}

type SessionAffinityConnection struct {
	SourceIP bool `json:"sourceIP"`
}

type SessionAffinityHeader struct {
	HeaderName string `json:"headerName"`
}

type SessionAffinity struct {
	// +kubebuilder:validation:Optional
	Cookie *SessionAffinityCookie `json:"cookie"`
	// +kubebuilder:validation:Optional
	Connection *SessionAffinityConnection `json:"connection"`
	// +kubebuilder:validation:Optional
	Header *SessionAffinityHeader `json:"header"`
}

// HttpBackendGroupSpec defines the desired state of HttpBackendGroup
type HttpBackendGroupSpec struct { // nolint:revive
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	SessionAffinity *SessionAffinity `json:"sessionAffinity"`
	Backends        []*HttpBackend   `json:"backends,omitempty"`
}

// HttpBackendGroupStatus defines the observed state of HttpBackendGroup
type HttpBackendGroupStatus struct { // nolint:revive
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type HttpHealthCheck struct { //nolint:revive
	Path string `json:"path"`
}

type HealthCheck struct {
	// +kubebuilder:validation:Optional
	HTTP *HttpHealthCheck `json:"http"`
	// +kubebuilder:validation:Optional
	GRPC *GrpcHealthCheck `json:"grpc"`

	Port *int64 `json:"port"`

	// Health check timeout.
	//
	// The timeout is the time allowed for the target to respond to a check.
	// If the target doesn't respond in time, the check is considered failed
	// +kubebuilder:validation:Optional
	Timeout *metav1.Duration `json:"timeout"`

	// Base interval between consecutive health checks.
	// +kubebuilder:validation:Optional
	Interval *metav1.Duration `json:"interval"`

	// Number of consecutive successful health checks required to mark an unhealthy target as healthy.
	//
	// Both `0` and `1` values amount to one successful check required.
	//
	// The value is ignored when a load balancer is initialized; a target is marked healthy after one successful check.
	//
	// Default value: `0`.
	// +kubebuilder:validation:Optional
	HealthyThreshold int64 `json:"healthyThreshold"`

	// Number of consecutive failed health checks required to mark a healthy target as unhealthy.
	//
	// Both `0` and `1` values amount to one unsuccessful check required.
	//
	// The value is ignored if a health check is failed due to an HTTP `503 Service Unavailable` response from the target
	// (not applicable to TCP stream health checks). The target is immediately marked unhealthy.
	//
	// Default value: `0`.
	// +kubebuilder:validation:Optional
	UnhealthyThreshold int64 `json:"unhealthyThreshold"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HttpBackendGroup is the Schema for the httpbackendgroups API
type HttpBackendGroup struct { // nolint:revive
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HttpBackendGroupSpec   `json:"spec,omitempty"`
	Status HttpBackendGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HttpBackendGroupList contains a list of HttpBackendGroup
type HttpBackendGroupList struct { // nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HttpBackendGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HttpBackendGroup{}, &HttpBackendGroupList{})
}
