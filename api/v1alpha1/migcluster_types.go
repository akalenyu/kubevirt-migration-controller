/*
Copyright 2025 The KubeVirt Authors.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MigClusterSpec defines the desired state of MigCluster
type MigClusterSpec struct {
	// Specifies if the cluster is host (where the controller is installed) or not. This is a required field.
	IsHostCluster bool `json:"isHostCluster"`

	// Stores the url of the remote cluster. The field is only required for the source cluster object.
	URL string `json:"url,omitempty"`

	ServiceAccountSecretRef *corev1.ObjectReference `json:"serviceAccountSecretRef,omitempty"`

	// If the migcluster needs SSL verification for connections a user can supply a custom CA bundle. This field is required only when spec.Insecure is set false
	CABundle []byte `json:"caBundle,omitempty"`

	// For azure clusters -- it's the resource group that in-cluster volumes use.
	AzureResourceGroup string `json:"azureResourceGroup,omitempty"`

	// If set false, user will need to provide CA bundle for TLS connection to the remote cluster.
	Insecure bool `json:"insecure,omitempty"`

	// An override setting to tell the controller that the source cluster restic needs to be restarted after stage pod creation.
	RestartRestic *bool `json:"restartRestic,omitempty"`

	// If set True, forces the controller to run a full suite of validations on migcluster.
	Refresh bool `json:"refresh,omitempty"`

	// Stores the path of registry route when using direct migration.
	ExposedRegistryPath string `json:"exposedRegistryPath,omitempty"`
}

// MigClusterStatus defines the observed state of MigCluster
type MigClusterStatus struct {
	Conditions      `json:","`
	ObservedDigest  string `json:"observedDigest,omitempty"`
	RegistryPath    string `json:"registryPath,omitempty"`
	OperatorVersion string `json:"operatorVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MigCluster is the Schema for the migclusters API.
type MigCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigClusterSpec   `json:"spec,omitempty"`
	Status MigClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MigClusterList contains a list of MigCluster.
type MigClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MigCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MigCluster{}, &MigClusterList{})
}
