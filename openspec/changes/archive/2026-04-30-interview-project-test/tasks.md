## 1. 工程骨架（已完成）

- [x] 1.1 在仓库根下创建目录 `interview-project-test/`
- [x] 1.2 初始化 `go.mod`，`module` 名与目录一致（如 `interview-project-test`）
- [x] 1.3 添加初始 `main.go`（后续由 CLI 替换占位输出）

## 2. 文档与基线验收（已完成）

- [x] 2.1 编写子工程 `README.md`：说明用途、`go build ./...` 与运行示例（当前为 `--help` 与规划中的测绘参数示例）
- [x] 2.2 在 `interview-project-test` 目录执行 `go build ./...` 通过；**占位 `main` 阶段** `go run .` 曾为零退出。**实现 CLI 后**须符合 `interview-project-test-bootstrap`：**无测绘必填参数时非零退出**、**`--help`/`-h` 为零退出**（届时以该规格重新验收）

## 3. mDNS 测绘 CLI（已完成）

- [x] 3.1 定义 CLI 参数：网段（CIDR，可预留 IP 起止）、端口范围、`--timeout`、`--concurrency`、`--max-hosts`、`-q` 查询名、`--format text|jsonl`
- [x] 3.2 实现网段与端口展开（流式/分批，避免大 CIDR OOM），超 `max-hosts` 时拒绝或需 `--i-understand`
- [x] 3.3 UDP 探测：向 `(ip,port)` 发送 DNS 格式 mDNS 查询，关联响应源地址与超时
- [x] 3.4 解析响应（推荐 `miekg/dns`）：提取 PTR/SRV/TXT/A/AAAA，按设计文档优先级推导 `host`
- [x] 3.5 构造 `banner` 基础层：`txt`、`srv`、`raw_hex`、`notes`（含 non-dns 分支）
- [x] 3.5b 实现 **`banner.dns_sd`**：`services[]`（port/transport/reg_type/instance_name/ipv4/ipv6/hostname/ttl/extras 与 `txt_flat` 或 `txt_kv`）、`answers.ptr[]`；与 **NAS 六服务 + 六 PTR** golden 对齐（`openspec` 规格中的示例契约）
- [x] 3.6 输出：`jsonl` 每行一对象且 **始终含 `banner.dns_sd` 键**；`text` 含 **`services:`/`answers:`** 深度摘要；无资产时退出码约定与 stderr 行为文档化
- [x] 3.7 更新 `interview-project-test/README.md`：完整示例、合规声明、与 OpenSpec `mdns-survey-cli` 规格对齐
- [x] 3.8 单元测试：参数解析、**banner.dns_sd** 构造、**NAS 类 golden**（六服务 + vendor TXT + 六 PTR）；`go test ./...` 通过

## 4. OpenSpec 变更内文档（本次）

- [x] 4.1 更新 `proposal.md` / `design.md` / `specs/**` / `tasks.md` 与 `openspec/changes/interview-project-test/README.md`
- [x] 4.2 实现完成后执行 `openspec validate interview-project-test` 与手工验收 CLI
