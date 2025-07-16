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

package migplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/google/uuid"

	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
)

// MigPlanReconciler reconciles a MigPlan object
type MigPlanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder
}

// +kubebuilder:rbac:groups=migrations.kubevirt.io,resources=migplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=migrations.kubevirt.io,resources=migplans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=migrations.kubevirt.io,resources=migplans/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=list;watch
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=list;watch
// +kubebuilder:rbac:groups=kubevirt.io,resources=kubevirts,verbs=list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MigPlan object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *MigPlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the MigPlan instance
	plan := &migrationsv1alpha1.MigPlan{}
	err := r.Get(context.TODO(), req.NamespacedName, plan)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get MigPlan")
		return reconcile.Result{}, err
	}

	// Report reconcile error.
	defer func() {
		// This should only be turned on in debug mode IMO,
		// in normal mode we should only show condition deltas.
		// log.Info("CR", "conditions", plan.Status.Conditions)
		plan.Status.Conditions.RecordEvents(plan, r.EventRecorder)
		// TODO: original controller was forgiving conflict errors, should we do the same?
		if err == nil {
			return
		}
		plan.Status.SetReconcileFailed(err)
		err := r.Status().Update(context.TODO(), plan)
		if err != nil {
			log.Error(err, "Failed to update MigPlan status on error")
			return
		}
	}()

	if plan.Status.Suffix == nil {
		// Generate suffix
		suffix := rand.String(4)
		plan.Status.Suffix = &suffix
	}

	// Begin staging conditions.
	plan.Status.BeginStagingConditions()

	// Plan Suspended
	err = r.planSuspended(ctx, plan)
	if err != nil {
		log.Error(err, "Failed to check if plan is suspended")
		return reconcile.Result{}, err
	}

	// Validations.
	err = r.validate(ctx, plan)
	if err != nil {
		log.Error(err, "Failed to validate plan")
		return reconcile.Result{}, err
	}

	// PV discovery
	err = r.updatePvs(ctx, plan)
	if err != nil {
		log.Error(err, "Failed to update discovered PVs")
		return reconcile.Result{}, err
	}

	// Validate PV actions

	planReadyCondition := plan.Status.HasCondition(PvsDiscovered)
	// Ready
	plan.Status.SetReady(
		planReadyCondition &&
			!plan.Status.HasBlockerCondition(),
		"The migration plan is ready.")

	// End staging conditions.
	plan.Status.EndStagingConditions()

	// Apply changes.
	markReconciled(plan)
	statusCopy := plan.Status.DeepCopy()
	// change this to patch so we're sure the only spec we fill in is persistent volumes
	err = r.Update(context.TODO(), plan)
	if err != nil {
		log.Error(err, "Failed to update MigPlan spec")
		return reconcile.Result{}, err
	}

	if statusCopy != nil {
		plan.Status = *statusCopy
	}
	err = r.Status().Update(context.TODO(), plan)
	if err != nil {
		log.Error(err, "Failed to update MigPlan status")
		return reconcile.Result{}, err
	}

	// Timed requeue on Plan conflict.
	if plan.Status.HasCondition(PlanConflict) {
		return reconcile.Result{RequeueAfter: time.Second * 10}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigPlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	c, err := controller.New("migplan-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MigPlan
	if err := c.Watch(source.Kind(mgr.GetCache(), &migrationsv1alpha1.MigPlan{},
		&handler.TypedEnqueueRequestForObject[*migrationsv1alpha1.MigPlan]{},
		predicate.TypedFuncs[*migrationsv1alpha1.MigPlan]{
			CreateFunc: func(e event.TypedCreateEvent[*migrationsv1alpha1.MigPlan]) bool { return true },
			DeleteFunc: func(e event.TypedDeleteEvent[*migrationsv1alpha1.MigPlan]) bool { return true },
			UpdateFunc: func(e event.TypedUpdateEvent[*migrationsv1alpha1.MigPlan]) bool {
				return !reflect.DeepEqual(e.ObjectOld.Spec, e.ObjectNew.Spec) ||
					!reflect.DeepEqual(e.ObjectOld.DeletionTimestamp, e.ObjectNew.DeletionTimestamp)
			},
		},
	)); err != nil {
		return err
	}

	// Watch for changes to MigClusters referenced by MigPlans

	// Watch for changes to MigMigrations

	return nil
}

// Determine whether the plan is `suspended`.
// A plan is considered `suspended` when a migration is running or the final migration has
// completed successfully. While suspended, reconcile is limited to basic validation
// and PV discovery and ensuring resources is not performed.
func (r *MigPlanReconciler) planSuspended(ctx context.Context, plan *migrationsv1alpha1.MigPlan) error {
	suspended := false

	// TODO: Implement this when MigMigration becomes available
	// migrations, err := plan.ListMigrations(ctx, r)
	// if err != nil {
	// 	return fmt.Errorf("error in listing migrations: %w", err)
	// }

	// // Sort migrations by timestamp, newest first.
	// sort.Slice(migrations, func(i, j int) bool {
	// 	ts1 := migrations[i].CreationTimestamp
	// 	ts2 := migrations[j].CreationTimestamp
	// 	return ts1.Time.After(ts2.Time)
	// })

	// for _, m := range migrations {
	// 	// If a migration is running, plan should be suspended
	// 	if m.Status.HasCondition(migrationsv1alpha1.Running) {
	// 		suspended = true
	// 		break
	// 	}
	// 	// If the newest final migration is successful, suspend plan
	// 	if m.Status.HasCondition(migrationsv1alpha1.Succeeded) && !m.Spec.Stage && !m.Spec.Rollback {
	// 		suspended = true
	// 		break
	// 	}
	// 	// If the newest migration is a successful rollback, unsuspend plan
	// 	if m.Status.HasCondition(migrationsv1alpha1.Succeeded) && m.Spec.Rollback {
	// 		suspended = false
	// 		break
	// 	}
	// }

	if suspended {
		plan.Status.SetCondition(migrationsv1alpha1.Condition{
			Type:     Suspended,
			Status:   True,
			Category: Advisory,
			Message: "The migrations plan is in suspended state; Limited validation enforced; PV discovery and " +
				"resource reconciliation suspended.",
		})
	}

	return nil
}

func markReconciled(plan *migrationsv1alpha1.MigPlan) {
	uuid, _ := uuid.NewUUID()
	if plan.Annotations == nil {
		plan.Annotations = map[string]string{}
	}
	plan.Annotations[migrationsv1alpha1.TouchAnnotation] = uuid.String()
	plan.Status.ObservedDigest = digest(plan.Spec)
}

// Generate a sha256 hex-digest for an object.
func digest(object interface{}) string {
	j, _ := json.Marshal(object)
	hash := sha256.New()
	hash.Write(j)
	digest := hex.EncodeToString(hash.Sum(nil))
	return digest
}
