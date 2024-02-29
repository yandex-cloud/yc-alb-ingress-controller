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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogDiscardRule discards a fraction of logs with certain codes.
// If neither codes nor intervals are provided, rule applies to all logs.
type LogDiscardRule struct {
	// HTTP codes that should be discarded.
	// +kubebuilder:validation:Optional
	HTTPCodes []int64 `json:"httpCodes"`

	// Groups of HTTP codes like 4xx that should be discarded.
	// +kubebuilder:validation:Optional
	HTTPCodeIntervals []string `json:"httpCodeIntervals"`

	// GRPC codes that should be discarded
	// +kubebuilder:validation:Optional
	GRPCCodes []string `json:"grpcCodes"`

	// Percent of logs to be discarded: 0 - keep all, 100 or unset - discard all
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=0
	DiscardPercent *int64 `json:"discardPercent"`
}

type LogOptions struct {
	// Cloud Logging log group ID to store access logs.
	// If not set then logs will be stored in default log group for the folder
	// where load balancer located.
	// +kubebuilder:validation:Optional
	LogGroupID string `json:"logGroupID"`

	// Ordered list of rules, first matching rule applies
	// +kubebuilder:validation:Optional
	DiscardRules []*LogDiscardRule `json:"discardRules"`

	// Do not send logs to Cloud Logging log group.
	// +kubebuilder:validation:Optional
	Disable bool `json:"disable"`
}

// +kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

type IngressGroupSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	LogOptions *LogOptions `json:"logOptions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// IngressGroupSettingsList contains a list of IngressGroupSettings
type IngressGroupSettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IngressGroupSettings `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressGroupSettings{}, &IngressGroupSettingsList{})
}
