package dnscore

import (
	"net"

	"github.com/miekg/dns"
)

type NotFoundUpstreamDns struct {
}

func (u NotFoundUpstreamDns) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	reply := new(dns.Msg)
	reply.SetReply(m)
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

func (u *UpstreamDns) HandleQuestion(m *dns.Msg) (*dns.Msg, error) {
	r, err := dns.Exchange(m, net.JoinHostPort(u.cfg.Servers[0], u.cfg.Port))
	return r, err
}
