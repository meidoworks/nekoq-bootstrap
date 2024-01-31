package wal

import (
	"math"
	"testing"
	"time"
)

func TestWalFilename(t *testing.T) {
	w := new(DiskWal)
	if w.filepath(0, 0, 0) != "00000000000000000000000000000000" {
		t.Fatal("00000000000000000000000000000000 failed")
	}
	if w.filepath(1, 1, 1) != "00000000000000010000000000000001" {
		t.Fatal("00000000000000010000000000000001 failed")
	}
	if w.filepath(math.MaxInt64, 1, 1) != "7fffffffffffffff0000000000000001" {
		t.Fatal("7fffffffffffffff0000000000000001 failed")
	}
	if w.filepath(1, math.MaxInt64, 1) != "00000000000000017fffffffffffffff" {
		t.Fatal("00000000000000017fffffffffffffff failed")
	}
	if w.filepath(1, 1, math.MaxInt64) != "00000000000000010000000000000001" {
		t.Fatal("00000000000000010000000000000001 failed")
	}
}

func TestPrepareDataBuf(t *testing.T) {
	w := new(DiskWal)
	w.maxPageSize = 4 * 1024
	buf := w.prepareDataBufByPage([]byte{1, 2, 3, 4}, 1)
	t.Log(buf)
}

func TestTTTT(t *testing.T) {
	w := NewDiskWal("testfolder")
	if id, err := w.Initialize(); err != nil {
		t.Fatal(err)
	} else {
		t.Log("id:", id)
	}
	for n := 0; n < 100; n++ {
		buf := make([]byte, 1256700)
		for idx, _ := range buf {
			buf[idx] = byte(idx%100) + 10
		}
		id, err := w.WriteEntryWithFullPage(buf)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(id)
		}
	}
	time.Sleep(5 * time.Second)
	t.Log(w.CurrentSequence())
}
