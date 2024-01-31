### SimplePrimaryStandbyKVStorage

* [ ] graceful shutdown
* [ ] Operational operation: including viewing current position, etc.
* [ ] Issue: primary -> standby connection broken will cause write block and standby out of date
* [ ] Issue: potential connection leak issue in primaryDetectorWorker
* [ ] Full/Wal sync rework when standby connected to primary
* [ ] wal sync and first log entry order race condition right after standby synced with primary
* [ ] Auth

New list:

* [Document: Wal](internal/wal/DocWal.md)

General feature:

* Check version api: for upgrade
* Connection poison: force client/server abandon the connection, to support graceful shutdown
* Service startup sequence: setup, warmup, start service
