package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var (
	_LISTEN_PORT int
	_LISTEN_IP   string

	_ENABLE_RELAY_REQUEST bool
)

func init() {
	flag.IntVar(&_LISTEN_PORT, "port", 53, "-port=53")
	flag.BoolVar(&_ENABLE_RELAY_REQUEST, "relay", false, "-relay=false")

	flag.Parse()

	fmt.Println("Listen on udp on: ", fmt.Sprintf("%s:%d", _LISTEN_IP, _LISTEN_PORT))
}

func main() {
	clientConfig := new(dns.ClientConfig)
	clientConfig.Servers = []string{}
	clientConfig.Search = make([]string, 0)
	clientConfig.Port = "53"
	clientConfig.Ndots = 1
	clientConfig.Timeout = 5
	clientConfig.Attempts = 2

	relayDnsHandler := new(RelayDnsHandler)
	relayDnsHandler.clientConfig = clientConfig
	relayDnsHandler.client = new(dns.Client)

	server := &dns.Server{
		Addr: fmt.Sprintf("%s:%d", _LISTEN_IP, _LISTEN_PORT),
		Net:  "udp",
	}
	server.Handler = relayDnsHandler
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}

	//client := new(dns.Client)
	//client.Exchange()

	//dns.A{}
	//dns.AAAA{}
	//dns.MX{}
	//dns.TXT{}
	//dns.SRV{}
	//dns.CNAME{}
	//HTTP dns
	//dns sec
}

type DnsHandler struct {
}

func (this *DnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	//TODO
	m := new(dns.Msg)
	m.SetReply(r)
	w.WriteMsg(m)
}

type RelayDnsHandler struct {
	clientConfig *dns.ClientConfig
	client       *dns.Client
}

func (this *RelayDnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// from local
	result := GetARecord(r.Question[0].Name, r)
	if result != nil {
		err := w.WriteMsg(result)
		if err != nil {
			//TODO should not panic
			panic(err)
		}
		return
	}

	// relay
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(this.clientConfig.Timeout)*time.Second)
	result, _, err := this.client.ExchangeContext(ctx, r, net.JoinHostPort(this.clientConfig.Servers[0], this.clientConfig.Port))
	if err != nil {
		//TODO should not panic
		panic(err)
	}
	err = w.WriteMsg(result)
	if err != nil {
		//TODO should not panic
		panic(err)
	}
}

func GetARecord(domain string, r *dns.Msg) *dns.Msg {
	if strings.HasSuffix(domain, ".servers.moetang.info.") {
		reply := new(dns.Msg)
		reply.SetReply(r)
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   net.ParseIP("0.0.0.0"),
		}
		reply.Answer = append(reply.Answer, rr)
		return reply
	}
	return nil
}
