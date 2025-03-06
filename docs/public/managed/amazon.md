[[_TOC_]]

# Introduction

Kafka Service chart allows you to deploy Kafka side services (AKHQ, Monitoring) without deploying Kafka, using Amazon Managed Streaming for
Apache Kafka (MSK) bootstrap servers.

# Prerequisites

## Global

* External Kafka bootstrap servers are available from Kubernetes cluster where you are going to deploy side services.
* Monitoring services and prometheus-operator should be pre-installed.

## Preparations for Monitoring

To monitor Amazon Kafka you need to perform the following steps before deploy:

1. Enable Open monitoring on MSK in AWS Console:

    * Navigate to your MSK cluster in AWS Console.
    * Edit Monitoring configuration and enable the following options: `Enable open monitoring with Prometheus`, `JMX Exporter`
      and `Node Exporter`.
      ![Open Monitoring Config](../images/open_monitoring.png)
    * Save changes.
    * Enable access to the monitoring ports (`11001` and `11002`) in corresponding VPC Security Group.
      ![Security Groups](../images/security_groups.png)
    * Check that monitoring endpoints are available from EKS cluster. For example:

      ```bash
      curl http://b-3.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11001
      curl http://b-3.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11002/metrics
      ```
  
    More information about Open Monitoring for Amazon MSK in 
    [Open Monitoring with Prometheus](https://docs.aws.amazon.com/msk/latest/developerguide/open-monitoring.html).

2. Configure Prometheus scrappers for Kafka exporters:

    * Navigate to `monitoring` namespace in EKS.
    * Find scrape-config secret:
      * `additional-scrape-config` if Prometheus is used.
      * `vm-additional-scrape-config` if VictoriaMetrics is used.
    * Edit the secret and add the section with information about Kafka monitoring endpoints.
    For example:
    
        ```yaml
        - job_name: kafka-node-exporter
          honor_timestamps: true
          scrape_interval: 30s
          scrape_timeout: 10s
          metrics_path: /metrics
          scheme: http
          static_configs:
            - targets:
                - b-1.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11002
              labels:
                service: kafka-1
                namespace: kafka-service
            - targets:
                - b-2.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11002
              labels:
                service: kafka-2
                namespace: kafka-service
            - targets:
                - b-3.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11002
              labels:
                service: kafka-3
                namespace: kafka-service
          tls_config:
            insecure_skip_verify: true
          metric_relabel_configs:
            - source_labels: [__name__]
              action: replace
              regex: node_exporter_build_info
              replacement: node_uname_info
              target_label: __name__
        - job_name: kafka-exporter
          honor_timestamps: true
          scrape_interval: 30s
          scrape_timeout: 10s
          metrics_path: /
          scheme: http
          static_configs:
            - targets:
                - b-1.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11001
              labels:
                service: kafka-1
                broker: kafka-1
                namespace: kafka-service
            - targets:
                - b-2.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11001
              labels:
                service: kafka-2
                broker: kafka-2
                namespace: kafka-service
            - targets:
                - b-3.test-kafka-cluster.test.kafka.us-east-1.amazonaws.com:11001
              labels:
                service: kafka-3
                broker: kafka-3
                namespace: kafka-service
          tls_config:
            insecure_skip_verify: true
        ```

        **Note:** Value of `job_name` and names of labels in `static_configs` are required and cannot be changed.
        
        **Pay attention**: This is an example, do not copy it as-is for your deployment. You need to change urls and number of brokers with
        corresponding labels for your specific configuration.

    * Save changes and make sure new targets have appeared in Prometheus or VictoriaMetrics.

## Preparations for Backup

**Note:** Using Amazon S3 is not required, but preferred option for storing Kafka configuration backups. RWO|RWX storage class can also be 
used.

To store backups to Amazon S3 storage you need to perform the following steps before deploy:

1. Create policy in AWS Console:

    * Navigate to IAM dashboard in AWS Console.
    * Choose **Policies** in the navigation pane on the left and press the **Create Policy** button.
      ![IAM Policies](../images/iam_policies.png)
    * Choose the **JSON** tab.
      ![Create Policy](../images/create_policy.png)
    * Paste the following policy document:

      ```yaml
        {
          "Version": "2012-10-17",
          "Statement": [
            {
              "Sid": "VisualEditor0",
              "Effect": "Allow",
              "Action": "s3:*",
              "Resource": [
                  "arn:aws:s3:::kafka-backups",
                  "arn:aws:s3:::kafka-backups/*"
              ]
            }
          ]
        }
      ```
    
      For details about the IAM policy language, see 
      [IAM JSON policy reference](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies.html).
    * When you are finished, press the **Next: Tags** button and then the **Next: Review** button.
    * On the **Review policy** page, type a **Name** and a **Description** (optional) for the policy that you are creating.
    * Press the **Create policy** button to save your work.

   More information about IAM policy creation in 
   [Creating IAM policies (console)](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_create-console.html).

2. Create user in AWS Console:

   * Navigate to IAM dashboard in AWS Console.
   * Choose **Users** in the navigation pane on the left and press the **Add users** button.
     ![IAM Users](../images/iam_users.png)
   * Type the **User name** for the new user and select **Programmatic access** checkbox.
     ![Set User Details](../images/set_user_details.png)
   * Press the **Next: Permissions** button.
   * On the **Set permissions** page, choose the **Attach existing policies directly** option.
     ![Set Permissions](../images/set_permissions.png)
   * Select the policy that you created before.
   * Press the **Next: Tags** button and then the **Next: Review** button.
   * When you are reviewed, press the **Create user** button to save your work.
     ![User Credentials](../images/user_credentials.png)
   * Store the user access keys (access key IDs and secret access keys).

   More information about IAM user creation in 
   [Creating an IAM user in your AWS account](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html).

3. Create bucket of S3 storage in AWS Console:

    * Navigate to your S3 storage in AWS Console.
    * Press the **Create bucket** button.
      ![S3 Buckets](../images/s3_buckets.png)
    * Type the **Bucket name** and choose the **AWS Region** where you want the bucket to be located.
      ![Create Bucket](../images/create_bucket.png)
    * When you are finished, press the **Create bucket** button to save your work.

   More information about S3 bucket creation in 
   [Creating a bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html).

## Preparation for Cruise Control

Prepare monitoring according to [Preparation for Monitoring](#preparations-for-monitoring).

Specify appropriate `cruiseControl.prometheusServerEndpoint` parameter. 

For instance `cruiseControl.prometheusServerEndpoint: prometheus-operated.monitoring:9090` in case of Prometheus using.

For instance `cruiseControl.prometheusServerEndpoint: vmauth-k8s.monitoring:8427` in case of VictoriaMetrics using.

# Example of Deploy Parameters

Example of deployment parameters for external Amazon MSK is presented below:

```yaml
global:
  name: kafka

  externalKafka:
    enabled: true
    bootstrapServers: "b-1.test-kafka-cluster.amazonaws.com:9094,b-2.test-kafka-cluster.us-east-1.amazonaws.com:9094,b-3.test-kafka-cluster.amazonaws.com:9094"
    enableSsl: true
    username: ""
    password: ""
  secrets:
    akhq:
      defaultUsername: "admin"
      defaultPassword: "admin"
    backupDaemon:
      s3:
        keyId: "AKIA2T3JJNXHAEF24HJN"
        keySecret: "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"
operator:
  replicas: 1
  kmmConfiguratorEnabled: false
kafka:
  install: false
monitoring:
  install: true
  kafkaTotalBrokerCount: 3
  securityContext: {
    "runAsUser": 1000
  }
  resources:
    requests:
      memory: 128Mi
      cpu: 50m
    limits:
      memory: 256Mi
      cpu: 200m
  lagExporter:
    enabled: true
    pollIntervalSeconds: 30
    cluster:
      name: "kafka"
      topicWhitelist:
        - ".*"
      groupWhitelist:
        - ".*"
      consumerProperties:
        security.protocol: SSL
      adminClientProperties:
        security.protocol: SSL
akhq:
  install: true
  ingress:
    host: akhq-kafka-service.aws.test.example.com
  securityContext: {
    "runAsUser": 1000
  }
  resources:
    requests:
      memory: 400Mi
      cpu: 50m
    limits:
      memory: 500Mi
      cpu: 300m
backupDaemon:
  install: true
  s3:
    enabled: true
    url: "https://s3.amazonaws.com"
    bucket: "kafka-backups"
  backupSchedule: "0 0 * * *"
  evictionPolicy: "0/1d,7d/delete"
  resources:
    requests:
      memory: 64Mi
      cpu: 25m
    limits:
      memory: 256Mi
      cpu: 200m
cruiseControl:
  install: true
  prometheusServerEndpoint: prometheus-operated.monitoring:9090
```

**NOTE:** This is an example, do not copy it as-is for your deployment, be sure about each parameter in your installation.

# Amazon MSK Features

## Security

Amazon MSK uses `SSL` security mechanism by default. It means that channel is encrypted but authentication is not required.
More detailed in [Amazon MSK Security](https://docs.aws.amazon.com/msk/latest/developerguide/security.html).

## Monitoring

Amazon MSK provides less information about its metrics, thus a lot of `Kafka Monitoring` dashboard panels are not applicable for it.
More information in [Monitoring](../monitoring.md).
