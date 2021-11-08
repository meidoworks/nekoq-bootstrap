package main

import (
	"fmt"

	"github.com/miekg/dns"
)

func main() {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("node1.example.dns"), dns.TypeA)
	in, rtt, err := c.Exchange(m, "127.0.0.1:8053")
	if err != nil {
		panic(err)
	}
	fmt.Println(rtt.String())
	if len(in.Answer) > 0 {
		if t, ok := in.Answer[0].(*dns.A); ok {
			// do something with t.Txt
			fmt.Println(t)
			fmt.Println(t.A.String())
		}
	} else {
		fmt.Println("no answer")
	}
}
