/*
Copyright 2024.

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

package v1

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceMonitorConfigSpec defines the desired state of ServiceMonitorConfig
type ServiceMonitorConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NameSpaceSpec monitoringv1.NamespaceSelector `json:"namespaceSpec,omitempty"`
}

// ServiceMonitorConfigStatus defines the observed state of ServiceMonitorConfig
type ServiceMonitorConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceMonitorConfig is the Schema for the servicemonitorconfigs API
type ServiceMonitorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceMonitorConfigSpec   `json:"spec,omitempty"`
	Status ServiceMonitorConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceMonitorConfigList contains a list of ServiceMonitorConfig
type ServiceMonitorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceMonitorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceMonitorConfig{}, &ServiceMonitorConfigList{})
}
