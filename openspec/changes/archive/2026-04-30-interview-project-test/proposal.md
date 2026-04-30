## Why

1. **工程隔离**：在 `interview-project-test` 独立 module 中承载实验性工具，避免污染主工程 `interview-project`。
2. **测绘需求**：内网与实验环境中常需基于 **IP 网段 + 端口范围** 发现 **mDNS（Multicast DNS）** 相关资产；运维与安全侧需要结构化输出（至少 **IP、端口、主机名/实例名**），并希望对响应做 **深度识别**（解码 DNS 语义 + 原始/半结构化 banner 线索），便于关联设备与服务指纹。

mDNS 虽以组播为主（如 `224.0.0.251:5353` / `ff02::fb`），但在现网中亦存在 **向单播地址的 5353/非标端口** 发送查询并得到响应的部署；本工具以 **可配置的网段与端口范围** 为输入，在授权范围内做探测与解析，统一输出资产视图。

## What Changes

- 在 `interview-project-test/` 内以 **Golang** 实现 **命令行测绘程序**（单二进制或可扩展子命令），能力包括：
  - **输入**：IPv4/IPv6 **CIDR 或等价网段表示**、**端口范围**（含上下界与可选默认）；
  - **输出**：在指定范围内发现的 **mDNS 协议相关响应** 资产记录，每条至少包含 **IP、端口、host（从 PTR/SRV/主机标签等推导）**；
  - **深度识别**：除结构化字段外，输出 **banner 层** 信息；**识别深度下限**以 **DNS-SD 浏览级** 为基准——即在同一资产上可复原 **多服务分块**（`9/tcp workstation`、`5000/tcp http`、`445/tcp smb`、`5000/tcp qdiscover`、`device-info`、`548/tcp afpovertcp` 等 **端口+传输+类型** 视图）、每块下的 **Name / IPv4 / IPv6 / Hostname / TTL**、**TXT 整段 vendor 指纹**（如 `accessType=...,model=...,fwVer=...`），以及 **`answers` 下列举全部相关 PTR**（如 `_workstation._tcp.local` … `_afpovertcp._tcp.local`）。**jsonl** 中固定嵌套 **`banner.dns_sd`**；**text** 格式须打印等价的 **`services:` / `answers:`** 摘要块（详见 `mdns-survey-cli` 规格与 `design.md`）。
- **输出格式**：须 **同时** 支持 **JSON Lines（`--format jsonl`）** 与 **人类可读文本（`--format text` 或默认 text）**，与 `mdns-survey-cli` 规格一致；字段名稳定、可版本化。
- **运行约束**：超时、并发上限、仅 UDP（mDNS 承载）等可在 CLI 中配置；文档中明确 **仅用于自有或已授权网络**。
- 更新子工程 **README** 与变更内 **design/specs/tasks**，与实现路径一致。

## Capabilities

### New Capabilities

- `interview-project-test-bootstrap`：子目录独立 `go.mod`、可构建、文档化入口（随 CLI 落地，入口行为从「Hello 占位」演进为 **正式 CLI**，见该能力规格中的 **MODIFIED** 条款）。
- `mdns-survey-cli`：**mDNS 测绘 CLI** 的完整行为契约——网段/端口解析、探测语义、资产字段、banner 深度识别、输出与错误码、性能与安全边界。

### Modified Capabilities

（无对 `openspec/specs/` 下已归档主规格的修改；本变更仅作用于 `interview-project-test` 子工程。）

## Impact

- 代码与依赖集中在 `interview-project-test/`，根 module `interview-project` 默认不改。
- 可能新增第三方或标准库扩展（如 `flag`/`cobra`、`net`、自研 DNS 解析）；需在 `go.mod` 中可追溯。
- 测绘能力涉及网络扫描，**文档与 `--help` 中须包含合规与授权提示**；CI 若后续加入，应避免在公网环境默认跑集成探测。
