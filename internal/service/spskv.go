package service

import (
	"sync"

	"github.com/fxamacker/cbor/v2"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
	"github.com/meidoworks/nekoq-bootstrap/internal/replication"
)

type SimplePrimaryStandbyKVStoreConfig struct {
	NodeName       string
	ClusterNodeMap map[string]struct {
		Addr string
	}
	ListenAddr string

	DataFolder string
	WalFolder  string
}

type SimplePrimaryStandbyKVStore struct {
	config *SimplePrimaryStandbyKVStoreConfig

	resp2Service *Resp2Service
	kv           *KVStorageAdvImpl

	replicator iface.PrimaryStandbyReplicator

	sync.Mutex
}

func NewSimplePrimaryStandbyKVStore(config *SimplePrimaryStandbyKVStoreConfig) (*SimplePrimaryStandbyKVStore, error) {
	s := new(SimplePrimaryStandbyKVStore)

	if kv, err := NewKVStorageAdv(&KVStorageAdvConfig{
		DataFolder: config.DataFolder,
		WalFolder:  config.WalFolder,
	}); err != nil {
		return nil, err
	} else {
		s.kv = kv
	}

	rs := NewResp2Service(&RespServiceConfig{
		Addr: "127.0.0.1:6379",
	})
	h := NewRespKVHandler(s.kv)
	h.Register(rs)
	s.resp2Service = rs

	s.replicator = replication.NewSimplePrimaryStandby(&replication.SimplePrimaryStandbyConfig{
		NodeName:                            config.NodeName,
		ClusterNodeMap:                      config.ClusterNodeMap,
		RegHandler:                          s.resp2Service,
		SequenceIdSupplier:                  s.sequenceIdSupplier,
		SnapshotSupplier:                    s.snapshotSupplier,
		ApplySnapshotAndIncrementalFunction: s.applySnapshotAndIncrementalFn,
		ApplyWalLog:                         s.applyWalLog,
	})

	return s, nil
}

func (s *SimplePrimaryStandbyKVStore) Initialize() error {
	if err := s.kv.Initialize(); err != nil {
		return err
	}

	if err := s.replicator.Initialize(); err != nil {
		return err
	}

	// have to unblock the initialize flow
	go func() {
		if err := s.resp2Service.ServeAndWait(); err != nil {
			panic(err)
		}
	}()
	return nil
}

func (s *SimplePrimaryStandbyKVStore) sequenceIdSupplier() (iface.SequenceId, error) {
	return s.kv.CurrentPosition()
}

func (s *SimplePrimaryStandbyKVStore) snapshotSupplier(seq iface.SequenceId) ([]byte, error) {
	v := new(SpskvSnapshotEntry)
	snapshot, err := s.kv.LatestSnapshotData(seq.Term)
	if err != nil {
		return nil, err
	}
	v.S = snapshot

	logs, err := s.kv.WalLogs(snapshot.SequenceId)
	if err != nil {
		return nil, err
	}
	v.L = logs

	return cbor.Marshal(v)
}

func (s *SimplePrimaryStandbyKVStore) applySnapshotAndIncrementalFn(dat []byte) error {
	v := new(SpskvSnapshotEntry)
	if err := cbor.Unmarshal(dat, v); err != nil {
		return err
	}
	for _, v := range v.S.Entries {
		if err := s.kv.kv.Put(v.K, v.V); err != nil {
			return err
		}
	}
	if err := s.kv.Replay(v.L); err != nil {
		return err
	}

	return nil
}

func (s *SimplePrimaryStandbyKVStore) applyWalLog(dat []byte) error {
	walLog := new(iface.WalLog)
	if err := cbor.Unmarshal(dat, walLog); err != nil {
		return err
	}

	if err := s.kv.Replay([]iface.WalLog{*walLog}); err != nil {
		return err
	}
	return nil
}

func (s *SimplePrimaryStandbyKVStore) Put(k, v []byte) error {
	s.Lock()
	defer s.Unlock()

	if err := s.kv.Put(k, v); err != nil {
		return err
	}
	seq := s.kv.wal.CurrentSequence()
	entry := new(iface.WalLog)
	entry.SequenceId = seq
	entry.K = k
	entry.V = v
	dat, err := cbor.Marshal(entry)
	if err != nil {
		return err
	}
	return s.replicator.Ship(dat)
}

func (s *SimplePrimaryStandbyKVStore) Get(k []byte) ([]byte, bool, error) {
	return s.kv.Get(k)
}

type SpskvSnapshotEntry struct {
	S *iface.Snapshot
	L []iface.WalLog
}
