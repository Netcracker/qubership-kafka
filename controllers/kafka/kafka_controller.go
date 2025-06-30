// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"context"
	"fmt"
	kafka "github.com/Netcracker/qubership-kafka/api/v1"
	"github.com/Netcracker/qubership-kafka/controllers"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	log = logf.Log.WithName("controller_kafka")
)

type ReconcileService interface {
	Reconcile() error
	Status() error
}

const (
	kafkaServiceConditionReason = "ReconcileCycleStatus"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	controllers.Reconciler
	StatusUpdater StatusUpdater
}

//+kubebuilder:rbac:groups=qubership.org,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qubership.org,resources=kafkas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qubership.org,resources=kafkas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kafka object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KafkaReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KafkaService")

	// Fetch the KafkaService instance
	instance := &kafka.Kafka{}
	var err error
	if err = r.Client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if !controllers.ApiGroupMatches(instance.APIVersion, r.ApiGroup) {
		return reconcile.Result{}, nil
	}
	r.StatusUpdater = NewStatusUpdater(r.Client, instance)

	specHash, err := util.Hash(instance.Spec)
	if err != nil {
		reqLogger.Info("error in hash function")
		return reconcile.Result{}, err
	}
	annotationsHash, err := util.Hash(instance.Annotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	isCustomResourceChanged := r.ResourceHashes["spec"] != specHash || r.ResourceHashes["annotations"] != annotationsHash
	if isCustomResourceChanged {
		instance.Status.Conditions = []kafka.StatusCondition{}
		if err = r.clearAllConditions(); err != nil {
			return reconcile.Result{}, err
		}
		if err = r.updateConditions(NewCondition(statusFalse,
			typeInProgress,
			kafkaServiceConditionReason,
			"Reconciliation cycle started")); err != nil {
			return reconcile.Result{}, err
		}
	}

	reconcilers := r.buildReconcilers(instance, log)

	for _, reconciler := range reconcilers {
		if err = reconciler.Reconcile(); err != nil {
			reqLogger.Error(err, "Error during reconciliation")
			r.writeFailedStatus(fmt.Sprintf("Reconciliation cycle failed for %T due to: %v", reconciler, err))
			return reconcile.Result{}, err
		}
	}

	if isCustomResourceChanged {
		if instance.Spec.WaitForPodsReady {
			if err = r.updateConditions(NewCondition(statusFalse,
				typeInProgress,
				kafkaServiceConditionReason,
				"Checking deployment readiness status")); err != nil {
				return reconcile.Result{}, err
			}

			for _, reconciler := range reconcilers {
				if err = reconciler.Status(); err != nil {
					r.writeFailedStatus(fmt.Sprintf("The status reconciliation cycle failed for %T due to: %v", reconciler, err))
					return reconcile.Result{}, err
				}
			}
		}

		var status *kafka.KafkaStatus
		status, err = r.StatusUpdater.GetStatus()
		if err != nil {
			r.writeFailedStatus(fmt.Sprintf("Cannot obtain status for Kafka custom resource: %v", err))
			return reconcile.Result{}, err
		}
		if hasFailedConditions(status) {
			if err = r.updateConditions(NewCondition(statusFalse,
				typeFailed,
				kafkaServiceConditionReason,
				"The deployment readiness status check failed")); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			if err = r.updateConditions(NewCondition(statusTrue,
				typeSuccessful,
				kafkaServiceConditionReason,
				"The deployment readiness status check is successful")); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	reqLogger.Info("Reconciliation cycle succeeded")
	r.ResourceHashes["annotations"] = annotationsHash
	r.ResourceHashes["spec"] = specHash
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	namespacePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == os.Getenv("OPERATOR_NAMESPACE")
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetNamespace() == os.Getenv("OPERATOR_NAMESPACE")
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetNamespace() == os.Getenv("OPERATOR_NAMESPACE")
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetNamespace() == os.Getenv("OPERATOR_NAMESPACE")
		},
	}
	dummyPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetResourceVersion() != e.ObjectOld.GetResourceVersion()
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kafka.Kafka{}).
		Owns(&corev1.Secret{}, builder.WithPredicates(namespacePredicate, dummyPredicate)).
		Complete(r)
}

func (r *KafkaReconciler) writeFailedStatus(errorMessage string) {
	if err := r.updateConditions(NewCondition(statusFalse, typeFailed, kafkaServiceConditionReason, errorMessage)); err != nil {
		log.Error(err, "An error occurred while updating the status condition")
	}
}

// buildReconcilers returns service reconcilers in accordance with custom resource.
func (r *KafkaReconciler) buildReconcilers(cr *kafka.Kafka, logger logr.Logger) []ReconcileService {
	var reconcilers []ReconcileService
	reconcilers = append(reconcilers, NewReconcileKafka(r, cr, logger))
	return reconcilers
}
