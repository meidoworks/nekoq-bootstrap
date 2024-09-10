package dnscore

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/miekg/dns"
)

type DnsEndpoint struct {
	Storage DnsStorage
	Server  *dns.Server
	Cache   DnsCache

	Addr string

	DebugPrintDnsRequest bool

	HandlerMapping map[uint16]DnsRecordHandler
}

func NewDnsEndpoint(addr string, storage DnsStorage, upstreams []string, enclosureDomainSuffixes []struct {
	Type   string
	Suffix string
}, debug bool) (*DnsEndpoint, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	endpoint := new(DnsEndpoint)
	endpoint.Addr = addr
	endpoint.Storage = storage
	endpoint.Cache = NewDnsMemCache()
	endpoint.Server = &dns.Server{
		Addr:    u.Host,
		Net:     u.Scheme,
		Handler: endpoint,
	}
	endpoint.DebugPrintDnsRequest = debug
	endpoint.HandlerMapping = map[uint16]DnsRecordHandler{}
	// init handlers
	{
		var parentHandler DnsRecordHandler
		if len(upstreams) > 0 {
			parentHandler = NewUpstreamDNSWithSingle(upstreams, enclosureDomainSuffixes)
		}
		endpoint.HandlerMapping[dns.TypeA] = NewRecordAHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
		endpoint.HandlerMapping[dns.TypeTXT] = NewRecordTxtHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
		endpoint.HandlerMapping[dns.TypeSRV] = NewRecordSRVHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
		endpoint.HandlerMapping[dns.TypePTR] = NewRecordPtrHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
	}

	return endpoint, nil
}

func (d *DnsEndpoint) StartSync() error {
	return d.Server.ListenAndServe()
}

func (d *DnsEndpoint) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	reqCtx := NewRequestContext()

	defer func() {
		err := recover()
		if err != nil {
			logger.Error("process dns request failed. information:", err)
			reqCtx.AddTraceInfo("error occurs:" + fmt.Sprint(err))
		}
		if d.DebugPrintDnsRequest {
			logger.Debug("Domain resolve info:", reqCtx.GetTraceInfoString())
		}
	}()

	reply := d.ProcessDnsMsg(r, reqCtx)
	if err := w.WriteMsg(reply); err != nil {
		panic(err)
	}
}

func (d *DnsEndpoint) ProcessDnsMsg(r *dns.Msg, ctx *RequestContext) *dns.Msg {
	if r.Opcode == dns.OpcodeQuery && len(r.Question) > 1 {
		// treat question count > 1 as incorrectly-formatted message according to rfc9619
		reply := new(dns.Msg)
		return reply.SetRcodeFormatError(r)
	}

	ctx.AddTraceInfo(fmt.Sprint("resolve:t=", r.Question[0].Qtype, ",domain:", r.Question[0].Name))
	// query cache
	if res := d.Cache.Get(r); res != nil {
		ctx.AddTraceInfoWithDnsAnswersIfNoError("hit_mem_cache", res, nil)
		return res
	}
	// query pipeline
	handler, ok := d.HandlerMapping[r.Question[0].Qtype]
	if !ok {
		ctx.AddTraceInfo("unknown request type:" + fmt.Sprint(r.Question[0].Qtype))
		result, err := NotFoundUpstreamDns{}.HandleQuestion(r, ctx)
		if err != nil {
			panic(errors.New("error handling question:" + err.Error()))
		}
		return result
	}
	res, err := handler.HandleQuestion(r, ctx)
	if err != nil {
		panic(errors.New("dns request failed. " + err.Error()))
	}
	// cache result
	d.Cache.Put(r, res)
	return res
}
