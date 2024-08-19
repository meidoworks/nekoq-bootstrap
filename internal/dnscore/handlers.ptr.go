package dnscore

import (
	"errors"
	"log"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type RecordPtrHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordPtrHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordPtrHandler{
		ParentRecordHandler: &ParentRecordHandler{Handler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordPtrHandler) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		log.Println("[DEBUG][RecordPtrHandler] domain:", domain)
	}

	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypePtr)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m)
	} else if err != nil {
		return nil, err
	}

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.PTR{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 0},
		Ptr: result,
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}