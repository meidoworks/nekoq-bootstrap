package main

import (
	"flag"

	"github.com/miekg/dns"
)

var (
	_LISTEN_PORT          int
	_ENABLE_RELAY_REQUEST bool
)

func init() {
	flag.IntVar(&_LISTEN_PORT, "port", 53, "-port=53")
	flag.BoolVar(&_ENABLE_RELAY_REQUEST, "relay", false, "-relay=false")

	flag.Parse()
}

func main() {
	clientConfig := new(dns.ClientConfig)
	clientConfig.Servers = make([]string, 0)
	clientConfig.Search = make([]string, 0)
	clientConfig.Port = "53"
	clientConfig.Ndots = 1
	clientConfig.Timeout = 5
	clientConfig.Attempts = 2

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
