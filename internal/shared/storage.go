package shared

import "errors"

type DomainType int

const (
	DomainTypeA = 1
)

var ErrStorageNotFound = errors.New("not found")
