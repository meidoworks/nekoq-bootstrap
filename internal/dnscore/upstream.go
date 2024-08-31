package dnscore

import (
	"net"
	"strings"

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

	enclosureSuffixMap map[string][]string
	enclosureDomainMap map[string][]string
}

func NewUpstreamDNSWithSingle(server []string, suffixes []struct {
	Type   string
	Suffix string
}) *UpstreamDns {
	enclosureSuffixMap := make(map[string][]string)
	for _, v := range suffixes {
		suffix := v.Suffix
		if !strings.HasPrefix(v.Suffix, ".") {
			suffix = "." + v.Suffix
		}
		enclosureSuffixMap[v.Type] = append(enclosureSuffixMap[v.Type], dns.Fqdn(suffix))
	}
	enclosureDomainMap := make(map[string][]string)
	for _, v := range suffixes {
		if !strings.HasSuffix(v.Suffix, ".") {
			enclosureDomainMap[v.Type] = append(enclosureDomainMap[v.Type], dns.Fqdn(v.Suffix))
		}
	}

	// default values are from ClientConfigFromReader method in the dns package
	cfg := dns.ClientConfig{
		Servers:  server,
		Search:   []string{},
		Port:     "53",
		Ndots:    1,
		Timeout:  5,
		Attempts: 2,
	}
	return &UpstreamDns{
		cfg: &cfg,

		enclosureSuffixMap: enclosureSuffixMap,
		enclosureDomainMap: enclosureDomainMap,
	}
}

func (u *UpstreamDns) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	res := u.handleEnclosureDomains(ctx, strings.ToLower(m.Question[0].Name), m.Question[0].Qtype, m)
	if res != nil {
		return res, nil
	}

	ctx.AddTraceInfo("UpstreamDns")
	r, err := dns.Exchange(m, net.JoinHostPort(u.cfg.Servers[0], u.cfg.Port))
	ctx.AddTraceInfoWithDnsAnswersIfNoError("UpstreamDns->", r, err)
	return r, err
}

func (u *UpstreamDns) handleEnclosureDomains(ctx *RequestContext, domain string, qtype uint16, raw *dns.Msg) *dns.Msg {
	var key string
	switch qtype {
	case dns.TypeA:
		key = "A"
	case dns.TypeAAAA:
		key = "AAAA"
	case dns.TypeSRV:
		key = "SRV"
	case dns.TypeTXT:
		key = "TXT"
	default:
		return nil // not supported type for enclosure domain matching
	}

	if suffixes, suffixesOk := u.enclosureSuffixMap[key]; suffixesOk {
		var found = false
		for _, suffix := range suffixes {
			if strings.HasSuffix(domain, suffix) {
				found = true
				break
			}
		}
		if found {
			// once the upstream dns handles enclosure domain suffixes, it means that the domain is not found
			ctx.AddTraceInfo("UpstreamDns-EnclosureDomainSuffix-Ended")
			r, _ := NotFoundUpstreamDns{}.HandleQuestion(raw, ctx)
			return r
		}
	}
	if domains, domainsOk := u.enclosureDomainMap[key]; domainsOk {
		var found = false
		for _, d := range domains {
			if strings.HasSuffix(domain, d) {
				found = true
				break
			}
		}
		if found {
			// once the upstream dns handles enclosure domain suffixes, it means that the domain is not found
			ctx.AddTraceInfo("UpstreamDns-EnclosureDomainSuffix-Ended")
			r, _ := NotFoundUpstreamDns{}.HandleQuestion(raw, ctx)
			return r
		}
	}

	return nil
}
