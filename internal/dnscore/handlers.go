package dnscore

import (
	"errors"
	"net"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type ParentRecordHandler struct {
	Handler DnsRecordHandler
}

func (c *ParentRecordHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	if c.Handler != nil {
		return c.Handler.HandleQuestion(m, ctx)
	}
	return NotFoundUpstreamDns{}.HandleQuestion(m, ctx)
}

type RecordAHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordAHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordAHandler{
		ParentRecordHandler: &ParentRecordHandler{Handler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordAHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		logger.Debug("[RecordAHandler] domain:", domain)
	}

	ctx.AddTraceInfo("RecordAHandler")
	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeA)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m, ctx)
	} else if err != nil {
		return nil, err
	}
	ctx.AddTraceInfo("RecordAHandler->" + result)

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: DefaultResponseTTL},
		A:   net.ParseIP(result),
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
