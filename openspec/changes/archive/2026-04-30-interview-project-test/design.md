## Context

- 子工程路径：`interview-project-test/`，独立 Go module，与仓库根 `interview-project` 解耦。
- mDNS 见 **RFC 6762**：通常基于 **UDP**，熟知端口 **5353**；报文为 **DNS 报文** 子集，常见查询/浏览 `_services._dns-sd._udp.local`、各 `_tcp.local` / `_udp.local` **PTR**，以及 **SRV、TXT、A、AAAA** 等。
- 用户需求为 **「网站测绘」语境下的 CLI**：输入 **IP 网段 + 端口范围**，输出该范围内的 **mDNS 资产**；「banner」在 mDNS 场景下定义为 **超越单字段主机名** 的 **响应级证据**：含 **解码后的 RR 集合**、**TXT 全文**、**SRV 目标与端口**、以及 **原始载荷的摘要/截断十六进制**（用于非标或畸形报文识别）。
- **Banner 深度基线（示例契约）**：验收以「类 QNAP / DNS-SD 浏览」文本为**信息粒度下限**——即在同一资产上，除 `txt`/`srv`/`raw_hex` 外，须能表达 **按端口+传输+服务类型展开的多条 `services` 记录**（每条含 **Name、IPv4、IPv6、Hostname、TTL** 及 **TXT 派生的 vendor 行**如 `accessType=...`），并单列 **`answers.ptr`** 枚举 `_workstation._tcp.local`、`_http._tcp.local` 等 PTR 名。JSON 中映射为 **`banner.dns_sd`**（见 `mdns-survey-cli` 规格）；`text` 格式须能打印**语义等价**的缩略版（见规格「文本格式」场景）。

## Goals / Non-Goals

**Goals:**

- 在授权网段内，对 **(ip, port)** 组合发送 **符合 mDNS 语义的 DNS 查询**（可配置 query name、QU/QM 行为在实现层固定或参数化其一），收集 **UDP 响应**。
- 将每条命中解析为 **资产记录**：**IP、端口、host**（从 **PTR/SRV 的 Target/标签**、**反向查询结果**、或 **查询名与响应名映射** 中按优先级推导），并附 **banner 深度结构**（见下节）。
- **CLI**：子命令或根级 flag 清晰；**超时、重试、最大并发 goroutine 数、每秒发包上限（可选）** 可配置。
- **输出**：默认 **stdout**；**JSON Lines** 便于管道；人类可读模式便于现场排障。

**Non-Goals:**

- 不实现完整通用端口扫描器（如 SYN mass scan）；**非 UDP 或非 DNS 形态** 的协议不在首版范围。
- 不保证跨所有厂商栈的 **单播 mDNS** 兼容性（部分 OS 仅响应组播）；设计层保留 **可选组播辅助模式**（Open Question）。
- 不在首版内置 **CVE 匹配** 或 **漏洞利用**；仅做 **发现与指纹线索**。

## Decisions

1. **传输与套接字**  
   使用 `net` 包 **UDP** `Dial`/`WriteTo` 向目标 `(ip,port)` 发送 DNS 编码查询；读响应时关联 **五元组上下文** 中的 **源 IP/源端口** 作为资产主键之一。

2. **查询构造**  
   默认携带 **至少一条** 典型 mDNS 查询名（如 `_services._dns-sd._udp.local` IN PTR，可配置 `-q`）；**ID** 随机或递增，**避免缓存污染** 在首版用短超时 + 单次查询为主。

3. **解析管道**  
   - 若载荷可解析为 DNS：走 **github.com/miekg/dns** 或 **自研最小头部+RR 解析**（决策：优先 **miekg/dns** 降低畸形包处理成本，若需零依赖再抽象接口）。  
   - **host 推导顺序**：SRV Target 去 `.local` 标签 → PTR 首条 dname → 否则 **从 Query 与 Answer 名空间拼接**；文档化优先级。  
   - **banner**：  
     - **基础层**：`txt`、`srv`、`raw_hex`、`notes`（与规格一致）。  
     - **dns_sd 层**：对 **PTR+SRV+TXT+A/AAAA** 做 **实例级关联**（同一 `Instance Name` / Target 归并），生成 `services[]`；从 **Answer/Additional** 抽取 **全部可解析 PTR** 填入 `answers.ptr[]`。  
     - **Vendor 行**：将 TXT 中 **逗号分隔 `k=v`** 保留为 `txt_flat` 或解析为 `txt_kv`，以满足「`accessType=https,model=...`」类深度指纹不丢失。  
     - **黄金集**：实现侧维护 **golden DNS 报文**（可由脱敏 PCAP 或手写 bytes 构造），使 CI 中断言 **`len(banner.dns_sd.services) ≥ 6`**（或与 JSON 路径等价），且 **`banner.dns_sd.answers.ptr`** 集合覆盖示例六类 `_…._tcp.local`（见 `mdns-survey-cli` 规格 golden 场景）。

4. **网段与端口展开**  
   - IPv4 CIDR：标准库或轻量依赖展开为 IP 列表 **流式迭代**（避免 /8 一次性内存爆炸）；提供 **`-max-hosts` 硬上限** 默认保护。  
   - 端口：`a-b` 与单端口列表统一解析为区间迭代。

5. **并发模型**  
   **worker pool**：任务为 `(ip,port)`；每个任务 **独立 deadline**；全局 **context** 支持 Ctrl+C 取消。

6. **CLI 形态**  
   首版推荐 **根包 `main` + `flag`** 或 **`cobra` 单命令 `survey`**；与 bootstrap 规格中「`--help` 可见」一致。

## Risks / Trade-offs

- **[Risk] 误扫未授权地址** → **缓解**：默认 `max-hosts` 较小；README 与 `-help` 法律声明；可选 `--dry-run` 只打印将扫描规模。  
- **[Risk] 组播与跨网段无效** → **缓解**：文档说明 mDNS 边界；日志中标记 **无响应** 与 **ICMP/超时**。  
- **[Trade-off] 深度 banner 与隐私** → TXT 可能含序列号；输出到文件时提示敏感信息脱敏责任。

## Migration Plan

- 已有 Hello 占位 **替换为 CLI**：`go run . --help` 稳定可用；**破坏性** 对依赖「单行 Hello」的脚本：在 README 标明 **CHANGELOG 一行**。  
- 回滚：恢复旧 `main.go` 并删测绘相关包。

## Open Questions

- 是否增加 **可选组播模式**（向 `224.0.0.251:5353` 发送、再监听窗口内所有响应）作为 **单播扫描** 的补充？  
- IPv6 链路本地与 **zone index**（`fe80::%eth0`）是否在首版强制支持？
