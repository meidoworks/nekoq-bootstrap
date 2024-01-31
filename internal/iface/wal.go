package iface

type Wal interface {
	WriteEntry([]byte) (SequenceId, error)
	WriteEntryWithFullPage([]byte) (SequenceId, error)
	CurrentSequence() SequenceId

	Initialize() (SequenceId, error)
	ReplayIncomplete(from SequenceId, f func(entry []byte) error) (SequenceId, error)
}

type SequenceId struct {
	Term       int64
	Collection int64
	Seq        int64
}

var EmptySequenceId = SequenceId{}

func (s SequenceId) Compare(to SequenceId) int {
	if s.Term < to.Term {
		return -1
	} else if s.Term > to.Term {
		return 1
	}

	if s.Collection < to.Collection {
		return -1
	} else if s.Collection > to.Collection {
		return 1
	}

	if s.Seq < to.Seq {
		return -1
	} else if s.Seq == to.Seq {
		return 0
	} else {
		return 1
	}
}
