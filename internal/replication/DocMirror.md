# Mirror

(Planning)

Feature:

* Simple: mirror data to cluster without coordinator
* Large Scale: support multi-tiers in single cluster

Best Practise:

* For better availability: put *same* data in 2 or more nodes in order for short period data loss once the source node
  offline
* To ensure consistency: put one change(*same* data) to desired nodes, after success, put next change

Assumption:

* Performance tradeoff: Regenerate complete data when one node offline. The possibility of node offline is rare.
