package main

import (
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
)

func TestGoldenNASStyleDNS_SD(t *testing.T) {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Id = 1

	const ttl = 10
	const target = "slw-nas.local."

	add := func(rr dns.RR) { msg.Answer = append(msg.Answer, rr) }

	// Browse PTR answers (subtype enumeration)
	for _, ptr := range []string{
		"_workstation._tcp.local.",
		"_http._tcp.local.",
		"_smb._tcp.local.",
		"_qdiscover._tcp.local.",
		"_device-info._tcp.local.",
		"_afpovertcp._tcp.local.",
	} {
		add(&dns.PTR{
			Hdr: dns.RR_Header{Name: "_services._dns-sd._udp.local.", Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: ptr,
		})
	}

	// SRV + TXT + addressing
	addSRV := func(owner string, port uint16, txt []string) {
		add(&dns.SRV{
			Hdr:      dns.RR_Header{Name: owner, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
			Priority: 0,
			Weight:   0,
			Port:     port,
			Target:   target,
		})
		if len(txt) > 0 {
			add(&dns.TXT{
				Hdr: dns.RR_Header{Name: owner, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
				Txt: txt,
			})
		}
	}

	addSRV("slw-nas._workstation._tcp.local.", 9, []string{"Name=slw-nas [24:5e:be:69:a3:13]"})
	addSRV("slw-nas._http._tcp.local.", 5000, []string{"path=/"})
	addSRV("slw-nas._smb._tcp.local.", 445, nil)
	addSRV("slw-nas._qdiscover._tcp.local.", 5000, []string{"accessType=https,accessPort=86,model=TS-X64,displayModel=TS-464C,fwVer=5.2.9,fwBuildNum=20260214"})
	addSRV("slw-nas._device-info._tcp.local.", 80, []string{"model=Xserve"})
	addSRV("slw-nas(AFP)._afpovertcp._tcp.local.", 548, nil)

	add(&dns.A{
		Hdr: dns.RR_Header{Name: target, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   mustParseIP(t, "192.0.2.10"),
	})
	add(&dns.AAAA{
		Hdr:  dns.RR_Header{Name: target, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
		AAAA: mustParseIP(t, "2001:db8::1"),
	})

	raw, err := msg.Pack()
	if err != nil {
		t.Fatal(err)
	}

	a := buildAsset(mustParseIP(t, "192.0.2.1"), 5353, raw, msg)

	if len(a.Banner.DnsSD.Services) < 6 {
		t.Fatalf("services: got %d want >= 6", len(a.Banner.DnsSD.Services))
	}
	wantRegs := []string{"workstation", "http", "smb", "qdiscover", "device-info", "afpovertcp"}
	got := map[string]struct{}{}
	for _, s := range a.Banner.DnsSD.Services {
		if s.RegType != "" {
			got[s.RegType] = struct{}{}
		}
	}
	for _, w := range wantRegs {
		if _, ok := got[w]; !ok {
			t.Fatalf("missing reg_type %q in %#v", w, got)
		}
	}

	ptrWant := []string{
		"_workstation._tcp.local",
		"_http._tcp.local",
		"_smb._tcp.local",
		"_qdiscover._tcp.local",
		"_device-info._tcp.local",
		"_afpovertcp._tcp.local",
	}
	for _, w := range ptrWant {
		if !containsPtr(a.Banner.DnsSD.Answers.Ptr, w) {
			t.Fatalf("missing answers.ptr %q: %#v", w, a.Banner.DnsSD.Answers.Ptr)
		}
	}

	var qtxt string
	for _, s := range a.Banner.DnsSD.Services {
		if s.RegType == "qdiscover" {
			qtxt = s.TxtFlat
		}
	}
	if !strings.Contains(qtxt, "accessType=https") || !strings.Contains(qtxt, "model=TS-X64") {
		t.Fatalf("vendor txt not preserved: %q", qtxt)
	}
}

func mustParseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("bad ip %q", s)
	}
	return ip
}

func containsPtr(list []string, want string) bool {
	for _, p := range list {
		if strings.TrimSuffix(p, ".") == strings.TrimSuffix(want, ".") {
			return true
		}
	}
	return false
}
