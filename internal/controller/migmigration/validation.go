package migmigration

import (
	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
)

// Statuses
const (
	True  = migrationsv1alpha1.True
	False = migrationsv1alpha1.False
)

// Categories
const (
	Critical = migrationsv1alpha1.Critical
	Advisory = migrationsv1alpha1.Advisory
)

// Reasons
const (
	NotSet         = "NotSet"
	NotFound       = "NotFound"
	Cancel         = "Cancel"
	ErrorsDetected = "ErrorsDetected"
	NotSupported   = "NotSupported"
	PvNameConflict = "PvNameConflict"
	NotDistinct    = "NotDistinct"
)

// Types
const (
	UnhealthyNamespaces                = "UnhealthyNamespaces"
	InvalidPlanRef                     = "InvalidPlanRef"
	PlanNotReady                       = "PlanNotReady"
	PlanClosed                         = "PlanClosed"
	HasFinalMigration                  = "HasFinalMigration"
	Postponed                          = "Postponed"
	SucceededWithWarnings              = "SucceededWithWarnings"
	ResticErrors                       = "ResticErrors"
	ResticVerifyErrors                 = "ResticVerifyErrors"
	VeleroInitialBackupPartiallyFailed = "VeleroInitialBackupPartiallyFailed"
	VeleroStageBackupPartiallyFailed   = "VeleroStageBackupPartiallyFailed"
	VeleroStageRestorePartiallyFailed  = "VeleroStageRestorePartiallyFailed"
	VeleroFinalRestorePartiallyFailed  = "VeleroFinalRestorePartiallyFailed"
	DirectImageMigrationFailed         = "DirectImageMigrationFailed"
	StageNoOp                          = "StageNoOp"
	RegistriesHealthy                  = "RegistriesHealthy"
	RegistriesUnhealthy                = "RegistriesUnhealthy"
	StaleSrcVeleroCRsDeleted           = "StaleSrcVeleroCRsDeleted"
	StaleDestVeleroCRsDeleted          = "StaleDestVeleroCRsDeleted"
	StaleResticCRsDeleted              = "StaleResticCRsDeleted"
	DirectVolumeMigrationBlocked       = "DirectVolumeMigrationBlocked"
	InvalidSpec                        = "InvalidSpec"
	ConflictingPVCMappings             = "ConflictingPVCMappings"
)
