# RaftLite

### Assumption and Design

* Based on raft which is clear and easy to understand consensus algorithm
* Complete decouple leader election and log replication to simply and split state machine into independent pieces
* Node role changes after network split/timeout(Loose requirement on safety)
* Fewer node fault or network issue

### List

* Leader election
* Log replication
* Snapshot & Log compaction
* Membership change: join and leave

### Advanced

* Compaction to save space for log and snapshot
* Large data support
* Enough transaction id room to avoid forcing rotate
* Speedup recovery from network split
* Less memory usage
* Simple configuration and less dependencies
* Jepsen test

### Data flow directions

* Leader Election
    * Election: Peer to Peer
    * KeepAlive: Leader -> Follower
* Log Replication
    * Quorum Alignment: Leader -> Follower
    * Log Ship: Leader -> Follower
    * New Follower Join Alignment: New Follower -> Leader -> Follower

### Low priority

* Optimization for network split or node crash happen frequently
* Optimization for the Byzantine Generals Problem
* Persistent delay when applying request from leader to follower which causes request not commit.
    * Note: request cannot be rollback from any other good node
    * Note2: keepalive is separate from log replication and may still be working
* Unified peer to peer channel which guarantees bi-direction communication between peers
* Network split may cause such scenario: node with newer record is network split and goes back after a certain time
    * Case1: no leader election for both network partitions, meaning term id is not changed -> log conflict
    * Case2: leader election happened, meaning term id increased
* Handling node unavailable in very short term, e.g. Trequest < t < Ttimeout
    * Meaning that role is not be changed as standard state transition(e.g. leader->candidate->new leader)
* Node shutdown and safety
