This chapter describes the security audit logging for Kafka.

The following topics are covered in this chapter:

[[_TOC_]]

# Common Information

Audit logs let you track key security events within Kafka and are useful for compliance purposes or in the aftermath of a security breach.

# Configuration

Kafka audit logging is partially supported and enabled by default for a part of audit events.

## Example of Events

The audit log format for events are described further:

### Unauthenticated event

```text
[2024-08-14T16:15:37,980][INFO][category=org.apache.kafka.common.network.Selector] [SocketServer listenerType=ZK_BROKER, nodeId=1] Failed authentication with /0:0:0:0:0:0:0:1 (channelId=0:0:0:0:0:0:0:1:9093-0:0:0:0:0:0:0:1:59568-1060) (Authentication failed during authentication due to invalid credentials with SASL mechanism SCRAM-SHA-512)
```

### Unauthorized event

```text
[2024-08-14T16:27:11,308][WARN][category=com.qubership.kafka.security.authorization.ExtendedAuthorizer] Principal = [User:test] is Denied Operation = [DESCRIBE] from host = [0:0:0:0:0:0:0:1] on resource = [ResourcePattern(resourceType=TOPIC, name=some_topic, patternType=LITERAL)]```
```

### Change Password / CRUD Users

```text
[2024-08-14T16:25:18,826][INFO][category=kafka.server.ZkConfigManager] Processing override for entityPath: users/test with config: Map(SCRAM-SHA-256 -> [hidden], SCRAM-SHA-512 -> [hidden])
```

### Modify Grants

```text
[2024-08-14T16:36:31,699][INFO][category=com.qubership.kafka.security.authorization.ExtendedAclAuthorizer] Processing Acl change notification for ResourcePattern(resourceType=GROUP, name=*, patternType=LITERAL), versionedAcls : Set(User:client has ALLOW permission for operations: ALL from hosts: *), zkVersion : 0
```
