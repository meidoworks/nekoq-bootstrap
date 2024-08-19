package dnscore

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/miekg/dns"
)

type DnsEndpoint struct {
	Storage DnsStorage
	Server  *dns.Server

	Addr string

	DebugPrintDnsRequest bool

	HandlerMapping map[uint16]DnsRecordHandler
}

func NewDnsEndpoint(addr string, storage DnsStorage, upstreams []string, debug bool) (*DnsEndpoint, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	endpoint := new(DnsEndpoint)
	endpoint.Addr = addr
	endpoint.Storage = storage
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
			parentHandler = NewUpstreamDNSWithSingle(upstreams[0])
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
	defer func() {
		err := recover()
		if err != nil {
			log.Println("[ERROR] process dns request failed. information:", err)
		}
	}()

	reply := d.ProcessDnsMsg(r)
	if err := w.WriteMsg(reply); err != nil {
		panic(err)
	}
}

func (d *DnsEndpoint) ProcessDnsMsg(r *dns.Msg) *dns.Msg {
	if r.Opcode == dns.OpcodeQuery && len(r.Question) > 1 {
		// treat question count > 1 as incorrectly-formatted message according to rfc9619
		reply := new(dns.Msg)
		return reply.SetRcodeFormatError(r)
	}

	handler, ok := d.HandlerMapping[r.Question[0].Qtype]
	if !ok {
		panic(errors.New("unknown request type:" + fmt.Sprint(r.Question[0].Qtype)))
	}
	res, err := handler.HandleQuestion(r)
	if err != nil {
		panic(errors.New("dns request failed. " + err.Error()))
	}
	return res
}
