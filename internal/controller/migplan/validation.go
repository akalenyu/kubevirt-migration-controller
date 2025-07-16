package migplan

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	kapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	virtv1 "kubevirt.io/api/core/v1"
	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
)

// Types
const (
	Suspended                                  = "Suspended"
	InvalidSourceClusterRef                    = "InvalidSourceClusterRef"
	InvalidDestinationClusterRef               = "InvalidDestinationClusterRef"
	InvalidStorageRef                          = "InvalidStorageRef"
	SourceClusterNotReady                      = "SourceClusterNotReady"
	DestinationClusterNotReady                 = "DestinationClusterNotReady"
	ClusterVersionMismatch                     = "ClusterVersionMismatch"
	SourceClusterNoRegistryPath                = "SourceClusterNoRegistryPath"
	DestinationClusterNoRegistryPath           = "DestinationClusterNoRegistryPath"
	StorageNotReady                            = "StorageNotReady"
	StorageClassConversionUnavailable          = "StorageClassConversionUnavailable"
	NsListEmpty                                = "NamespaceListEmpty"
	InvalidDestinationCluster                  = "InvalidDestinationCluster"
	NsNotFoundOnSourceCluster                  = "NamespaceNotFoundOnSourceCluster"
	NsNotFoundOnDestinationCluster             = "NamespaceNotFoundOnDestinationCluster"
	NsLimitExceeded                            = "NamespaceLimitExceeded"
	NsLengthExceeded                           = "NamespaceLengthExceeded"
	NsNotDNSCompliant                          = "NamespaceNotDNSCompliant"
	NsHaveNodeSelectors                        = "NamespacesHaveNodeSelectors"
	DuplicateNsOnSourceCluster                 = "DuplicateNamespaceOnSourceCluster"
	DuplicateNsOnDestinationCluster            = "DuplicateNamespaceOnDestinationCluster"
	PodLimitExceeded                           = "PodLimitExceeded"
	SourceClusterProxySecretMisconfigured      = "SourceClusterProxySecretMisconfigured"
	DestinationClusterProxySecretMisconfigured = "DestinationClusterProxySecretMisconfigured"
	PlanConflict                               = "PlanConflict"
	PvNameConflict                             = "PvNameConflict"
	PvInvalidAction                            = "PvInvalidAction"
	PvNoSupportedAction                        = "PvNoSupportedAction"
	PvInvalidStorageClass                      = "PvInvalidStorageClass"
	PvInvalidAccessMode                        = "PvInvalidAccessMode"
	PvNoStorageClassSelection                  = "PvNoStorageClassSelection"
	PvWarnAccessModeUnavailable                = "PvWarnAccessModeUnavailable"
	PvInvalidCopyMethod                        = "PvInvalidCopyMethod"
	PvCapacityAdjustmentRequired               = "PvCapacityAdjustmentRequired"
	PvUsageAnalysisFailed                      = "PvUsageAnalysisFailed"
	PvNoCopyMethodSelection                    = "PvNoCopyMethodSelection"
	PvWarnCopyMethodSnapshot                   = "PvWarnCopyMethodSnapshot"
	NfsNotAccessible                           = "NfsNotAccessible"
	NfsAccessCannotBeValidated                 = "NfsAccessCannotBeValidated"
	PvLimitExceeded                            = "PvLimitExceeded"
	StorageEnsured                             = "StorageEnsured"
	RegistriesEnsured                          = "RegistriesEnsured"
	RegistriesHealthy                          = "RegistriesHealthy"
	PvsDiscovered                              = "PvsDiscovered"
	Closed                                     = "Closed"
	SourcePodsNotHealthy                       = "SourcePodsNotHealthy"
	GVKsIncompatible                           = "GVKsIncompatible"
	InvalidHookRef                             = "InvalidHookRef"
	InvalidResourceList                        = "InvalidResourceList"
	HookNotReady                               = "HookNotReady"
	InvalidHookNSName                          = "InvalidHookNSName"
	InvalidHookSAName                          = "InvalidHookSAName"
	HookPhaseUnknown                           = "HookPhaseUnknown"
	HookPhaseDuplicate                         = "HookPhaseDuplicate"
	IntraClusterMigration                      = "IntraClusterMigration"
	KubeVirtNotInstalledSourceCluster          = "KubeVirtNotInstalledSourceCluster"
	KubeVirtVersionNotSupported                = "KubeVirtVersionNotSupported"
	KubeVirtStorageLiveMigrationNotEnabled     = "KubeVirtStorageLiveMigrationNotEnabled"
)

// Categories
const (
	Advisory = migrationsv1alpha1.Advisory
	Critical = migrationsv1alpha1.Critical
	Error    = migrationsv1alpha1.Error
	Warn     = migrationsv1alpha1.Warn
)

// Reasons
const (
	NotSet                 = "NotSet"
	NotFound               = "NotFound"
	KeyNotFound            = "KeyNotFound"
	NotDistinct            = "NotDistinct"
	LimitExceeded          = "LimitExceeded"
	LengthExceeded         = "LengthExceeded"
	NotDNSCompliant        = "NotDNSCompliant"
	NotDone                = "NotDone"
	IsDone                 = "Done"
	Conflict               = "Conflict"
	NotHealthy             = "NotHealthy"
	NodeSelectorsDetected  = "NodeSelectorsDetected"
	DuplicateNs            = "DuplicateNamespaces"
	ConflictingNamespaces  = "ConflictingNamespaces"
	ConflictingPermissions = "ConflictingPermissions"
	NotSupported           = "NotSupported"
)

// Messages
const (
	KubeVirtNotInstalledSourceClusterMessage      = "KubeVirt is not installed on the source cluster"
	KubeVirtVersionNotSupportedMessage            = "KubeVirt version does not support storage live migration, Virtual Machines will be stopped instead"
	KubeVirtStorageLiveMigrationNotEnabledMessage = "KubeVirt storage live migration is not enabled, Virtual Machines will be stopped instead"
)

// Statuses
const (
	True  = migrationsv1alpha1.True
	False = migrationsv1alpha1.False
)

// OpenShift NS annotations
const (
	openShiftMCSAnnotation       = "openshift.io/sa.scc.mcs"
	openShiftSuppGroupAnnotation = "openshift.io/sa.scc.supplemental-groups"
	openShiftUIDRangeAnnotation  = "openshift.io/sa.scc.uid-range"
)

// Valid kubevirt feature gates
const (
	VolumesUpdateStrategy = "VolumesUpdateStrategy"
	VolumeMigrationConfig = "VolumeMigration"
	VMLiveUpdateFeatures  = "VMLiveUpdateFeatures"
	storageProfile        = "auto"
)

// Valid AccessMode values
var validAccessModes = []kapi.PersistentVolumeAccessMode{kapi.ReadWriteOnce, kapi.ReadOnlyMany, kapi.ReadWriteMany, storageProfile}

// Validate the plan resource.
func (r *MigPlanReconciler) validate(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	// // Source cluster
	// err := r.validateSourceCluster(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Destination cluster
	// err = r.validateDestinationCluster(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // validates possible migration options available for this plan
	// err = r.validatePossibleMigrationTypes(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Storage
	// err = r.validateStorage(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Migrated namespaces.
	// err = r.validateNamespaces(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Validates pod properties (e.g. limit of number of active pods, presence of node-selectors)
	// // within each namespace.
	// err = r.validatePodProperties(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Required namespaces.
	// err = r.validateRequiredNamespaces(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// Conflict
	if err := r.validateConflict(ctx, plan); err != nil {
		return fmt.Errorf("error validating conflict: %w", err)
	}

	// // Registry proxy secret
	// err = r.validateRegistryProxySecrets(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Validate health of Pods
	// err = r.validatePodHealth(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Hooks
	// err = r.validateHooks(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Included Resources
	// err = r.validateIncludedResources(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // GVK
	// err = r.compareGVK(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	// // Versions
	// err = r.validateOperatorVersions(ctx, plan)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }

	if err := r.validateLiveMigrationPossible(ctx, plan); err != nil {
		return fmt.Errorf("err checking if live migration is possible: %w", err)
	}

	return nil
}

// Validate the plan does not conflict with another plan.
func (r *MigPlanReconciler) validateConflict(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	planList := migrationsv1alpha1.MigPlanList{}
	err := r.Client.List(ctx, &planList)
	if err != nil {
		return err
	}
	list := []string{}
	for _, p := range planList.Items {
		if plan.UID == p.UID {
			continue
		}
		if plan.HasConflict(&p) {
			list = append(list, p.Name)
		}
	}
	if len(list) > 0 {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     PlanConflict,
			Status:   True,
			Reason:   Conflict,
			Category: Error,
			Message:  "The migration plan is in conflict with [].",
			Items:    list,
		})
	}

	return nil
}

func (r *MigPlanReconciler) validateLiveMigrationPossible(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	// Check if kubevirt is installed, if not installed, return nil
	// srcCluster, err := plan.GetSourceCluster(r)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }
	// if err := r.validateCluster(ctx, srcCluster, plan); err != nil {
	// 	return liberr.Wrap(err)
	// }
	// dstCluster, err := plan.GetDestinationCluster(r)
	// if err != nil {
	// 	return liberr.Wrap(err)
	// }
	// return r.validateCluster(ctx, dstCluster, plan)

	if err := r.validateKubeVirtInstalled(ctx, plan); err != nil {
		return err
	}
	return nil
}

func (r *MigPlanReconciler) validateKubeVirtInstalled(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	log := logf.FromContext(ctx)
	kubevirtList := &virtv1.KubeVirtList{}
	if err := r.Client.List(ctx, kubevirtList); err != nil {
		if meta.IsNoMatchError(err) {
			return nil
		}
		return fmt.Errorf("error listing kubevirts: %w", err)
	}
	if len(kubevirtList.Items) == 0 || len(kubevirtList.Items) > 1 {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     KubeVirtNotInstalledSourceCluster,
			Status:   True,
			Reason:   NotFound,
			Category: Advisory,
			Message:  KubeVirtNotInstalledSourceClusterMessage,
		})
		return nil
	}
	kubevirt := kubevirtList.Items[0]
	operatorVersion := kubevirt.Status.OperatorVersion
	major, minor, bugfix, err := parseKubeVirtOperatorSemver(operatorVersion)
	if err != nil {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     KubeVirtVersionNotSupported,
			Status:   True,
			Reason:   NotSupported,
			Category: Warn,
			Message:  KubeVirtVersionNotSupportedMessage,
		})
		return nil
	}
	log.V(3).Info("KubeVirt operator version", "major", major, "minor", minor, "bugfix", bugfix)
	// Check if kubevirt operator version is at least 1.3.0 if live migration is enabled.
	if major < 1 || (major == 1 && minor < 3) {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     KubeVirtVersionNotSupported,
			Status:   True,
			Reason:   NotSupported,
			Category: Warn,
			Message:  KubeVirtVersionNotSupportedMessage,
		})
		return nil
	}
	// Check if the appropriate feature gates are enabled
	if kubevirt.Spec.Configuration.VMRolloutStrategy == nil ||
		*kubevirt.Spec.Configuration.VMRolloutStrategy != virtv1.VMRolloutStrategyLiveUpdate ||
		kubevirt.Spec.Configuration.DeveloperConfiguration == nil ||
		isStorageLiveMigrationDisabled(&kubevirt, major, minor) {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     KubeVirtStorageLiveMigrationNotEnabled,
			Status:   True,
			Reason:   NotSupported,
			Category: Warn,
			Message:  KubeVirtStorageLiveMigrationNotEnabledMessage,
		})
		return nil
	}
	return nil
}

func parseKubeVirtOperatorSemver(operatorVersion string) (int, int, int, error) {
	// example versions: v1.1.1-106-g0be1a2073, or: v1.3.0-beta.0.202+f8efa57713ba76-dirty
	tokens := strings.Split(operatorVersion, ".")
	if len(tokens) < 3 {
		return -1, -1, -1, fmt.Errorf("version string was not in semver format, != 3 tokens")
	}

	if tokens[0][0] == 'v' {
		tokens[0] = tokens[0][1:]
	}
	major, err := strconv.Atoi(tokens[0])
	if err != nil {
		return -1, -1, -1, fmt.Errorf("major version could not be parsed as integer")
	}

	minor, err := strconv.Atoi(tokens[1])
	if err != nil {
		return -1, -1, -1, fmt.Errorf("minor version could not be parsed as integer")
	}

	bugfixTokens := strings.Split(tokens[2], "-")
	bugfix, err := strconv.Atoi(bugfixTokens[0])
	if err != nil {
		return -1, -1, -1, fmt.Errorf("bugfix version could not be parsed as integer")
	}

	return major, minor, bugfix, nil
}

func isStorageLiveMigrationDisabled(kubevirt *virtv1.KubeVirt, major, minor int) bool {
	if major == 1 && minor >= 5 || major > 1 {
		// Those are all GA from 1.5 and onwards
		// https://github.com/kubevirt/kubevirt/releases/tag/v1.5.0
		return false
	}

	return !slices.Contains(kubevirt.Spec.Configuration.DeveloperConfiguration.FeatureGates, VolumesUpdateStrategy) ||
		!slices.Contains(kubevirt.Spec.Configuration.DeveloperConfiguration.FeatureGates, VolumeMigrationConfig) ||
		!slices.Contains(kubevirt.Spec.Configuration.DeveloperConfiguration.FeatureGates, VMLiveUpdateFeatures)
}
