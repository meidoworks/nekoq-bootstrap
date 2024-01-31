package service

import (
	"testing"
)

func TestAAAAA(t *testing.T) {
	kv, err := NewKVStorageAdv(&KVStorageAdvConfig{
		DataFolder: "data",
		WalFolder:  "wal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := kv.Initialize(); err != nil {
		t.Fatal(err)
	}

	rs := NewResp2Service(&RespServiceConfig{
		Addr: "127.0.0.1:6379",
	})

	h := NewRespKVHandler(kv)
	h.Register(rs)

	if err := rs.ServeAndWait(); err != nil {
		t.Log(err)
	}
}
