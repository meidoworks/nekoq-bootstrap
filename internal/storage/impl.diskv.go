package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/peterbourgon/diskv/v3"
)

type DiskvStorage struct {
	diskv *diskv.Diskv
}

func NewDiskvStroage(folder string) (*DiskvStorage, error) {
	f, err := filepath.Abs(folder)
	if err != nil {
		return nil, err
	}
	d := diskv.New(diskv.Options{
		BasePath: f,
		Transform: func(s string) []string {
			return []string{diskvSha256prefix(s)}
		},
		CacheSizeMax: 1024 * 1024,
	})

	return &DiskvStorage{
		diskv: d,
	}, nil
}

func (d *DiskvStorage) Put(k, v []byte) error {
	key := string(k)
	if !validateKeyFormat(key) {
		return ErrKeyFormatInvalid
	}
	return d.diskv.Write(key, v)
}

func (d *DiskvStorage) Get(k []byte) ([]byte, bool, error) {
	key := string(k)
	if !validateKeyFormat(key) {
		return nil, false, ErrKeyFormatInvalid
	}
	dat, err := d.diskv.Read(key)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return dat, true, nil
}

func diskvSha256prefix(s string) string {
	v := sha256.Sum256([]byte(s))
	return hex.EncodeToString(v[:])[:4]
}
