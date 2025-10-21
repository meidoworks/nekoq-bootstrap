package dnscore

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

type NotFoundUpstreamDns struct {
	RespondNXDomainForAAAA bool
}

func (u NotFoundUpstreamDns) HandleQuestion(m *dns.Msg, ctx *RequestContext) (*dns.Msg, error) {
	ctx.AddTraceInfo("NotFoundUpstreamDns")
	reply := new(dns.Msg)
	//FIXME Not good to directly respond NXDomain due to the reason in the below.
	// Should check if there is any other types to the same domain name and determine to respond NXDomain or NOERROR.
	var defaultRcode = dns.RcodeNameError
	//Debug tool: resolvectl show-cache
	//Example output for a domain:
	// docker-pull.registry.internal IN A 10.11.0.3
	// docker-pull.registry.internal IN ANY NXDOMAIN
	//The resolver will send A and AAAA in parallel.
	//The 2nd entry should be related to AAAA type rather than ANY, which causes the issue.
	//Solutions:
	// 1. Respond NOERROR when resolving AAAA rather than NXDOMAIN.
	//    Explain: NXDOMAIN means "there are no records that have that name". If there are other records with the same name, but different types, the response you will see is NOERROR with zero records in the answer section.
	if m.Question[0].Qtype == dns.TypeAAAA && !u.RespondNXDomainForAAAA {
		defaultRcode = dns.RcodeSuccess
	}
	reply.SetRcode(m, defaultRcode)
	ctx.AddTraceInfo(fmt.Sprint("NotFoundUpstreamDns-rcode:[", reply.Rcode, "]"))
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
	case dns.TypePTR:
		key = "PTR"
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
