package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/dnscore"
)

type DnsEndpoint struct {
	Storage Storage
	Server  *dns.Server

	Addr string

	DebugPrintDnsRequest bool

	HandlerMapping map[uint16]dnscore.DnsRecordHandler
}

func NewDnsEndpoint(addr string, storage Storage, upstreams []string, debug bool) (*DnsEndpoint, error) {
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
	endpoint.HandlerMapping = map[uint16]dnscore.DnsRecordHandler{}
	// init handlers
	{
		var parentHandler dnscore.DnsRecordHandler
		if len(upstreams) > 0 {
			parentHandler = dnscore.NewUpstreamDNSWithSingle(upstreams[0])
		}
		endpoint.HandlerMapping[dns.TypeA] = dnscore.NewRecordAHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
		endpoint.HandlerMapping[dns.TypeTXT] = dnscore.NewRecordTxtHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
		endpoint.HandlerMapping[dns.TypeSRV] = dnscore.NewRecordSRVHandler(parentHandler, storage, endpoint.DebugPrintDnsRequest)
	}

	return endpoint, nil
}

func (this *DnsEndpoint) StartSync() error {
	return this.Server.ListenAndServe()
}

func (this *DnsEndpoint) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	defer func() {
		err := recover()
		if err != nil {
			log.Println("[ERROR] process dns request failed. information:", err)
		}
	}()

	reply := this.processDnsMsg(r)
	if err := w.WriteMsg(reply); err != nil {
		panic(err)
	}
}

func (this *DnsEndpoint) processDnsMsg(r *dns.Msg) *dns.Msg {
	if r.Opcode == dns.OpcodeQuery && len(r.Question) > 1 {
		// treat question count > 1 as incorrectly-formatted message according to rfc9619
		reply := new(dns.Msg)
		return reply.SetRcodeFormatError(r)
	}

	handler, ok := this.HandlerMapping[r.Question[0].Qtype]
	if !ok {
		panic(errors.New("unknown request type:" + fmt.Sprint(r.Question[0].Qtype)))
	}
	res, err := handler.HandleQuestion(r)
	if err != nil {
		panic(errors.New("dns request failed. " + err.Error()))
	}
	return res
}
