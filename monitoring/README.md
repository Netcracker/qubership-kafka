# Overview
kafka-monitoring is extended original telegraf image for monitoring of kafka cluster.

# Description

This service provides custom telegraf sh plugin which collect metrics from dynamic cluster of kafka.
Dynamic means that cluster support scale up/down operations.

## What does plugin do?

1. Sh script get all broker from <kafka-root>/brokers/ids.
2. After from each broker script get response
3. Convert response to influx command line

## Telegraf configuration file
For including this sh input plugin, you need add next configuration in telegraf.conf file
```
 [[inputs.exec]]
   ## Commands array
   commands = [
     "python3 /opt/kafka-monitoring/exec-scripts/kafka_metric.py"
   ]

   ## Timeout for each command to complete.
   timeout = "10s"

   ## measurement name suffix (for separating different commands)
   name_prefix = "kafka_"

   ## Data format to consume.
   ## Each data format has it's own unique set of configuration options, read
   ## more about them here:
   ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
   data_format = "influx"
```
If you use OpenShift, prefix name parameter is set in `setEnv.sh` for OpenShift deployer script.

## Grafana
### Dashboard exporting
```bash
curl -XGET -k -u admin:admin http://localhost:3000/api/dashboards/db/kafka-monitoring \
  | jq . > dashboard/kafka-dashboard.json
```

  Where:
   
   * `admin:admin` grafana user login and password
   * `http://localhost:3000` grafana URL
   * `kafka-monitoring` dashboard name
 
### Dashboard importing
Dashboard can be imported using the following command:

```bash
curl -XPOST \
  -u admin:admin \
  --data @./dashboard/kafka-dashboard.json \
  -H 'Content-Type: application/json'  \
  -k \
   http://localhost:3000/api/dashboards/db
```
  Where:
   
   * `admin:admin` grafana user login and password
   * `http://localhost:3000` grafana URL


## Zabbix
Zabbix template for monitoring kafka cluster state consists of items and triggers for monitoring CPU, memory, disk usage and state of the cluster.

### Importing Template
Template can be imported in Zabbix UI from templates page (Configuration -> Templates) by using Import button.

Following macroses are used in template items and triggers and should be specified either on template or on hosts that this template is going to be used with:
* {$KAFKA_PROJECT_NAME} - name of openshift project where kafka is deployed
* {$KAFKA_PV_NAMES} - names of persistent volumes (as regex) that are used in kafka pods
* {$DR_SIDE} - name of side (`left`, `right`) if Kafka is deployed in DR mode. Should be empty if Kafka is not deployed in DR mode.

### Template Items and Triggers
Triggers for tracking following problems are included in template:
* *Cluster degradation*: Related item - *Kafka Offline Partition Count*. If offline partition count is not 0 then trigger will report problem with High severity.
* *Service is down*: Related item - *Kafka Cluster Size*. If cluster size is 0 then trigger will report problem with Disaster severity.
* *CPU usage threshold reached*: Related items - *Kafka CPU limit* and *Kafka CPU usage*. If CPU usage is more than 95% of the limit then trigger will report problem with High Severity.
* *Memory threshold reached*: Related item - *Kafka Memory Usage*. If memory usage is more than 95% of the limit then trigger will report problem with High Severity.
* *Disk space threshold reached*: Related item - *Kafka Disk Usage*. If disk usage is more than 90% of the limit then trigger will report problem with Average Severity or Disaster Severity if usage is more than 98%.

### DR Mode
If you have Kafka which is deployed in DR mode you need to create two hosts: 
for left and for right side and to specify the side as value (`left`, `right`) for the macros `{$DR_SIDE}`.
If you have Kafka without DR just leave this macros empty.
