package dnscore

import "errors"

const DefaultResponseTTL = 3600

var (
	ErrDoNotRespondResult = errors.New("internal: do not respond")
)
