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

package kafkauser

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	kafka "github.com/Netcracker/qubership-kafka/operator/api/v1"
	"github.com/Netcracker/qubership-kafka/operator/controllers"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
	"time"
)

const (
	successState           = "success"
	failureState           = "failure"
	processingState        = "processing"
	disabledState          = "disabled"
	kafkaUserFinalizerName = "kafka-user-controller"
	specName               = "spec"
	labelsName             = "labels"
	annotationsName        = "annotations"
	bootstrapServersLabel  = "kafka.qubership.org/bootstrap.servers"
	authorizationDisabled  = "Security features are disabled"
	kafkaScheme            = "kafka"
	saslPlaintextProtocol  = "sasl_plaintext"
)

// KafkaUserReconciler reconciles a KafkaUser object
type KafkaUserReconciler struct {
	BootstrapServers      string
	Client                client.Client
	SecretCreatingEnabled bool
	Namespace             string
	ReconciliationPeriod  int
	ResourceHashes        map[string]string
	Scheme                *runtime.Scheme
	KafkaSecret           string
	KafkaSaslMechanism    string
	KafkaSslEnabled       bool
	KafkaSslSecret        string
	ApiGroup              string
}

//+kubebuilder:rbac:groups=qubership.org,resources=kafkausers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qubership.org,resources=kafkausers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qubership.org,resources=kafkausers/finalizers,verbs=update

func (r *KafkaUserReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := logf.Log.WithName("controller_kafka_user").
		WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling KafkaUser")
	kafkaUserFinalizer := fmt.Sprintf("%s/%s", r.ApiGroup, kafkaUserFinalizerName)
	instance := &kafka.KafkaUser{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllers.ApiGroupMatches(instance.APIVersion, r.ApiGroup) {
		return ctrl.Result{}, nil
	}

	specHashKey := util.JoinNames(instance.Namespace, instance.Name, specName)
	specHash, err := util.Hash(instance.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}
	labelsHashKey := util.JoinNames(instance.Namespace, instance.Name, labelsName)
	labelsHash, err := util.Hash(instance.Labels)
	if err != nil {
		return ctrl.Result{}, err
	}
	annotationsHashKey := util.JoinNames(instance.Namespace, instance.Name, annotationsName)
	annotationsHash, err := util.Hash(instance.Annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	customResourceChanged := r.ResourceHashes[specHashKey] != specHash ||
		r.ResourceHashes[labelsHashKey] != labelsHash ||
		r.ResourceHashes[annotationsHashKey] != annotationsHash

	customResourceUpdater := NewCustomResourceUpdater(r.Client, instance)

	if customResourceChanged {
		if err := customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
			cr.Status.State = processingState
			cr.Status.Message = "Processing of custom resource is in progress"
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	kafkaClient, err := r.newKafkaAdminClient(logger)
	if err != nil {
		logger.Error(err, "Reconciliation cycle failed")
		if err := customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
			cr.Status.State = failureState
			cr.Status.Message = fmt.Sprintf("Custom resource was not applied due to: %v", err)
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Duration(r.ReconciliationPeriod) * time.Second}, err
	}
	kafkaUserProvider := NewUserProvider(kafkaClient, logger)

	if !instance.DeletionTimestamp.IsZero() {
		if util.Contains(kafkaUserFinalizer, instance.GetFinalizers()) {
			username := fmt.Sprintf("%s_%s", instance.Namespace, instance.Name)
			logger.Info("Deleting Kafka User")
			reconcileError := kafkaUserProvider.deleteKafkaUser(username, instance.Spec.Authentication.Type)
			if reconcileError != nil {
				return r.processError(reconcileError, customResourceUpdater, logger)
			}
			logger.Info("Deleting Kafka User ACLs")
			reconcileError = kafkaUserProvider.deleteACLs(username)
			if reconcileError != nil {
				return r.processError(reconcileError, customResourceUpdater, logger)
			}
			if err := customResourceUpdater.UpdateWithRetry(func(cr *kafka.KafkaUser) {
				controllerutil.RemoveFinalizer(cr, kafkaUserFinalizer)
			}); err != nil {
				return ctrl.Result{}, err
			}
		}
		delete(r.ResourceHashes, specHashKey)
		delete(r.ResourceHashes, labelsHashKey)
		delete(r.ResourceHashes, annotationsHashKey)
		return ctrl.Result{}, nil
	}
	if !util.Contains(kafkaUserFinalizer, instance.GetFinalizers()) {
		if err := customResourceUpdater.UpdateWithRetry(func(cr *kafka.KafkaUser) {
			controllerutil.AddFinalizer(cr, kafkaUserFinalizer)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	if customResourceChanged {
		if instance.Spec.Authentication.Secret.Generate {
			if instance.Namespace != r.Namespace && !r.SecretCreatingEnabled {
				return r.processAuthenticationError(fmt.Errorf("grants to create secret in separate namespace are not provided"), customResourceUpdater, logger)
			}
			if reconcileError := r.generateSecret(instance, kafkaUserProvider, logger); reconcileError != nil {
				return r.processAuthenticationError(reconcileError, customResourceUpdater, logger)
			}
		}

		if instance.Spec.Authentication.WatchSecret == nil || *instance.Spec.Authentication.WatchSecret {
			if instance.Spec.Authentication.Secret == nil {
				return r.processAuthenticationError(fmt.Errorf("user secret is not specified"), customResourceUpdater, logger)
			}
			logger.Info("Collecting Kafka User secret")
			secret, reconcileError := r.FindSecret(instance.Spec.Authentication.Secret.Name, instance.Namespace, logger)
			if reconcileError != nil {
				return r.processAuthenticationError(reconcileError, customResourceUpdater, logger)
			}

			logger.Info("Extracting Kafka User connection properties")
			username, password, reconcileError := kafkaUserProvider.extractConnectionProperties(secret, instance.Spec.Authentication.Secret.Format)
			if reconcileError != nil {
				return r.processAuthenticationError(reconcileError, customResourceUpdater, logger)
			}

			if username != fmt.Sprintf("%s_%s", instance.Namespace, instance.Name) {
				reconcileError = fmt.Errorf("username must be specified in format {cr.namespace}_{cr.name}")
				return r.processAuthenticationError(reconcileError, customResourceUpdater, logger)
			}

			if secret.ResourceVersion != instance.Status.AuthenticationStatus.ResourceVersion {
				logger.Info("Creating Kafka User")
				reconcileError = kafkaUserProvider.upsertKafkaUser(username, password, instance.Spec.Authentication.Type)
				if reconcileError != nil {
					return r.processAuthenticationError(reconcileError, customResourceUpdater, logger)
				}
				logger.Info("Kafka user is updated")

				if err = customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
					cr.Status.AuthenticationStatus.State = successState
					cr.Status.AuthenticationStatus.ResourceVersion = secret.ResourceVersion
					if instance.Spec.Authentication.Secret.Format == connectionUriKey {
						cr.Status.AuthenticationStatus.ConnectionUri = kafka.SecretKey{Key: connectionUriKey, Name: instance.Spec.Authentication.Secret.Name}
					} else {
						cr.Status.AuthenticationStatus.Username = kafka.SecretKey{Key: usernameKey, Name: instance.Spec.Authentication.Secret.Name}
						cr.Status.AuthenticationStatus.Password = kafka.SecretKey{Key: passwordKey, Name: instance.Spec.Authentication.Secret.Name}
					}
				}); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			logger.Info("Secret watching is disabled for KafkaUser")
			if err = customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
				cr.Status.AuthenticationStatus.ConnectionUri = kafka.SecretKey{}
				cr.Status.AuthenticationStatus.Username = kafka.SecretKey{}
				cr.Status.AuthenticationStatus.Password = kafka.SecretKey{}
				cr.Status.AuthenticationStatus.ResourceVersion = ""
			}); err != nil {
				return ctrl.Result{}, err
			}
		}

		logger.Info("Creating Kafka ACLs")
		aclsCreated, reconcileError := kafkaUserProvider.createACLs(instance.Namespace, instance.Spec.Authorization.Role, fmt.Sprintf("%s_%s", instance.Namespace, instance.Name))
		if reconcileError != nil {
			if strings.Contains(reconcileError.Error(), authorizationDisabled) {
				logger.Info("Kafka Authorization is disabled")
				if err = customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
					cr.Status.AuthorizationStatus.State = disabledState
					cr.Status.AuthorizationStatus.Acls = []string{}
				}); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return r.processAuthorizationError(reconcileError, customResourceUpdater, logger)
			}
		} else {
			logger.Info("Kafka ACLs are updated")
			if err = customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
				cr.Status.AuthorizationStatus.State = successState
				cr.Status.AuthorizationStatus.Acls = aclsCreated
			}); err != nil {
				return ctrl.Result{}, err
			}
		}

		r.ResourceHashes[specHashKey] = specHash
		r.ResourceHashes[labelsHashKey] = labelsHash
		r.ResourceHashes[annotationsHashKey] = annotationsHash
	}

	if err := customResourceUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
		cr.Status.ObservedGeneration = instance.Generation
		cr.Status.State = successState
		cr.Status.Message = "Custom resource is successfully processed"
	}); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciliation cycle succeeded")
	return ctrl.Result{}, nil
}

func (r *KafkaUserReconciler) newKafkaAdminClient(logger logr.Logger) (sarama.ClusterAdmin, error) {
	adminUsername, adminPassword, err := r.getKafkaCredentials(logger)
	if err != nil {
		return nil, err
	}
	sslCertificates, err := r.getKafkaCertificates(logger)
	if err != nil {
		return nil, err
	}
	saslSettings := &controllers.SaslSettings{
		Mechanism: r.KafkaSaslMechanism,
		Username:  adminUsername,
		Password:  adminPassword,
	}
	kafkaClient, err := controllers.NewKafkaAdminClient(r.BootstrapServers, saslSettings, r.KafkaSslEnabled, sslCertificates)
	return kafkaClient, err
}

// FindSecret finds secret by name
func (r *KafkaUserReconciler) FindSecret(name string, namespace string, logger logr.Logger) (*corev1.Secret, error) {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] secret", name))
	foundSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundSecret)
	return foundSecret, err
}

func (r *KafkaUserReconciler) generateSecret(instance *kafka.KafkaUser, userProvider *UserProvider, logger logr.Logger) error {
	kafkaUserSecret := instance.Spec.Authentication.Secret.Name
	_, err := r.FindSecret(kafkaUserSecret, instance.Namespace, logger)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Creating a new secret - %s", kafkaUserSecret))

			username := fmt.Sprintf("%s_%s", instance.Namespace, instance.Name)
			password, err := userProvider.generatePassword()
			if err != nil {
				return err
			}
			var stringData map[string]string
			switch instance.Spec.Authentication.Secret.Format {
			case connectionPropertiesKey:
				stringData = map[string]string{
					"bootstrap.servers": r.BootstrapServers,
					"username":          username,
					"password":          password,
					"security.protocol": saslPlaintextProtocol,
					"sasl.mechanism":    instance.Spec.Authentication.Type,
				}
			case connectionUriKey:
				values := url.Values{
					"security.protocol": []string{saslPlaintextProtocol},
					"sasl.mechanism":    []string{instance.Spec.Authentication.Type},
				}
				connectionUri := url.URL{
					Scheme:   kafkaScheme,
					Host:     r.BootstrapServers,
					User:     url.UserPassword(username, password),
					RawQuery: values.Encode(),
				}
				stringData = map[string]string{
					"connection-uri": connectionUri.String(),
				}
			}
			userSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Spec.Authentication.Secret.Name,
					Namespace: instance.Namespace,
				},
				StringData: stringData,
			}

			return r.Client.Create(context.TODO(), &userSecret)
		} else {
			return err
		}
	}
	logger.Info(fmt.Sprintf("Secret [%s] already exists", kafkaUserSecret))
	return nil
}

func (r *KafkaUserReconciler) getKafkaCredentials(logger logr.Logger) (string, string, error) {
	foundSecret, err := r.FindSecret(r.KafkaSecret, r.Namespace, logger)
	if err != nil {
		return "", "", err
	}
	username := string(foundSecret.Data["admin-username"])
	password := string(foundSecret.Data["admin-password"])
	return username, password, nil
}

func (r *KafkaUserReconciler) getKafkaCertificates(logger logr.Logger) (*controllers.SslCertificates, error) {
	if r.KafkaSslEnabled && r.KafkaSslSecret != "" {
		sslCertificates, err := r.GetSslCertificates(r.KafkaSslSecret, r.Namespace, logger)
		return sslCertificates, err
	}
	return &controllers.SslCertificates{}, nil
}

// GetSslCertificates get ssl certificates from secret
func (r *KafkaUserReconciler) GetSslCertificates(secretName, namespace string, logger logr.Logger) (*controllers.SslCertificates, error) {
	foundSecret, err := r.FindSecret(secretName, namespace, logger)
	if err != nil {
		return nil, err
	}
	caCert := foundSecret.Data["ca.crt"]
	tlsCert := foundSecret.Data["tls.crt"]
	tlsKey := foundSecret.Data["tls.key"]
	if len(caCert) == 0 {
		return nil, fmt.Errorf("TLS certificates must be provided by secret with name: %s", secretName)
	}
	return &controllers.SslCertificates{CaCert: caCert, TlsCert: tlsCert, TlsKey: tlsKey}, nil
}

func (r *KafkaUserReconciler) processError(reconcileError error,
	crUpdater CustomResourceUpdater, logger logr.Logger) (ctrl.Result, error) {
	var result ctrl.Result
	var err error
	result.RequeueAfter = time.Duration(r.ReconciliationPeriod) * time.Second
	err = crUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
		cr.Status.State = failureState
		cr.Status.Message = fmt.Sprintf("During custom resource processing error occurred: %s",
			reconcileError.Error())
	})
	logger.Error(reconcileError, "Problem during custom resource reconciliation")
	return result, err
}

func (r *KafkaUserReconciler) processAuthenticationError(reconcileError error,
	crUpdater CustomResourceUpdater, logger logr.Logger) (ctrl.Result, error) {
	var result ctrl.Result
	var err error
	result.RequeueAfter = time.Duration(r.ReconciliationPeriod) * time.Second
	err = crUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
		cr.Status.State = failureState
		cr.Status.AuthenticationStatus.State = failureState
		cr.Status.Message = fmt.Sprintf("During custom resource processing error occurred: %s",
			reconcileError.Error())
	})
	logger.Error(reconcileError, "Problem during KafkaUser authentication")
	return result, err
}

func (r *KafkaUserReconciler) processAuthorizationError(reconcileError error,
	crUpdater CustomResourceUpdater, logger logr.Logger) (ctrl.Result, error) {
	var result ctrl.Result
	var err error
	result.RequeueAfter = time.Duration(r.ReconciliationPeriod) * time.Second
	err = crUpdater.UpdateStatusWithRetry(func(cr *kafka.KafkaUser) {
		cr.Status.State = failureState
		cr.Status.AuthorizationStatus.State = failureState
		cr.Status.Message = fmt.Sprintf("During custom resource processing error occurred: %s",
			reconcileError.Error())
	})
	logger.Error(reconcileError, "Problem during KafkaUser authorization")
	return result, err
}

// kafkaHostFilterFunction returns whether to handle CR depending on target Kafka cluster
func (r *KafkaUserReconciler) kafkaHostFilterFunction(annotations map[string]string) bool {
	if bootstrapServers, ok := annotations[bootstrapServersLabel]; ok {
		return bootstrapServers == r.BootstrapServers
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	statusPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
				!util.AreMapsEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}

	kafkaHostPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.kafkaHostFilterFunction(e.Object.GetAnnotations())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.kafkaHostFilterFunction(e.ObjectNew.GetAnnotations())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.kafkaHostFilterFunction(e.Object.GetAnnotations())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return r.kafkaHostFilterFunction(e.Object.GetAnnotations())
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kafka.KafkaUser{},
			builder.WithPredicates(predicate.And(statusPredicate, kafkaHostPredicate))).
		Complete(r)
}
