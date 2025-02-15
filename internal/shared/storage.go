package shared

import "errors"

type DomainType int

const (
	DomainTypeA DomainType = iota + 1
	DomainTypeTxt
	DomainTypeSrv
	DomainTypePtr
	DomainTypeAAAA
)

var ErrStorageNotFound = errors.New("not found")
