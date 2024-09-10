package dnscore

import (
	"errors"

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

func (r *RecordTxtHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		logger.Debug("[RecordTxtHandler] domain:", domain)
	}

	ctx.AddTraceInfo("RecordTxtHandler")
	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeTxt)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m, ctx)
	} else if err != nil {
		return nil, err
	}
	ctx.AddTraceInfo("RecordTxtHandler->" + result)

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: []string{result},
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
