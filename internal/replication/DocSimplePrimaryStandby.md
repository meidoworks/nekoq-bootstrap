# SimplePrimaryStandby

# 1. Design

## 1.0 Introduction

* All nodes have other nodes in the same cluster as peers to simplify configuration and operation.
* Simple primary-standby replication without strict validation.
* Very simple primary-standby cluster.
* Assumption 1: once a primary offline, it takes *long time*(> several seconds) to recover by selecting a new primary

## 1.1 Startup

All nodes start from standby role.

Startup Process Sequence:

1. (out of replicator) Read data and wal files and setup current position.
2. (out of replicator) Check and recovery from wal.

After startup process, Finding Primary Sequence:

1. Check peer role, and find primary: standby -> standby(maybe)
2. Check operation with current position and get sync decision: standby -> primary
3. If current position > primary position or current position < earliest wal position, then start full sync: standby ->
   primary
4. Start wal sync and register as standby in primary (atomic operation in order to guarantee timeline): standby ->
   primary
5. Note 1: Before sync and register to primary, both wal replication(standby control) and write operation(primary
   control) are blocked

No primary found, waiting for primary:

1. Looping all peers with interval

Turning into primary, see #1.4 Maintenance - Promote .

## 1.2 Write Operation

Process Sequence(primary only):

1. (out of replicator) wal write
2. (out of replicator) do data change with wal SequenceId update
3. replicate wal to standby nodes (including wal write & in-progress & finish) with mandatory node count -> unblocked:
   primary -> standby

Insufficient standby node situation:

* Write Operation is blocked until timeout

## 1.3 Read Operation

* If in primary role, read operations are permitted.
* If in standby role, read operations are denied.

## 1.4 Maintenance - Promote

Note: DO NOT promote more than once node at the same time. Undefined behavior. Please promote the standby with latest
wal only.

On one standby node:

* Promote from standby to primary
