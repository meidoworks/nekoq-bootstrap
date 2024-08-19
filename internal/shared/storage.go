package shared

import "errors"

type DomainType int

const (
	DomainTypeA DomainType = iota + 1
	DomainTypeTxt
	DomainTypeSrv
)

var ErrStorageNotFound = errors.New("not found")
