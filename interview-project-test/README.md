# interview-project-test

与仓库根目录 `interview-project` 主模块解耦的 **独立 Go module**，提供 **`mdns-survey`**：在授权范围内，对 **IP 网段（CIDR）× UDP 端口范围** 发送 **mDNS 语义 PTR 查询**，解析响应并输出 **ip / port / host** 与 **`banner`（含 `dns_sd` 深度视图）**。

> **合规**：仅在你拥有或已书面授权的网络与系统上使用本工具。未经授权的扫描可能违法。

## 构建

```bash
go build -o mdns-survey .
go build ./...
go test ./...
```

## 用法

```bash
./mdns-survey --help

# 示例：扫描局域网一段地址的 5353（mDNS 常用端口），JSON Lines 输出
./mdns-survey -cidr 192.168.1.0/24 -ports 5353 -format jsonl -timeout 800ms -concurrency 64 -max-hosts 1024

# 人类可读（默认）
./mdns-survey -cidr 192.168.1.0/24 -ports 5353-5354 -format text -verbose
```

### 主要参数

| 参数 | 说明 |
|------|------|
| `-cidr` | **必填**，IPv4/IPv6 CIDR。 |
| `-ports` | **必填**，单端口、`a-b` 区间或逗号列表。 |
| `-q` | PTR 查询名，默认 `_services._dns-sd._udp.local.` |
| `-timeout` | 单目标 UDP 超时，默认 `800ms`。 |
| `-concurrency` | 最大并发探测数，默认 `64`。 |
| `-max-hosts` | 每轮最多扫描的 **IP 个数**（从 CIDR 起始顺序计数），默认 `4096`；IPv4 若 CIDR 展开地址数 **大于** 该值且未带 `-i-understand`，程序会拒绝启动。 |
| `-format` | `text`（默认）或 `jsonl`。 |
| `-verbose` | 将跳过的目标原因打到 **stderr**。 |
| `-i-understand` | 允许 **超过** `-max-hosts` 的 IPv4 CIDR 预检（仍只扫描前 `-max-hosts` 个地址，详见实现）。 |

### JSON 字段约定

- 每条资产包含：`ip`、`port`、`host`、`banner`。
- **`banner.raw_hex`**：UDP 载荷前 **64** 字节的小写十六进制（规格要求 32～128 之间，本实现固定 **64**）。
- **`banner.dns_sd`**：**始终存在**；无足够 RR 时为 `{"services":[],"answers":{"ptr":[]}}`；有数据时含 `services[]` 与 `answers.ptr[]`（与 OpenSpec `mdns-survey-cli` 对齐）。

### 退出码

- `0`：正常结束（含 **无响应**，此时 stderr 会提示 `no DNS responses collected`）。
- `2`：参数错误、CIDR/端口非法、或超出 `-max-hosts` 预检等。

## 规格与变更跟踪

- OpenSpec 变更：`openspec/changes/interview-project-test`
- 行为细节：`openspec/changes/interview-project-test/specs/mdns-survey-cli/spec.md`

## 与主工程关系

不依赖根目录 `interview-project` 的 `go.mod`；在本目录单独执行 `go` 命令即可。
