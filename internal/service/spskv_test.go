package service

import (
	"testing"
)

func TestBasic(t *testing.T) {
	//
	//var primary iface.ClusterKVStorage
	//// node 1
	//{
	//	var kvs iface.ClusterKVStorage = NewSimplePrimaryStandbyKVStore(&SimplePrimaryStandbyKVStoreConfig{
	//		NodeName: "node1",
	//		ClusterNodeMap: map[string]struct {
	//			Addr string
	//		}{
	//			"node1": {"127.0.0.1:12081"},
	//			"node2": {"127.0.0.1:12082"},
	//		},
	//		ListenAddr: "127.0.0.1:12081",
	//		Dependencies: struct {
	//			Wal        iface.Wal
	//			Replicator iface.PrimaryStandbyReplicator
	//			KVStorage  iface.KVStorage
	//		}{},
	//	})
	//
	//	// stage 1: initialize
	//	if err := kvs.Initialize(); err != nil {
	//		t.Fatal(err)
	//	}
	//	primary = kvs
	//}
	//// node 2
	//{
	//	var kvs iface.ClusterKVStorage = NewSimplePrimaryStandbyKVStore(&SimplePrimaryStandbyKVStoreConfig{
	//		NodeName: "node2",
	//		ClusterNodeMap: map[string]struct {
	//			Addr string
	//		}{
	//			"node1": {"127.0.0.1:12081"},
	//			"node2": {"127.0.0.1:12082"},
	//		},
	//		ListenAddr: "127.0.0.1:12082",
	//		Dependencies: struct {
	//			Wal        iface.Wal
	//			Replicator iface.PrimaryStandbyReplicator
	//			KVStorage  iface.KVStorage
	//		}{},
	//	})
	//
	//	// stage 1: initialize
	//	if err := kvs.Initialize(); err != nil {
	//		t.Fatal(err)
	//	}
	//}
	//
	//// stage 2: promote primary
	//if err := primary.Promote(); err != nil {
	//	t.Fatal(err)
	//}
	//
	//// wait for a while(just let standby syncup with primary)
	//time.Sleep(5 * time.Second)
	//
	//// stage 3: write operation
	//if err := primary.Put([]byte("test_key1"), []byte("test value 111")); err != nil {
	//	t.Fatal(err)
	//}
	//
	//// wait for cluster sync
	//time.Sleep(1 * time.Hour)
}
