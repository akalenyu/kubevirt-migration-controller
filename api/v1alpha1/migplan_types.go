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
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MigPlanSpec defines the desired state of MigPlan
type MigPlanSpec struct {

	// Holds all the persistent volumes found for the namespaces included in migplan. Each entry is a persistent volume with the information. Name - The PV name. Capacity - The PV storage capacity. StorageClass - The PV storage class name. Supported - Lists of what is supported. Selection - Choices made from supported. PVC - Associated PVC. NFS - NFS properties. staged - A PV has been explicitly added/updated.
	PersistentVolumes `json:",inline"`

	// Holds names of all the namespaces to be included in migration.
	Namespaces []string `json:"namespaces,omitempty"`

	SrcMigClusterRef *corev1.ObjectReference `json:"srcMigClusterRef,omitempty"`

	DestMigClusterRef *corev1.ObjectReference `json:"destMigClusterRef,omitempty"`

	MigStorageRef *corev1.ObjectReference `json:"migStorageRef,omitempty"`

	// LabelSelector optional label selector on the included resources in Velero Backup
	// +kubebuilder:validation:Optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// MigPlanStatus defines the observed state of MigPlan
type MigPlanStatus struct {
	UnhealthyResources `json:",inline"`
	Conditions         `json:",inline"`
	Incompatible       `json:",inline"`
	Suffix             *string        `json:"suffix,omitempty"`
	ObservedDigest     string         `json:"observedDigest,omitempty"`
	ExcludedResources  []string       `json:"excludedResources,omitempty"`
	SrcStorageClasses  []StorageClass `json:"srcStorageClasses,omitempty"`
	DestStorageClasses []StorageClass `json:"destStorageClasses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MigPlan is the Schema for the migplans API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.srcMigClusterRef.name"
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.destMigClusterRef.name"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=".spec.migStorageRef.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MigPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigPlanSpec   `json:"spec,omitempty"`
	Status MigPlanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MigPlanList contains a list of MigPlan.
type MigPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MigPlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MigPlan{}, &MigPlanList{})
}

// GetSourceNamespaces get source namespaces without mapping
func (r *MigPlan) GetSourceNamespaces() []string {
	includedNamespaces := []string{}
	for _, namespace := range r.Spec.Namespaces {
		namespace = strings.Split(namespace, ":")[0]
		includedNamespaces = append(includedNamespaces, namespace)
	}

	return includedNamespaces
}

// GetDestinationNamespaces get destination namespaces without mapping
func (r *MigPlan) GetDestinationNamespaces() []string {
	includedNamespaces := []string{}
	for _, namespace := range r.Spec.Namespaces {
		namespaces := strings.Split(namespace, ":")
		if len(namespaces) > 1 {
			includedNamespaces = append(includedNamespaces, namespaces[1])
		} else {
			includedNamespaces = append(includedNamespaces, namespaces[0])
		}
	}

	return includedNamespaces
}

// GetNamespaceMapping gets a map of src to dest namespaces
func (r *MigPlan) GetNamespaceMapping() map[string]string {
	nsMapping := make(map[string]string)
	for _, namespace := range r.Spec.Namespaces {
		namespaces := strings.Split(namespace, ":")
		if len(namespaces) > 1 {
			nsMapping[namespaces[0]] = namespaces[1]
		} else {
			nsMapping[namespaces[0]] = namespaces[0]
		}
	}

	return nsMapping
}

func (r *MigPlan) GetSuffix() string {
	if r.Status.Suffix != nil {
		return *r.Status.Suffix
	}
	return StorageConversionPVCNamePrefix
}

// Get whether the plan conflicts with another.
// Plans conflict when:
//   - Have any of the clusters in common AND
//   - Have any of the namespaces in common
func (r *MigPlan) HasConflict(plan *MigPlan) bool {
	if !refEquals(r.Spec.SrcMigClusterRef, plan.Spec.SrcMigClusterRef) &&
		!refEquals(r.Spec.DestMigClusterRef, plan.Spec.DestMigClusterRef) &&
		!refEquals(r.Spec.SrcMigClusterRef, plan.Spec.DestMigClusterRef) &&
		!refEquals(r.Spec.DestMigClusterRef, plan.Spec.SrcMigClusterRef) {
		return false
	}
	nsMap := map[string]bool{}
	for _, name := range plan.Spec.Namespaces {
		nsMap[name] = true
	}
	for _, name := range r.Spec.Namespaces {
		if _, foundNs := nsMap[name]; foundNs {
			return true
		}
	}

	return false
}

func refEquals(refA, refB *corev1.ObjectReference) bool {
	return refA.Namespace == refB.Namespace && refA.Name == refB.Name
}

// Collection of PVs
// List - The collection of PVs.
// index - List index.
// --------
// Example:
// plan.Spec.BeginPvStaging()
// plan.Spec.AddPv(pvA)
// plan.Spec.AddPv(pvB)
// plan.Spec.AddPv(pvC)
// plan.Spec.EndPvStaging()
type PersistentVolumes struct {
	List    []PV           `json:"persistentVolumes,omitempty"`
	index   map[string]int `json:"-"`
	staging bool           `json:"-"`
}

// Allocate collections.
func (r *PersistentVolumes) init() {
	if r.List == nil {
		r.List = []PV{}
	}
	if r.index == nil {
		r.buildIndex()
	}
}

// Build the index.
func (r *PersistentVolumes) buildIndex() {
	r.index = map[string]int{}
	for i, pv := range r.List {
		r.index[pv.Name] = i
	}
}

// Begin staging PVs.
func (r *PersistentVolumes) BeginPvStaging() {
	r.init()
	r.staging = true
	for i := range r.List {
		pv := &r.List[i]
		pv.staged = false
	}
}

// End staging PVs and delete un-staged PVs from the list.
// The PVs are sorted by Name.
func (r *PersistentVolumes) EndPvStaging() {
	r.init()
	r.staging = false
	kept := []PV{}
	for _, pv := range r.List {
		if pv.staged {
			kept = append(kept, pv)
		}
	}
	sort.Slice(
		kept,
		func(i, j int) bool {
			return kept[i].Name < kept[j].Name
		})
	r.List = kept
	r.buildIndex()
}

// Find a PV
func (r *PersistentVolumes) FindPv(pv PV) *PV {
	r.init()
	i, found := r.index[pv.Name]
	if found {
		return &r.List[i]
	}

	return nil
}

// Find a PVC
func (r *PersistentVolumes) FindPVC(namespace string, name string) *PVC {
	for i := range r.List {
		pv := &r.List[i]
		if pv.PVC.GetSourceName() == name && pv.PVC.Namespace == namespace {
			return &pv.PVC
		}
	}
	return nil
}

// Add (or update) Pv to the collection.
func (r *PersistentVolumes) AddPv(pv PV) {
	r.init()
	pv.staged = true
	foundPv := r.FindPv(pv)
	if foundPv == nil {
		r.List = append(r.List, pv)
		r.index[pv.Name] = len(r.List) - 1
	} else {
		foundPv.Update(pv)
	}
}

// Delete a PV from the collection.
func (r *PersistentVolumes) DeletePv(names ...string) {
	r.init()
	for _, name := range names {
		i, found := r.index[name]
		if found {
			r.List = append(r.List[:i], r.List[i+1:]...)
			r.buildIndex()
		}
	}
}

// Reset PVs collection.
func (r *PersistentVolumes) ResetPvs() {
	r.init()
	r.List = []PV{}
	r.buildIndex()
}

// Name - The PV name.
// Capacity - The PV storage capacity.
// StorageClass - The PV storage class name.
// Supported - Lists of what is supported.
// Selection - Choices made from supported.
// PVC - Associated PVC.
// NFS - NFS properties.
// staged - A PV has been explicitly added/updated.
type PV struct {
	Name              string                  `json:"name,omitempty"`
	Capacity          resource.Quantity       `json:"capacity,omitempty"`
	StorageClass      string                  `json:"storageClass,omitempty"`
	Supported         Supported               `json:"supported"`
	Selection         Selection               `json:"selection"`
	PVC               PVC                     `json:"pvc,omitempty"`
	NFS               *corev1.NFSVolumeSource `json:"-"`
	staged            bool                    `json:"-"`
	CapacityConfirmed bool                    `json:"capacityConfirmed,omitempty"`
	ProposedCapacity  resource.Quantity       `json:"proposedCapacity,omitempty"`
}

// TODO: Candidate for dropping since in a VM only world we don't have any other owner types
type OwnerType string

const (
	VirtualMachine   OwnerType = "VirtualMachine"
	Deployment       OwnerType = "Deployment"
	DeploymentConfig OwnerType = "DeploymentConfig"
	StatefulSet      OwnerType = "StatefulSet"
	ReplicaSet       OwnerType = "ReplicaSet"
	DaemonSet        OwnerType = "DaemonSet"
	Job              OwnerType = "Job"
	CronJob          OwnerType = "CronJob"
	Unknown          OwnerType = "Unknown"
)

const (
	StorageConversionPVCNamePrefix = "new"
	TouchAnnotation                = "touch"
)

// PV Actions.
const (
	PvMoveAction = "move"
	PvCopyAction = "copy"
	PvSkipAction = "skip"
)

// PV Copy Methods.
const (
	PvBlockCopyMethod = "block"
)

// PVC
type PVC struct {
	Namespace    string                              `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	Name         string                              `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	AccessModes  []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty" protobuf:"bytes,1,rep,name=accessModes,casttype=PersistentVolumeAccessMode"`
	VolumeMode   corev1.PersistentVolumeMode         `json:"volumeMode,omitempty"`
	HasReference bool                                `json:"hasReference,omitempty"`
	OwnerType    OwnerType                           `json:"ownerType,omitempty"`
}

// GetTargetName returns name of the target PVC
func (p PVC) GetTargetName() string {
	names := strings.Split(p.Name, ":")
	if len(names) > 1 {
		return names[1]
	}
	if len(names) > 0 {
		return names[0]
	}
	return p.Name
}

// GetSourceName returns name of the source PVC
func (p PVC) GetSourceName() string {
	names := strings.Split(p.Name, ":")
	if len(names) > 0 {
		return names[0]
	}
	return p.Name
}

// Supported
// Actions     - The list of supported actions
type Supported struct {
	Actions []string `json:"actions"`
}

// Selection
// Action - The PV migration action (move|copy|skip)
// StorageClass - The PV storage class name to use in the destination cluster.
// AccessMode   - The PV access mode to use in the destination cluster, if different from src PVC AccessMode
type Selection struct {
	Action       string                            `json:"action,omitempty"`
	StorageClass string                            `json:"storageClass,omitempty"`
	AccessMode   corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty" protobuf:"bytes,1,rep,name=accessMode,casttype=PersistentVolumeAccessMode"`
}

// Update the PV with another.
func (r *PV) Update(pv PV) {
	r.StorageClass = pv.StorageClass
	r.Supported.Actions = pv.Supported.Actions
	r.Capacity = pv.Capacity
	r.PVC = pv.PVC
	r.NFS = pv.NFS
	if len(r.Supported.Actions) == 1 {
		r.Selection.Action = r.Supported.Actions[0]
	}
	r.staged = true
}

// StorageClass is an available storage class in the cluster
// Name - the storage class name
// Provisioner - the dynamic provisioner for the storage class
// Default - whether or not this storage class is the default
// AccessModes - access modes supported by the dynamic provisioner
type StorageClass struct {
	Name              string                 `json:"name,omitempty"`
	Provisioner       string                 `json:"provisioner,omitempty"`
	Default           bool                   `json:"default,omitempty"`
	VolumeAccessModes []VolumeModeAccessMode `json:"volumeAccessModes,omitempty"`
}

type VolumeModeAccessMode struct {
	VolumeMode  corev1.PersistentVolumeMode         `json:"volumeMode,omitempty"`
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty" protobuf:"bytes,1,rep,name=accessModes,casttype=PersistentVolumeAccessMode"`
}
