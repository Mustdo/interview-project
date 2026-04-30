package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type job struct {
	ip   net.IP
	port int
}

func main() {
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			printHelp()
			os.Exit(0)
		}
	}

	var (
		cidr        = flag.String("cidr", "", "IPv4/IPv6 CIDR to scan (required)")
		portsFlag   = flag.String("ports", "", "Ports: e.g. 5353 or 5353-5356 or 5353,5355 (required)")
		timeout     = flag.Duration("timeout", 800*time.Millisecond, "per-target UDP timeout")
		concurrency = flag.Int("concurrency", 64, "max concurrent UDP probes")
		maxHosts    = flag.Int("max-hosts", 4096, "max IP addresses to scan per run (see README)")
		q           = flag.String("q", "_services._dns-sd._udp.local.", "PTR query name")
		format      = flag.String("format", "text", "output format: text|jsonl")
		verbose     = flag.Bool("verbose", false, "log per-target skips to stderr")
		understand  = flag.Bool("i-understand", false, "allow IPv4 CIDR larger than --max-hosts (dangerous)")
	)
	flag.Usage = printHelp
	flag.Parse()

	if *cidr == "" || *portsFlag == "" {
		fmt.Fprintln(os.Stderr, "mdns-survey: -cidr and -ports are required")
		printHelp()
		os.Exit(2)
	}
	if *format != "text" && *format != "jsonl" {
		fmt.Fprintln(os.Stderr, "mdns-survey: -format must be text or jsonl")
		os.Exit(2)
	}
	if *concurrency < 1 {
		*concurrency = 1
	}

	n, err := parseCIDR(*cidr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdns-survey: bad -cidr: %v\n", err)
		os.Exit(2)
	}

	ports, err := parsePortSpec(*portsFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdns-survey: bad -ports: %v\n", err)
		os.Exit(2)
	}

	if n.IP.To4() != nil {
		cnt, err := ipv4HostCount(n)
		if err == nil && cnt > uint64(*maxHosts) && !*understand {
			fmt.Fprintf(os.Stderr, "mdns-survey: CIDR expands to %d addresses (> -max-hosts=%d). Use a narrower CIDR or pass --i-understand (see README).\n", cnt, *maxHosts)
			os.Exit(2)
		}
	} else {
		ones, bits := n.Mask.Size()
		if bits == 128 && ones < 120 && !*understand {
			fmt.Fprintf(os.Stderr, "mdns-survey: IPv6 prefix /%d is wider than /120; narrow the CIDR or pass --i-understand (see README).\n", ones)
			os.Exit(2)
		}
	}

	jobs := make(chan job, *concurrency*4)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var assets []asset

	workers := *concurrency
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				raw, msg, err := probe(j.ip, j.port, *q, *timeout)
				if err != nil {
					if *verbose {
						fmt.Fprintf(os.Stderr, "skip %s:%d: %v\n", j.ip, j.port, err)
					}
					continue
				}
				if msg == nil {
					if *verbose {
						fmt.Fprintf(os.Stderr, "skip %s:%d: non-DNS UDP payload\n", j.ip, j.port)
					}
					continue
				}
				if len(msg.Answer)+len(msg.Extra)+len(msg.Ns) == 0 {
					continue
				}
				a := buildAsset(j.ip, j.port, raw, msg)
				mu.Lock()
				assets = append(assets, a)
				mu.Unlock()
			}
		}()
	}

	go func() {
		defer close(jobs)
		var ipSeen int
		_ = iterCIDRHosts(n, func(ip net.IP) bool {
			if ipSeen >= *maxHosts {
				return false
			}
			ipSeen++
			for _, p := range ports {
				jobs <- job{ip: append(net.IP(nil), ip...), port: p}
			}
			return true
		})
	}()

	wg.Wait()

	for _, a := range assets {
		if *format == "jsonl" {
			if err := printAssetJSON(os.Stdout, a); err != nil {
				fmt.Fprintf(os.Stderr, "mdns-survey: encode: %v\n", err)
				os.Exit(1)
			}
		} else {
			printAssetText(os.Stdout, a)
		}
	}

	if len(assets) == 0 {
		fmt.Fprintln(os.Stderr, "mdns-survey: no DNS responses collected")
	}
	os.Exit(0)
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `mdns-survey — mDNS / DNS-SD 风格 UDP 测绘（实验性）

用法:
  mdns-survey -cidr <CIDR> -ports <端口说明> [选项]

必填:
  -cidr    IPv4/IPv6 CIDR，例如 192.168.1.0/24
  -ports   单端口、a-b 区间或逗号列表，例如 5353 或 5353-5354

常用选项:
  -timeout       单目标 UDP 超时（默认 %s）
  -concurrency   最大并发探测数（默认 64）
  -max-hosts     每轮最多扫描的 IP 数（默认 4096）
  -q             PTR 查询名（默认 _services._dns-sd._udp.local.）
  -format        text|jsonl（默认 text）
  -verbose       将跳过原因输出到 stderr
  -i-understand  允许 IPv4 CIDR 展开地址数大于 -max-hosts 的预检（危险，见 README）

合规:
  仅扫描自有或已书面授权的网络与系统；未经授权的扫描可能违法。

`, (800 * time.Millisecond).String())
}
