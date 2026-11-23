# 部署

## 原生开发

`bash scripts/dev.sh` 会显式配置本地密钥，只启动当前检出的 MySQL 服务，构建平台，预置 `admin` 账号，通过显式脚本安装缺失的可观测性生成列，应用 `migrations/seed/*.sql`，并在 8090 端口启动服务端。已有的目录定义由 `INSERT IGNORE` 保留。请把 `.env`、`.local/` 和数据库数据卷放在版本控制之外。

原生安装时，在 API 宿主上使用 Node.js 22+，并安装你计划使用的 CLI。以 Codex 为例：

```bash
npm install --global @openai/codex@0.162.1
codex --version
```

在个人设置中配置该 CLI 的凭据。若选择 CodeBuddy，请使用它的网络环境，并把它的可执行文件放进 PATH 或设置 `AGENTFLOW_CODEBUDDY_BIN`。生产镜像内置 Codex、DeepSeek Harness，并默认内置 CodeBuddy。按需 CLI 缺失时，会话中会显示安装操作。使用已发布镜像的 SOP 沙箱会共享 Claude Code、OpenCode、Kimi Code 和 Pi 的首次使用安装。已连接的 runner 自行管理其 CLI 安装。

WeCom 通知要求平台服务的 PATH 上存在 `wecom-cli`。由本 Dockerfile 构建的平台镜像已包含 `@wecom/cli@1.3.4`；已发布的镜像会在下个版本加入。原生安装时请单独执行 `npm install --global @wecom/cli@1.3.4`，确保系统 CA 证书已安装，并以服务用户的身份验证 `wecom-cli --version`。发送通知仍然需要配置 WeCom 机器人。

把 `AGENTFLOW_MANAGED_CLI_ROOT` 设为持久化的绝对目录，即可在执行运行记录的平台进程（`worker` 或 `all`）上启用每日 CLI 更新器。默认检查 Claude Code、CodeBuddy 和 Codex。把 `AGENTFLOW_MANAGED_CLIS` 设为你希望更新的完整逗号分隔列表，例如 `claude-code,codebuddy,codex,pi`。其余提供商 ID 为 `agy`、`cursor`、`dsh`、`kimi`、`opencode` 和 `pi`。更新器会忽略空条目，包括末尾的逗号。不含任何名称的列表使用默认值，出现未知 ID 则拒绝本次更新。该列表适用于定时检查、批量 CLI 更新命令以及 Runner 上的托管 CLI 更新。已有安装和回滚仍然可用。请在共享同一个托管根目录的 API 与 worker 进程上设置相同的列表。API 在解析 CodeBuddy 时使用该列表，worker 则在每个根目录上每天认领一次检查。

更新器每天检查上游元数据，并在需要时下载符合条件的版本。它在运行 npm 生命周期脚本之前，用子资源完整性（SRI）校验 npm 包。Antigravity 归档文件按官方清单的 SHA512 校验。官方 Cursor 安装器提供带版本号的 URL，但不提供内容校验和。Cursor 下载的完整性依赖 TLS 以及匹配的强 ETag，即归档服务端给出的不透明标识。[Cursor 安装脚本](https://cursor.com/install)

worker 需要能访问所选 CLI 的来源，并可在需要时配置 `HTTPS_PROXY`。更新器会按提供商报告下载失败。

所有候选版本都要经过 24 小时观察窗口，并通过版本与 CLI 参数检查后才能启用。更新器会把已安装版本保留在磁盘上，用于回滚和工作流锁定版本，因此随着安装的版本增多，占用空间也会增长。在启用更多提供商之前，如果你打算回滚平台版本，请备份托管根目录。较旧的平台版本只接受包含最初三个提供商的清单。把兼容的备份还原到原始绝对路径，以保留已记录的可执行文件路径。CLI 回滚之后，自动更新会等待比被拒绝版本更新的版本。

原生执行时，平台优先使用显式的可执行文件覆盖项，其次使用托管清单。平台自有的原生工作流会锁定其托管根目录内的清单。接管运行记录的 worker 若以相同的绝对路径访问同一个持久托管目录，且可执行文件兼容，就可以复用该锁定版本。使用独立根目录的 worker 从自己的清单解析 CLI。

更新器会跨每日检查保留对 Cursor 与 Antigravity 版本的观察记录，这样即使厂商持续发布，每个版本仍能走完观察期。Docker 沙箱使用其镜像或沙箱安装器提供的 CLI。ZCode 使用部署提供的可执行文件。

基于自定义 Dockerfile 且不含 Node.js、npm 或 Python 3 的会话工作区会使用平台自带的副本。首次使用时，平台把自身的 Node.js、npm 与可重定位的 Python 复制到 `$LOOPFORGE_DATA_DIR/cli/toolchain/<build>/`，并在某个沙箱首次需要时把每个内置 CLI（CodeBuddy、Codex、DeepSeek Harness）一并复制进去，然后以只读方式挂载到沙箱 `PATH` 的末尾。镜像自带的工具优先。只有与平台架构一致的 glibc 镜像才使用这套机制；Alpine（musl）镜像需要自备 Node.js、npm 和 Python 3。新的平台构建会创建新目录并保留旧目录：升级后请删除其余的 `cli/toolchain/<build>/` 目录（每个约 200 MB，另加复制进去的各 CLI）。

用 Ctrl-C 停止前台服务，再用同一命令启动它。MySQL 数据保存在其 Compose 数据卷中。`docker compose stop mysql` 会停止数据库但不删除它。

自有 MySQL 部署请使用 `AGENTFLOW_MYSQL_DSN`。全新的账号命令或服务端连接会创建缺失的 schema 对象。对已有安装做数据迁移时，请先审阅各迁移的前置条件，再逐个执行；不要盲目把全部历史迁移重放到新数据库上。

## 选择执行器

- 需求澄清、Bot 回复、SOP 通知报告、运行评估和定时 AI 任务的模型选择器都接受受支持 CLI 的模型。API 客户端可以发送 `codex:<model>`、`claude-code:<model>` 或其他带 CLI 前缀的模型 ID。早期的裸模型 ID 仍视为 CodeBuddy ID。省略模型或只写 `default` 时，如果执行账号配置了个人默认模型就使用它。回退顺序中现有的 CodeBuddy 默认值优先。具体模型会原样传给对应的 CLI。
- 在**个人配置**中，为每个仓库宿主添加 Git 凭据，为你使用的每个 CLI 添加执行器凭据。Git 按仓库协议、宿主和端口选择凭据。第一个配置的执行器会成为个人默认执行器。你可以更改它，并选择默认模型、推理强度和对话审批模式。CodeBuddy 除网络环境外还可以配置可选的 Base URL。把该 URL 留空则使用网络环境的官方端点。在远程执行机上使用自定义 CodeBuddy URL 需要 Runner 0.11.2 或更高版本。
- 澄清使用经办人的凭据，未指定经办人时回退到项目所有者。Bot 回复使用绑定的发送账号，报告使用运行记录上的负责账号，评估使用发起请求的账号。辅助审批使用该轮对话的 CLI 与所选模型。
- SOP 角色与静态预览都可以各自选择 CLI。新建的角色和静态预览仍默认使用 CodeBuddy。要让内置模板改用其他 CLI，可把它复制到项目中并更新其角色，或以平台管理员身份编辑该模板。
- 在平台上设置 `AGENTFLOW_BASELINE_ALIGN_EXECUTOR=codex`，即可用 Codex 运行定时基线冲突消解。该设置接收内置执行器 ID，包括用于 Claude Code 的 `claude`。该设置为空时，冲突消解默认使用 CodeBuddy。新的运行记录会把配置的执行器写入快照。
- 要在源码构建中去掉 CodeBuddy，可向 Docker 传入 `--build-arg CODEBUDDY_PACKAGE=`，或给开发用 Compose 构建加上 `CODEBUDDY_PACKAGE=` 前缀。把该镜像用作平台沙箱的 `AGENTFLOW_ISOLATED_IMAGE`。没有共享镜像时，为内置回退 Dockerfile 设置 `AGENTFLOW_SKIP_CODEBUDDY_INSTALL=1`。自定义 Dockerfile 自行管理软件包。使用该标志后，默认托管 CLI 列表也会去掉 CodeBuddy。设置 `AGENTFLOW_MANAGED_CLIS` 可显式管理提供商列表。
- 追踪系统 MCP 认证使用目录中显式配置的请求头，或部署的 `AGENTFLOW_MCP_TOKEN`。项目级追踪系统使用自己的 MCP 授权请求头。
- 把 `AGENTFLOW_GONGFENG_HOST` 设为托管的 Git 宿主（主机名与可选的 HTTPS 端口），即可启用其仓库与合并请求链接以及令牌检查。工蜂 OAuth 还需要其应用凭据和回调。已有的 OAuth 端点覆盖设置仍然优先。
- 需要时把 `AGENTFLOW_TAPD_HOST` 设为部署专用的 TAPD Web 宿主。生成的 TAPD 工作项链接默认使用 `www.tapd.cn`。

## 容器安装

```bash
./start.sh                        # install or start in the background (same as bash scripts/install.sh)
./start.sh logs                   # follow platform logs
./start.sh stop                   # stop; ./start.sh down removes containers but keeps data volumes
./start.sh restart                # restart the platform container
./start.sh pull                   # refresh images; times ghcr.io against ghcr.nju.edu.cn
MIRROR=ghcr.io ./start.sh         # install from ghcr.io without the speed test
MIRROR=ghcr.nju.edu.cn ./start.sh # force that host instead of the speed test
./start.sh dev                    # build this checkout instead (compose.dev.yaml)
./start.sh release                # switch a source build back to the published image
```

安装器会检查 Docker、Compose、Python 3 和 curl。它会在 Linux 上用 apt、dnf 或 yum（macOS 上用 Homebrew）安装缺失的 Python/curl 软件包，用官方安装器安装 Docker，并用经过 SHA-256 校验的固定官方版本安装缺失的 Linux x86_64/arm64 Compose 插件。插件会为当前使用 Docker 的用户安装，需要 sudo 时则安装到系统范围。macOS 的 Compose 来自 Docker Desktop。设置 `LOOPFORGE_YES=1` 可自动接受安装提示；仍然需要网络访问以及包管理器/管理员权限。安装器会生成带唯一密钥的 `.env`，拉取并启动容器，安装可观测性列并应用 `migrations/seed/*.sql`。重复运行是安全的。若默认端口已被占用，可在首次运行前在环境中设置 `AGENTFLOW_HTTP_PORT` 或 `LOOPFORGE_MYSQL_PORT`（也可以稍后编辑 `.env`）。

请以安装所有者的身份运行它，普通 Linux 用户也可以。如果该用户不能直接访问 Docker，安装器会校验 sudo，并一致地使用 `sudo -n docker` 执行 Compose、安装表结构和种子数据。每次操作前它都会检查 sudo 凭据是否在长时间构建中过期，在终端中重新提示，或在无人值守模式下报错并给出说明。它不要求把用户加入 Docker 组，也不要求重新登录。无人值守安装请先执行 `sudo -v`；`LOOPFORGE_YES=1` 只接受安装提示，不会绕过操作系统的权限。没有 sudo 或 Docker 访问权限时，它会立即失败并给出说明。Rootless Docker 使用其当前上下文的 Unix socket；远程 Docker 上下文需要另行配置部署，因为沙箱工作区是本地绑定挂载。

`compose.yaml` 是已发布镜像的编排栈，从不执行构建：`./start.sh`、`scripts/install.sh` 以及直接执行 `docker compose --profile platform up` 都会拉取其中固定的 GHCR 发布镜像（当前为 `ghcr.io/tencent/loopforge:0.4.2`）。该固定版本与检出中的迁移和种子数据相匹配，因此安装时请检出对应的发布标签。仅在需要覆盖时设置 `LOOPFORGE_IMAGE`（例如设为某个 `@sha256:` 摘要）；保存在 `.env` 中的值优先于之后对固定版本的修改。

升级时，先更新检出，再刷新容器：

```bash
git fetch --tags && git checkout vX.Y.Z
./start.sh pull
```

新的 `compose.yaml` 会同时带来镜像固定版本和新的平台设置，只要其中有变化，`./start.sh pull` 就会重建平台容器。`./start.sh dev` 产生的源码构建不会被 `./start.sh pull` 重建；更新检出后需要再次运行 `./start.sh dev`。只运行 `./start.sh pull` 会保留已固定的版本。若 `.env` 中保存了 `LOOPFORGE_IMAGE`，也要把它改成所检出的发布版本；较新的镜像搭配较旧的 `compose.yaml` 可能缺少该版本需要的设置。

在 `ghcr.io` 下载受阻的环境中，`./start.sh` 和 `./start.sh pull` 会对 `ghcr.io` 与 [ghcr.nju.edu.cn](https://doc.nju.edu.cn/books/e1654/page/ghcr) 做测速。只有南京站的清单摘要与刚从 ghcr.io 读到的摘要一致时才会使用它；随后拉取的是该摘要，再标记回 `compose.yaml` 中的名称，因为该文件信任那个宿主。官方摘要缺失、摘要不一致或镜像站拉取失败时，都会留在 `ghcr.io`。`MIRROR=ghcr.io` 强制使用上游仓库；其他任何 `MIRROR` 宿主都会强制使用该仓库、跳过测速且不再回退。该选择不会写入 `.env`。安装时的拉取策略仍为 `missing`，因此本地已有镜像会被保留，也不会再次测速；`./start.sh pull` 会重新测速各仓库并从胜出者刷新。GitHub Release 会记录镜像摘要；执行 `docker image inspect --format '{{index .RepoDigests 0}}' ghcr.io/tencent/loopforge:<pin>` 后，`@` 之后的摘要应当一致。Docker 的 `registry-mirrors` 设置（包括 `https://docker.nju.edu.cn`）只作用于 Docker Hub，不能用作 `MIRROR`。不是 `ghcr.io` 名称:标签形式的 `LOOPFORGE_IMAGE` 值会保持原样。

```bash
./start.sh
MIRROR=ghcr.io ./start.sh pull
MIRROR=ghcr.nju.edu.cn ./start.sh pull
```

`./start.sh dev`（或 `LOOPFORGE_BUILD_FROM_SOURCE=1 bash scripts/install.sh`）会从 `compose.dev.yaml` 把当前检出构建为 `loopforge-platform:local`。它把 `COMPOSE_FILE=compose.yaml:compose.dev.yaml` 保存在 `.env` 中，因此后续运行和直接执行的 `docker compose` 命令会继续使用源码构建，直到执行 `./start.sh release`。源码构建会复用 npm、Go 模块和 Go 编译器的缓存。

在 `main` 上推送 `vX.Y.Z` 标签时，只要该提交上的 `platform` 工作流已通过，就会运行 `release` 工作流。被标记的 `compose.yaml` 必须已经固定 `loopforge:X.Y.Z`。该工作流会把 `ghcr.io/<owner>/loopforge` 发布为 `X.Y.Z`、`X.Y`、`latest` 和 `sha-<commit>`，用 `./start.sh` 对已推送的摘要做冒烟测试，并创建 GitHub Release。复刻仓库以自己的所有者名义发布；分发镜像前请为你的团队设置 GHCR 软件包的可见性/访问权限。默认只在镜像缺失时才拉取；更新可变标签时请改用不可变标签/摘要，或设置 `LOOPFORGE_PULL_POLICY=always`。已有的 `.env` 值优先；更新安装时请在那里修改已保存的镜像或 URL。Compose 会在安装 schema 扩展和种子数据之前，以有界超时等待 API 的健康检查。

打开平台并在浏览器中创建第一个账号；该账号会获得平台管理权限。在该账号创建之前，登录页只提供这个初始化步骤。

初始设置期间，请把云安全组或防火墙限制为运维人员的 IP。在管理员账号创建完成之前，不要把首账号设置页面暴露给任意互联网客户端。

平台监听 `LOOPFORGE_BIND`，默认 `0.0.0.0`，不会提示输入访问 URL。安装器会打印公网、局域网和本地地址；在腾讯云、阿里云、AWS、GCP 和 Azure 上，它会从提供商的链路本地元数据服务读取公网 IP，其他位于 NAT 之后的环境需要你自行填写公网 IP。`LOOPFORGE_PUBLIC_URL` 是可选项，只改变打印出来的地址。请在云安全组/防火墙中放行 HTTP 端口，或使用 HTTPS 反向代理。已有的 `.env` 文件会被保留；没有绑定设置的旧安装现在也会监听 `0.0.0.0`。仅供本地使用的安装可以设置 `LOOPFORGE_BIND=127.0.0.1`。请代理 WebSocket 和流式请求。在代理处终止 HTTPS 时设置 `AGENTFLOW_SECURE_COOKIES=true`。请让 `AGENTFLOW_SESSION_SECRET` 在多个 API 实例之间和重启之后保持不变。

企业认证网关可以通过 `AGENTFLOW_TRUST_IDENTITY_HEADERS=true` 以请求头方式登录用户。该选项默认关闭。此时 API 只接受来自回环对端的 `Staffname` / `x-staff-name`，并且只有在能判定请求来自网关时才接受：

- 把 `AGENTFLOW_IDENTITY_HEADER_SECRET` 设为随机值，并让网关通过 `X-AgentFlow-Gateway-Secret` 发送它。网关必须覆盖而不是透传客户端的 `Staffname`、`Staffid`、`x-staff-name` 和 `X-AgentFlow-Gateway-Secret` 请求头。
- 未设置密钥时，任何携带 `X-Forwarded-For`、`Forwarded` 或 `X-Real-IP` 的请求都会被当作经代理转发的客户端，其身份请求头会被忽略。这只保护会添加这些头之一的代理：同宿主上不添加这些头、直接透传客户端请求头的代理（例如未设置 `proxy_set_header X-Forwarded-For` 的 nginx `proxy_pass`），以及宿主上的其他进程，仍然可以冒充任意用户登录。只要开启请求头登录就请设置密钥；缺少密钥时平台会在启动时记录一条警告。

### Docker 沙箱

即使尚未安装目录 SQL，平台也会提供一个内置的默认沙箱（`local`）。它是内存中的回退实现，不会插入数据库行。已保存的 `local` 配置文件优先，包括其配置的驱动和环境。默认沙箱无法删除；要改变任务的运行方式，请编辑其配置或创建另一个配置文件。完整安装器仍会导入入门 SOP 和欢迎项目。

对于以普通宿主用户身份运行的原生服务，default/root Docker 镜像会以该服务用户的 UID/GID 运行，从而在不放宽权限、不添加 capability 的情况下保持私有工作区可访问。显式选择非 root 用户的镜像会保留该身份；运维人员必须为这些镜像安排兼容的绑定挂载属主。

Compose 部署会把宿主的 Docker socket 挂载进平台容器，使内置 AI SOP 开箱即用。`AGENTFLOW_ISOLATED_IMAGE` 指向已安装的平台镜像：默认沙箱会用同一镜像启动同级的独立容器，无需再次下载 CLI 或构建镜像。它们不挂载 Docker socket。自定义 Dockerfile 仍会构建并缓存自己的镜像。运行工作区挂载在 Compose 网络的 `/workspace`。由于绑定挂载由宿主 Docker 守护进程解析，`LOOPFORGE_DATA_DIR`（由 `scripts/configure.sh` 写入 `.env`，默认 `./.data`）会以相同的绝对路径挂载进平台容器；工作区和 CLI 状态都存放在该目录下。原生会话工作区也以同样方式绑定挂载其会话根目录，因此 `compose.yaml` 把 `AGENTFLOW_CONVERSATION_DIR` 设为 `${LOOPFORGE_DATA_DIR}/conversations`。Compose 部署中的会话工作区使用 Docker 沙箱：原生（bwrap）沙箱需要用户命名空间和 PID 命名空间，非特权的平台容器无法创建，而且镜像不提供 bubblewrap。原生沙箱请使用原生安装或已连接的执行机。附件和其他按用户区分的助手状态保存在 `AGENTFLOW_ASSISTANT_SESSION_DIR`（`platform-data` 数据卷中的 `/data/assistant`），该目录永远不会挂载进工作区。未设置 `AGENTFLOW_CONVERSATION_DIR` 的平台容器也会把会话根目录放在那里，此时打开会话工作区会以 `mkdirat ... read-only file system` 失败。

使用标准 CLI 执行器的 Docker SOP 节点会先在沙箱用户的 PATH 上查找该 CLI。找不到时，LoopForge 会在有共享安装的情况下使用平台镜像的共享安装，否则在容器中运行与会话工作区相同的安装器。安装会使用沙箱配置的代理，该容器中后续的节点会复用该 CLI。在容器内安装需要可写的文件系统和访问官方软件源。只读沙箱可以使用镜像中的 CLI 或挂载的工具链。ZCode 使用镜像提供的官方可执行文件。

访问 Docker socket 等同于拥有宿主上的 root 权限。由此有两条防护措施：平台会在沙箱容器中运行每个任务，包括配置为 `local` 驱动的 SOP（`AGENTFLOW_LOCAL_SANDBOX=docker`），并且只有平台管理员才能把仓库指向宿主上的源路径。即便如此，任何能运行 SOP 的人都能在该宿主的容器中执行代码，因此只给信任的人开账号，也不要把这个服务暴露给不受信任的网络。Agent CLI 还会按各自的原生规则应用所绑定仓库自己的配置，无人值守的 SOP 运行同样如此：`AGENTS.md`/`CLAUDE.md` 指令、`.codebuddy/`、`.claude/settings.json`（钩子与环境变量）和 `.mcp.json` 都能在沙箱内启动命令或 MCP 服务器。只绑定你信任其内容的仓库。如果要在不使用 socket 的情况下运行，请从 `compose.yaml` 中移除 `docker.sock` 卷和 `AGENTFLOW_LOCAL_SANDBOX` 设置，再为 AI SOP 连接一个独立的、具备 Docker 能力的 worker 或执行机；此时仅含脚本的欢迎 SOP 会在平台容器内运行。

### 慢速网络：克隆与镜像构建限制

克隆仓库和构建自定义沙箱镜像都会在执行过程中报告进度，出错时会保留 Git 或 Docker 输出的最后几行。会话会在平台上和已连接的 runner 上显示当前的 Docker 构建步骤。耗时且无输出的 Dockerfile 步骤会一直继续，直到其调用方或配置的上限停止构建。所有设置都接受 Go 时长，例如 `90s` 或 `20m`。设为 `0` 即关闭限制。请在运行该命令的平台进程或执行机上设置这些变量。

Runner 会话工作区的准备工作跟随会话请求的生命周期。停止该轮对话或断开 runner 控制连接都会取消准备工作。

| 设置 | 默认值 | 适用对象 |
|---|---|---|
| `AGENTFLOW_GIT_CLONE_STALL_TIMEOUT` | `5m` | 无任何输出的 `git clone`。平台从仓库的本地源路径克隆时不适用，因为其最后的 repack 是静默的。 |
| `AGENTFLOW_GIT_CLONE_TIMEOUT` | 平台默认关闭，执行机默认 `10m` | 整个 `git clone`。执行机保留其原先固定的 10 分钟上限作为默认值。 |
| `AGENTFLOW_DOCKER_BUILD_STALL_TIMEOUT` | 关闭 | `docker build` 可选的无输出限制。静默的 `RUN` 步骤也算无输出。 |
| `AGENTFLOW_DOCKER_BUILD_TIMEOUT` | 关闭 | 整个 `docker build`。 |

这样被停止的克隆不会重试，因为重试仍会等待同一网络。当执行机访问 GitHub 或 Debian 镜像源很慢时，请让 git 走代理（`git config --global http.proxy`），并在 Dockerfile 中使用就近的软件包镜像源，而不是调高这些上限。

### 通过 HTTPS 访问自托管 Git 仓库

在**平台管理 → Git 宿主信任**中添加 HTTPS 宿主，端口不是 443 时也要包含端口。解析主机名或手动输入已批准的 IP 地址，然后审阅并确认列表。对于 IPv4 服务器，在宿主字段中输入其主机名，并在授权中包含其 IPv4 地址。直接使用 IPv6 连接时，端口 443 输入 `fd00::2`，自定义端口输入 `[fd00::2]:8443`。授权覆盖该 IP，并且 Git 服务端的 TLS 证书必须在其主体备用名称中包含该 IP。克隆 URL 使用带方括号的形式，例如 `https://[fd00::2]:8443/team/repo.git`。

已登录用户可以在**个人设置 → Git 访问令牌**中看到实例的主机名或 IPv6 地址以及 Git 提供商。他们可以选择该实例并输入自己的令牌。已批准的 IP 地址列表显示在平台管理中。当 GitLab 返回的合并请求链接使用不同的宿主或端口时，LoopForge 会用其查询过的已授权端点来跟踪该 MR。

平台会在拨号进行令牌校验、拉取、合并请求发现和状态同步之前检查已批准的地址。HTTPS 连接使用平台的常规 TLS 信任链。对于每个已注册的主机名和端口，平台只拨号该授权中的地址。`AGENTFLOW_GIT_EGRESS_ALLOW_CIDRS` 只对没有 HTTPS 授权的宿主生效。

从部署 CIDR 迁移宿主时，请把平台应能访问的每个 IP 都包含进授权。要轮换地址，请编辑已保存的列表，并在确认前审阅新增和移除的地址。

### 通过 SSH 访问 Git 仓库

对于 GitLab.com、自托管 GitLab、Gitea、Gogs、GitHub 以及其他提供该传输方式的 Git 宿主，请使用 Git SSH。在**项目设置 → 代码仓库**或**快速访问**中，输入克隆地址，例如 `git@gitlab.com:group/repo.git` 或 `ssh://git@git.example.org:2222/group/repo.git`，并选择已有的默认分支。GitHub HTTPS 仍然使用 GitHub 授权或 PAT。

1. GitHub（`github.com:22` 与 `ssh.github.com:443`）使用内置的宿主密钥。对于其他宿主，由平台管理员打开**平台管理 → Git 宿主信任**，输入主机名和可选端口，并选择**读取密钥并审阅指纹**。平台会通过受 `AGENTFLOW_GIT_EGRESS_ALLOW_CIDRS` 约束的 SSH 握手读取公钥。对于私有宿主，请选择一台在线、能访问该服务器并支持宿主密钥发现的个人 Runner。你也可以手动输入 OpenSSH 公钥，包括为同一宿主输入多个密钥。确认前请把显示的 SHA256 指纹与 Git 提供商或服务器管理员核对。注册会对该确切的主机和端口在整个平台生效。修改和删除会在下一次平台托管的 Git 操作或新启动的工作区 CLI 或终端会话中生效。
2. Runner **0.11.1** 及更高版本会在任务或工作区请求需要 Git SSH 时收到匹配的平台条目。部署也可以把 `AGENTFLOW_GIT_SSH_KNOWN_HOSTS` 指向一个专用的 OpenSSH `known_hosts` 文件。使用 Compose 时，把该文件放在 `LOOPFORGE_DATA_DIR` 绑定挂载下，并在 `.env` 中设置其绝对路径。平台文件和平台管理页面中的条目会合并。文件中请使用确切的主机名或 IP 地址，非默认端口使用 `[host]:port`。对于同一主机和端口，平台提供的受信密钥优先。当平台只提供吊销信息或该端点没有条目时，Runner 本地的密钥生效。两个来源中的显式 `@revoked` 条目都会生效。每次分发的编码条目上限为 1 MiB。Runner **0.9.3** 至 **0.11.0** 仍需自己的文件和变量。独立的 worker 可以从共享数据库读取注册表，或使用自己的部署文件。

   如果数据库注册表不可用，平台会记录错误并继续使用部署文件和内置 GitHub 密钥。Runner 本地密钥仍按相同的优先级规则可用。配置的 `known_hosts` 文件无法读取时仍会阻断 Git SSH 操作，因为它可能携带吊销信息。平台管理列表会把无效条目标记为待移除，主机名有效的条目可以就地修复。无效条目不提供任何受信密钥。

   请在执行机和自定义沙箱镜像中安装 Git 与 OpenSSH 客户端。Docker 沙箱命令需要沙箱与执行进程共享同一 Linux 内核，以支持签名 agent 的 socket。若 Docker VM 与原生 runner 分离，请使用托管的检出与交付，或为交互式 Git 命令使用原生沙箱。
3. 为无人值守执行生成不带口令的专用 SSH 密钥。把每个公钥添加到你的托管账号，并授予其所需仓库的访问权限。若运行需要推送，请授予写权限。在**个人设置 → Git SSH 密钥**中，为每个 SSH 宿主添加一个私钥，例如 `gitlab.com` 或 `git.example.org:2222`。省略端口表示 22。编辑该宿主的条目即可替换其密钥。

仓库会自动使用当前用户针对其 SSH 宿主和端口的密钥；同一宿主和端口上的仓库共享该密钥。仅当该宿主没有个人密钥时，仓库表单才会显示红色警告。该提醒不会阻止保存仓库或完成设置；凭据和宿主信任可以在首次 Git 操作之前配置。不存在仓库专属密钥或默认回退密钥。检出和 SOP 执行使用负责用户的密钥。会话提交和工作区终端使用发起请求用户的密钥。临时签名 agent 会在命令或终端结束时关闭。终端的 agent 绝不会留在共享工作区中：使用 `bwrap` 驱动时，它位于一个仅绑定到该终端的私有目录；使用 `local` 驱动（无隔离）时，它位于工作区之外的一个 0700 目录中。Docker 和 seatbelt 工作区在所有进程间共享同一文件系统视图，因此其终端不会获得个人 SSH 密钥；请在那里使用平台的提交操作进行推送。`git@github.com:` 和 `ssh://git@github.com/` 地址会被保存为 GitHub 仓库（HTTPS + 令牌，自动创建 PR），除非显式把 `provider` 设为 `ssh`。要轮换密钥，请在设置中替换它，并在宿主上吊销旧的公钥。

SSH 负责克隆、拉取和推送。会话交付会返回分支和提交 SHA。你还可以选择添加仓库的 Web URL，以便打开宿主并手动创建 PR/MR。Issue 导入和 PR/MR 自动化需要托管 API 凭据。要自动化这些操作，请在**设置 → 资产 → MCP** 中添加托管方专用的 MCP，为它分配单独的作用域令牌，并把它附加到相关的 agent 或工作流上。[CNB 不支持 Git SSH](https://docs.cnb.cool/en/guide/git-access.html)；它使用 HTTPS 加 CNB 访问令牌，不在本集成范围内。

## 账号

在浏览器中创建的第一个账号会成为平台管理员。此后，只要 `AGENTFLOW_ALLOW_SIGNUP` 为 `true`（默认值），任何能访问登录页的人都可以注册为普通用户。注册用户可以在你的 worker 上创建需求和运行 SOP，因此对于任何可从不受信任网络访问的安装，请设置 `AGENTFLOW_ALLOW_SIGNUP=false`，改为显式创建账号。

平台管理员在**平台管理 → 成员账号**中创建账号：输入登录名和初始密码，可选地授予管理权限，然后把密码交给同事。同一页面会列出所有账号，并可重置密码账号的密码；通过 GitHub 或网关登录的账号无法在那里设置密码。新账号会像自助注册一样加入欢迎项目。

LoopForge 假定团队是可信的。每个已登录账号都能列出所有项目、需求和运行记录。项目设置和仓库只有项目所有者或平台管理员才能修改，个人凭据只有其所有者才能修改。不要用一套安装来隔离相互不可信的租户。

没有浏览器时，导出原生安装的 `.env` 值并运行显式的账号命令：

```bash
set -a; source .env; set +a
read -r -s -p 'Password: ' account_password; echo
printf '%s\n' "$account_password" | go run ./cmd/loopforge-account -user alice
unset account_password
```

仅在需要授予平台管理权限时使用 `-admin`。再次运行该命令会重置该账号的密码。密码哈希使用 bcrypt。浏览器会话使用签名的 HttpOnly、SameSite Cookie，并在七天后过期。不会有任何具名个人获得隐式管理员角色；其他管理员可以通过现有的平台管理界面管理。

## Worker 与 runner

上游的进程角色仍为 `-role api`、`-role worker` 和 `-role all`。API 和 worker 实例必须共享 MySQL 数据库以及适合其部署的配置。分离控制与执行时，设置 `AGENTFLOW_CONTROL_TOKEN`、`AGENTFLOW_CONTROL_URL` 和 `AGENTFLOW_WORKER_URL`。把 `AGENTFLOW_PLATFORM_MCP_URL` 设为执行容器可访问的实际平台 URL。

个人 runner 通过执行机页面用短期绑定码连接。安装脚本位于 `/runners/install.sh`。`AGENTFLOW_RUNNER_BINARY` 可以指向带 `{os}` 和 `{arch}` 替换的二进制文件。容器镜像内置 Linux/macOS amd64/arm64 的 runner 与 gate 二进制文件。原生源码安装可以按需用 Go 构建。

macOS 上的 Docker 工作区使用 Linux Runner 可执行文件来处理 Git 凭据和 MCP 调用。安装器会选择已安装 macOS Runner 的架构，包括在 Rosetta 下运行的 Intel 二进制文件。要为 Apple Silicon 提供通用 Runner，它会安装 arm64 辅助程序，并在可用时添加 amd64 辅助程序。前台启动和 `--no-start` 打印的命令会选择原生 arm64 切片，从 Rosetta shell 启动时也一样。在 Intel Mac 上，安装器会安装 amd64 辅助程序。默认情况下，它把辅助程序放在宿主二进制文件旁边。Runner 侧带 `{os}` 的 `AGENTFLOW_RUNNER_BINARY` 模板会选择安装路径和运行时路径。要同时安装两种辅助程序，请使用 `{arch}` 并为每个架构指定不同的目标位置。安装器会保存该模板以便升级，并把它写入 launchd 以及 `--no-start` 打印的启动命令中。辅助程序路径必须解析为与 macOS Runner 分离的文件。

在替换已安装的 Runner 之前，安装器会对照暂存的 macOS 二进制文件检查辅助程序的目标位置，并安装所需的 Gate MCP 辅助程序。当已有 Runner 带有可用的 Docker 辅助程序时，升级要求新 Runner 配置的路径上存在兼容的辅助程序，然后才会替换二进制文件并重启服务。已安装 launchd 配置中的相对模板使用该任务的工作目录。安装器和更新器在替换 Linux 可执行文件之前会校验其格式、架构和加载段。自定义安装会保留兼容的现有辅助程序。从本地检出构建的 Runner 可以从该检出构建其配套程序。带 `{os}` 占位符的下载 URL 在作为 Runner 下载源使用时可以提供这两个二进制文件。

启用平台更新后，macOS runner 会在启动时于后台同步辅助程序。版本检查失败会在重新扫描期间重试，包括旧辅助程序仍可用的情况。在同步进行期间，可读取且兼容的本地 ELF 或本地配置的脚本包装器可以被复制到 Docker 工作区并赋予可执行权限。需要 Git 或 MCP 缺失辅助程序的操作最多可以等待初始同步五秒。后台下载会在其自有的传输超时内继续。

平台管理员可以在**平台管理 → 平台共享 CodeBuddy Key**下共享一个 CodeBuddy API Key：选择签发它的网络环境以及成员可使用的模型（第一个为默认）。没有自己密钥的成员便可以用它聊天和运行 SOP，但仅限于这些模型；若某个角色配置了其他模型，则以默认模型运行，运行记录中会说明这一点。拥有自己密钥的成员继续用自己的密钥使用任意模型。澄清、报告、Bot 和运行评估使用相关账号的个人凭据。定时提示词遵循所选 CLI 的凭据与赞助设置。该密钥与个人密钥一样存储在数据库中，保存后不再显示。`AGENTFLOW_ASSISTANT_SPONSOR_LOGIN` 曾借用某个账号的密钥并限定为 `deepseek-v4.1-flash`，在未配置共享密钥时仍然生效。

请在平台 UI 中配置提供商凭据，而不是宿主全局的 CLI 配置。凭据由上游运行时绑定到负责的执行主体。首页输入框和 SOP 角色各自独立选择内置 CLI。在首页模型选择器旁选择执行器；它的 API Key（以及可选的 Codex/Claude Base URL）来自你的个人设置。切换执行器会保留草稿；已保存的会话会记录所使用的执行器。主 Bot 的写入仍会经过平台工具网关及其审批检查。定时助手任务可以选择任何受支持的 CLI，并作为普通对话轮次运行；Codex 和 Claude Code 仍需要该账号自己的凭据。实时模型测试为可选项；仅为需要加载本地凭据的验收套件设置 `AGENTFLOW_LIVE_TESTS=1`。

## 可观测性 schema

基础表创建完成后，为 Compose 数据库运行 `bash scripts/install-schema-extensions.sh`。它会从上游迁移 `001_run_obs_generated_columns.sql` 安装缺失的 `run_telemetry.adapter` 和 `run_logs.level` 列及其索引，然后按该顺序重放 `INSERT IGNORE` 起始目录以及 `031_generic_backend_openssh.sql`、`033_generic_sop_mr_node.sql` 和 `038_generic_sop_host_credentials.sql` 中的内置目录重写。这些重写可重复执行，且只作用于匹配的默认值。安装器会在打印地址之前运行它，`./start.sh pull` 会在重建容器之后再次运行它，因此把较旧的数据卷升级到较新的镜像时，新代码读取的列和新代码内置的 SOP 图都能补齐。应用启动不会运行数据迁移。

对于外部 MySQL 部署，请审阅该迁移，并在执行相应语句之前检查列是否存在。存储的生成列可能重建已有的大表；请据此安排操作时间。

### 升级需要哪些迁移

`migrations/` 中存放三类脚本。请查看发布说明了解某个版本新增了哪些文件，并在运行之前阅读脚本自身的头部说明。

| 类别 | 示例 | 运维人员需要做什么 |
| --- | --- | --- |
| 已由安装器和 `./start.sh pull` 应用 | `001_run_obs_generated_columns.sql`、`migrations/seed/*.sql`、`031_generic_backend_openssh.sql`、`033_generic_sop_mr_node.sql`、`038_generic_sop_host_credentials.sql` | 无需操作；上面的脚本已覆盖 |
| 必需的升级迁移 | `009_bot_interactions_utf8mb4.sql`、`036_git_http_credentials.sql` | 遵循各迁移的时机与前置条件。[迁移已有的 GitHub 凭据](#migrating-existing-github-credentials) |
| 可选的现有行回填与重写 | `016_public_optional_repositories.sql`、`021_assistant_session_sources.sql` | 若该安装需要转换这些行，则运行它们 |

## 更新已有的公共 SOP

所有内置通用 SOP 的仓库绑定都是可选的。要在已有安装上更新原始目录指令，请显式执行：

```bash
docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" mysql -u loopforge loopforge' < migrations/016_public_optional_repositories.sql
```

这段可重复执行的 SQL 只修改那些仍与原始公共种子匹配的指令字段。它会保留运维人员的修改和项目副本。新安装已经通过公共目录种子获得这些指令；应用启动不会迁移数据。

在 Compose 安装中，安装器和 `./start.sh pull` 会给内置的后端、前端和缺陷修复 SOP 添加「发起 MR」节点。它们先应用 `033_generic_sop_mr_node.sql` 添加该步骤，再应用 `038_generic_sop_host_credentials.sql` 更新原始 MR 角色指令。要手动应用这些 SQL，请按相同顺序运行两个迁移。

```bash
cat migrations/033_generic_sop_mr_node.sql migrations/038_generic_sop_host_credentials.sql \
  | docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" mysql -u loopforge loopforge'
```

迁移 033 会检查节点图和交付汇总的位置是否仍与随附模板一致。它只在交付汇总提示词未被编辑时才更新它。迁移 038 会把原有的 Git 认证指令替换为使用托管 CLI、已配置的 MCP 或 `git credential fill` 进行 API 访问的指引。两个迁移都会保留项目副本且可重复执行。运行负责用户的 Git 凭据需要具备推送权限，新步骤才能完成。

## GitHub 凭据

用户可以在个人凭据中或在添加仓库时选择**连接 GitHub**，复制显示的 PIN，并在 GitHub 上批准。平台会把授权保存到该已登录用户已有的 Git 凭据槽位。仓库选择器会列出该账号拥有的、与其共享的、或通过其组织可访问的仓库，支持分页和对已加载仓库的搜索，并把所选仓库的真实默认分支填入表单。仍然可以手动输入 PAT。

新安装使用内置的 LoopForge OAuth App。用户无需创建应用、配置 Client ID 或申请 PAT：选择**连接 GitHub**，复制 PIN 并在 GitHub 上批准 LoopForge 应用即可。GitHub 的确认无法跳过；设备授权不是零交互登录。

应用配置属于部署层面的事务，不会显示在个人凭据中。需要自有应用的运维人员可以注册 OAuth App，启用 **Device Flow**，在支持刷新之前保持令牌过期关闭，并在 `.env` 中用 `AGENTFLOW_GITHUB_CLIENT_ID` 设置其公开的 Client ID。不需要客户端密钥。部署环境会覆盖内置应用和任何数据库覆盖值。仅管理员可用的 `PUT /api/auth/github/config` 端点仍然保留，用于显式的数据库覆盖；空的 Client ID 会显式禁用设备授权，而不是静默恢复默认值。已有的显式配置会被保留。

该流程申请 OAuth 的 `repo` 作用域，以便私有克隆、推送、Issue 和 PR 操作正常工作。GitHub 的 OAuth 作用域覆盖可访问的仓库；在 LoopForge 中选择某个仓库只会把它绑定到项目，不会收窄 OAuth 授权。组织 SSO 或第三方应用限制可能需要组织批准。参见 [GitHub 的设备授权文档](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow)。待处理的授权会过期，且只保存在 API 内存中：在多个 API 实例后面运行时，请为 `/api/auth/github/*` 使用会话亲和性；API 重启后需要重新发起授权。令牌和 GitHub 设备码永远不会发送到浏览器。

GitHub Issue 导入和 PR 状态同步使用项目所有者的 Git 令牌。Git 操作和可选的 GitHub MCP 集成使用负责用户的令牌。公开读取可能无需令牌即可工作；私有仓库和写入需要仓库访问权限。GitHub MCP 必须显式安装，并且只把凭据发送到其官方 HTTPS 端点。

平台创建的 Git 提交使用负责的本地账号名和不可投递的 `@users.noreply.loopforge.invalid` 邮箱。本地账号并不等同于企业邮箱或经验证的 GitHub 身份。

### 迁移已有的 GitHub 凭据

从 v0.4.0 升级到按宿主读取 HTTPS 凭据的版本时，请停止旧的 API 和 worker，并保持 MySQL 运行。在启动升级后的平台或运行 `./start.sh pull` 之前，手动应用目标版本的 `036_git_http_credentials.sql`。使用 Compose 时执行：

```bash
docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" mysql -u loopforge loopforge' < migrations/036_git_http_credentials.sql
```

回填会把已有的 GitHub 个人访问令牌和设备流授权复制到 `github.com` 条目中。如果某次安装已经完成升级，但用户的令牌仍在旧列中，应用回填即可恢复缺失的条目。已有的 `github.com` 条目优先。

升级后请校验凭据。回滚窗口结束后，用同一命令并换成清理文件名手动运行 `037_retire_plain_git_token.sql`。只有当对应的 `github.com` 条目保存相同值时，它才会清除旧令牌。若值不同，两者都会保留。

## CodeBuddy 认证环境

在个人设置中选择属于该 API Key 的网络环境；「获取 API Key」链接会打开该版本的密钥页面。某个版本的密钥发送到另一个版本时会返回 HTTP 401，来自第三方模型提供商（DeepSeek、OpenAI 等）的密钥也不是 CodeBuddy 密钥。该设置按用户区分；公共安装不会假定企业环境。

| 版本 | 存储值 | 密钥页面 |
| --- | --- | --- |
| 国际版 | 空 | https://www.codebuddy.ai/profile/keys |
| 中国版 | `internal` | https://copilot.tencent.com/profile/ |
| iOA | `ioa` | https://tencent.sso.copilot.tencent.com/profile/keys |

`PUT /api/users/:id/credentials` 只接受这些值（`public` 和 `China` 也能识别），其他任何值都会以 HTTP 400 拒绝，因为对于无法识别的值，CLI 会静默回退到国际版。

其他专用/自托管配置见 [CodeBuddy 官方身份文档](https://www.codebuddy.ai/docs/cli/iam)。用 `AGENTFLOW_ASSISTANT_MODEL` 和 `AGENTFLOW_ASSISTANT_MODELS` 配置模型目录；普通角色可以独立选择模型。

## 隔离容器验收

运行 `python3 scripts/test-install.py -v` 执行安装器回归测试，覆盖 sudo 选择、直接 Docker 访问、无人值守权限失败、重复安装、镜像模式和访问 URL。若已有本地构建的平台镜像，`LOOPFORGE_INSTALL_TEST_IMAGE=your-image bash scripts/test-install-live.sh` 会在随机端口上的隔离 Compose 项目中验证真实的 root 与 UID 65534 安装、非回环 HTTP 访问、首账号创建和重复安装。该实机测试需要一个能降权的 Linux root 测试运行机；它会在结束后删除自己创建的容器和数据卷。若还要在一次性 UID 1000 安装器容器内测试真实 sudo，请用 `--build-arg LOOPFORGE_TEST_BASE=your-image` 构建 `scripts/testdata/install-sudo.Dockerfile`，再把该测试镜像作为 `LOOPFORGE_INSTALL_SUDO_TEST_IMAGE` 传入。生产安装器本身不需要 root。

在仓库根目录，安装 `e2e/` 依赖和 Chromium 后运行 `bash scripts/test-container.sh`。它会构建生产镜像，在 8093/33317 端口上创建隔离的 Compose 项目和数据库，预置测试账号，显式应用可观测性基线和公共目录，检查内置 CLI 以及跨平台的 runner/gate 二进制文件，并运行公共浏览器测试套件。它的容器和数据卷会在结束后删除。端口被占用时用 `LOOPFORGE_CONTAINER_TEST_PORT` 和 `LOOPFORGE_CONTAINER_MYSQL_PORT` 覆盖。这项干净安装检查会禁用模型调用。
