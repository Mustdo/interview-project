## ADDED Requirements

### Requirement: 独立 Go 模块可构建

`interview-project-test` 工程 MUST 在仅包含该子树上下文时能够通过 `go build ./...` 成功编译。

#### Scenario: 在子目录根执行构建

- **WHEN** 在 `interview-project-test` 目录下执行 `go build ./...`
- **THEN** 命令以零退出码结束且无编译错误

### Requirement: 文档说明用途与用法

工程 MUST 包含 `README.md`，说明本项目的定位，以及如何构建与运行。

#### Scenario: 新贡献者阅读 README

- **WHEN** 阅读 `interview-project-test/README.md`
- **THEN** 文档中 SHALL 包含用于构建与运行的 `go` 命令示例

### Requirement: 可运行入口（测绘 CLI）

系统 MUST 提供 `main` 包入口；默认行为 SHALL 为 **测绘 CLI**（见 `mdns-survey-cli` 能力规格），且 MUST 支持通过 **`--help` / `-h`** 打印用法说明。若缺少必填参数（如未指定网段或端口范围），程序 SHALL 打印简要用法或错误提示并以 **非零退出码** 结束。

#### Scenario: 查看帮助

- **WHEN** 在 `interview-project-test` 目录下执行 `go run . --help` 或 `go run . -h`
- **THEN** 标准输出包含子命令或全局参数说明（至少包含网段、端口范围相关说明或占位），进程以退出码 **0** 结束

#### Scenario: 缺少必填参数

- **WHEN** 在 `interview-project-test` 目录下执行 `go run .` 且未提供测绘所需的最小参数集合
- **THEN** 标准错误（或实现文档明确约定的诊断输出）中包含错误原因或用法提示，进程以 **非零** 退出码结束
