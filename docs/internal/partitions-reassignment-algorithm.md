## Smart Partitions Reassignment Plan

This is an algorithm of partitions reassignment used on Kafka cluster scaling.

```bash
for topic in topics {

    // Prepare data for assessment: calculate brokers skew and prepare structs
        Get topics metadata.
        Calculate global partitions count from topics metadata.
        Calculate partitions count per broker from topics metadata.
        Calculate brokers skew as it is done in Kafka Monitoring. Including new brokers.
        Check skew for all brokers - if it is less than abs(5%) than break the loop. The end of the algorithm. 
    
        Create struct:
        type BrokerInfo{
          BrokerId
          Rack
          PartitionsCount
          Skew
        }
          
        Create a slice brokersInfo []*BrokerInfo with all brokers.
        
        Get currentReplicaAssignment for the topic.
        Copy currentReplicaAssignment to newReplicaAssignment.
	
	previousAssignment = -1
	for partition in newReplicaAssignment {
	  	  
	  Calculate brokers skew as it is done in Kafka Monitoring. Including new brokers.

	  Sort brokersInfo by Skew in descending order

      brokersWithMostPartitions should go in the beginning of the brokersInfo (first elements with skew > 0)
	  brokersWithLeastPartitions should go in the end of the brokersInfo (last elements with skew < 0). 
	  Sort brokersWithLeastPartitions by Skew in ascending order
	 
      // Find broker holding most partitions among this partition replicas.
      // Further we will change this replica to the broker with least partitions.   	 
          brokerWithMostPartitionsToSwap = -1
          idx = -1
          label:
          for broker in brokersWithMostPartitions {
             for replica in newReplicaAssignment[partition] {
                if broker == replica {
                   brokerWithMostPartitionsToSwap = broker
                   idx = index of replica in newReplicaAssignment[partition]
                   break label
                }
             }
          }
          
          if brokerWithMostPartitionsToSwap = -1 {
             continue
          }	  
	  
      // Find broker holding least partitions among this partition replicas, we will use it instead of brokerWithMostPartitionsToSwap.
      // Always need to check !(newReplicaAssignment[partition] contains brokerWithLeastPartitionsToSwap),
      // because all partition's replicas must be different.
          brokerWithLeastPartitionsToSwap = -1
          currentRacks = Get racks for all brokers from newReplicaAssignment[partition]
          for broker in brokersWithLeastPartitions {
             if !(currentRacks contains broker.rack) && !(newReplicaAssignment[partition] contains brokerWithLeastPartitionsToSwap) {		 
                 brokerWithLeastPartitionsToSwap = broker // unique rack for partition replica is ideal
                 break
             }
          }
          
          if (brokerWithLeastPartitionsToSwap == -1){
              for broker in brokersWithLeastPartitions {
                 if brokerWithMostPartitionsToSwap.rack == broker.rack && !(newReplicaAssignment[partition] contains brokerWithLeastPartitionsToSwap){
                     brokerWithLeastPartitionsToSwap = broker // the same rack does not change the current assignment
                     break
                 }
              }
          }
          
          if (brokerWithLeastPartitionsToSwap == -1){     
            for broker in brokersWithLeastPartitions {
              if !(newReplicaAssignment[partition] contains brokerWithLeastPartitionsToSwap){
                brokerWithLeastPartitionsToSwap = brokersWithLeastPartitions[0] // just get a broker with least partitions if we have no better choice
              }
            }		
          }	  
          
          if brokerWithLeastPartitionsToSwap = -1 {
             continue
          }
	  
	  // For topics with RF = 1 we do not want a single replica of all partitions to just move to another broker. 
      // They should be distributed. For that purpose we will take into account previous replacement for that topic.
	  if topic.ReplicationFactor == 1 && previousAssignment == brokerWithLeastPartitionsToSwap {
			previousAssignment = -1
			continue
	  } else {
			previousAssignment = brokerWithLeastPartitionsToSwap
	  }
		
	  newReplicaAssignment[partition][idx] = brokerWithLeastPartitionsToSwap
	  
	  brokersInfo[brokerWithLeastPartitionsToSwap].partitionsCount++
	  brokersInfo[brokerWithMostPartitionsToSwap].partitionsCount--
	}
	
	Run PartitionsReassignment with newReplicaAssignment
	Wait until ListPartitionsReassignment for the topic is empty (it means it is done)
}
```
