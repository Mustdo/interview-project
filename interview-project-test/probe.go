package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

func probe(ip net.IP, port int, q string, timeout time.Duration) ([]byte, *dns.Msg, error) {
	d := net.Dialer{}
	addr := net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port))
	conn, err := d.Dial("udp", addr)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(q), dns.TypePTR)
	m.RecursionDesired = false
	var id uint16
	_ = binary.Read(rand.Reader, binary.BigEndian, &id)
	if id == 0 {
		id = 1
	}
	m.Id = id

	buf, err := m.Pack()
	if err != nil {
		return nil, nil, err
	}
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, nil, err
	}
	if _, err := conn.Write(buf); err != nil {
		return nil, nil, err
	}

	rb := make([]byte, 65535)
	n, err := conn.Read(rb)
	if err != nil {
		return nil, nil, err
	}
	raw := append([]byte(nil), rb[:n]...)
	rm := new(dns.Msg)
	if err := rm.Unpack(raw); err != nil {
		return raw, nil, nil
	}
	if rm.Response {
		return raw, rm, nil
	}
	return raw, rm, nil
}
