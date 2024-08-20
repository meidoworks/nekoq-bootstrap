package dnscore

import (
	"net"

	"github.com/miekg/dns"
)

type NotFoundUpstreamDns struct {
}

func (u NotFoundUpstreamDns) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	ctx.AddTraceInfo("NotFoundUpstreamDns")
	reply := new(dns.Msg)
	reply.SetRcode(m, dns.RcodeNameError)
	return reply, nil
}

type UpstreamDns struct {
	cfg *dns.ClientConfig
}

func NewUpstreamDNSWithSingle(server string) *UpstreamDns {
	// default values are from ClientConfigFromReader method in the dns package
	cfg := dns.ClientConfig{
		Servers:  []string{server},
		Search:   []string{},
		Port:     "53",
		Ndots:    1,
		Timeout:  5,
		Attempts: 2,
	}
	return &UpstreamDns{
		cfg: &cfg,
	}
}

func (u *UpstreamDns) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	ctx.AddTraceInfo("UpstreamDns")
	r, err := dns.Exchange(m, net.JoinHostPort(u.cfg.Servers[0], u.cfg.Port))
	ctx.AddTraceInfoWithDnsAnswersIfNoError("UpstreamDns->", r, err)
	return r, err
}
