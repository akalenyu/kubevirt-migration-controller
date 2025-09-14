package migmigration

// Itinerary names
const (
	StageItineraryName    = "Stage"
	FinalItineraryName    = "Final"
	CancelItineraryName   = "Cancel"
	FailedItineraryName   = "Failed"
	RollbackItineraryName = "Rollback"
)

// Itinerary defines itinerary
type Itinerary struct {
	Name   string
	Phases []Phase
}

var StageItinerary = Itinerary{
	Name: StageItineraryName,
	Phases: []Phase{
		{Name: Created, Step: StepPrepare},
		{Name: Started, Step: StepPrepare},
		{Name: CleanStaleAnnotations, Step: StepPrepare},
		{Name: CleanStaleStagePods, Step: StepPrepare},
		{Name: WaitForStaleStagePodsTerminated, Step: StepPrepare},
		{Name: QuiesceSourceApplications, Step: StepStageBackup, all: Quiesce},
		{Name: EnsureSrcQuiesced, Step: StepStageBackup, all: Quiesce},
		{Name: CreateDirectVolumeMigrationStage, Step: StepStageBackup, all: DirectVolume | EnableVolume},
		{Name: WaitForDirectVolumeMigrationToComplete, Step: StepDirectVolume, all: DirectVolume | EnableVolume},
		{Name: SwapPVCReferences, Step: StepCleanup, all: StorageConversion | Quiesce},
		{Name: Completed, Step: StepCleanup},
	},
}

var FinalItinerary = Itinerary{
	Name: FinalItineraryName,
	Phases: []Phase{
		{Name: Created, Step: StepPrepare},
		{Name: Started, Step: StepPrepare},
		{Name: StartRefresh, Step: StepPrepare},
		{Name: WaitForRefresh, Step: StepPrepare},
		{Name: CleanStaleAnnotations, Step: StepPrepare},
		{Name: CleanStaleResticCRs, Step: StepPrepare},
		{Name: CleanStaleVeleroCRs, Step: StepPrepare},
		{Name: RestartVelero, Step: StepPrepare},
		{Name: CleanStaleStagePods, Step: StepPrepare},
		{Name: WaitForStaleStagePodsTerminated, Step: StepPrepare},
		{Name: CreateRegistries, Step: StepPrepare, all: IndirectImage | EnableImage | HasISs},
		{Name: WaitForVeleroReady, Step: StepPrepare},
		{Name: WaitForRegistriesReady, Step: StepPrepare, all: IndirectImage | EnableImage | HasISs},
		{Name: EnsureCloudSecretPropagated, Step: StepPrepare},
		{Name: CreateDirectImageMigration, Step: StepBackup, all: DirectImage | EnableImage},
		{Name: CreateDirectVolumeMigrationStage, Step: StepStageBackup, all: DirectVolume | EnableVolume},
		{Name: EnsureInitialBackup, Step: StepBackup},
		{Name: InitialBackupCreated, Step: StepBackup},
		{Name: QuiesceSourceApplications, Step: StepStageBackup, all: Quiesce},
		{Name: EnsureSrcQuiesced, Step: StepStageBackup, all: Quiesce},
		{Name: EnsureStagePodsFromRunning, Step: StepStageBackup, all: HasPVs | IndirectVolume},
		{Name: EnsureStagePodsFromTemplates, Step: StepStageBackup, all: HasPVs | IndirectVolume},
		{Name: EnsureStagePodsFromOrphanedPVCs, Step: StepStageBackup, all: HasPVs | IndirectVolume},
		{Name: StagePodsCreated, Step: StepStageBackup, all: HasStagePods},
		{Name: RestartRestic, Step: StepStageBackup, all: HasStagePods},
		{Name: AnnotateResources, Step: StepStageBackup, all: HasStageBackup},
		{Name: WaitForResticReady, Step: StepStageBackup, anyf: HasPVs | HasStagePods},
		{Name: CreateDirectVolumeMigrationFinal, Step: StepStageBackup, all: DirectVolume | EnableVolume},
		{Name: EnsureStageBackup, Step: StepStageBackup, all: HasStageBackup},
		{Name: StageBackupCreated, Step: StepStageBackup, all: HasStageBackup},
		{Name: EnsureStageBackupReplicated, Step: StepStageBackup, all: HasStageBackup},
		{Name: EnsureStageRestore, Step: StepStageRestore, all: HasStageBackup},
		{Name: StageRestoreCreated, Step: StepStageRestore, all: HasStageBackup},
		{Name: EnsureStagePodsDeleted, Step: StepStageRestore, all: HasStagePods},
		{Name: EnsureStagePodsTerminated, Step: StepStageRestore, all: HasStagePods},
		{Name: EnsureAnnotationsDeleted, Step: StepStageRestore, all: HasStageBackup},
		{Name: WaitForDirectImageMigrationToComplete, Step: StepDirectImage, all: DirectImage | EnableImage},
		{Name: WaitForDirectVolumeMigrationToComplete, Step: StepDirectVolume, all: DirectVolume | EnableVolume},
		{Name: EnsureInitialBackupReplicated, Step: StepRestore},
		{Name: EnsureFinalRestore, Step: StepRestore},
		{Name: FinalRestoreCreated, Step: StepRestore},
		{Name: UnQuiesceDestApplications, Step: StepRestore},
		{Name: SwapPVCReferences, Step: StepCleanup, all: StorageConversion | Quiesce},
		{Name: DeleteRegistries, Step: StepCleanup},
		{Name: Verification, Step: StepCleanup, all: HasVerify},
		{Name: Completed, Step: StepCleanup},
	},
}

var CancelItinerary = Itinerary{
	Name: CancelItineraryName,
	Phases: []Phase{
		{Name: Canceling, Step: StepCleanupVelero},
		{Name: DeleteBackups, Step: StepCleanupVelero},
		{Name: DeleteRestores, Step: StepCleanupVelero},
		{Name: DeleteRegistries, Step: StepCleanupHelpers},
		{Name: DeleteHookJobs, Step: StepCleanupHelpers},
		{Name: DeleteDirectVolumeMigrationResources, Step: StepCleanupHelpers, all: DirectVolume},
		{Name: DeleteDirectImageMigrationResources, Step: StepCleanupHelpers, all: DirectImage},
		{Name: EnsureStagePodsDeleted, Step: StepCleanupHelpers, all: HasStagePods},
		{Name: EnsureAnnotationsDeleted, Step: StepCleanupHelpers, all: HasStageBackup},
		{Name: UnQuiesceSrcApplications, Step: StepCleanupUnquiesce},
		{Name: Canceled, Step: StepCleanup},
		{Name: Completed, Step: StepCleanup},
	},
}

var FailedItinerary = Itinerary{
	Name: FailedItineraryName,
	Phases: []Phase{
		{Name: MigrationFailed, Step: StepCleanupHelpers},
		{Name: DeleteRegistries, Step: StepCleanupHelpers},
		{Name: EnsureAnnotationsDeleted, Step: StepCleanupHelpers, all: HasStageBackup},
		{Name: Completed, Step: StepCleanup},
	},
}

var RollbackItinerary = Itinerary{
	Name: RollbackItineraryName,
	Phases: []Phase{
		{Name: Rollback, Step: StepCleanupVelero},
		{Name: DeleteBackups, Step: StepCleanupVelero},
		{Name: DeleteRestores, Step: StepCleanupVelero},
		{Name: DeleteRegistries, Step: StepCleanupHelpers},
		{Name: EnsureStagePodsDeleted, Step: StepCleanupHelpers},
		{Name: EnsureAnnotationsDeleted, Step: StepCleanupHelpers, anyf: HasPVs | HasISs},
		{Name: QuiesceDestinationApplications, Step: StepCleanupMigrated, all: DirectVolume | EnableVolume},
		{Name: EnsureDestQuiesced, Step: StepCleanupMigrated, all: DirectVolume | EnableVolume},
		{Name: SwapPVCReferences, Step: StepCleanupMigrated, all: StorageConversion},
		{Name: CreateDirectVolumeMigrationRollback, Step: StepRollbackLiveMigration, all: DirectVolume | EnableVolume | LiveVmMigration},
		{Name: WaitForDirectVolumeMigrationRollbackToComplete, Step: StepRollbackLiveMigration, all: DirectVolume | EnableVolume | LiveVmMigration},
		{Name: DeleteMigrated, Step: StepCleanupMigrated},
		{Name: EnsureMigratedDeleted, Step: StepCleanupMigrated},
		{Name: UnQuiesceSrcApplications, Step: StepCleanupUnquiesce},
		{Name: Completed, Step: StepCleanup},
	},
}
