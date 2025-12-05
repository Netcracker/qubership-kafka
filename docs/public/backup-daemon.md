This section provides information about backup and recovery procedures using API for Kafka Service.

Kafka Backup Daemon is based on common [Backup Daemon](https://github.com/Netcracker/qubership-backup-daemon) project, follow the link to find more information about its abilities.

For POST operations you must specify the username and password from `KAFKA_DAEMON_API_CREDENTIALS_USERNAME` and
`KAFKA_DAEMON_API_CREDENTIALS_PASSWORD` environment variables so that you can use REST API to run backup tasks.

# Backup

Kafka Backup Daemon provides abilities to create and restore snapshots of Kafka topic and ACL configuration.

**Note**: Only topic configuration stored during Kafka Service backup. Messages are not subject to back up.

You can back up data for Kafka Service per your requirement. You can select any one of the following options for backup:

* Full manual backup
* Granular backup
* Not Evictable Backup

## Full Manual Backup

To back up data of all Kafka topics, run the following command:

```
curl -u username:password -XPOST http://localhost:8080/backup
```

After executing the command, you receive a name of the folder where the backup is stored. For example, `20190321T080000`.

## Granular Backup

To back up data for specified topics, you can specify them in the `dbs` parameter. For example:

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"dbs":["topic1","topic2"]}' http://localhost:8080/backup
```

Alternatively, you can specify regular expression for topic names to back up in the `topic-regex` parameter. For example:

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"topic_regex":"topic[1-9]"}'  http://localhost:8080/backup
```

## Not Evictable Backup

If you do not want the backup to be evicted automatically, you need to add the `allow_eviction` property
with value as `False` in the request body. For example:

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"allow_eviction":"False"}' http://localhost:8080/backup
```

## Backup Eviction by ID

To remove a specific backup, run the following command:

```
curl -u username:password -XPOST http://localhost:8080/evict/<backup_id>
```

where `backup_id` is the name of specific backup to be evicted, for example, `20190321T080000`.
If the operation is successful, the following text displays: `Backup <backup_id> successfully removed`.

## Backup ACL

If you also want to back up configured ACL, you can specify `"mode":"acl"` parameter to enable ACL backup. In this case ACLs are stored additionally to topics configurations.

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"mode":"acl"}'  http://localhost:8080/backup
```

**Note**: ACL Backup/Restore requires authorization enabled in Kafka configurations to be performed. Make sure that `enableAuthorization` property is set to `true`. Check [Kafka Installation Guide](./installation.md#kafka).

## Backup Status

When the backup is in progress, you can check its status using the following command:

```
curl -XGET http://localhost:8080/jobstatus/<backup_id>
```

where `backup_id` is the backup name received at the backup execution step. The result is JSON with
the following information:

* `status` - Status of operation. The possible options are:
  * Successful
  * Queued
  * Processing
  * Failed
* `message` - Description of error (optional field)
* `vault` - The name of vault used in recovery
* `type` - The type of operation. The possible options are:
  * backup
  * restore
* `err` - None if no error, last 5 lines of log if `status=Failed`
* `task_id` - The identifier of the task

## Backup Information

To get the backup information, use the following command:

```
curl -XGET http://localhost:8080/listbackups/<backup_id>
```

where `backup_id` is the name of specific backup. The command returns JSON string with data about
specific backup:

* `ts` - The Unix timestamp of backup.
* `spent_time` - The time spent on backup (in ms)
* `db_list` - The list of stored topics (only for granular backup)
* `id` - The name of backup
* `size` - The size of backup in bytes
* `evictable` - Whether backup is evictable, _true_ if backup is evictable and _false_ otherwise
* `locked` - Whether backup is locked. _true_ if backup is locked (either process is not finished, or it failed somehow)
* `exit_code` - The exit code of backup script
* `failed` - Whether backup failed or not. _true_ if backup failed and _false_ otherwise
* `valid`- Backup is valid or not. _true_ if backup is valid and _false_ otherwise

# Recovery

To recover data from a specific backup, you need to specify JSON with information about a backup name (`vault`) or timestamp(`ts`, optional). 

If vault parameter presented timestamp will be ignored, if timestamp presented backup with equal or newer timestamp will be restored.

You must specify databases in the dbs list. You will not be able to run a recovery without database specified.

You also can start a recovery with specifying topics (`dbs`). In this case only snapshots for specified topics are restored.

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"vault":"20250513T095536"}' http://localhost:8080/restore
```

Alternatively, you can specify regular expression (`topic-regex`) for topic names to restore.

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"vault":"20190321T080000", "topic_regex":"topic[1-9]"}' http://localhost:8080/restore
```

Example of restore from backup with timestamp specified:

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d  '{"ts":"1689762600000"}' http://localhost:8080/restore
```

As a response, you receive `task_id`, which can be used to check _Recovery Status_.

**Note**: If some of provided topics are already exists in your project or cannot be created due to misconfiguration (for example, incorrect replication factor), 
no one topic created during this restore. Only full recovery of selected snapshot supported.

## Recovery ACL

If you also want to recover ACL snapshot, you can specify `"mode":"acl"` parameter to enable ACL recovery. In this case ACLs from snapshot are additionally restored with topics configurations.

```
curl -u username:password -XPOST -v -H "Content-Type: application/json" -d '{"vault":"20190321T080000", "mode":"acl"}'  http://localhost:8080/restore
```

**Note**: ACL Backup/Restore requires authorization enabled in Kafka configurations to be performed. Make sure that `enableAuthorization` property is set to `true`. Check [Kafka Installation Guide](installation.md#kafka).

## Recovery Status

When the recovery is in progress, you can check its status using the following command:

```
curl -XGET http://localhost:8080/jobstatus/<task_id>
```

where `task_id` is task ID received at the recovery execution step.

# Backups List

To receive list of collected backups, use the following command:

```
curl -XGET http://localhost:8080/listbackups
```

It returns JSON with list of backup names.

# Find Backup

To find the backup with timestamp equal or newer than specified, use the following command:

For full backups:

```
curl -XGET -u username:password -v -H "Content-Type: application/json" -d  '{"ts":"1689762600000"}' localhost:8080/find
```

For incremental backups:

```
curl -XGET -u username:password -v -H "Content-Type: application/json" -d  '{"ts":"1689762600000"}' localhost:8080/icremental/find
```

This command will return a JSON string with stats about particular backup or the first backup newer that specified timestamp:

* `ts`: Unix timestamp of backup
* `spent_time`: time spent on backup (in ms)
* `db_list`: List of backed up databases
* `id`: vault name
* `size`: Size of backup in bytes
* `evictable`: whether backup is evictable
* `locked`: whether backup is locked (either process isn't finished, or it failed somehow)
* `exit_code`: exit code of backup script
* `failed`: whether backup failed or not
* `valid`: is backup valid or not
* `is_granular`: Whether the backup is granular
* `sharded`: Whether the backup is sharded
* `custom_vars`: Custom variables with values that were used in backup preparation. It is specified if `.custom_vars` file exists in backup

An example is given below:

```
{"is_granular": false, "db_list": "full backup", "id": "20220113T230000", "failed": false, "locked": false, "sharded": false, "ts": 1642114800000, "exit_code": 0, "spent_time": "9312ms", "size": "25283b", "valid": true, "evictable": true, "custom_vars": {"mode": "hierarchical"}}
```

# Backup Daemon Health

To know the state of Backup Daemon, use the following command:

```
curl -XGET http://localhost:8080/health
```

You receive JSON with the following information:

```
"status": status of backup daemon   
"backup_queue_size": backup daemon queue size (if > 0 then there are 1 or tasks waiting for execution)
 "storage": storage info:
  "total_space": total storage space in bytes
  "dump_count": number of backup
  "free_space": free space left in bytes
  "size": used space in bytes
  "total_inodes": total number of inodes on storage
  "free_inodes": free number of inodes on storage
  "used_inodes": used number of inodes on storage
  "last": last backup metrics
    "metrics['exit_code']": exit code of script 
    "metrics['exception']": python exception if backup failed
    "metrics['spent_time']": spent time
    "metrics['size']": backup size in bytes
    "failed": is failed or not
    "locked": is locked or not
    "id": vault name of backup
    "ts": timestamp of backup  
  "lastSuccessful": last successfull backup metrics
    "metrics['exit_code']": exit code of script 
    "metrics['spent_time']": spent time
    "metrics['size']": backup size in bytes
    "failed": is failed or not
    "locked": is locked or not
    "id": vault name of backup
    "ts": timestamp of backup
```
