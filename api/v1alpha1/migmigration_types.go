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

// Cache Indexes.
const (
	PlanIndexField = "MigPlanRef"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MigMigrationSpec defines the desired state of MigMigration
type MigMigrationSpec struct {
	MigPlanRef *corev1.ObjectReference `json:"migPlanRef,omitempty"`

	// Invokes the stage operation, when set to true the migration controller switches to stage itinerary. This is a required field.
	Stage bool `json:"stage"`

	// Invokes the state migration operation
	MigrateState bool `json:"migrateState,omitempty"`

	// Specifies whether to quiesce the application Pods before migrating Persistent Volume data.
	QuiescePods bool `json:"quiescePods,omitempty"`

	// Specifies whether to retain the annotations set by the migration controller or not.
	KeepAnnotations bool `json:"keepAnnotations,omitempty"`

	// Specifies whether to verify the health of the migrated pods or not.
	Verify bool `json:"verify,omitempty"`

	// Invokes the cancel migration operation, when set to true the migration controller switches to cancel itinerary. This field can be used on-demand to cancel the running migration.
	Canceled bool `json:"canceled,omitempty"`

	// Invokes the rollback migration operation, when set to true the migration controller switches to rollback itinerary. This field needs to be set prior to creation of a MigMigration.
	Rollback bool `json:"rollback,omitempty"`

	// If set True, run rsync operations with escalated privileged, takes precedence over setting RunAsUser and RunAsGroup
	RunAsRoot *bool `json:"runAsRoot,omitempty"`

	// If set, runs rsync operations with provided user id. This provided user id should be a valid one that falls within the range of allowed UID of user namespace
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// If set, runs rsync operations with provided group id. This provided user id should be a valid one that falls within the range of allowed GID of user namespace
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`
}

// MigMigrationStatus defines the observed state of MigMigration
type MigMigrationStatus struct {
	Conditions         `json:",inline"`
	UnhealthyResources `json:",inline"`
	ObservedDigest     string       `json:"observedDigest,omitempty"`
	StartTimestamp     *metav1.Time `json:"startTimestamp,omitempty"`
	Phase              string       `json:"phase,omitempty"`
	Pipeline           []*Step      `json:"pipeline,omitempty"`
	Itinerary          string       `json:"itinerary,omitempty"`
	Errors             []string     `json:"errors,omitempty"`
}

// Step defines a task in a step of migration
type Step struct {
	Timed `json:",inline"`

	Name     string   `json:"name"`
	Phase    string   `json:"phase,omitempty"`
	Message  string   `json:"message,omitempty"`
	Progress []string `json:"progress,omitempty"`
	Failed   bool     `json:"failed,omitempty"`
	Skipped  bool     `json:"skipped,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MigMigration is the Schema for the migmigrations API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Plan",type=string,JSONPath=".spec.migPlanRef.name"
// +kubebuilder:printcolumn:name="Stage",type=string,JSONPath=".spec.stage"
// +kubebuilder:printcolumn:name="Rollback",type=string,JSONPath=".spec.rollback"
// +kubebuilder:printcolumn:name="Itinerary",type=string,JSONPath=".status.itinerary"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MigMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigMigrationSpec   `json:"spec,omitempty"`
	Status MigMigrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MigMigrationList contains a list of MigMigration.
type MigMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MigMigration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MigMigration{}, &MigMigrationList{})
}
