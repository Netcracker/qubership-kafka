The following topics are covered in this chapter:

[[_TOC_]]

# Introduction

Cruise Control is a project that helps operate Kafka.

Cruise Control provides the following features out of the box:

* Balance Kafka cluster with respect to disk, network and CPU utilization.
* When a broker fails, automatically reassign replicas that were on that broker to other brokers in the cluster and restore the original
  replication factor.
* Identifying the topic-partitions that consume the most resources in the cluster.
* Support one-click cluster expansions and broker decommissions.
* Support heterogeneous Kafka clusters and multiple brokers per machine.

# REST API

Cruise control provides a REST API to allow users to interact with it.
The REST API supports querying the load and optimization proposals of the Kafka cluster, as well as triggering admin operations.

For full list of actions, refer to [Cruise Control REST API](https://github.com/linkedin/cruise-control/wiki/REST-APIs).

**Note**: REST API is protected with basic authentication by default, make sure you use appropriate credentials while working with
Cruise Control REST API. 

## Change Kafka topic configuration

You can check all the available parameters to change in the appropriate documentation
[Change Kafka Topic Configuration](https://github.com/linkedin/cruise-control/wiki/REST-APIs#change-kafka-topic-configuration).

Below is the example for how to change topic replication factor via REST API.

The standardized request is as follows:

```text
POST /kafkacruisecontrol/topic_configuration?topic=[topic_regex]&replication_factor=[target_replication_factor]
```

And the following is the example of POST request that should be invoked from the inside of the cruise-control pod via curl for `__TestTopic`

```sh
curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/topic_configuration?topic=__TestTopic&replication_factor=3&dryrun=false"
```

For more information, refer to
[Change Topic Replication Factor](https://github.com/linkedin/cruise-control/wiki/Change-topic-replication-factor-through-Cruise-Control).

When you try to decrease topic replication factor, you may have trouble reaching the default Goals:

See full Goals list [Cruise Control Goals](https://github.com/linkedin/cruise-control/wiki/Pluggable-Components#goals).

For example, we try to decrease replication factor from 3 to 1:

```sh
curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/topic_configuration?topic=test_topic&replication_factor=1&dryrun=false"
```

In the output we see something like this:

```text
Error processing POST request '/topic_configuration' due to: 'com.linkedin.kafka.cruisecontrol.exception.OptimizationFailureException: [DiskCapacityGoal] Insufficient capacity for disk (Utilization 2713.14, Allowed Capacity 2457.6
0, Threshold: 0.80). Add at least 1 broker with the same disk capacity (1024.00) as broker-1. || Tips: [1] The optimization is limited to replicas to be added to/removed from brokers. Potential mitigation: First, rebalance the clu
ster using REBALANCE endpoint with a superset of hard-goals defined via hard.goals config. Then, re-run your original request. Add at least 1 broker with the same disk capacity (1024.00) as broker-1.'.kafka-cruise-control-5f54f9c9
```

This means, that default DiskCapacityGoal is violated by changing replication factor operation, and you have to perform appropriate steps
described in the output to resolve this issue.

In case you can't do that, you can skip default goals and set appropriate goal that is not related to the problem, for instance, if you have
a problem with DiskCapacityGoal, you can set Repli
caDistributionGoal instead.

Below is the example, where we've skipped default goals check and provided another goal, which doesn't get violated by this operation:  

```bash
curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/topic_configuration?topic=test_topic&replication_factor=1&dryrun=false&skip_hard_goal_check=true&goals={ReplicaDistributionGoal}"
```

Here we've added following parameters:

* `skip_hard_goal_check=true` - Skips default Goals during topic configuration changing.
* `goals={ReplicaDistributionGoal}` - Instead of suing all Goals list, we explicitly set only one Goal, which doesn't get violated.

**Important** : We do not recommend avoiding capacity problems during RF changing in that way, because it might cause trouble in the future
during Cruise Control Proposals calculations.

## Rebalance Cluster

This section describes how to invoke cluster rebalance via REST API.

For full parameters' description, refer to
[Trigger workload balance](https://github.com/linkedin/cruise-control/wiki/REST-APIs#trigger-a-workload-balance).

**Note** : Rebalance is not possible until valid window prepared.

Following is the example request:

```bash
curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/rebalance
```

In the console output you will see how cluster would be balanced by Cruise Control proposals.

To make it actually start rebalance you have to disable `dryrun` mode, like in the following example:

```bash
curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/rebalance?dryrun=false"
```

Description for some crucial default parameters:

* `dryrun=true` - specifies whether to invoke rebalance or just show calculated rebalance result according to proposals and goals.
* `skip_hard_goal_check=false` - means that all the hard goals would be checked during rebalance. 
* `data_from=VALID_WINDOW` - whether generate proposal from available valid partitions or valid windows.

While you invoke rebalance, two possible scenarios might take place: 

1. Rebalance succeeded

    Then you'll see the output how cluster was rebelanced and which goals taken place like with `dryrun=true` option.

2. Error NotEnoughValidWindowsException

    In this case, rebalance hasn't started because there's not enough valid windows by which Cruise Control models its Kafka Cluster
    representation.
   
    As said previously, default data source for rebalance is `VALID_WINDOW`, for this mode this may occur for several reasons:

   * Cruise Control was just deployed, and it hasn't gathered enough samples from cluster yet: 
   
        You have to wait some time until it makes it's first window (usually it takes from 10 to 15 minutes).
   
        Also, you can manipulate window size during installation by `cruiseControl.config.broker.metrics.window.ms` and
        `cruiseControl.config.partition.metrics.window.ms` parameters.
   
   * Something is wrong with Kafka Cluster or its partitions:
   
        There might be a problem, for instance, with a large topic about which information can't be gathered. 
   
        By default, for window construction, it takes data only about valid partitions and the default parameter that measures the ratio for
        valid/all partitions is `min.valid.partition.ratio: 0.95`
     
        In this case, you can change ratio parameter like `min.valid.partition.ratio: 0.70`.
   
   In case none of this helped, you can use VALID_PARTITIONS mode. This mode uses only valid partitions for proposals construction and
   cluster re-balance.

## Kafka Broker Manipulation

1. Remove Broker

    The following POST request removes a list of brokers from the Kafka cluster:

    ```bash
    curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/remove_broker?brokerid=[id1,id2...]&dryrun=false"
    ```

   For detailed information, refer to
   [Decommission a list of brokers from the Kafka cluster](https://github.com/linkedin/cruise-control/wiki/REST-APIs#decommission-a-list-of-brokers-from-the-kafka-cluster)

2. Demote Broker

   The following POST request moves all the leader replicas away from a list of brokers:

    ```bash
    curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/demote_broker?brokerid=[id1, id2...]&dryrun=false"
    ```

   User can also request to move all the leader replicas away from the list of disks via:

    ```bash
    curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/demote_broker?brokerid_and_logdirs=[id1-logdir1, id2-logdir2...]&dryrun=false"
    ```
   
    For detailed information, refer to [Demote a list of brokers from the Kafka cluster](https://github.com/linkedin/cruise-control/wiki/REST-APIs#demote-a-list-of-brokers-from-the-kafka-cluster).

3. Add Broker

   The following POST request adds the given brokers to the Kafka cluster:

    ```bash
    curl -X POST -u admin:admin "http://localhost:9090/kafkacruisecontrol/add_broker?brokerid=[id1,id2...]&dryrun=false"
    ```

   For detailed information, refer tot [Add a list of new brokers to Kafka Cluster](https://github.com/linkedin/cruise-control/wiki/REST-APIs#add-a-list-of-new-brokers-to-kafka-cluster).

**Note** It means, these operations only applicable for existing brokers, means you have to make changes via REST API and then add/delete 
brokers via upgrade job.

Following is the short instruction of removing broker as an example:

* Make request via REST API to remove broker, in the console output you'll see that this broker doesn't contain partitions anymore.

    **Important** : In case you want to remove broker from cluster, appropriate way to do so is to remove broker with the highest ID,
    because during upgrade broker with the highest ID will be deleted.

    **Note** : Broker still exists as a kubernetes entity, Cruise Control can't manipulate K8s entities via its REST API.

* Make upgrade via deployer job and decrease replicas number for kafka via `kafka.replicas` parameter.

  **Note** Kafka Broker would be scaled-in only if `kafka.scaling.brokerDeploymentScaleInEnabled: true`.
   By default, this parameter is `false`.

* Run rebalance via REST API/Cruise Control UI or add broker to cluster via REST API 

# Cruise Control UI

Cruise Control UI is the project that provides convenient way to interact with Cruise Control REST API via GUI.

Full description can be found in the official documentation [Cruise Control UI](https://github.com/linkedin/cruise-control-ui/wiki).

On the `Cruise Control Proposals` tab you can find desired Kafka Cluster state by Cruise Control calculations.

On the `Cruise Control State` tab you can find current state of executor, monitor, analyzer and anomaly detector:

* Executor shows current task in progress, like rebalance, PLE, etc.

* Monitor shows total loaded kafka partitions, valid partitions, snapshots and how ready is the model in percentage.

* Analyzer shows Goals readiness.

* Anomaly Detector shows any detected problems with Brokers, Partitions and Goals Violations. Also, if `self.healing.enabled: true` it'll
  show for which issues it enables self-healing process.

## Rebalance Cluster

To run rebalance go the Kafka Cluster Administration tab and click `Rebalance Cluster` checkbox.

**Note** : Rebalance is not possible until valid window prepared.

![Cruise Control UI Rebalace Tab](/docs/public/images/rebalance.png).

You'll see Available Goals and Additional properties.

Those goals which are in bold considered as mandatory, but in case you want to use only few of all goals, you can activate
`Skip Hard Goal Check` checkbox.

So the least necessary configuration for rebalance is following:

* Choose at least one goal by which rebalance should be performed.

* Activate `Skip Hard Goal Check` checkbox if not all necessary goals needed.

* Disable `Dryrun` checkbox.

Then after Rebalance Execution there might be the same scenarios as for the REST API:

For instance, Error NotEnoughValidWindowsException.

There we have `Use Data From` parameter, that by Default have `Valiw Windows` value.

You can change it to `Valid Partitions` to perform rebalance by already processed partitions on the cluster.

## Kafka Broker Manipulation

On the same pages as rebalance you can click on the checkbox in broker list and then proceed with allowed operations: add/remove/demote

**Note** : During Remove operation you might face with the error `Insufficient number of racks to distribute each replica`.
This happens when there's no enough broker to satisfy RackAwareGoal after deleting the broker.

After remove operation, do the same steps as for REST API removal:

* Make upgrade via deployer job where `kafka.replicas` parameter was decreased.

* Start rebalance or add broker to cluster manually via GUI

# Self Healing

Self-healing is additional feature of Cruise Control which allows, if enabled, Cruise Control to deal with detected anomalies in
Kafka Cluster.

To enable self-healing you have to add `cruiseControl.config.self.healing.enabled=true` in `cruiseControl.config` map.

Here's the full list of possible anomalies and how to deal with them [Cruise Control Anomaly Detector](https://github.com/linkedin/cruise-control/wiki/Overview#anomaly-detector).

Among all the anomalies we're going to take a look at `Broker Failure` anomaly, because all the others do nothing in case there's no
explicit implementation.

You can explicitly configure threshold time to start anomaly fix execution by adding `broker.failure.alert.threshold.ms` and
`broker.failure.self.healing.threshold.ms` in `cruiseControl.Config` map.

In this case, after certain amount of time `broker.failure.alert.threshold.ms` broker will be considered as failed.

Then after defined delay `broker.failure.self.healing.threshold.ms` Cruise Control will start to move all the partitions from
the failed broker to others. Then failed broker will be deleted from the cluster.

By default, when you set `cruiseControl.config.self.healing.enabled=true`, self-healing enables for all possible anomalies.

You can explicitly enable/disable self-healing for each kind of anomaly.

For full parameters list, refer to
[Self Healing Configurations](https://github.com/linkedin/cruise-control/wiki/Configurations#selfhealingnotifier-configurations).

# External Kafka

This guide provides explanation how to install Kafka Service with Cruise Control with external Kafka Cluster.

Cruise Control requires configured monitoring, see
[Preparation for AWS Kafka Monitoring](managed/amazon.md#preparations-for-monitoring).

See main Kafka installation guide [Managed Kafka](managed/amazon.md).

For Cruise Control you have to provide following parameters:

* `global.externalKafka.enabled: true` - Configures Cruise Control to work with External Kafka cluster
* `cruiseControl.prometheusServerEndpoint: "prometheus-example.monitoring:9090"` - Endpoint for Cruise Control's PrometheusMetricCollector.

After that Cruise Control will be able to gather Kafka metrics from Prometheus to build its model and appropriately manage Kafka Cluster.
