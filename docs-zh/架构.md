# 架构

平台保留上游的 Go 模块名 `agentflow`、包边界和 `AGENTFLOW_` 环境变量，以减少持续的合并冲突。

| 目录 | 职责 |
| --- | --- |
| `cmd/agentflow` | API、worker 或二者合一的进程 |
| `cmd/agentflow-runner` | 个人执行机与池化执行机 |
| `cmd/agentflow-mcp`, `cmd/agentflow-gate-mcp` | 公共 MCP 与结构化门禁结果 |
| `internal/engine` | SOP 图、租约、调度、沙箱生命周期与交付。就绪分支在运行工作区内一起执行；取消后重跑其中一个分支会恢复被取消的兄弟分支 |
| `internal/repo` | MySQL 持久化，每个业务领域一个 Repo |
| `internal/httpapi` | HTTP/MCP handler，每个业务领域一个 handler |
| `internal/runtime`, `internal/runner` | 执行器协议、凭据、隔离与制品 |
| `internal/tracker` | 远程 Issue/PR 适配器 |
| `web` | 内嵌的上游 Web 界面、偏好设置与公开登录 |
| `migrations` | 显式的数据迁移与可选的公共目录 |
| `local` | 本地工作流与宿主集成；用户指南位于 `docs/local/` |

公共版本新增账号/密码登录和 GitHub 适配器。它们接入现有的用户、令牌、仓库和交付契约；不存在单独的公共引擎，也不存在内存态的生产存储。以 `github:` 开头的远程仓库标识用于区分 GitHub 的 owner/repository 名与旧版跟踪系统标识。上游的 `gongfengProjectId` JSON 字段为兼容起见承载该标识。

CLI 的 Classic 与 Portable 工作流资产彼此独立，也与平台角色定义相互独立。共享行为契约优于复制运行时资产。平台 worker 不需要任何 Classic 宿主包。
