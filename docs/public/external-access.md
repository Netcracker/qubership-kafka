You can enable access to Kafka for external clients outside OpenShift or Kubernetes.

To enable external access to Kafka:

1. Create services with external IP addresses and ports for **each broker** manually or use the corresponding deployment parameter
   (`kafka.createExternalServices`) to create them automatically.
2. Specify parameters `kafka.externalHostNames` and `kafka.externalPorts` parameters and deploy Kafka. 

The following sections provide information about how to create services with external IP addresses and ports.

# NodePort

For development purposes, there is a way to provide external access to Kafka by using the `NodePort` service.

To configure Kafka with enabled external access:

1. You can create NodePort services manually or deploy procedure can do it. 

   To create `NodePort` service during installation you need to enable it with `kafka.createExternalServices`
   and fill node port numbers for each Kafka broker in parameter `kafka.externalPorts`. 
   Pay attention, the provided node ports should be free.

   The following is an example of template file which you can use to create NodePort services 
   for three-nodes Kafka (`external-template.yaml`) manually:

   ```yaml
    kind: Service
    apiVersion: v1
    metadata:
      name: external-kafka-1
    spec:
      type: NodePort
      externalTrafficPolicy: Local
      ports:
        - name: kafka-external-client
          port: 9094
          protocol: TCP
          targetPort: 9094
      selector:
        name: kafka-1
        component: kafka
        clusterName: kafka
   ---
    kind: Service
    apiVersion: v1
    metadata:
      name: external-kafka-2
    spec:
      type: NodePort
      externalTrafficPolicy: Local
      ports:
        - name: kafka-external-client
          port: 9094
          protocol: TCP
          targetPort: 9094
      selector:
        name: kafka-2
        component: kafka
        clusterName: kafka
   ---
    kind: Service
    apiVersion: v1
    metadata:
      name: external-kafka-3
    spec:
      type: NodePort
      externalTrafficPolicy: Local
      ports:
        - name: kafka-external-client
          port: 9094
          protocol: TCP
          targetPort: 9094
      selector:
        name: kafka-3
        component: kafka
        clusterName: kafka
   ```

   where `externalTrafficPolicy` is an optional parameter. You can omit it.

   If you set the value for `service.spec.externalTrafficPolicy` to `Local`, it allows only proxy requests to local endpoints, 
   and never forwards traffic to other nodes, thereby preserving the original source IP address. 
   If there are no local endpoints, packets sent to the node are dropped. 
   You can therefore rely on the correct source-ip in any packet processing rules, which you might apply to a packet to make it through
   to the endpoint.

   Check that each `external-kafka-1(2,3)` service routes to appropriate Kafka pod. If it does not, fix the `selector`
   for each `external-kafka-1(2,3)` service.

   After all services are created, you can get information about the created services and copy the values of `NodePort`
   that are generated automatically. 

   `NodePorts` exposes the service NodePort on all nodes in the OpenShift/Kubernetes cluster.
   `NodePorts` are in the 30000-32767 range by default, which means a `NodePort` is unlikely to match a service's intended port.

   Generated NodePorts are used as `kafka.externalPort` parameters.

2. Find out external IP addresses of OpenShift/Kubernetes nodes where Kafka brokers should be placed.

   Go to **Pods** > **kafka-1(2,3)**. In the `Yaml` you can see information about a node, 
   for example `hostIP: 10.105.105.105`. Check the IP address whether it is external 
   by performing the `ping` command for this address from your local computer: `ping 10.105.105.105`. 
   If `ping` command succeeds, then it is external IP address, and you can use it as `kafka.externalHostName` parameter.
   If `ping` command fails, then it is not external, and to find out external IP address of the node 
   you need to go to `OpenStack` interface and find out information about desired nodes.

3. Deploy Kafka specifying the values for external access that you found using the steps above.
   Example of deploy parameters for external access:
   
   ```yaml
   kafka:
      install: true
      storage:
         size: 5Gi
         volumes:
            - kafka-pv1
            - kafka-pv2
            - kafka-pv3
         className:
            - hostpath
         nodes:
            - node-1
            - node-2
            - node-3
      zookeeperConnect: zookeeper.zookeeper-service:2181
      createExternalServices: true
      externalTrafficPolicy: Local
      externalHostNames:
         - 10.10.10.11
         - 10.10.10.12
         - 10.10.10.13
      externalPorts:
         - 31001
         - 31002
         - 31003
   ```

**Note**: The administrator must ensure the external IPs are routed to the nodes and local firewall rules on all nodes 
allow access to the open port. If external access to Kafka is denied it means that your Node Ports are not open
in OpenStack configuration and the administrator should open them. 

The disadvantages of the `NodePort` approach are:

* Only one service in the cluster can use this port.
* Only 30000–32767 ports are available.
* Ports numbers are generated automatically, so you do not know them in advance (`NodePort` value can be specified manually,
  but it is not recommended).
* It is a problem if the IP addresses of nodes change.

However, it is an inexpensive and fast way to organize external access to your application for development purposes.

# LoadBalancer

The best option for a production environment is to configure a **LoadBalancer** service for each broker.

**Note**: It is important to note that the datapath for this functionality is provided by load balancers
**external to the Kubernetes cluster**. 
You need to ensure that the number of external load balancers is equal to or greater than the number of Kafka brokers. 
**Do not confuse the LoadBalancer services built into Kubernetes with the required ***external*** load balancers.**
External cloud load balancers or physical load balancers are not free.

A load balancer service allocates a unique IP from a configured pool. 

To configure a load balancer service, set the `type` field to `LoadBalancer`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-kafka-1
spec:
  type: LoadBalancer
  selector:
    name: kafka-1
    component: kafka
    clusterName: kafka
  ports:
    - protocol: TCP
      port: 9094
      targetPort: 9094  
```

where `9094` is a predefined value for broker's container port, do not change it. 

After all the services are created, copy the values of the allocated IP addresses and
then specify them in the `kafka.externalHostNames` parameter in a deployment job.
Leave the `kafka.externalPorts` parameter empty - the default value `9094` is set in `Deployment` automatically for each broker.

The disadvantage of the `LoadBalancer` approach is the cost of external cloud load balancers or physical load balancers.

# Client Connection

Use the provided IP addresses and ports for client connection. For example:

`bootstrap-servers: 10.10.10.11:31001,10.10.10.12:31002,10.10.10.13:31003`
