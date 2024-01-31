package iface

type ReplicatorRole int

const (
	RolePrimary ReplicatorRole = iota + 1
	RoleStandby
)

type PrimaryStandbyReplicator interface {
	Initialize() error

	Promote() error
	Role() ReplicatorRole
	Writable() bool

	Ship([]byte) error
}

type SequenceIdSupplier func() (SequenceId, error)
type SnapshotSupplier func(id SequenceId) ([]byte, error)
type ApplySnapshotAndIncrementalFunction func([]byte) error
type ApplyWalLog func([]byte) error
