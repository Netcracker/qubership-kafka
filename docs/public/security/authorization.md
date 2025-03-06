Authorizer implementation uses ZooKeeper to store all the Access Control Lists (ACLs). It is important to set ACLs
else access to resources is limited to superusers when an Authorizer is configured.
By default, if a resource has no associated ACLs, then no one except the superusers are allowed to access the resource.

Producers and consumers need to be authorized to perform operations on topics, but they should be
configured with different principals compared to the servers. The main operations that producers
require authorization to execute are `WRITE` and `READ`. Admin users can use command line tools,
but require authorization for certain operations. Operations that an admin user might need 
authorization for are `DELETE`, `CREATE`, and `ALTER`. You can use wildcards for producers and 
consumers so that you only have to set it once.

You can give topic and group wildcard access to users who have permission to access all topics and
groups, for example `admin users`. If you use this method, you do not have to create a separate rule for
each topic and group for the user. If you want to restrict the rights for user, it is necessary to
remember the following:

* To create a topic, the principal of the client requires the `CREATE` and `DESCRIBE`
  operations on the topic resource.
* To produce for a topic, the principal of the producer requires the `WRITE` operation on the topic
  resource.
* To consume from a topic, the principal of the consumer requires the `READ` operation on the topic
  and group resources.

**Note**: To be able to create, produce, and consume, the servers need to be configured with
the appropriate ACLs. The servers need authorization to update metadata (CLUSTER_ACTION) and to read
from a topic (READ) for replication purposes.

For more information about Kafka Authorization and ACLs, refer to the
_Apache Kafka Official Documentation_ at 
[http://kafka.apache.org/documentation/#security_authz](http://kafka.apache.org/documentation/#security_authz).

# ACL Operations

To enable authorization with ACLs, the authorizer class is specified in Kafka as `server.properties`
(under the property `authorizer.class.name`). It stores the collection of ACLs that have the ability 
to allow or deny operations for specific users. To add, remove, or list ACLs, the `bin/kafka-acls.sh` 
command line interface is used. If authentication is enabled in Kafka, a property file with credentials 
for a privileged user should be created and specified in the `--command-config` option for each command 
used in the `bin/kafka-acls.sh` CLI. For more information, refer to 
[Kafka Client Configuration](kafka.md#kafka-client-configuration).

For example, the following command provides wildcard access to the `client` user:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal User:client --operation All --topic '*' --group '*'
```

The `--allow-principal` option allows you to specify the principal in the `principalType:name` format. For 
`Basic` authorization, it is necessary to specify `User` as the type and the user's name as the name, 
because there is no other information about user rights. For `OAuth2.0` authorization, there 
is an ability to give users the rights for operations via the roles stored in the token, so 
the principal is configured with `Role` as the type and the role's name as the name.

For example, if it is necessary to allow users with `Administrator` role `Read` and `Write` access to any topic, use the following command:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal Role:Administrator --operation Read --operation Write --topic '*'
```

For more information about other `bin/kafka-acls.sh` CLI options, refer to `bin/kafka-acls.sh` CLI description or 
_Apache Kafka Official Documentation_ at 
[http://kafka.apache.org/documentation/#security_authz](http://kafka.apache.org/documentation/#security_authz).

## Adding ACLs

By default, all principals that do not have an explicit ACL that allows an operation to access a resource
are denied. In rare cases, where an ACL that allows access for all principals except of one, is desired, you can use the `--deny-principal`
and `--deny-host` options.
For example, use the following command to allow all users to `Read` from `test-topic`, but only deny `User:BadClient`
from IP `198.51.100.3`:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal User:'*' --allow-host '*' --deny-principal User:BadClient \
  --deny-host 198.51.100.3 --operation Read --topic test-topic
```

**Note**: The `--allow-host` and `deny-host` options only support IP addresses (host names are not 
supported). Also note that IPv6 addresses are supported, and that you can use them in ACLs.

The preceding example adds ACLs to a topic by specifying `--topic <topic-name>` as the resource 
pattern option. Similarly, one can add ACLs to a cluster by specifying `--cluster` and to a group 
by specifying `--group <group-name>`.
You can add ACLs on prefixed resource patterns. For example, you can add an ACL for the user `client`
to produce any topic whose name starts with `test-`. You can do that by executing the CLI with
the following options:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal User:client --producer --topic test- --resource-pattern-type prefixed
```

**Note**: The `--resource-pattern-type` is `literal` by default, which only affects resources with the
exact same name or in the case of the wildcard resource name "*", a resource with any name.

## Removing ACLs

You can remove ACLs. It is similar to adding, but the `--remove` option should be specified instead of `--add`.
If you want to remove the ACL added to the prefixed resource pattern, execute the CLI with
the following options:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --remove --allow-principal User:client --producer --topic test- --resource-pattern-type prefixed
```

## Listing ACLs

You can list the ACLs for a given resource by specifying the `--list` option and the resource.
For example, to list all ACLs for `test-topic`, execute the following command:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --list --topic test-topic
```

However, this only returns the ACLs that have been added to this exact resource pattern. Other ACLs
can exist that affect access to the topic, for example any ACLs on the topic wildcard "*", or any ACLs on
prefixed resource patterns. ACLs on the wildcard resource pattern can be queried explicitly:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties --list
```

It is not necessarily possible to explicitly query for ACLs on prefixed resource patterns that match
`test-topic`, as the name of such patterns may not be known. We can list all ACLs affecting `test-topic`
by using `--resource-pattern-type match`. For example:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --list --topic test-topic --resource-pattern-type match
```

## Adding or Removing a Principal as Producer or Consumer

For ACL management, you can add/remove a principal as a producer or consumer.
To add `User:client` as a producer of `test-topic`, execute the following command:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal User:client --producer --topic test-topic
```

To add `User:client` as a consumer of `test-topic` with group `kafka-group`, specify the
`--consumer` and `--group` options:

```sh
bin/kafka-acls.sh \
  --bootstrap-server localhost:9092 --command-config ./bin/adminclient.properties \
  --add --allow-principal User:client --consumer --topic test-topic --group kafka-group
```

To remove a principal from a producer or consumer role, specify the `--remove` option.
