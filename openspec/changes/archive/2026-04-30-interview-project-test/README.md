# Change: interview-project-test

本变更在仓库内落地独立子工程 **`interview-project-test/`**（独立 Go module），并定义 **mDNS 测绘 CLI** 的产品与技术契约。

## 阅读顺序

1. [`proposal.md`](./proposal.md) — 动机、范围、能力拆分、影响面  
2. [`design.md`](./design.md) — 协议边界、CLI/并发/解析/输出设计决策与风险  
3. [`specs/interview-project-test-bootstrap/spec.md`](./specs/interview-project-test-bootstrap/spec.md) — 子工程可构建、README、**CLI 入口** 行为  
4. [`specs/mdns-survey-cli/spec.md`](./specs/mdns-survey-cli/spec.md) — 网段/端口、mDNS 探测、资产字段、**`banner` 基础层 + `banner.dns_sd`（DNS-SD 浏览级深度，含 NAS 类示例契约与 golden 场景）**、输出与安全上限  

## 实现状态

- **工程骨架**：已完成（见 [`tasks.md`](./tasks.md) 第 1～2 节）。  
- **mDNS 测绘 CLI**：见 `tasks.md` 第 3 节，与 `mdns-survey-cli` 规格逐项对齐后实现。

文档或规格修改后建议运行：`openspec validate interview-project-test`（与实现是否完成无关）。
