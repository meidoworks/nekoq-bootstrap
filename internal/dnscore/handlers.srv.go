package dnscore

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type RecordSRVHandler struct {
	*ParentRecordHandler
	DnsStorage

	debugOutput bool
}

func NewRecordSRVHandler(parent DnsRecordHandler, storage DnsStorage, debug bool) DnsRecordHandler {
	return &RecordSRVHandler{
		ParentRecordHandler: &ParentRecordHandler{Handler: parent},
		DnsStorage:          storage,
		debugOutput:         debug,
	}
}

func (r *RecordSRVHandler) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	domain := m.Question[0].Name
	if r.debugOutput {
		log.Println("[DEBUG][RecordSRVHandler] domain:", domain)
	}

	ctx.AddTraceInfo("RecordSRVHandler")
	result, err := r.DnsStorage.ResolveDomain(domain, shared.DomainTypeSrv)
	if errors.Is(err, shared.ErrStorageNotFound) {
		return r.ParentRecordHandler.HandleQuestion(m, ctx)
	} else if err != nil {
		return nil, err
	}
	ctx.AddTraceInfo("RecordSRVHandler->" + result)

	var srvData = struct {
		Priority uint16 `json:"priority"`
		Weight   uint16 `json:"weight"`
		Port     uint16 `json:"port"`
		Target   string `json:"target"`
	}{}
	if err := json.Unmarshal([]byte(result), &srvData); err != nil {
		return nil, err
	}

	reply := new(dns.Msg)
	reply.SetReply(m)
	rr := &dns.SRV{
		Hdr:      dns.RR_Header{Name: domain, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 0},
		Priority: srvData.Priority,
		Weight:   srvData.Weight,
		Port:     srvData.Port,
		Target:   dns.Fqdn(srvData.Target),
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}
