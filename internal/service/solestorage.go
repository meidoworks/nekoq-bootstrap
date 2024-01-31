package service

import (
	"sync"

	"github.com/fxamacker/cbor/v2"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
	"github.com/meidoworks/nekoq-bootstrap/internal/storage"
	"github.com/meidoworks/nekoq-bootstrap/internal/wal"
)

type WalEntry struct {
	K []byte
	V []byte
	D bool
}

type KVStorageAdvImpl struct {
	sync.RWMutex

	wal iface.Wal
	kv  iface.KVStorageAdv

	config *KVStorageAdvConfig
}

type KVStorageAdvConfig struct {
	DataFolder string
	WalFolder  string
}

func NewKVStorageAdv(config *KVStorageAdvConfig) (*KVStorageAdvImpl, error) {
	kv, err := storage.NewDiskvStroage(config.DataFolder)
	if err != nil {
		return nil, err
	}
	w := wal.NewDiskWal(config.WalFolder)
	return &KVStorageAdvImpl{
		wal:    w,
		kv:     kv,
		config: config,
	}, nil
}

func (k *KVStorageAdvImpl) Initialize() error {
	if _, err := k.wal.Initialize(); err != nil {
		return err
	} else {
		// check if recover required
		seq, found, err := k.readVersion()
		if err != nil {
			return err
		}
		if !found {
			seq = wal.StartWalSequence
		}
		if seq, err := k.wal.ReplayIncomplete(seq, k.recoverFn); err != nil {
			return err
		} else if err := k.writeVersion(seq); err != nil {
			return err
		}
	}
	return nil
}

func (k *KVStorageAdvImpl) readVersion() (iface.SequenceId, bool, error) {
	dat, found, err := k.kv.Get(storage.VersionKey)
	if err != nil {
		return iface.SequenceId{}, false, err
	}
	if !found {
		return iface.SequenceId{}, false, nil
	}
	var seq iface.SequenceId
	if err := cbor.Unmarshal(dat, &seq); err != nil {
		return iface.SequenceId{}, false, err
	} else {
		return seq, true, nil
	}
}

func (k *KVStorageAdvImpl) writeVersion(seq iface.SequenceId) error {
	dat, err := cbor.Marshal(seq)
	if err != nil {
		return err
	}
	return k.kv.Put(storage.VersionKey, dat)
}

func (k *KVStorageAdvImpl) recoverFn(buf []byte) error {
	entry := new(WalEntry)
	if err := cbor.Unmarshal(buf, entry); err != nil {
		return err
	}
	return k.kv.Put(entry.K, entry.V)
}

func (k *KVStorageAdvImpl) Put(key, val []byte) error {
	entry := new(WalEntry)
	entry.K = key
	entry.V = val
	if seq, err := k.writeWalEntry(entry); err != nil {
		return err
	} else {
		if err := k.kv.Put(key, val); err != nil {
			return err
		}
		return k.writeVersion(seq)
	}
}

func (k *KVStorageAdvImpl) writeWalEntry(entry *WalEntry) (iface.SequenceId, error) {
	dat, err := cbor.Marshal(entry)
	if err != nil {
		return iface.SequenceId{}, err
	}
	return k.wal.WriteEntry(dat)
}

func (k *KVStorageAdvImpl) Get(key []byte) ([]byte, bool, error) {
	return k.kv.Get(key)
}

func (k *KVStorageAdvImpl) LatestSnapshotData(term int64) (*iface.Snapshot, error) {
	ver, _, err := k.readVersion()
	if err != nil {
		return nil, err
	}
	snapshot, err := k.kv.LatestSnapshotData(term)
	if err != nil {
		return nil, err
	}
	if snapshot.SequenceId == iface.EmptySequenceId {
		snapshot.SequenceId = ver
	}
	return snapshot, nil
}

func (k *KVStorageAdvImpl) CurrentPosition() (iface.SequenceId, error) {
	seq, _, err := k.readVersion()
	return seq, err
}

func (k *KVStorageAdvImpl) WalLogs(from iface.SequenceId) ([]iface.WalLog, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KVStorageAdvImpl) Replay(i []iface.WalLog) error {
	//TODO implement me
	panic("implement me")
}
