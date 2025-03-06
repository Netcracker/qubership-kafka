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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/go-logr/logr"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	backupDaemonConditionReason = "KafkaBackupDaemonReadinessStatus"
	scaleMessageTemplate        = "Timeout occurred during Backup Daemon scale %s"
	scaleTimeout                = time.Duration(120) * time.Second
	backupRestoreTimeout        = time.Duration(60) * time.Second
	certificateFilePath         = "/backupTLS/ca.crt"
)

type ReconcileBackupDaemon struct {
	cr              *kafkaservice.KafkaService
	reconciler      *KafkaServiceReconciler
	logger          logr.Logger
	username        string
	password        string
	backupDaemonUrl string
}

type StatusResponse struct {
	Status string `json:"status"`
	Vault  string `json:"vault"`
	Type   string `json:"type"`
	Err    string `json:"err"`
	TaskId string `json:"task_id"`
}

type BackupInfoResponse struct {
	IsGranular bool              `json:"is_granular"`
	DBList     string            `json:"db_list"`
	Id         string            `json:"id"`
	Failed     bool              `json:"failed"`
	Locked     bool              `json:"locked"`
	TS         int               `json:"ts"`
	SpentTime  string            `json:"spent_time"`
	Valid      bool              `json:"valid"`
	Evictable  bool              `json:"evictable"`
	CustomVars map[string]string `json:"custom_vars,omitempty"`
}

func NewReconcileBackupDaemon(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger) ReconcileBackupDaemon {
	return ReconcileBackupDaemon{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r ReconcileBackupDaemon) Reconcile() error {
	kafkaServicesSecret, err := r.reconciler.WatchSecret(fmt.Sprintf("%s-services-secret", r.cr.Name), r.cr, r.logger)
	if err != nil {
		return err
	}
	backupDaemonDeployment, err := r.reconciler.FindDeployment(fmt.Sprintf("%s-backup-daemon", r.cr.Name), r.cr.Namespace, r.logger)
	if err != nil {
		return err
	}
	if kafkaServicesSecret.Annotations != nil && kafkaServicesSecret.Annotations[autoRestartAnnotation] == "true" {
		r.addDeploymentAnnotation(backupDaemonDeployment, fmt.Sprintf(resourceVersionAnnotationTemplate, kafkaServicesSecret.Name), kafkaServicesSecret.ResourceVersion)
	}
	r.logger.Info("Updating the found Deployment",
		"Deployment.Namespace", backupDaemonDeployment.Namespace, "Deployment.Name", backupDaemonDeployment.Name)
	if err := r.reconciler.Client.Update(context.TODO(), backupDaemonDeployment); err != nil {
		return err
	}
	if r.cr.Spec.DisasterRecovery == nil || !r.cr.Spec.DisasterRecovery.TopicsBackup.Enabled {
		r.logger.Info("TopicsBackup is not enabled, skipping disaster recovery operations.")
		return nil
	}
	if r.cr.Status.DisasterRecoveryStatus.Mode != "" && r.cr.Status.DisasterRecoveryStatus.Mode != r.cr.Spec.DisasterRecovery.Mode || r.cr.Status.DisasterRecoveryStatus.Status == "failed" {
		r.username, r.password = r.getBackupDaemonCredentials()
		_, err := os.ReadFile(certificateFilePath)
		if err == nil {
			r.backupDaemonUrl = fmt.Sprintf("https://%s-backup-daemon.%s:8443", r.cr.Name, r.cr.Namespace)
		} else {
			r.backupDaemonUrl = fmt.Sprintf("http://%s-backup-daemon.%s:8080", r.cr.Name, r.cr.Namespace)
		}
		r.logger.Info(fmt.Sprintf("Start switchover with mode: %s and no-wait: %t, current status mode is: %s",
			r.cr.Spec.DisasterRecovery.Mode,
			r.cr.Spec.DisasterRecovery.NoWait,
			r.cr.Status.DisasterRecoveryStatus.Mode))
		if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "active" {
			r.logger.Info("Scaling up backup daemon")
			err := r.scaleDeploymentWithCheck(1, waitingInterval, scaleTimeout)
			if err != nil {
				return err
			}
			r.logger.Info("Backup Daemon started")

			r.logger.Info("Restoring last backup")
			lastFullBackup, jobId, err := r.restoreLastBackup(waitingInterval, backupRestoreTimeout)
			if err != nil {
				return err
			}

			if lastFullBackup != "" {
				if err = r.checkRestoreStatus(jobId, waitingInterval, backupRestoreTimeout); err != nil {
					return err
				}
			}
		} else if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "standby" &&
			strings.ToLower(r.cr.Status.DisasterRecoveryStatus.Mode) != "disable" {
			r.logger.Info("Backup started")
			vaultId, err := r.performBackup(waitingInterval, backupRestoreTimeout)
			if err != nil {
				return err
			}

			r.logger.Info(fmt.Sprintf("Backup was performed: %s, check status", vaultId))
			if err = r.checkBackupStatus(vaultId, waitingInterval, backupRestoreTimeout); err != nil {
				return err
			}

			r.logger.Info("Backup Daemon scale-down started")
			if err = r.scaleDeploymentWithCheck(0, waitingInterval, backupRestoreTimeout); err != nil {
				return err
			}
			r.logger.Info("Backup Daemon scale-down completed")
		} else if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "disable" {
			r.logger.Info("Backup Daemon scale-down started for disable mode")
			if err := r.scaleDeploymentWithCheck(0, waitingInterval, backupRestoreTimeout); err != nil {
				return err
			}
			r.logger.Info("Backup Daemon scale-down for disable mode completed")
		}
		r.logger.Info("Switchover finished successfully")
		return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
			instance.Status.DisasterRecoveryStatus.Status = "done"
			instance.Status.DisasterRecoveryStatus.Message = ""
		})
	}
	return nil
}

func (r ReconcileBackupDaemon) restoreLastBackup(interval, timeout time.Duration) (string, string, error) {
	var lastFullBackup string
	var jobId string
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		response, err := r.processRequest("GET", fmt.Sprintf("%s/listbackups", r.backupDaemonUrl), nil)
		if err != nil {
			r.logger.Error(err, "Can't get a list of backups, attempt failed")
			return false, nil
		}

		backupList, err := r.getBackupList(response)
		if err != nil {
			r.logger.Error(err, "Can't get a list of backups, attempt failed")
			return false, nil
		}

		lastFullBackup, err = r.getLastFullBackup(backupList)
		if err != nil {
			return false, nil
		}

		if lastFullBackup == "" {
			if r.cr.Spec.DisasterRecovery.NoWait {
				r.logger.Error(err, "There is no backup to restore. Switchover process will be continued because noWait parameter is True")
				return true, nil
			} else {
				r.logger.Error(err, "There is no backup to restore, attempt failed")
				return false, nil
			}
		}

		data := map[string]string{}
		data["vault"] = lastFullBackup
		byteData, err := json.Marshal(data)
		if err != nil {
			return false, nil
		}
		r.logger.Info(fmt.Sprintf("Trying to restore backup %s", lastFullBackup))
		response, err = r.processRequest("POST", fmt.Sprintf("%s/restore", r.backupDaemonUrl), bytes.NewBuffer(byteData))
		if err != nil {
			r.logger.Error(err, "Restore attempt failed")
			return false, nil
		}

		jobId = string(response)
		return true, nil
	})
	if err != nil {
		r.logger.Error(err, "Restore request failed")
		return "", "", err
	}
	return lastFullBackup, jobId, nil
}

func (r ReconcileBackupDaemon) checkRestoreStatus(jobId string, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		r.logger.Info(fmt.Sprintf("Check restore status of job %s", jobId))

		response, err := r.processRequest("GET", fmt.Sprintf("%s/jobstatus/%s", r.backupDaemonUrl, jobId), nil)
		if err != nil {
			r.logger.Error(err, "Job status check failed")
			return false, nil
		}

		restoreStatus, err := r.getBackupStatus(response)
		if err != nil {
			return false, nil
		}

		if restoreStatus != "Successful" {
			r.logger.Error(err, "Restore job is not successful")
			return false, nil
		}
		r.logger.Info("Restore successful")
		return true, nil
	})
	if err != nil {
		r.logger.Error(err, "Restore failed")
		return err
	}
	return nil
}

func (r ReconcileBackupDaemon) performBackup(interval, timeout time.Duration) (string, error) {
	var vaultId string
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		response, err := r.processRequest("POST", fmt.Sprintf("%s/backup", r.backupDaemonUrl), nil)
		if err != nil {
			return false, nil
		}
		vaultId = string(response)
		return true, nil
	})
	if err != nil {
		r.logger.Error(err, "Backup request failed")
		return "", err
	}
	return vaultId, nil
}

func (r ReconcileBackupDaemon) checkBackupStatus(vaultId string, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		response, err :=
			r.processRequest("GET", fmt.Sprintf("%s/jobstatus/%s", r.backupDaemonUrl, vaultId), nil)
		if err != nil {
			return false, nil
		}

		backupStatus, err := r.getBackupStatus(response)
		if err != nil {
			return false, nil
		}
		if backupStatus != "Successful" {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		r.logger.Error(err, "Backup failed")
		return err
	}
	r.logger.Info(fmt.Sprintf("Backup %s was performed successfully", vaultId))
	return nil
}

func (r ReconcileBackupDaemon) scaleDeploymentWithCheck(replicas int32, interval, timeout time.Duration) error {
	err := r.scaleDeployment(replicas)
	if err != nil {
		r.logger.Error(err, "Deployment update failed")
		return err
	}
	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		labels := r.GetBackupDaemonLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		direction := "up"
		if replicas == 0 {
			direction = "down"
		}
		r.logger.Error(err, fmt.Sprintf(scaleMessageTemplate, direction))
		return err
	}
	return nil
}

func (r ReconcileBackupDaemon) Status() error {
	if r.cr.Spec.DisasterRecovery != nil && r.cr.Spec.DisasterRecovery.Mode == "standby" && r.cr.Spec.DisasterRecovery.TopicsBackup.Enabled {
		r.logger.Info("Skip Kafka Backup Daemon status check because of standby mode")
		return nil
	}
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		backupDaemonConditionReason,
		"Kafka Backup Daemon health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of Kafka Backup Daemon pod")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.Global.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.GetBackupDaemonLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, backupDaemonConditionReason, "Kafka backupDaemonConditionReason pod is not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, backupDaemonConditionReason, "Kafka backupDaemonConditionReason pod is ready"))
}

func (r *ReconcileBackupDaemon) processRequest(method string, url string, body io.Reader) ([]byte, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	caCert, err := os.ReadFile(certificateFilePath)
	if err == nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return responseData, nil
}

func (r *ReconcileBackupDaemon) scaleDeployment(replicas int32) error {
	backupDaemonDeployment, err := r.reconciler.FindDeployment(fmt.Sprintf("%s-backup-daemon", r.cr.Name), r.cr.Namespace, r.logger)
	if err == nil {
		backupDaemonDeployment.Spec.Replicas = pointer.Int32Ptr(replicas)
		err := r.reconciler.Client.Update(context.TODO(), backupDaemonDeployment)
		return err
	}
	return err
}

func (r *ReconcileBackupDaemon) getBackupDaemonCredentials() (string, string) {
	foundSecret, err := r.reconciler.FindSecret(fmt.Sprintf("%s-backup-daemon-secret", r.cr.Name), r.cr.Namespace, r.logger)
	if err != nil {
		return "", ""
	}
	username := string(foundSecret.Data["username"])
	password := string(foundSecret.Data["password"])
	return username, password
}

func (r ReconcileBackupDaemon) GetBackupDaemonLabels() map[string]string {
	return map[string]string{
		"component": "kafka-backup-daemon",
		"name":      fmt.Sprintf("%s-backup-daemon", r.cr.Name),
	}
}

func (r *ReconcileBackupDaemon) getBackupStatus(response []byte) (status string, err error) {
	var backupStatusResponse StatusResponse
	err = json.Unmarshal(response, &backupStatusResponse)
	if err != nil {
		return "", err
	}
	return backupStatusResponse.Status, nil
}

func (r *ReconcileBackupDaemon) getBackupList(response []byte) (backupList []string, err error) {
	var backupListResponse []string
	err = json.Unmarshal(response, &backupListResponse)
	if err != nil {
		return []string{}, err
	}
	return backupListResponse, nil
}

func (r *ReconcileBackupDaemon) getLastFullBackup(backups []string) (vaultId string, err error) {
	// Assuming that vault id of backup is close to timestamp and can be used to find the last full backup.
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	for _, backup := range backups {
		response, err := r.processRequest("GET", fmt.Sprintf("%s/listbackups/%s", r.backupDaemonUrl, backup), nil)
		if err != nil {
			return "", err
		}
		var backupInfoResponse BackupInfoResponse
		err = json.Unmarshal(response, &backupInfoResponse)
		if err != nil {
			return "", err
		}
		if backupInfoResponse.DBList == "full backup" {
			if len(backupInfoResponse.CustomVars) != 0 {
				if backupInfoResponse.CustomVars["region"] != r.cr.Spec.DisasterRecovery.Region {
					return backup, nil
				}
			}
		}
	}
	return "", nil
}

func (r ReconcileBackupDaemon) addDeploymentAnnotation(deployment *appsv1.Deployment, annotationName string, annotationValue string) {
	r.logger.Info(fmt.Sprintf("Add annotation '%s: %s' to deployment '%s'",
		annotationName, annotationValue, deployment.Name))
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[annotationName] = annotationValue
}
