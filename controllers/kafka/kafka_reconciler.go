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
	"bytes"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	kafka "github.com/Netcracker/qubership-kafka/api/v1"
	"github.com/Netcracker/qubership-kafka/controllers"
	"github.com/Netcracker/qubership-kafka/controllers/provider"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	kafkaConditionReason              = "KafkaReadinessStatus"
	kafkaHashName                     = "spec"
	autoRestartAnnotation             = "kafkaservice.qubership.org/auto-restart"
	resourceVersionAnnotationTemplate = "%s/resource-version"
)

type ReconcileKafka struct {
	cr            *kafka.Kafka
	reconciler    *KafkaReconciler
	logger        logr.Logger
	kafkaProvider provider.KafkaResourceProvider
}

func NewReconcileKafka(r *KafkaReconciler, cr *kafka.Kafka, logger logr.Logger) ReconcileKafka {
	return ReconcileKafka{
		cr:            cr,
		logger:        logger,
		reconciler:    r,
		kafkaProvider: provider.NewKafkaResourceProvider(cr, logger),
	}
}

func (r ReconcileKafka) Reconcile() error {
	kafkaSecret, err := r.reconciler.WatchSecret(r.cr.Spec.SecretName, r.cr, r.logger)
	if err != nil {
		return err
	}

	kafkaSpecHash, err := util.Hash(r.cr.Spec)
	if err != nil {
		return err
	}

	kafkaConfigurationChanged := r.reconciler.ResourceHashes[kafkaHashName] != kafkaSpecHash ||
		(kafkaSecret.Name != "" && r.reconciler.ResourceVersions[kafkaSecret.Name] != kafkaSecret.ResourceVersion)

	if !kafkaConfigurationChanged {
		r.logger.Info("Kafka configuration didn't change, skipping reconcile loop")
	} else {
		if r.cr.Spec.Replicas > 0 {
			if err = r.processKafkaReplicas(kafkaSecret); err != nil {
				return err
			}
		}

		if err := r.updateKafkaStatus(); err != nil {
			return err
		}
	}

	if kafkaSecret.ResourceVersion != r.reconciler.ResourceVersions[kafkaSecret.Name] {
		kafkaServicesSecret, err := r.reconciler.FindSecret(fmt.Sprintf("%s-services-secret", r.cr.Name), r.cr.GetNamespace(), r.logger)
		if err != nil {
			if errors.IsNotFound(err) {
				r.logger.Info("Kafka Services secret not found, skipping secret update")
			} else {
				return err
			}
		} else {
			r.logger.Info("Updating Kafka Services secret")
			kafkaServicesSecret.Data["admin-username"] = kafkaSecret.Data["client-username"]
			kafkaServicesSecret.Data["admin-password"] = kafkaSecret.Data["client-password"]
			kafkaServicesSecret.Data["client-username"] = kafkaSecret.Data["client-username"]
			kafkaServicesSecret.Data["client-password"] = kafkaSecret.Data["client-password"]
			if err := r.reconciler.UpdateSecret(kafkaServicesSecret, r.logger); err != nil {
				return err
			}
		}
	}

	r.reconciler.ResourceVersions[kafkaSecret.Name] = kafkaSecret.ResourceVersion
	r.reconciler.ResourceHashes[kafkaHashName] = kafkaSpecHash
	return nil
}

func (r ReconcileKafka) processKafkaReplicas(kafkaSecret *corev1.Secret) error {
	kafkaSpec := r.cr.Spec

	err := checkParamsForExternalAccess(r.cr, kafkaSpec.Replicas)
	if err != nil {
		return err
	}
	err = r.checkRacksConfig(kafkaSpec.Replicas)
	if err != nil {
		return err
	}

	clientService := r.kafkaProvider.NewKafkaClientServiceForCR()
	if err := controllerutil.SetControllerReference(r.cr, clientService, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateService(clientService, r.logger); err != nil {
		return err
	}

	domainClientService := r.kafkaProvider.NewKafkaDomainClientServiceForCR()
	if err := controllerutil.SetControllerReference(r.cr, domainClientService, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateService(domainClientService, r.logger); err != nil {
		return err
	}

	trustCertsSecret := r.kafkaProvider.NewEmptySecret(fmt.Sprintf("%s-trusted-certs", r.cr.Name))
	if err := r.reconciler.CreateSecret(trustCertsSecret, r.logger); err != nil {
		return err
	}

	publicCertsSecret := r.kafkaProvider.NewEmptySecret(fmt.Sprintf("%s-public-certs", r.cr.Name))
	if err := r.reconciler.CreateSecret(publicCertsSecret, r.logger); err != nil {
		return err
	}

	serviceAccount := provider.NewServiceAccount(r.kafkaProvider.GetServiceAccountName(), r.cr.Namespace, r.cr.Spec.DefaultLabels)
	if err := r.reconciler.CreateOrUpdateServiceAccount(serviceAccount, r.logger); err != nil {
		return err
	}

	currentReplicas, err := r.getCurrentDeploymentsCount()
	if err != nil {
		return err
	}

	if currentReplicas < 3 {
		r.logger.Info("RollingUpdate value is set to false")
		r.cr.Spec.RollingUpdate = false
	}

	r.logger.Info(fmt.Sprintf("Update brokers set: current replicas count is [%d], new replicas count is [%d].", currentReplicas, kafkaSpec.Replicas))
	kraft := r.cr.Spec.Kraft.Enabled
	if r.cr.Spec.Kraft.Migration {
		kraft = false
	}
	if err := r.rolloutBrokers(kafkaSpec.Replicas, kraft, kafkaSecret); err != nil {
		return err
	}

	if currentReplicas > 0 && currentReplicas < kafkaSpec.Replicas {
		if err := r.reassignPartitionsWithStatusUpdate(int32(kafkaSpec.Replicas), true); err != nil {
			return err
		}
	} else {
		if err := r.reassignPartitionsWithStatusUpdate(int32(kafkaSpec.Replicas), false); err != nil {
			return err
		}
		if currentReplicas > kafkaSpec.Replicas && r.kafkaProvider.IsBrokerScalingInEnabled() {
			if err = r.performBrokerScalingIn(currentReplicas, kafkaSpec.Replicas); err != nil {
				return err
			}
		}
	}

	if r.cr.Spec.Kraft.Migration {
		var status *kafka.KafkaStatus
		status, err = r.reconciler.StatusUpdater.GetStatus()
		if err != nil {
			log.Info("Can not get previous Kraft migration status, will start from the beginning")
			status.KraftMigrationStatus.Status = "Initial step"
		}
		step := 0
		switch status.KraftMigrationStatus.Status {
		case "Created Kraft controller with connection to ZooKeeper":
			step = 1
		case "Migrated metadata from ZooKeeper to Kraft in controller":
			step = 2
		case "Removed ZooKeeper connection and created Kraft cluster":
			step = 3
		case "Migration finished succesfully":
			step = 4
		default:
			step = 0
		}
		log.Info("Starting ZooKeeper to Kraft migration")

		// ZooKeeper -> Kraft migration initial step, getting ZooKeeper cluster ID
		zkClusterID, err := r.getZooKeeperClusterID()
		if err != nil {
			return err
		}

		if step < 1 {
			// ZooKeeper -> Kraft migration step one, creating Kraft controller
			log.Info("Creating Kraft controller")
			if err := r.createMigrationControllerEntities(zkClusterID); err != nil {
				return err
			}
			if err := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
				instance.Status.KraftMigrationStatus.Status = "Created Kraft controller with connection to ZooKeeper"
			}); err != nil {
				return err
			}
		}

		if step < 2 {
			// ZooKeeper -> Kraft migration step two, updating brokers and waiting for migration
			log.Info("Updating brokers and waiting for migration")
			if err := r.updateBrokersAndWaitMigrationResult(zkClusterID, currentReplicas); err != nil {
				return err
			}
			if err := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
				instance.Status.KraftMigrationStatus.Status = "Migrated metadata from ZooKeeper to Kraft in controller"
			}); err != nil {
				return err
			}
		}

		if step < 3 {
			// ZooKeeper -> Kraft migration step three, updating Kraft controller and brokers to remove ZooKeeper connection
			// and creating Kraft cluster

			log.Info("Updating Kraft controller and brokers to remove ZooKeeper connection and creating Kraft cluster")

			if err := r.updateBrokersWithoutZooKeeper(zkClusterID, currentReplicas); err != nil {
				return err
			}

			if err := r.updateMigrationControllerWithoutZooKeeper(zkClusterID); err != nil {
				return err
			}

			if err := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
				instance.Status.KraftMigrationStatus.Status = "Removed ZooKeeper connection and created Kraft cluster"
			}); err != nil {
				return err
			}
		}

		if step < 4 {
			// ZooKeeper -> Kraft migration step four, removing Kraft controller and finishing migration
			log.Info("Updating brokers to remove Kraft controller from voters list")
			if err := r.updateBrokersWithoutKraftMigrationController(zkClusterID, currentReplicas); err != nil {
				return err
			}
			log.Info("Removing Kraft controller entities")
			if err := r.removeMigrationControllerEntities(zkClusterID); err != nil {
				return err
			}
			if err := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
				instance.Status.KraftMigrationStatus.Status = "Migration finished succesfully"
			}); err != nil {
				return err
			}
		}

		log.Info("ZooKeeper to Kraft migration finished succesfully or not needed")
	}

	return nil
}

func (r ReconcileKafka) rolloutBrokers(replicas int, kraft bool, kafkaSecret *corev1.Secret) error {
	r.logger.Info("Perform brokers rollout procedure")
	for brokerId := 1; brokerId <= replicas; brokerId++ {
		if err := r.rolloutBroker(brokerId, kraft, kafkaSecret); err != nil {
			return err
		}
		if r.cr.Spec.RollingUpdate {
			if err := r.waitUntilBrokerIsReady(brokerId, 300); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r ReconcileKafka) reassignPartitionsWithStatusUpdate(replicas int32, clusterScaling bool) error {
	r.logger.Info(fmt.Sprintf("Reassign partitions with cluster scaling enabled: %t", clusterScaling))
	if err := r.reassignPartitions(replicas, clusterScaling); err != nil {
		err2 := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
			instance.Status.PartitionsReassignmentStatus.Status = "Failed"
		})
		if err2 != nil {
			return err2
		}
		return err
	}
	return nil
}

func (r ReconcileKafka) performBrokerScalingIn(currentReplicas int, requiredReplicas int) error {
	r.logger.Info(fmt.Sprintf("There is an attempt to downscale Kafka with %d replicas to Kafka with %d replicas. For correct work excess Kafka deployments need to be scaled down.", currentReplicas, requiredReplicas))
	for i := requiredReplicas + 1; i <= currentReplicas; i++ {
		if err := r.reconciler.ScaleDeployment(fmt.Sprintf("%s-%d", r.cr.Name, i), 0, r.cr.Namespace, r.logger); err != nil {
			return err
		}
	}
	return nil
}

func (r ReconcileKafka) Status() error {
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		kafkaConditionReason,
		"Kafka health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of Kafka pods")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.kafkaProvider.GetSelectorLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, kafkaConditionReason, "Kafka pods are not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, kafkaConditionReason, "Kafka pods are ready"))
}

func (r *ReconcileKafka) rolloutBroker(brokerId int, kraft bool, kafkaSecret *corev1.Secret) error {
	brokerService := r.kafkaProvider.NewKafkaBrokerServiceForCR(brokerId)
	if err := controllerutil.SetControllerReference(r.cr, brokerService, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateService(brokerService, r.logger); err != nil {
		return err
	}

	persistentVolumeClaim := r.kafkaProvider.NewKafkaPersistentVolumeClaimForCR(brokerId)
	if persistentVolumeClaim != nil {
		if err := r.reconciler.CreatePersistentVolumeClaim(persistentVolumeClaim, r.logger); err != nil {
			return err
		}
	}

	rack, err := r.getRack(brokerId, r.logger)
	if err != nil {
		return err
	}
	brokerDeployment := r.kafkaProvider.NewKafkaBrokerDeploymentForCR(brokerId, rack, kraft, "")
	if err := controllerutil.SetControllerReference(r.cr, brokerDeployment, r.reconciler.Scheme); err != nil {
		return err
	}
	if kafkaSecret.Annotations != nil && kafkaSecret.Annotations[autoRestartAnnotation] == "true" {
		r.addDeploymentAnnotation(brokerDeployment, fmt.Sprintf(resourceVersionAnnotationTemplate, kafkaSecret.Name), kafkaSecret.ResourceVersion)
	}
	if err := r.reconciler.CreateOrUpdateDeployment(brokerDeployment, r.logger); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileKafka) updateBrokerDeploymentForMigration(brokerId int, replicas int, zkClusterID string, migrated bool) error {
	rack, err := r.getRack(brokerId, r.logger)
	if err != nil {
		return err
	}
	var brokerDeployment *appsv1.Deployment
	var additionalEnvs []corev1.EnvVar
	if !migrated {
		var voters []string
		voters = append(voters, fmt.Sprintf("3000@%s-%s:9092", r.cr.Name, "kraft-controller"))
		additionalEnvs = []corev1.EnvVar{
			{Name: "MIGRATION_BROKER", Value: "true"},
			{Name: "VOTERS", Value: strings.Join(voters, ",")},
		}
		brokerDeployment = r.kafkaProvider.NewKafkaBrokerDeploymentForCR(brokerId, rack, false, zkClusterID)
		brokerDeployment.Spec.Template.Spec.Containers[0].Env = append(brokerDeployment.Spec.Template.Spec.Containers[0].Env, additionalEnvs...)
	} else {
		var voters []string
		for i := 1; i <= replicas; i++ {
			voters = append(voters, fmt.Sprintf("%d@%s-%d.kafka-broker.%s:9096", i, r.cr.Name, i, r.cr.Namespace))
		}
		voters = append(voters, fmt.Sprintf("3000@%s-%s:9092", r.cr.Name, "kraft-controller"))
		additionalEnvs = []corev1.EnvVar{
			{Name: "VOTERS", Value: strings.Join(voters, ",")},
		}
		brokerDeployment = r.kafkaProvider.NewKafkaBrokerDeploymentForCR(brokerId, rack, true, zkClusterID)
		brokerDeployment.Spec.Template.Spec.Containers[0].Env = append(brokerDeployment.Spec.Template.Spec.Containers[0].Env, additionalEnvs...)
	}
	if err := controllerutil.SetControllerReference(r.cr, brokerDeployment, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateDeployment(brokerDeployment, r.logger); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileKafka) createMigrationControllerEntities(zkClusterID string) error {
	controllerService := r.kafkaProvider.NewKafkaControllerServiceForCR()
	if err := controllerutil.SetControllerReference(r.cr, controllerService, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateService(controllerService, r.logger); err != nil {
		return err
	}

	persistentVolumeClaim := r.kafkaProvider.NewKafkaControllerPersistentVolumeClaimForCR()
	if persistentVolumeClaim != nil {
		if err := r.reconciler.CreatePersistentVolumeClaim(persistentVolumeClaim, r.logger); err != nil {
			return err
		}
	}
	controllerDeployment := r.kafkaProvider.NewKafkaKraftControllerDeploymentForCR(zkClusterID, false, true)
	if err := controllerutil.SetControllerReference(r.cr, controllerDeployment, r.reconciler.Scheme); err != nil {
		return err
	}

	if err := r.reconciler.CreateOrUpdateDeployment(controllerDeployment, r.logger); err != nil {
		return err
	}

	if err := r.waitUntilControllerIsReady(r.cr.Spec.Kraft.MigrationTimeout); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileKafka) removeMigrationControllerEntities(zkClusterID string) error {
	controllerService := r.kafkaProvider.NewKafkaControllerServiceForCR()

	if err := r.reconciler.DeleteService(controllerService, r.logger); err != nil {
		return err
	}

	persistentVolumeClaim := r.kafkaProvider.NewKafkaControllerPersistentVolumeClaimForCR()
	if persistentVolumeClaim != nil {
		if err := r.reconciler.DeletePersistentVolumeClaim(persistentVolumeClaim, r.logger); err != nil {
			return err
		}
	}
	controllerDeployment := r.kafkaProvider.NewKafkaKraftControllerDeploymentForCR(zkClusterID, false, true)
	if err := r.reconciler.DeleteDeployment(controllerDeployment, r.logger); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileKafka) updateMigrationControllerWithoutZooKeeper(zkClusterID string) error {
	controllerDeployment := r.kafkaProvider.NewKafkaKraftControllerDeploymentForCR(zkClusterID, true, false)
	if err := controllerutil.SetControllerReference(r.cr, controllerDeployment, r.reconciler.Scheme); err != nil {
		return err
	}

	if err := r.reconciler.CreateOrUpdateDeployment(controllerDeployment, r.logger); err != nil {
		return err
	}

	if err := r.waitUntilControllerIsReady(r.cr.Spec.Kraft.MigrationTimeout); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileKafka) updateBrokersAndWaitMigrationResult(zkClusterID string, currentReplicas int) error {
	for brokerId := 1; brokerId <= currentReplicas; brokerId++ {
		if err := r.updateBrokerDeploymentForMigration(brokerId, currentReplicas, zkClusterID, false); err != nil {
			return err
		}
		if err := r.waitUntilBrokerIsReady(brokerId, r.cr.Spec.Kraft.MigrationTimeout); err != nil {
			return err
		}
	}

	if err := r.waitUntilMigrationCompleted(r.cr.Spec.Kraft.MigrationTimeout); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileKafka) updateBrokersWithoutZooKeeper(zkClusterID string, currentReplicas int) error {
	for brokerId := 1; brokerId <= currentReplicas; brokerId++ {
		if err := r.updateBrokerDeploymentForMigration(brokerId, currentReplicas, zkClusterID, true); err != nil {
			return err
		}
	}
	for brokerId := 1; brokerId <= currentReplicas; brokerId++ {
		if err := r.waitUntilBrokerIsReady(brokerId, r.cr.Spec.Kraft.MigrationTimeout); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileKafka) updateBrokersWithoutKraftMigrationController(zkClusterID string, currentReplicas int) error {
	for brokerId := 1; brokerId <= currentReplicas; brokerId++ {
		rack, err := r.getRack(brokerId, r.logger)
		if err != nil {
			return err
		}
		brokerDeployment := r.kafkaProvider.NewKafkaBrokerDeploymentForCR(brokerId, rack, true, zkClusterID)
		if err := controllerutil.SetControllerReference(r.cr, brokerDeployment, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateDeployment(brokerDeployment, r.logger); err != nil {
			return err
		}
	}

	for brokerId := 1; brokerId <= currentReplicas; brokerId++ {
		if err := r.waitUntilBrokerIsReady(brokerId, r.cr.Spec.Kraft.MigrationTimeout); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileKafka) runCommandInPod(podName string, container string, namespace string, command []string) (string, error) {
	config := kubeconfig.GetConfigOrDie()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	request := kubeClient.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Stdout:    true,
			Stderr:    true,
			Container: container,
			Command:   command,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(config, "POST", request.URL())
	if err != nil {
		return "", fmt.Errorf("unable execute command via SPDY: %+v", err)
	}
	var execOut bytes.Buffer
	var execErr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    false,
	})
	if err != nil {
		if strings.Contains(err.Error(), "command terminated with exit code 127") {
			return "", fmt.Errorf("the command %v is not found in '%s' pod", command, podName)
		}
		return "", fmt.Errorf("there is a problem during command execution: %+v", err)
	}
	if execErr.Len() > 0 {
		if !strings.Contains(execErr.String(), "zookeeper.ssl.keyStore.location not specified") {
			return "", fmt.Errorf("there is a problem during command execution: %s", execErr.String())
		}
	}
	log.Info(fmt.Sprintf("Executed command output: %s", execOut.String()))
	return execOut.String(), nil
}

func (r ReconcileKafka) getZooKeeperClusterID() (string, error) {
	labels := r.kafkaProvider.GetSelectorLabels()
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return "", err
	}
	podNames := controllers.GetActualPodNames(foundPodList.Items)
	zkClusterID, commandErr := r.runCommandInPod(podNames[0], "kafka", r.cr.Namespace,
		[]string{"/bin/sh", "-c", "${KAFKA_HOME}/bin/get-cluster-id.sh"})
	return strings.TrimSpace(zkClusterID), commandErr
}

func (r ReconcileKafka) getMigrationStatus() bool {
	labels := make(map[string]string)
	labels["name"] = fmt.Sprintf("%s-%s", r.cr.Name, "kraft-controller")
	labels["component"] = "kafka-controller"
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		log.Error(err, "Cannot find controller pod")
		return false
	}
	podNames := controllers.GetActualPodNames(foundPodList.Items)
	status, commandErr := r.runCommandInPod(podNames[0], "kafka", r.cr.Namespace,
		[]string{"/bin/sh", "-c", "${KAFKA_HOME}/bin/get-kraft-migration-status.sh"})
	if commandErr != nil {
		log.Error(err, "Cannot get migration status from controller pod exec")
		return false
	}
	status = strings.TrimSpace(status)
	if status == "true" {
		return true
	} else {
		return false
	}
}

func (r *ReconcileKafka) reassignPartitions(newBrokersCount int32, clusterScaling bool) error {
	reassignPartitionsEnabled := r.kafkaProvider.IsReassignPartitionsEnabled(clusterScaling)
	allBrokersStartTimeoutSeconds := r.kafkaProvider.GetAllBrokersStartTimeoutSeconds()
	topicReassignmentTimeoutSeconds := r.kafkaProvider.GetTopicReassignmentTimeoutSeconds()
	if !reassignPartitionsEnabled {
		r.logger.Info("Partitions reassignment is disabled")
		return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
			instance.Status.PartitionsReassignmentStatus.Status = "Disabled"
		})
	}
	// for cluster scaling we run reassignment without taking into account Status,
	// because we can scale a cluster several times and always want to reassign
	// despite the Finished status from previous reassignment
	if clusterScaling || r.cr.Status.PartitionsReassignmentStatus.Status != "Finished" {
		r.logger.Info(fmt.Sprintf("Partitions reassignment is enabled, allBrokersStartTimeoutSeconds is %d, topicReassignmentTimeoutSeconds is %d", allBrokersStartTimeoutSeconds, topicReassignmentTimeoutSeconds))
		err := r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
			instance.Status.PartitionsReassignmentStatus.Status = "In Progress"
		})
		if err != nil {
			return err
		}
		username, password, err := r.getKafkaCredentials()
		if err != nil {
			return err
		}
		sslEnabled := r.cr.Spec.Ssl.Enabled
		sslCertificates, err := r.getKafkaCertificates()
		if err != nil {
			return err
		}
		kafkaClient, err := controllers.NewKafkaClient(
			r.kafkaProvider.GetServiceName(),
			username,
			password,
			sslEnabled,
			sslCertificates,
			newBrokersCount,
			allBrokersStartTimeoutSeconds,
			topicReassignmentTimeoutSeconds)
		if err != nil {
			return err
		}
		err = kafkaClient.ReassignPartitions()
		if err != nil {
			return err
		}
		return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
			instance.Status.PartitionsReassignmentStatus.Status = "Finished"
		})
	}
	r.logger.Info("Partitions are already reassigned. Skip reassignment")
	return nil
}

func (r *ReconcileKafka) getKafkaCredentials() (string, string, error) {
	foundSecret, err := r.reconciler.FindSecret(r.cr.Spec.SecretName, r.cr.Namespace, r.logger)
	if err != nil {
		return "", "", err
	}
	username := string(foundSecret.Data["client-username"])
	password := string(foundSecret.Data["client-password"])
	return username, password, nil
}

func (r *ReconcileKafka) getKafkaCertificates() (*controllers.SslCertificates, error) {
	if r.cr.Spec.Ssl.Enabled && r.cr.Spec.Ssl.SecretName != "" {
		sslCertificates, err :=
			r.reconciler.GetSslCertificates(r.cr.Spec.Ssl.SecretName, r.cr.Namespace, r.logger)
		if err != nil {
			return nil, err
		}
		return sslCertificates, nil
	}
	return &controllers.SslCertificates{}, nil
}

func (r *ReconcileKafka) getCurrentDeploymentsCount() (int, error) {
	deployments, err := r.reconciler.FindKafkaDeployments(r.cr)
	if err != nil {
		return 0, err
	}

	deploymentsWithPods := 0
	for _, deployment := range deployments.Items {
		if *deployment.Spec.Replicas > 0 {
			deploymentsWithPods++
		}
	}

	return deploymentsWithPods, nil
}

func checkParamsForExternalAccess(cr *kafka.Kafka, replicasCount int) error {
	externalHostNamesCount := len(cr.Spec.ExternalHostNames)
	externalPortsCount := len(cr.Spec.ExternalPorts)

	if externalHostNamesCount > 0 {
		if externalHostNamesCount != replicasCount {
			return fmt.Errorf("the number of external host names must be equal to replicas")
		}
		if externalPortsCount > 0 && externalPortsCount != replicasCount {
			return fmt.Errorf("external ports must be empty or their number must be equal to replicas")
		}
	}
	return nil
}

func (r *ReconcileKafka) isGetRacksFromNodeLabelsEnabled() bool {
	if r.cr.Spec.GetRacksFromNodeLabels != nil {
		return *r.cr.Spec.GetRacksFromNodeLabels
	}
	return false
}

func (r *ReconcileKafka) checkRacksConfig(replicasCount int) error {
	if r.isGetRacksFromNodeLabelsEnabled() {
		nodesCount := len(r.cr.Spec.Storage.Nodes)
		if r.cr.Spec.NodeLabelNameForRack == "" || nodesCount != replicasCount {
			return fmt.Errorf("when GetRacksFromNodeLabels=true, nodeLabelNameForRack and Storage.Nodes must be specified")
		}
		return nil
	}

	racksCount := len(r.cr.Spec.Racks)
	if racksCount > 0 && racksCount != replicasCount {
		return fmt.Errorf("the number of Racks must be equal to replicas")
	}

	return nil
}

// Get rack for broker if GetRacksFromNodeLabels configured or explicit list of racks' names is provided
func (r *ReconcileKafka) getRack(brokerId int, logger logr.Logger) (string, error) {
	if r.isGetRacksFromNodeLabelsEnabled() {
		return r.reconciler.GetNodeLabel(r.cr.Spec.Storage.Nodes[brokerId-1], r.cr.Spec.NodeLabelNameForRack, logger)
	} else {
		if len(r.cr.Spec.Racks) > 0 {
			return r.cr.Spec.Racks[brokerId-1], nil
		}
	}
	return "", nil
}

func (r *ReconcileKafka) updateKafkaStatus() error {
	labels := r.kafkaProvider.GetSelectorLabels()
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return err
	}
	podNames := controllers.GetPodNames(foundPodList.Items)
	sort.Strings(podNames)
	return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafka.Kafka) {
		instance.Status.KafkaBrokerStatus.Brokers = podNames
	})
}

func (r *ReconcileKafka) waitUntilControllerIsReady(maxWaitingInterval int) error {
	r.logger.Info("Waiting for kafka-kraft-controller deployment.")
	time.Sleep(waitingInterval)
	err := wait.PollImmediate(waitingInterval, time.Duration(maxWaitingInterval)*time.Second, func() (done bool, err error) {
		labels := make(map[string]string)
		labels["name"] = fmt.Sprintf("%s-%s", r.cr.Name, "kraft-controller")
		labels["component"] = "kafka-controller"
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		r.logger.Error(err, "Deployment kafka-kraft-controller failed.")
		return err
	}
	return nil
}

func (r *ReconcileKafka) waitUntilBrokerIsReady(brokerId int, maxWaitingInterval int) error {
	r.logger.Info(fmt.Sprintf("Waiting for kafka-%d deployment.", brokerId))
	time.Sleep(waitingInterval)
	err := wait.PollImmediate(waitingInterval, time.Duration(maxWaitingInterval)*time.Second, func() (done bool, err error) {
		kafkaLabels := r.kafkaProvider.GetSelectorLabels()
		kafkaLabels["name"] = fmt.Sprintf("%s-%d", r.cr.Name, brokerId)
		return r.reconciler.AreDeploymentsReady(kafkaLabels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		r.logger.Error(err, fmt.Sprintf("Deployment kafka-%d failed.", brokerId))
		return err
	}
	return nil
}

func (r *ReconcileKafka) waitUntilMigrationCompleted(maxWaitingInterval int) error {
	r.logger.Info("Waiting for ZooKeeper to Kraft migration to complete.")
	time.Sleep(waitingInterval)
	err := wait.PollImmediate(waitingInterval, time.Duration(maxWaitingInterval)*time.Second, func() (done bool, err error) {
		return r.getMigrationStatus(), nil
	})
	if err != nil {
		r.logger.Error(err, "ZooKeeper to Kraft migration not completed")
		return err
	}
	return nil
}

func (r ReconcileKafka) addDeploymentAnnotation(deployment *appsv1.Deployment, annotationName string, annotationValue string) {
	r.logger.Info(fmt.Sprintf("Add annotation '%s: %s' to deployment '%s'",
		annotationName, annotationValue, deployment.Name))
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[annotationName] = annotationValue
}
