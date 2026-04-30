## ADDED Requirements

### Requirement: 接受 IP 网段与端口范围参数

CLI MUST 接受表示 **IPv4 或 IPv6 网段** 的参数（至少支持 **CIDR** 记法；实现可选用 **起止 IP** 作为扩展），以及 **端口范围**（支持 `单端口` 与 `起始-结束` 闭区间形式）。

#### Scenario: 解析合法 IPv4 CIDR 与端口区间

- **WHEN** 用户传入有效 IPv4 CIDR（如 `192.168.1.0/24`）与端口范围（如 `5353-5353` 或 `5353`）
- **THEN** 程序 SHALL 将该组合展开为待探测的 `(IP, 端口)` 任务集合（受全局 `max-hosts` 或等价上限约束时可截断或拒绝并提示），且在**仅进行参数解析与展开**阶段不得崩溃

#### Scenario: 非法网段或端口

- **WHEN** 用户传入无法解析的网段或逆序/越界端口范围
- **THEN** 程序 SHALL 向标准错误输出可读错误信息并以 **非零退出码** 退出

### Requirement: 发送 mDNS 语义探测并接收 UDP 响应

对每个待测 `(目标IP, 目标端口)`，程序 MUST 通过 **UDP** 发送 **DNS 格式的 mDNS 查询**（默认查询名可配置；首版 MUST 至少支持一种与 **RFC 6762** 一致的 PTR 类查询模板），并在超时内尝试读取 **至少一次** 响应。

#### Scenario: 开放端口返回 DNS 形态 UDP 数据

- **WHEN** 目标在超时内向源返回 UDP 载荷且载荷可被解析为 DNS 报文（含 **QR=1** 的响应）
- **THEN** 程序 SHALL 将该事件纳入结果处理管道并生成 **至多一条** 与该源地址关联的资产记录候选（若多包合并策略在实现中定义，则文档化）

#### Scenario: 超时无响应

- **WHEN** 在配置的超时内未收到 UDP 响应
- **THEN** 程序 SHALL 静默跳过该 `(IP,端口)` 或按可选 **verbose** 模式记录跳过原因，且不将该组合记为有效资产

### Requirement: 资产记录包含 IP、端口与 host

每条有效资产输出 MUST 包含：**`ip`**（响应源 IP 字符串）、**`port`**（响应源端口整数或字符串与文档一致）、**`host`**（从 **PTR/SRV/名称标签** 按实现文档中的优先级规则推导的人类可读主机或服务实例名；若无法推导则使用 **空字符串或 `<unknown>`** 之一并在字段 **`host_confidence`** 或等价元数据中标明 **`low`**）。

#### Scenario: 响应含 SRV 与 PTR

- **WHEN** 解析结果中包含可关联到同一服务的 **SRV** 与 **PTR** 记录
- **THEN** 输出记录中 **`host`** SHALL 非空且与 **SRV Target** 或 **PTR 目标** 之一一致（按设计文档优先级），**`ip`** 与 **`port`** 与 UDP 响应源一致

### Requirement: 深度识别 banner（基础层）

每条资产 MUST 包含名为 **`banner`** 的对象。其 MUST **至少**包含下列 **基础层** 字段（键名保持稳定并在 README 列出）：

1. **`txt`**：从 **TXT** RR 提取的 **原始字符串** 或 **键值对** 列表（与报文语义等价）；  
2. **`srv`**：若存在 **SRV**，则为对象或数组，包含 **target、port、priority、weight** 中可获得的字段；  
3. **`raw_hex`**：UDP 载荷前 **N** 字节（N 在 **32～128** 间固定并在 README 声明）的 **连续小写十六进制**；  
4. **`notes`**：字符串数组，标记 **截断、附加区未解析、non-dns** 等。

#### Scenario: 含 TXT 的 mDNS 响应

- **WHEN** 响应解析后存在至少一条 **TXT** RR
- **THEN** **`banner.txt`** SHALL 非空且与报文中 **TXT 字符串** 语义等价

#### Scenario: 畸形或非 DNS UDP

- **WHEN** UDP 载荷无法被解析为 DNS 报文
- **THEN** 程序可选择 **不输出资产** 或在 **`verbose`** 下输出 **`banner.raw_hex` 与 `notes` 标明 `non-dns`**；不得 panic

### Requirement: Banner DNS-SD 深度视图（不低于示例）

在 **同一条资产** 的 **`banner`** 下，MUST 提供嵌套对象 **`dns_sd`**（名称固定；不得仅放在人类可读文本里而不出现在 **jsonl** 中），用于表达 **DNS-SD / mDNS 浏览级** 聚合信息。当底层 RR 足以支撑时，其信息深度 **不得低于** 下列「示例契约」所展示的粒度（示例为 QNAP 类 NAS 多服务宣告；字段值随目标变化，**结构能力**为验收基准）：

**示例契约（节选，验收时对照信息是否可复原）：**

```text
services:
9/tcp workstation:
Name=slw-nas [24:5e:be:69:a3:13]
IPv4=x.x.x.x
IPv6=fe80::265e:beff:fe69:a313
Hostname=slw-nas.local
TTL=10
5000/tcp http:
Name=slw-nas
...
path=/
445/tcp smb:
...
5000/tcp qdiscover:
...
accessType=https,accessPort=86,model=TS-X64,displayModel=TS-464C,fwVer=5.2.9,fwBuildNum=20260214
device-info:
Name=slw-nas(AFP)
...
model=Xserve
548/tcp afpovertcp:
...
answers:
PTR:
_workstation._tcp.local
_http._tcp.local
_smb._tcp.local
_qdiscover._tcp.local
_device-info._tcp.local
_afpovertcp._tcp.local
```

**对 `banner.dns_sd` 的规范映射：**

1. **`services`**：数组；**每个元素** 表示一个 **按「端口/传输/服务类型」可区分** 的服务视图（对应示例中 `9/tcp workstation:`、`5000/tcp http:` 等行首三元信息）。每个元素 MUST 支持下列键（值缺失时用 **null** 或空数组，但键名保留）：  
   - **`port`**（整数）、**`transport`**（`"tcp"` 或 `"udp"`）、**`reg_type`** 或等价 **`service_label`**（如 `workstation`、`http`、`smb`、`qdiscover`、`device-info`、`afpovertcp` 等 **从 PTR/SRV 推导的短名**）；  
   - **`instance_name`**（对应 `Name=`，可含厂商括号内 MAC 文本）；  
   - **`ipv4`**、**`ipv6`**（字符串数组，与 A/AAAA 一致）；  
   - **`hostname`**（如 `*.local`）；**`ttl`**（整数秒，与 RR TTL 一致或取合并策略文档化值）；  
   - **`extras`**：对象或字符串数组，承载 **非标准但高价值** 行（如 **`path=/`**、仅出现在 TXT 的 **`path`** 键），以及 **整段 TXT 解码为 `k=v` 逗号行** 的 **`txt_flat`** 或 **`txt_kv`**（二者实现其一即可，但 MUST 能无损复原 `accessType=...` 这类 vendor banner）。
2. **`answers.ptr`**：字符串数组，列出本响应合并上下文中解析出的 **PTR 名称**（对应示例 `answers:` 下列出的 `_workstation._tcp.local` 等）；**不得**在已知可解析情况下整类遗漏（允许顺序不同）。

#### Scenario: 多服务 NAS 类 golden 输入

- **WHEN** 使用测试夹具（内置 golden DNS 报文或脱敏 PCAP 转报文）作为输入，其 RR 集合语义上等价于示例中 **6 条 PTR 服务类型 + 多 SRV/TXT/A/AAAA** 的联合响应
- **THEN** 输出 JSON 中 **`banner.dns_sd.services` 长度 SHALL ≥ 6**，且 **分别** 能区分 **workstation / http / smb / qdiscover / device-info / afpovertcp** 所对应条目（通过 `reg_type`+`port`+`transport` 或文档规定的稳定复合键）；**`banner.dns_sd.answers.ptr` SHALL 包含** 上述六个 `_tcp.local` 形式名称（允许额外元素）

#### Scenario: 厂商 TXT 指纹行必须保留

- **WHEN** 某服务 TXT 中存在形如 `accessType=https,accessPort=86,model=TS-X64,...` 的连续 `k=v` 串
- **THEN** 该串 MUST 以 **原文或键值展开** 形式出现在 **对应服务条目** 的 `txt_flat` / `txt_kv` / `extras` 之一中，且可被 **无损拼接** 回等价语义（顺序不要求与原文逐字相同，但 **键集合与值** 不得丢失）

#### Scenario: 仅部分 RR 存在

- **WHEN** 响应仅有 PTR+SRV 而无 AAAA
- **THEN** **`banner.dns_sd.services` 对应条目的 `ipv6` SHALL 为空数组或 null**，且 **`instance_name`/`hostname` 仍按可得 RR 尽最大努力填充**；**不得**因缺失 AAAA 而省略该服务条目（只要 SRV/PTR 可关联）

### Requirement: 机器可读输出

程序 MUST 支持 **`--format jsonl`**（或等价长选项）使 **每条资产一行 JSON** 输出至标准输出；字段名保持稳定，且 **至少**包含 **`ip`、`port`、`host`、`banner`**。**`banner.dns_sd` 键 MUST 始终存在**：无足够 RR 聚合时其值可为 **`{}`**（空对象）；当 RR 足以支撑 DNS-SD 聚合时，**不得**用空对象代替已可得的服务或 PTR 枚举（即不得无故丢弃可解析信息）。

#### Scenario: JSON Lines 模式

- **WHEN** 用户使用 `jsonl` 格式且扫描到至少一条资产
- **THEN** 标准输出每一行 SHALL 为独立 JSON 对象，且每行 **可被 `json.Unmarshal` 解析**，并包含上述必填字段

### Requirement: 人类可读输出

程序 MUST 支持 **`--format text`**（或默认格式为 text），以 **列对齐或固定键值行** 打印资产，便于终端阅读。

#### Scenario: 文本格式包含深度摘要

- **WHEN** 用户指定文本格式且 **`banner.dns_sd` 非空**
- **THEN** 标准输出 SHALL 在单条资产块内包含 **`services:` 与 `answers:`（或标题语义等价）** 两段摘要，且 **每个服务至少一行** 包含 **port/transport/类型** 与 **Name= / Hostname= / TTL=** 中的 **至少三项**（其余项若可得则不得故意隐藏）

### Requirement: 可配置超时与并发

程序 MUST 提供 **单目标超时**（如 `--timeout`）与 **最大并发探测数**（如 `--concurrency`）配置；缺省时 SHALL 使用文档中的安全默认值。

#### Scenario: 超时生效

- **WHEN** 用户将单目标超时设为极小正值且目标无响应
- **THEN** 程序 SHALL 在约该超时量级内结束对该任务的等待且不泄漏 goroutine（可通过压力测试或 race 检测验证）

### Requirement: 扫描规模保护

程序 MUST 提供 **`--max-hosts`**（或等价）上限，用于限制从 CIDR 展开的最大地址数；超过时 SHALL **拒绝启动** 或 **要求确认 flag**（二选一并在 README 说明）。

#### Scenario: 超过上限

- **WHEN** 展开后的 IP 数量大于 `max-hosts`
- **THEN** 程序 SHALL 非零退出并在标准错误中说明原因

### Requirement: 合规提示

**`--help` / `-h`** 输出与 **`interview-project-test/README.md`** 中 MUST 包含 **「仅扫描自有或已授权网络」** 或语义等价的中文声明。

#### Scenario: 帮助文本含授权提示

- **WHEN** 用户执行 **`--help` 或 `-h`**
- **THEN** 帮助文本中 SHALL 出现授权/合规相关提示语
