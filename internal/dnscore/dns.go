package dnscore

import (
	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type DnsRecordHandler interface {
	HandleQuestion(m *dns.Msg) (*dns.Msg, error)
}

type DnsStorage interface {
	ResolveDomain(domain string, domainType shared.DomainType) (string, error)
}
