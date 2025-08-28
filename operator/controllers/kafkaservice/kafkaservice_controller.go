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

package kafkaservice

import (
	"context"
	"fmt"
	kafkaservice "github.com/Netcracker/qubership-kafka/operator/api/v7"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	kafkaServiceConditionReason       = "ReconcileCycleStatus"
	globalHashName                    = "spec.global"
	autoRestartAnnotation             = "kafkaservice.qubership.org/auto-restart"
	resourceVersionAnnotationTemplate = "%s/resource-version"
)

var (
	log            = logf.Log.WithName("controller_kafkaservice")
	globalSpecHash = ""
)

type ReconcileService interface {
	Reconcile() error
	Status() error
}

//+kubebuilder:rbac:groups=qubership.org,resources=kafkaservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qubership.org,resources=kafkaservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qubership.org,resources=kafkaservices/finalizers,verbs=update

func (r *KafkaServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KafkaService")

	// Fetch the KafkaService instance
	instance := &kafkaservice.KafkaService{}
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
	r.StatusUpdater = NewStatusUpdater(r.Client, instance)

	specHash, err := util.Hash(instance.Spec)
	if err != nil {
		reqLogger.Info("error in hash function")
		return reconcile.Result{}, err
	}
	globalSpecHash, err = util.Hash(instance.Spec.Global)
	if err != nil {
		reqLogger.Info("error in hash function for global section")
		return reconcile.Result{}, err
	}
	annotationsHash, err := util.Hash(instance.Annotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	isCustomResourceChanged := r.ResourceHashes["spec"] != specHash || r.ResourceHashes["annotations"] != annotationsHash
	if isCustomResourceChanged {
		instance.Status.Conditions = []kafkaservice.StatusCondition{}
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

	drChecked := false
	if instance.Spec.DisasterRecovery != nil &&
		(instance.Status.DisasterRecoveryStatus.Mode != instance.Spec.DisasterRecovery.Mode ||
			instance.Status.DisasterRecoveryStatus.Status == "running" ||
			instance.Status.DisasterRecoveryStatus.Status == "failed") {
		checkNeeded := isCheckNeeded(instance)

		if err = r.updateDisasterRecoveryStatus(instance,
			"running",
			"The switchover process for Kafka has been started"); err != nil {
			return reconcile.Result{}, err
		}

		status := "done"
		message := "replication has finished successfully"
		if checkNeeded {
			replicationAuditor := NewKafkaReplicationAuditor(instance, r)
			checkCompleted, errSwitchover := replicationAuditor.CheckFullReplication(time.Second * 300)
			if !checkCompleted {
				status = "failed"
				message = "timeout occurred during replication check"
			} else {
				if errSwitchover != nil {
					status = "failed"
					message = fmt.Sprintf("Error is occurred during switching: %v", errSwitchover)
				} else {
					drChecked = true
				}
			}
		} else {
			message = "Switchover mode has been changed without replication check"
			drChecked = true
		}

		defer func() {
			if status == "failed" {
				_ = r.updateDisasterRecoveryStatus(instance, status, message)
			} else {
				if err != nil {
					status = "failed"
					message = fmt.Sprintf("Error is occurred during Kafka switching: %v", err)
				}
				_ = r.updateDisasterRecoveryStatus(instance, status, message)
			}
		}()
	}

	reconcilers := r.buildReconcilers(instance, log, drChecked)

	for _, reconciler := range reconcilers {
		if err = reconciler.Reconcile(); err != nil {
			reqLogger.Error(err, "Error during reconciliation")
			r.writeFailedStatus(fmt.Sprintf("Reconciliation cycle failed for %T due to: %v", reconciler, err))
			return reconcile.Result{}, err
		}
	}

	if isCustomResourceChanged {
		if instance.Spec.Global != nil && instance.Spec.Global.WaitForPodsReady {
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
			// Gives Kubernetes time to update the status of the last component being checked.
			time.Sleep(5 * time.Second)
		}

		var status *kafkaservice.KafkaServiceStatus
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
			err = fmt.Errorf("status for Kafka service contains failed conditions")
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
	r.ResourceHashes[globalHashName] = globalSpecHash
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	statusPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
				!util.EqualPlainMaps(e.ObjectOld.GetAnnotations(), e.ObjectNew.GetAnnotations())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
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
		For(&kafkaservice.KafkaService{}, builder.WithPredicates(statusPredicate, namespacePredicate)).
		Owns(&corev1.Secret{}, builder.WithPredicates(namespacePredicate, dummyPredicate)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(namespacePredicate, dummyPredicate)).
		Owns(&v1.Deployment{}, builder.WithPredicates(namespacePredicate, dummyPredicate)).
		Complete(r)
}

func isCheckNeeded(instance *kafkaservice.KafkaService) bool {
	if instance.Spec.DisasterRecovery.NoWait {
		return false
	}
	if !instance.Spec.DisasterRecovery.MirrorMakerReplication.Enabled {
		return false
	}
	specMode := strings.ToLower(instance.Spec.DisasterRecovery.Mode)
	statusMode := strings.ToLower(instance.Status.DisasterRecoveryStatus.Mode)
	switchoverStatus := strings.ToLower(instance.Status.DisasterRecoveryStatus.Status)
	return specMode == "active" && (statusMode != "active" || statusMode == "active" && switchoverStatus == "failed")
}

func (r *KafkaServiceReconciler) writeFailedStatus(errorMessage string) {
	if err := r.updateConditions(NewCondition(statusFalse, typeFailed, kafkaServiceConditionReason, errorMessage)); err != nil {
		log.Error(err, "An error occurred while updating the status condition")
	}
}

// buildReconcilers returns service reconcilers in accordance with custom resource.
func (r *KafkaServiceReconciler) buildReconcilers(cr *kafkaservice.KafkaService, logger logr.Logger,
	drChecked bool) []ReconcileService {
	var reconcilers []ReconcileService
	if cr.Spec.Monitoring != nil {
		reconcilers = append(reconcilers, NewReconcileMonitoring(r, cr, logger))
	}
	if cr.Spec.Akhq != nil {
		reconcilers = append(reconcilers, NewReconcileAkhq(r, cr, logger))
	}
	if cr.Spec.MirrorMaker != nil {
		reconcilers = append(reconcilers, NewReconcileMirrorMaker(r, cr, logger, drChecked))
	}
	if cr.Spec.MirrorMakerMonitoring != nil {
		reconcilers = append(reconcilers, NewReconcileMirrorMakerMonitoring(r, cr, logger, drChecked))
	}
	if cr.Spec.IntegrationTests != nil {
		reconcilers = append(reconcilers, NewReconcileIntegrationTests(r, cr, logger))
	}
	if cr.Spec.BackupDaemon != nil {
		reconcilers = append(reconcilers, NewReconcileBackupDaemon(r, cr, logger))
	}
	return reconcilers
}

// updateDisasterRecoveryStatus updates state of Disaster Recovery switchover
func (r *KafkaServiceReconciler) updateDisasterRecoveryStatus(cr *kafkaservice.KafkaService, status string, message string) error {
	return r.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		instance.Status.DisasterRecoveryStatus.Mode = cr.Spec.DisasterRecovery.Mode
		instance.Status.DisasterRecoveryStatus.Status = status
		instance.Status.DisasterRecoveryStatus.Message = message
	})
}
