package dnscore

import "testing"

func TestIsValidIPAddress(t *testing.T) {
	if IsValidIPAddress("999.0.111.0") {
		t.Fatal("999.0.111.0 failed")
	}
}

func TestFromIPAddressToPtrFqdn(t *testing.T) {
	if FromIPAddressToPtrFqdn("127.0.0.1") != "1.0.0.127.in-addr.arpa." {
		t.Fatal("127.0.0.1 failed")
	}
	if FromIPAddressToPtrFqdn("192.168.0.1") != "1.0.168.192.in-addr.arpa." {
		t.Fatal("192.168.0.1 failed")
	}
	if FromIPAddressToPtrFqdn("8.8.8.8") != "8.8.8.8.in-addr.arpa." {
		t.Fatal("8.8.8.8 failed")
	}

	if FromIPAddressToPtrFqdn("2002:7f00:1::") != "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.f.7.2.0.0.2.ip6.arpa." {
		t.Fatal("2002:7f00:1:: failed")
	}
	if FromIPAddressToPtrFqdn("2002:7f00:0001:0000:0000:0000:0000:0000") != "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.0.f.7.2.0.0.2.ip6.arpa." {
		t.Fatal("2002:7f00:0001:0000:0000:0000:0000:0000 failed")
	}
}
