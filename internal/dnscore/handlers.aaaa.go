package dnscore

import (
	"errors"
	"net"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type RecordAAAAHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordAAAAHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordAAAAHandler{
		ParentRecordHandler: &ParentRecordHandler{Handler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordAAAAHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		logger.Debug("[RecordAAAAHandler] domain:", domain)
	}

	ctx.AddTraceInfo("RecordAAAAHandler")
	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeAAAA)
	if errors.Is(err, shared.ErrStorageNotFound) {
		//TODO may need to check if the A record with the same name exists
		// In this case, empty AAAA or no AAAA response will be generated.
		return r.ParentRecordHandler.HandleQuestion(m, ctx)
	} else if err != nil {
		return nil, err
	}
	ctx.AddTraceInfo("RecordAAAAHandler->" + result)

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.AAAA{
		Hdr:  dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: DefaultResponseTTL},
		AAAA: net.ParseIP(result),
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
