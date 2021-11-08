package bootstrap

import (
	"errors"
	"log"
	"net"
	"net/url"

	"github.com/miekg/dns"
)

type DnsEndpoint struct {
	Storage Storage
	Server  *dns.Server

	Addr string

	DebugPrintDnsRequest bool
}

func NewDnsEndpoint(addr string, storage Storage) (*DnsEndpoint, error) {
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

	reply := processDnsMsg(r, this.Storage, this.DebugPrintDnsRequest)
	if err := w.WriteMsg(reply); err != nil {
		panic(err)
	}
}

func processDnsMsg(r *dns.Msg, storage Storage, debugOutput bool) *dns.Msg {
	if r.Question[0].Qtype != dns.TypeA {
		panic(errors.New("Request type is not A record"))
	}
	domain := r.Question[0].Name
	if debugOutput {
		log.Println("[DEBUG] domain:", domain)
	}

	result, err := storage.ResolveDomain(domain, DomainTypeA)
	if err == ErrStorageNotFound {
		reply := new(dns.Msg)
		reply.SetReply(r)
		return reply
	}

	reply := new(dns.Msg)
	reply.SetReply(r)
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   net.ParseIP(result),
	}
	reply.Answer = append(reply.Answer, rr)
	return reply
}
