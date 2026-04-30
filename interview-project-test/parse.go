package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const rawHexN = 64

func parsePortSpec(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty ports")
	}
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			a, b, ok := strings.Cut(part, "-")
			if !ok {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(a))
			end, err2 := strconv.Atoi(strings.TrimSpace(b))
			if err1 != nil || err2 != nil || start < 0 || end < 0 || start > 65535 || end > 65535 || start > end {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			for p := start; p <= end; p++ {
				out = append(out, p)
			}
			continue
		}
		p, err := strconv.Atoi(part)
		if err != nil || p < 0 || p > 65535 {
			return nil, fmt.Errorf("invalid port %q", part)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ports parsed")
	}
	return dedupePorts(out), nil
}

func dedupePorts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	var out []int
	for _, p := range in {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func ipv4HostCount(n *net.IPNet) (uint64, error) {
	ip4 := n.IP.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("not IPv4")
	}
	ones, bits := n.Mask.Size()
	if bits != 32 {
		return 0, fmt.Errorf("internal: expected IPv4 mask")
	}
	if ones < 0 || ones > 32 {
		return 0, fmt.Errorf("invalid IPv4 mask")
	}
	return uint64(1) << uint(32-ones), nil
}

func parseCIDR(s string) (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	return ipnet, nil
}

func iterCIDRHosts(n *net.IPNet, yield func(net.IP) bool) error {
	if ip4 := n.IP.To4(); ip4 != nil {
		masked := ip4.Mask(n.Mask)
		start := binary.BigEndian.Uint32(masked)
		ones, bits := n.Mask.Size()
		if bits != 32 {
			return fmt.Errorf("internal: expected IPv4")
		}
		hostCount := uint32(1) << uint(32-ones)
		end := start + hostCount - 1
		for v := start; v <= end; v++ {
			b := make(net.IP, 4)
			binary.BigEndian.PutUint32(b, v)
			if !yield(b) {
				return nil
			}
		}
		return nil
	}
	ip := n.IP.Mask(n.Mask)
	if len(ip) != net.IPv6len {
		return fmt.Errorf("invalid IPv6 CIDR")
	}
	for {
		if !n.Contains(ip) {
			break
		}
		if !yield(append(net.IP(nil), ip...)) {
			return nil
		}
		if !incIP(ip) {
			break
		}
	}
	return nil
}

func incIP(ip net.IP) bool {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			return true
		}
	}
	return false
}
