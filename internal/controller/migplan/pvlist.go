package migplan

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
	componenthelpers "kubevirt.io/kubevirt-migration-controller/pkg/component-helpers"
	migref "kubevirt.io/kubevirt-migration-controller/pkg/reference"
)

type PvMap map[k8sclient.ObjectKey]corev1.PersistentVolume
type Claims []migrationsv1alpha1.PVC

var (
	suffixMatcher = regexp.MustCompile(`(.*)-mig-([\d|[:alpha:]]{4})$`)
)

const (
	// Identifies associated migplan
	// to allow migplan restored resources rollback
	// The value is Task.PlanResources.MigPlan.UID
	MigPlanLabel       = "migration.openshift.io/migrated-by-migplan"              // (migplan UID)
	MigrationSourceFor = "migration.openshift.io/source-for-directvolumemigration" // (dvm UID)
)

// Update the PVs listed on the plan.
func (r *MigPlanReconciler) updatePvs(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	log := logf.FromContext(ctx)

	if plan.Status.HasAnyCondition(Suspended) {
		plan.Status.StageCondition(PvsDiscovered)
		plan.Status.StageCondition(PvNoSupportedAction)
		plan.Status.StageCondition(PvNoStorageClassSelection)
		plan.Status.StageCondition(PvWarnAccessModeUnavailable)
		plan.Status.StageCondition(PvWarnCopyMethodSnapshot)
		plan.Status.StageCondition(PvLimitExceeded)
		return nil
	}
	if plan.Status.HasAnyCondition(
		InvalidSourceClusterRef,
		SourceClusterNotReady,
		InvalidDestinationClusterRef,
		DestinationClusterNotReady,
		NsNotFoundOnSourceCluster,
		NsLimitExceeded,
		NsListEmpty) {
		return nil
	}

	log.Info("PV Discovery: Starting for Migration Plan",
		"migPlan", path.Join(plan.Namespace, plan.Name),
		"migPlanNamespaces", plan.Spec.Namespaces)

	// Get srcMigCluster
	srcMigCluster, err := componenthelpers.GetSourceCluster(r.Client, plan)
	if err != nil {
		return fmt.Errorf("failed to get source cluster: %w", err)
	}
	if srcMigCluster == nil || !srcMigCluster.Status.IsReady() {
		return nil
	}

	// Get StorageClasses
	srcStorageClasses, err := componenthelpers.GetStorageClasses(r.Client)
	if err != nil {
		return fmt.Errorf("failed to get storage classes: %w", err)
	}
	plan.Status.SrcStorageClasses = srcStorageClasses
	plan.Status.DestStorageClasses = srcStorageClasses

	plan.Spec.BeginPvStaging()
	// Build PV map.
	pvMap, err := r.getPvMap(r.Client, plan)
	if err != nil {
		return fmt.Errorf("failed to get pv map: %w", err)
	}
	claims, err := r.getClaims(r.Client, plan)
	if err != nil {
		return fmt.Errorf("failed to get claims: %w", err)
	}
	for _, claim := range claims {
		key := k8sclient.ObjectKey{
			Namespace: claim.Namespace,
			Name:      claim.Name,
		}
		pv, found := pvMap[key]
		if !found {
			continue
		}
		selection, err := r.getDefaultSelection(ctx, pv, claim, plan, srcStorageClasses, srcStorageClasses)
		if err != nil {
			return fmt.Errorf("failed to get default selection: %w", err)
		}
		plan.Spec.AddPv(
			migrationsv1alpha1.PV{
				Name:         pv.Name,
				Capacity:     pv.Spec.Capacity[corev1.ResourceStorage],
				StorageClass: pv.Spec.StorageClassName,
				Supported: migrationsv1alpha1.Supported{
					Actions: getSupportedActions(pv),
				},
				Selection: selection,
				PVC:       claim,
				NFS:       pv.Spec.NFS,
			})
	}

	// Set the condition to indicate that discovery has been performed.
	plan.Status.SetCondition(migrationsv1alpha1.Condition{
		Type:     PvsDiscovered,
		Status:   True,
		Reason:   IsDone,
		Category: migrationsv1alpha1.Required,
		Message:  "The `persistentVolumes` list has been updated with discovered PVs.",
	})

	plan.Spec.PersistentVolumes.EndPvStaging()

	// Limits
	limit := 100
	count := len(plan.Spec.PersistentVolumes.List)
	if count > limit {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     PvLimitExceeded,
			Status:   True,
			Reason:   LimitExceeded,
			Category: Warn,
			Message:  fmt.Sprintf("PV limit: %d exceeded, found: %d.", limit, count),
		})
	}

	log.Info("PV Discovery: Finished for Migration Plan",
		"migPlan", path.Join(plan.Namespace, plan.Name),
		"migPlanNamespaces", plan.Spec.Namespaces)

	return nil
}

// Gets the default selection values for a PV
func (r *MigPlanReconciler) getDefaultSelection(ctx context.Context,
	pv corev1.PersistentVolume,
	claim migrationsv1alpha1.PVC,
	plan *migrationsv1alpha1.MigPlan,
	srcStorageClasses []migrationsv1alpha1.StorageClass,
	destStorageClasses []migrationsv1alpha1.StorageClass) (migrationsv1alpha1.Selection, error) {

	log := logf.FromContext(ctx)
	selectedStorageClass, err := r.getInitialDestStorageClass(ctx, pv, claim, plan, srcStorageClasses, destStorageClasses)
	if err != nil {
		return migrationsv1alpha1.Selection{}, err
	}
	actions := getSupportedActions(pv)
	selectedAction := ""
	// if there's only one action, make that the default, otherwise select "copy" (if available)
	if len(actions) == 1 {
		selectedAction = actions[0]
	} else {
		for _, a := range actions {
			if a == migrationsv1alpha1.PvCopyAction {
				selectedAction = a
				break
			}
		}
	}

	log.Info("PV Discovery: Setting default selections for discovered PV.",
		"persistentVolume", pv.Name,
		"pvSelectedAction", selectedAction,
		"pvSelectedStorageClass", selectedStorageClass,
	)

	return migrationsv1alpha1.Selection{
		Action:       selectedAction,
		StorageClass: selectedStorageClass,
	}, nil
}

// Determine the supported PV actions.
func getSupportedActions(pv corev1.PersistentVolume) []string {
	supportedActions := []string{}
	supportedActions = append(supportedActions, migrationsv1alpha1.PvSkipAction)
	if pv.Spec.HostPath != nil {
		return supportedActions
	}

	return append(supportedActions, migrationsv1alpha1.PvCopyAction)
}

// Determine the initial value for the destination storage class.
func (r *MigPlanReconciler) getInitialDestStorageClass(ctx context.Context,
	pv corev1.PersistentVolume,
	claim migrationsv1alpha1.PVC,
	plan *migrationsv1alpha1.MigPlan,
	srcStorageClasses []migrationsv1alpha1.StorageClass,
	destStorageClasses []migrationsv1alpha1.StorageClass) (string, error) {

	log := logf.FromContext(ctx)
	srcStorageClassName := pv.Spec.StorageClassName
	srcProvisioner := findProvisionerForName(srcStorageClassName, srcStorageClasses)
	targetProvisioner := ""
	targetStorageClassName := ""

	targetProvisioner = srcProvisioner
	matchingStorageClasses := findStorageClassesForProvisioner(targetProvisioner, destStorageClasses)
	if len(matchingStorageClasses) > 0 {
		if findProvisionerForName(srcStorageClassName, matchingStorageClasses) != "" {
			targetStorageClassName = srcStorageClassName
		} else {
			targetStorageClassName = matchingStorageClasses[0].Name
		}
	} else {
		log.Info("PV Discovery: No matching storage class found for PV.",
			"persistentVolume", pv.Name,
			"pvProvisioner", srcProvisioner)
	}

	return targetStorageClassName, nil
}

// Get a table (map) of PVs keyed by PVC namespaced name.
func (r *MigPlanReconciler) getPvMap(client k8sclient.Client, plan *migrationsv1alpha1.MigPlan) (PvMap, error) {
	pvMap := PvMap{}
	list := corev1.PersistentVolumeList{}
	err := client.List(context.TODO(), &list, &k8sclient.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pv := range list.Items {
		if pv.Status.Phase != corev1.VolumeBound {
			continue
		}
		claim := pv.Spec.ClaimRef
		if migref.RefSet(claim) {
			key := k8sclient.ObjectKey{
				Namespace: claim.Namespace,
				Name:      getMappedNameForPVC(claim, plan),
			}
			pvMap[key] = pv
		}
	}

	return pvMap, nil
}

// Get a list of PVCs found within the specified namespaces.
func (r *MigPlanReconciler) getClaims(client k8sclient.Client, plan *migrationsv1alpha1.MigPlan) (Claims, error) {
	claims := Claims{}
	pvcList := []corev1.PersistentVolumeClaim{}
	for _, namespace := range plan.GetSourceNamespaces() {
		list := &corev1.PersistentVolumeClaimList{}
		err := client.List(context.TODO(), list, k8sclient.InNamespace(namespace))
		if err != nil {
			return nil, fmt.Errorf("failed to get pvc list: %w", err)
		}
		for _, pvc := range list.Items {
			// Skip PVCs that are created by the plan
			if strings.HasSuffix(pvc.Name, "-mig-"+plan.GetSuffix()) {
				continue
			}
			// Skip temporary PVCs owned by other PVCs.
			owner := metav1.GetControllerOf(&pvc)
			if owner != nil && owner.Kind == "PersistentVolumeClaim" {
				continue
			}
			pvcList = append(pvcList, pvc)
		}
	}

	alreadyMigrated := func(pvc corev1.PersistentVolumeClaim) bool {
		if planuid, exists := pvc.Labels[MigPlanLabel]; exists {
			if planuid == string(plan.UID) {
				return true
			}
		}
		return false
	}

	migrationSourceOtherPlan := func(pvc corev1.PersistentVolumeClaim) bool {
		if planuid, exists := pvc.Labels[MigrationSourceFor]; exists {
			if planuid != string(plan.UID) {
				return true
			}
		}
		return false
	}

	for _, pvc := range pvcList {
		if alreadyMigrated(pvc) || migrationSourceOtherPlan(pvc) {
			continue
		}

		pv := plan.Spec.FindPv(migrationsv1alpha1.PV{Name: pvc.Spec.VolumeName})
		volumeMode := corev1.PersistentVolumeFilesystem
		accessModes := pvc.Spec.AccessModes
		if pv == nil {
			if pvc.Spec.VolumeMode != nil {
				volumeMode = *pvc.Spec.VolumeMode
			}
		} else {
			volumeMode = pv.PVC.VolumeMode
			accessModes = pv.PVC.AccessModes
		}
		claims = append(
			claims, migrationsv1alpha1.PVC{
				Namespace: pvc.Namespace,
				Name: getMappedNameForPVC(&corev1.ObjectReference{
					Name:      pvc.Name,
					Namespace: pvc.Namespace,
				}, plan),
				AccessModes: accessModes,
				VolumeMode:  volumeMode,
			})
	}

	return claims, nil
}

// How to determine the new mapped name for a new target PVC.
// If the original PVC doesn't contain -mig-XXXX where XXXX is 4 random letters, then we append -mig-XXXX to the end of the PVC name.
// If the original PVC contains -mig-XXXX, then we replace -mig-XXXX with -mig-YYYY where YYYY is 4 random letters.
// YYYY can be found in the mig plan.
func getMappedNameForPVC(pvcRef *corev1.ObjectReference, plan *migrationsv1alpha1.MigPlan) string {
	pvcName := pvcRef.Name
	existingPVC := plan.Spec.FindPVC(pvcRef.Namespace, pvcRef.Name)
	if existingPVC != nil && (existingPVC.GetSourceName() != existingPVC.GetTargetName()) {
		pvcName = existingPVC.GetTargetName()
	}
	// create a new unique name for the destination PVC
	pvcName = trimSuffix(pvcName)
	// we cannot append a suffix when pvcName itself is 247 characters or more because:
	// 1. total length of new pvc name cannot exceed 253 characters
	// 2. we append '-mig-' char before prefix and pvc names cannot end with '-'
	if len(pvcName) > 247 {
		pvcName = pvcName[:247]
	}
	pvcName = fmt.Sprintf("%s-mig-%s", pvcName, plan.GetSuffix())
	if len(pvcName) > 253 {
		pvcName = pvcName[:253]
	}

	return fmt.Sprintf("%s:%s", pvcRef.Name, pvcName)
}

func trimSuffix(pvcName string) string {
	suffix := "-new"
	if suffixMatcher.MatchString(pvcName) {
		suffixFixCols := suffixMatcher.FindStringSubmatch(pvcName)
		suffix = "-mig-" + suffixFixCols[2]
	}
	return strings.TrimSuffix(pvcName, suffix)
}

func findProvisionerForName(name string, storageClasses []migrationsv1alpha1.StorageClass) string {
	for _, storageClass := range storageClasses {
		if name == storageClass.Name {
			return storageClass.Provisioner
		}
	}

	return ""
}

func findStorageClassesForProvisioner(provisioner string, storageClasses []migrationsv1alpha1.StorageClass) []migrationsv1alpha1.StorageClass {
	var matchingClasses []migrationsv1alpha1.StorageClass

	for _, storageClass := range storageClasses {
		if provisioner == storageClass.Provisioner {
			matchingClasses = append(matchingClasses, storageClass)
		}
	}

	return matchingClasses
}
