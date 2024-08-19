package dnscore

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

func IsValidIPAddress(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil
}

func FromIPAddressToPtrFqdn(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		panic(errors.New("not a valid IP address"))
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		//ipv4
		return dns.Fqdn(fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", ipv4[3], ipv4[2], ipv4[1], ipv4[0]))
	} else {
		//ipv6
		ipv6 := ip.To16()
		r := []byte(hex.EncodeToString(ipv6))
		slices.Reverse(r)
		strbuilder := new(strings.Builder)
		for _, v := range r {
			strbuilder.WriteByte(v)
			strbuilder.WriteByte('.')
		}
		return dns.Fqdn(strbuilder.String() + "ip6.arpa")
	}
}

func AddIpReverseDnsToStorage(storage DnsStorage, ipStr, domain string) error {
	if !IsValidIPAddress(ipStr) {
		return errors.New("invalid IP address:" + ipStr)
	}
	rDomain := FromIPAddressToPtrFqdn(ipStr)
	storage.PutDomain(rDomain, dns.Fqdn(domain), shared.DomainTypePtr)
	return nil
}
