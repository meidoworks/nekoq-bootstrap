package iface

type ClosableStorage interface {
	Close() error
}

type KVStorage interface {
	Put(k, v []byte) error
	Get(k []byte) ([]byte, bool, error)
}

type InitializingStorage interface {
	Initialize() error
}

type SnapshotStorage interface {
	LatestSnapshotData(term int64) (*Snapshot, error)
}

type WalLogSupportedStorage interface {
	CurrentPosition() (SequenceId, error)
	WalLogs(from SequenceId) ([]WalLog, error)
	Replay([]WalLog) error
}

type KVStorageAdv interface {
	KVStorage
	InitializingStorage
	SnapshotStorage
}

type KVStorageAdvAdv interface {
	KVStorageAdv
	WalLogSupportedStorage
}

type ClusterOperation interface {
	Promote() error
}

type ClusterKVStorage interface {
	KVStorageAdv
	ClusterOperation
}

type Snapshot struct {
	SequenceId SequenceId
	Entries    []struct {
		K []byte
		V []byte
	}
	Metadata map[string]string
}

type WalLog struct {
	K []byte
	V []byte
	D bool
	SequenceId
}
