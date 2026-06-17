AKHQ is an open source GUI for Apache Kafka to manage topics, topics data, consumer groups, Kafka Connect and more.
For more information and full list of features, refer to the [Official AKHQ Documentation](https://github.com/tchiotludo/akhq/tree/0.17.0).

# Protobuf Deserialization

To deserialize topics containing data in Protobuf format, you can set topics mapping as follows:

* For each `topic-regex`, you can specify `descriptor-file-base64`, the descriptor file encoded to Base64 format,
  or you can add the descriptor files in the `descriptors-folder` and specify the `descriptor-file` name.
* Specify corresponding message types for keys and values. If, for example, keys are not in Protobuf format,
  `key-message-type` can be omitted, the same for `value-message-type`.

To apply Protobuf configuration to AKHQ:

1. Prepare YAML configuration. For example:

   ```yaml
   deserialization:
     protobuf:          
       descriptors-folder: "/app/config"
       topics-mapping:            
         - topic-regex: "dev-.*|test_.*"
           descriptor-file: "streaming.desc"
           key-message-type: "com.qubership.protobuf.serialization.protocol.Row"
           value-message-type: "com.qubership.protobuf.serialization.protocol.Envelope"
         - topic-regex: "album.*"
           descriptor-file-base64: "Cs4BCgthbGJ1bS5wcm90bxIXY29tLm5ldGNyYWNrZXIucHJvdG9idWYidwoFQWxidW0SFAoFdGl0bGUYASABKAlSBXRpdGxlEhYKBmFydGlzdBgCIAMoCVIGYXJ0aXN0EiEKDHJlbGVhc2VfeWVhchgDIAEoBVILcmVsZWFzZVllYXISHQoKc29uZ190aXRsZRgEIAMoCVIJc29uZ1RpdGxlQiUKF2NvbS5uZXRjcmFja2VyLnByb3RvYnVmQgpBbGJ1bVByb3RvYgZwcm90bzM="
           value-message-type: "com.qubership.protobuf.Album"
   ```

   **Note**: Be careful with topic regexes, they should match only the desired topics.

   **Note**: The `streaming.desc` file is a descriptor file for Streaming Platform Protobuf message format.
   It exists in the `/app/config` folder of AKHQ.

   **Note**: To create a `.desc` file from `.proto` file execute the following command:

   ```bash
   protoc --descriptor_set_out=<desired_name_for_desc_file>.desc --include_imports <name_of_proto_file>.proto
   ```

   **Note**: To encode your custom `.desc` files to Base64 format,
   you can use online resource [base64encode.org](https://www.base64encode.org/) for encoding.
   
   **Note**: `key-message-type` and `value-message-type` use full-qualified proto names. For instance, we have the following proto file
   
   ```protobuf
   syntax = "proto3";
   package com.qubership.protobuf.serialization.protocol;

   option java_package = "com.qubership.proto";
   option java_outer_classname = "StreamingProtocol";
   option optimize_for = SPEED;
   
   message Row {
   // Row describing
   }
   
   message Envelope {
   // Envelop describing
   }
   ```
   
   Full-qualified names for `Row` and `Envelope` are `com.qubership.protobuf.serialization.protocol.Row` and
   `com.qubership.protobuf.serialization.protocol.Envelope` respectively.

   Copy the configuration you created.

2. Go to `akhq-protobuf-configuration` Config Map in OpenShift/Kubernetes and paste the configuration from the previous step in
   the `config` field.

   Or, alternatively, open the [akhq-protobuf-configuration.yaml](configs/akhq-protobuf-configuration.yaml), paste the configuration from
   the previous step instead of the provided example configuration, and apply this Config Map using the command:
   
   ```sh
   kubectl apply -f configs/akhq-protobuf-configuration.yaml -n ${NAMESPACE}
   ```
   
   where

   * `${NAMESPACE}` is the name of the Kafka namespace.

3. AKHQ pod reboots automatically for the change to take effect. 

   **Note**: The `akhq-protobuf-configuration` Config Map is not deleted on service upgrade,
   so the descriptor configuration is not removed on service upgrade.

   **Note**: For operator setup, the `akhq-protobuf-configuration` Config Map is deleted on Custom Resource deletion.
   In this case, the descriptor configuration is lost.

# Streaming Platform Protobuf Deserialization

To deserialize Streaming Platform messages produced in Protobuf format perform the steps described in
[Protobuf Deserialization](#protobuf-deserialization) section.

Use the following configuration and set the appropriate `topic-regex` that matches the topics containing data in Protobuf format:

```yaml
deserialization:
  protobuf:          
    descriptors-folder: "/app/config"
    topics-mapping:            
      - topic-regex: "set appropriate regex"
        descriptor-file: "streaming.desc"
        key-message-type: "com.qubership.protobuf.serialization.protocol.Row"
        value-message-type: "com.qubership.protobuf.serialization.protocol.Envelope"     
```
