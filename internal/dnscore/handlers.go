package dnscore

import (
	"errors"
	"log"
	"net"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type ParentRecordHandler struct {
	ParentRecordHandler DnsRecordHandler
}

func (c *ParentRecordHandler) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	if c.ParentRecordHandler != nil {
		return c.ParentRecordHandler.HandleQuestion(m)
	} else {
		return NotFoundUpstreamDns{}.HandleQuestion(m)
	}
}

type RecordAHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordAHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordAHandler{
		ParentRecordHandler: &ParentRecordHandler{ParentRecordHandler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordAHandler) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		log.Println("[DEBUG][RecordAHandler] domain:", domain)
	}

	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeA)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m)
	} else if err != nil {
		return nil, err
	}

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   net.ParseIP(result),
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
