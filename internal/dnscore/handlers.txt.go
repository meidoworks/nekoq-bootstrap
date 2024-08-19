package dnscore

import (
	"errors"
	"log"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type RecordTxtHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordTxtHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordTxtHandler{
		ParentRecordHandler: &ParentRecordHandler{Handler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordTxtHandler) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		log.Println("[DEBUG][RecordTxtHandler] domain:", domain)
	}

	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeTxt)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m)
	} else if err != nil {
		return nil, err
	}

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: []string{result},
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
