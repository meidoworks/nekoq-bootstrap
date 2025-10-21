package dnscore

import (
	"errors"

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

func (r *RecordPtrHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		logger.Debug("[RecordPtrHandler] domain:", domain)
	}

	ctx.AddTraceInfo("RecordPtrHandler")
	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypePtr)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m, ctx)
	} else if err != nil {
		return nil, err
	}
	ctx.AddTraceInfo("RecordPtrHandler->" + result)

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.PTR{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: DefaultResponseTTL},
		Ptr: result,
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
