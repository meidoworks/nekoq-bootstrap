package iface

type ClosableStorage interface {
	Close() error
}

type KVStorage interface {
	Put(k, v []byte) error
	Get(k []byte) ([]byte, bool, error)
}
