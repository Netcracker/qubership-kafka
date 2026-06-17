This section provides information about Affinity.

# Difference between nodeSelector and Affinity Rules

The `nodeSelector` provides a very simple way to constrain pods to nodes with specific labels. The
affinity/anti-affinity feature greatly expands the types of constraints you can express. The key
enhancements are:

* The language is more expressive.
* You can indicate that the rule is `soft` rather than a `hard` requirement, so if the scheduler
  cannot satisfy the rule, the pod will still be scheduled.
* You can constrain against labels on other pods running on the node (or other topological domain),
  rather than against labels on the node itself, which allows rules about which pods can and cannot
  be co-located.

The affinity feature consists of two types of affinity, `node affinity` and
`inter-pod affinity/anti-affinity`. Node affinity is like the existing nodeSelector but with the first two benefits listed above.
Inter-pod affinity/anti-affinity constrains against pod labels rather than node labels, as described in the third item listed above,
in addition to having the first and second properties listed above.

The `nodeSelector` continues to work as usual, but will eventually be deprecated, as `node affinity` can
express everything that `nodeSelector` can express.

# Node Affinity

The `node affinity` is a set of rules used by the scheduler to determine where a pod can be placed. The
rules are defined using custom `labels` on nodes and `label selectors` specified in the pods. The `node affinity`
allows a pod to specify an affinity towards a group of nodes it can be placed on. The node does not
have control over the placement. The `node affinity` is conceptually similar to `nodeSelector`.

The `node affinity` rule in the deployment template is shown below.

```yaml
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: label-key
            operator: In
            values:
            - label-value
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 50
        preference:
          matchExpressions:
          - key: another-label-key
            operator: In
            values:
            - another-label-value
```

# Inter-pod Affinity and Anti-affinity

The `inter-pod affinity and anti-affinity` allow specifying rules about how pods should be placed
relative to other pods. The rules are defined using custom labels on nodes and label selectors
specified in pods. The `pod affinity/anti-affinity` allows a pod to specify an affinity (or anti-affinity)
towards a group of pods it can be placed with. The node does not have control over the placement.
The rules are of the form `this pod should (or, in the case of anti-affinity, should not) run
in an X node if that X node is already running one or more pods that meet rule Y`. `Y` is expressed
as a LabelSelector with an associated list of namespaces; unlike nodes, because pods are namespaced
(and therefore the labels on pods are implicitly namespaced), a label selector over pod labels must
specify which namespaces the selector should apply to. Conceptually `X` is a topology domain like
node, rack, cloud provider zone, cloud provider region, and so on. To express it, it is necessary to use
a `topologyKey` which is the key for the node label that the system uses to denote such a topology
domain.

* `Pod affinity` can tell the scheduler to locate a new pod on the same node as other pods if the
  label selector on the new pod matches the label on the current pod.
* `Pod anti-affinity` can prevent the scheduler from locating a new pod on the same node as pods
  with the same labels if the label selector on the new pod matches the label on the current pod.

The `pod affinity` rule in the deployment template is shown below.

```yaml
spec:
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: security
            operator: In
            values:
            - S1
        topologyKey: failure-domain.beta.kubernetes.io/zone
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 50
        podAffinityTerm:
          labelSelector:
            matchExpressions:
            - key: security
              operator: In
              values:
              - S2
          topologyKey: kubernetes.io/hostname
```

In principle, the `topologyKey` can be any legal label-key. However, for performance and security
reasons, there are some constraints on `topologyKey`:

* For `affinity` and for `requiredDuringSchedulingIgnoredDuringExecution` `pod anti-affinity`,
  empty topologyKey is not allowed.
* For `preferredDuringSchedulingIgnoredDuringExecution` `pod anti-affinity`, empty topologyKey is
  interpreted as `all topologies` (`all topologies` here is now limited to the combination of
  `kubernetes.io/hostname`, `failure-domain.beta.kubernetes.io/zone` and
  `failure-domain.beta.kubernetes.io/region`).
* Except for the above cases, the `topologyKey` can be any legal label-key.

# Affinity Types

There are two types of affinity:

* requiredDuringSchedulingIgnoredDuringExecution (`hard`) specifies rules that must be met for
  a pod to be scheduled onto a node.
  If you specify multiple `matchExpressions`, then the pod can be scheduled onto a node only if all
  `matchExpressions` can be satisfied.
* preferredDuringSchedulingIgnoredDuringExecution (`soft`) specifies preferences that the scheduler
  will try to enforce but will not guarantee.
  The `weight` field is in the range 1-100. For each node that meets all of the scheduling
  requirements, the scheduler will compute a sum by iterating through the elements of this field and
  adding `weight` to the sum if the node matches the corresponding `matchExpressions`. This score is
  then combined with the scores of other priority functions for the node. The node(s) with the
  highest total score are the most preferred.

The `IgnoredDuringExecution` part of the names means that, similar to how `nodeSelector` works,
if labels on a node change at runtime such that the affinity rules on a pod are no longer met,
the pod will still continue to run on the node.
