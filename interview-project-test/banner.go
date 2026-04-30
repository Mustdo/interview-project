package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

type srvView struct {
	Target   string `json:"target"`
	Port     int    `json:"port"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
}

type serviceView struct {
	Port          int                    `json:"port"`
	Transport     string                 `json:"transport"`
	RegType       string                 `json:"reg_type"`
	InstanceName  string                 `json:"instance_name"`
	IPv4          []string               `json:"ipv4"`
	IPv6          []string               `json:"ipv6"`
	Hostname      string                 `json:"hostname"`
	TTL           uint32                 `json:"ttl"`
	Extras        map[string]string      `json:"extras,omitempty"`
	TxtFlat       string                 `json:"txt_flat,omitempty"`
	TxtKV         map[string]string      `json:"txt_kv,omitempty"`
}

type ptrAnswers struct {
	Ptr []string `json:"ptr"`
}

type dnsSD struct {
	Services []serviceView `json:"services"`
	Answers  ptrAnswers    `json:"answers"`
}

type banner struct {
	Txt    []string  `json:"txt"`
	Srv    []srvView `json:"srv,omitempty"`
	RawHex string    `json:"raw_hex"`
	Notes  []string  `json:"notes"`
	DnsSD  dnsSD     `json:"dns_sd"`
}

type asset struct {
	IP               string  `json:"ip"`
	Port             int     `json:"port"`
	Host             string  `json:"host"`
	HostConfidence   string  `json:"host_confidence,omitempty"`
	Banner           banner  `json:"banner"`
}

func rawHexPrefix(b []byte, n int) string {
	if n > len(b) {
		n = len(b)
	}
	return strings.ToLower(hex.EncodeToString(b[:n]))
}

func parseMdnsName(fqdn string) (instance, regType, transport string) {
	fqdn = dns.Fqdn(fqdn)
	lbl := dns.SplitDomainName(fqdn)
	if len(lbl) < 3 {
		return "", "", ""
	}
	if lbl[len(lbl)-1] != "local" {
		return "", "", ""
	}
	tl := lbl[len(lbl)-2]
	switch tl {
	case "_tcp":
		transport = "tcp"
	case "_udp":
		transport = "udp"
	default:
		return "", "", ""
	}
	reg := lbl[len(lbl)-3]
	regType = strings.TrimPrefix(reg, "_")
	if len(lbl) > 3 {
		instance = strings.Join(lbl[:len(lbl)-3], ".")
		instance = strings.TrimSuffix(instance, ".")
	}
	return instance, regType, transport
}

func collectRRs(msg *dns.Msg) []dns.RR {
	var out []dns.RR
	out = append(out, msg.Answer...)
	out = append(out, msg.Ns...)
	out = append(out, msg.Extra...)
	return out
}

func buildAsset(srcIP net.IP, srcPort int, raw []byte, msg *dns.Msg) asset {
	a := asset{
		IP:   srcIP.String(),
		Port: srcPort,
		Banner: banner{
			RawHex: rawHexPrefix(raw, rawHexN),
			Notes:  nil,
			DnsSD: dnsSD{
				Services: []serviceView{},
				Answers:  ptrAnswers{Ptr: []string{}},
			},
		},
	}
	if msg == nil {
		a.Host = ""
		a.HostConfidence = "low"
		a.Banner.Notes = append(a.Banner.Notes, "non-dns")
		return a
	}

	rrs := collectRRs(msg)
	var txts []string
	var srvs []srvView
	ptrSet := map[string]struct{}{}
	txtByName := map[string][]string{}
	for _, rr := range rrs {
		switch t := rr.(type) {
		case *dns.TXT:
			txtByName[strings.ToLower(t.Hdr.Name)] = append(txtByName[strings.ToLower(t.Hdr.Name)], t.Txt...)
			for _, s := range t.Txt {
				txts = append(txts, s)
			}
		case *dns.SRV:
			srvs = append(srvs, srvView{
				Target:   strings.TrimSuffix(t.Target, "."),
				Port:     int(t.Port),
				Priority: int(t.Priority),
				Weight:   int(t.Weight),
			})
		case *dns.PTR:
			ptr := strings.TrimSuffix(t.Ptr, ".")
			if ptr != "" {
				ptrSet[ptr] = struct{}{}
			}
		case *dns.A:
			// handled when merging by target hostname
		case *dns.AAAA:
		}
	}
	sort.Strings(txts)
	a.Banner.Txt = txts
	a.Banner.Srv = srvs

	ptrs := keysSorted(ptrSet)
	a.Banner.DnsSD.Answers.Ptr = ptrs

	// services from SRV records
	seen := map[string]struct{}{}
	for _, rr := range rrs {
		srv, ok := rr.(*dns.SRV)
		if !ok {
			continue
		}
		inst, reg, tr := parseMdnsName(srv.Hdr.Name)
		key := fmt.Sprintf("%s/%s/%d", tr, reg, srv.Port)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		tname := strings.ToLower(srv.Hdr.Name)
		txtParts := append([]string(nil), txtByName[tname]...)
		txtFlat := strings.Join(txtParts, ",")
		txtKV := parseKVComma(txtFlat)

		host := strings.TrimSuffix(srv.Target, ".")
		var extras map[string]string
		if p, ok := txtKV["path"]; ok {
			extras = map[string]string{"path": p}
		}
		sv := serviceView{
			Port:         int(srv.Port),
			Transport:    tr,
			RegType:      reg,
			InstanceName: inst,
			IPv4:         []string{},
			IPv6:         []string{},
			Hostname:     host,
			TTL:          srv.Hdr.Ttl,
			TxtFlat:      txtFlat,
			TxtKV:        txtKV,
			Extras:       extras,
		}
		mergeARecords(rrs, dns.Fqdn(srv.Target), &sv)
		a.Banner.DnsSD.Services = append(a.Banner.DnsSD.Services, sv)
	}

	// PTR-only rows: subtype enumeration when no SRV row exists for same transport+reg_type
	srvReg := map[string]struct{}{}
	for _, sv := range a.Banner.DnsSD.Services {
		if sv.RegType != "" {
			srvReg[sv.Transport+"/"+sv.RegType] = struct{}{}
		}
	}
	for ptr := range ptrSet {
		if strings.Contains(ptr, "._tcp.local") || strings.Contains(ptr, "._udp.local") {
			if _, reg, tr := parseMdnsName(dns.Fqdn(ptr)); reg != "" {
				if _, ok := srvReg[tr+"/"+reg]; ok {
					continue
				}
				key := fmt.Sprintf("%s/%s/ptr", tr, reg)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				a.Banner.DnsSD.Services = append(a.Banner.DnsSD.Services, serviceView{
					Port:         0,
					Transport:    tr,
					RegType:      reg,
					InstanceName: "",
					IPv4:         []string{},
					IPv6:         []string{},
					Hostname:     "",
					TTL:          0,
				})
			}
		}
	}

	sort.Slice(a.Banner.DnsSD.Services, func(i, j int) bool {
		if a.Banner.DnsSD.Services[i].Port != a.Banner.DnsSD.Services[j].Port {
			return a.Banner.DnsSD.Services[i].Port < a.Banner.DnsSD.Services[j].Port
		}
		return a.Banner.DnsSD.Services[i].RegType < a.Banner.DnsSD.Services[j].RegType
	})

	a.Host, a.HostConfidence = deriveHost(rrs)
	return a
}

func mergeARecords(rrs []dns.RR, targetFQDN string, sv *serviceView) {
	t := strings.ToLower(targetFQDN)
	for _, rr := range rrs {
		switch x := rr.(type) {
		case *dns.A:
			if strings.ToLower(dns.Fqdn(x.Hdr.Name)) == t {
				sv.IPv4 = append(sv.IPv4, x.A.String())
			}
		case *dns.AAAA:
			if strings.ToLower(dns.Fqdn(x.Hdr.Name)) == t {
				sv.IPv6 = append(sv.IPv6, x.AAAA.String())
			}
		}
	}
}

func deriveHost(rrs []dns.RR) (host, conf string) {
	for _, rr := range rrs {
		if srv, ok := rr.(*dns.SRV); ok {
			h := strings.TrimSuffix(srv.Target, ".")
			if h != "" {
				return h, ""
			}
		}
	}
	for _, rr := range rrs {
		if p, ok := rr.(*dns.PTR); ok {
			h := strings.TrimSuffix(p.Ptr, ".")
			if h != "" && strings.Contains(h, ".local") {
				return h, ""
			}
		}
	}
	return "", "low"
}

func parseKVComma(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func keysSorted(m map[string]struct{}) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
