package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func printAssetJSON(w io.Writer, a asset) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(a)
}

func printAssetText(w io.Writer, a asset) {
	fmt.Fprintf(w, "ip=%s port=%d host=%s\n", a.IP, a.Port, a.Host)
	if a.HostConfidence != "" {
		fmt.Fprintf(w, "host_confidence=%s\n", a.HostConfidence)
	}
	if len(a.Banner.Txt) > 0 {
		fmt.Fprintf(w, "txt:\n")
		for _, t := range a.Banner.Txt {
			fmt.Fprintf(w, "  - %s\n", t)
		}
	}
	if len(a.Banner.Srv) > 0 {
		fmt.Fprintf(w, "srv:\n")
		for _, s := range a.Banner.Srv {
			fmt.Fprintf(w, "  - target=%s port=%d pri=%d w=%d\n", s.Target, s.Port, s.Priority, s.Weight)
		}
	}
	fmt.Fprintf(w, "raw_hex=%s\n", a.Banner.RawHex)
	if len(a.Banner.Notes) > 0 {
		fmt.Fprintf(w, "notes: %s\n", strings.Join(a.Banner.Notes, "; "))
	}
	if hasDNSSDDepth(a.Banner.DnsSD) {
		fmt.Fprintln(w, "services:")
		for _, sv := range a.Banner.DnsSD.Services {
			line := fmt.Sprintf("%d/%s/%s:", sv.Port, sv.Transport, sv.RegType)
			fmt.Fprintf(w, "  %s\n", line)
			if sv.InstanceName != "" {
				fmt.Fprintf(w, "    Name=%s\n", sv.InstanceName)
			}
			for _, ip := range sv.IPv4 {
				fmt.Fprintf(w, "    IPv4=%s\n", ip)
			}
			for _, ip := range sv.IPv6 {
				fmt.Fprintf(w, "    IPv6=%s\n", ip)
			}
			if sv.Hostname != "" {
				fmt.Fprintf(w, "    Hostname=%s\n", sv.Hostname)
			}
			if sv.TTL != 0 {
				fmt.Fprintf(w, "    TTL=%d\n", sv.TTL)
			}
			if sv.TxtFlat != "" {
				fmt.Fprintf(w, "    %s\n", sv.TxtFlat)
			}
			for k, v := range sv.Extras {
				fmt.Fprintf(w, "    %s=%s\n", k, v)
			}
		}
		fmt.Fprintln(w, "answers:")
		fmt.Fprintln(w, "PTR:")
		for _, p := range a.Banner.DnsSD.Answers.Ptr {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	fmt.Fprintln(w, "---")
}

func hasDNSSDDepth(d dnsSD) bool {
	if len(d.Services) > 0 {
		return true
	}
	if len(d.Answers.Ptr) > 0 {
		return true
	}
	return false
}
