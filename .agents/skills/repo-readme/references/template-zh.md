# 中文 README 模板 (`README.zh.md`)

此模板为中文版 `README.zh.md` 的标准架构基准，与 `README.md` 保持严格的 1:1 结构对应与代码块一致性。

```markdown
<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

Z-Library 命令行客户端与终端交互界面。

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/release.yml)](https://github.com/heartleo/zlib/actions)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases)
[![Downloads](https://img.shields.io/github/downloads/heartleo/zlib/total)](https://github.com/heartleo/zlib/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[English](README.md) | 中文

[功能特性](#功能特性) · [安装](#安装) · [快速开始](#快速开始) · [常用命令](#常用命令) · [Agent 接入](#agent-接入) · [文档](#文档)

</div>

搜索和下载图书，浏览个人收藏与推荐，把文件发送到 Kindle。可以在终端界面中选书，也可以在脚本和 AI Agent 中通过 JSON 输出调用。

![交互式搜索演示](docs/demo-search.gif)

## 功能特性

- **终端界面**：基于 Bubble Tea 的终端 Shell，支持实时搜索、图书详情卡片（ISBN、出版社、出版年、格式）、自适应排版和斜杠命令（`/search`、`/favorites`、`/theme`、`/doctor`、`/config`）。
- **原生 EAPI**：直接调用 Z-Library 官方 JSON 接口，不解析 HTML 网页，不受前端验证码和 DiamWall 拦截影响。
- **Kindle 推送**：内置 SMTP 发送，支持把下载的电子书直接发送到 Kindle 设备。
- **配色主题**：内置 20 多款亮暗主题（Catppuccin、Dracula、Nord、Gruvbox、Tokyo Night、Solarized）。
- **Agent 技能**：内置 Claude Code 与 Codex 技能包，所有查询命令均支持 `--json` 输出。
- **网络诊断**：并发探测官方镜像节点的连通性与延迟，快速找出可用域名。

## 安装

### 包管理器

macOS / Linux（Homebrew）：

```bash
brew install heartleo/tap/zlib
```

Windows（winget）：

```powershell
winget install --id heartleo.zlib --exact
```

### 安装脚本

macOS / Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/heartleo/zlib/main/install.sh | sh
```

### 预编译二进制

从 [Releases](https://github.com/heartleo/zlib/releases/latest) 下载适用于 macOS、Linux 或 Windows（x86_64、ARM64）的压缩包，解压后将 `zlib`（Windows 上为 `zlib.exe`）移动到系统的 `PATH` 目录。

### Go 安装

需要 Go 1.26 或更高版本：

```bash
go install github.com/heartleo/zlib/cmd/zlib@latest
```

确保 `$GOPATH/bin`（Windows 上为 `%USERPROFILE%\go\bin`）已添加到系统 `PATH`。

### 源码编译

需要 Go 环境：

```bash
git clone https://github.com/heartleo/zlib.git
cd zlib
go build -o zlib ./cmd/zlib
```

Windows（PowerShell）：

```powershell
go build -o zlib.exe ./cmd/zlib
```

随时运行 `zlib update` 即可按当前安装渠道更新到最新版本。完整校验和与构建说明见[安装指南](docs/installation.md)。

## 快速开始

使用前需要一个 Z-Library 账号。

1. 查找当前网络可用的镜像域名：
   ```bash
   zlib doctor
   ```

2. 使用标为 `Available` 的域名登录（登录状态会保存在本地）：
   ```bash
   zlib login --domain https://z-lib.gd
   ```
   若遇到验证码拦截，可[导入浏览器 Cookie 登录](docs/troubleshooting.md#login-returns-authorization-failed)。

3. 打开终端交互界面：
   ```bash
   zlib tui
   ```
   - 输入书名并按 `Enter` 搜索。
   - 使用 `↑/↓` 选书，使用 `←/→` 翻页，按 `Enter` 下载。
   - 输入 `/` 打开命令菜单（如 `/theme`、`/favorites`、`/doctor`），输入 `/quit` 退出。

## 常用命令

| 操作目标 | 命令 | 输出与说明 |
| --- | --- | --- |
| 打开终端交互界面 | `zlib tui` | 全屏终端 Shell |
| 交互式搜索 | `zlib search` | 终端列表选择 |
| 搜索图书 | `zlib search "dune" --ext epub` | 纯文本表格输出 |
| 搜索并输出 JSON | `zlib search "dune" --json --count 10` | JSON 对象（含 `books`、`page`、`total_pages`） |
| 按 ID 下载 | `zlib download "id:hash"` | 下载图书到本地 |
| 浏览收藏图书 | `zlib favorites` | 交互式列表或 `--json` |
| 浏览推荐图书 | `zlib recommended` | 推荐书单 |
| 浏览热门图书 | `zlib most-popular` | 全站热门书单 |
| 浏览下载历史 | `zlib history` | 分页历史记录 |
| 查看账号与配额 | `zlib profile` | 每日配额与账号信息 |
| 发送图书到 Kindle | `zlib kindle send ./book.epub` | 通过配置的 SMTP 邮箱发送 |
| 切换配色主题 | `zlib theme nord` | 设置主题（运行 `zlib theme` 打开选择器） |
| 探测镜像连通性 | `zlib doctor` | 延迟与可用性检查 |
| 更新版本 | `zlib update` | 检测安装方式并自动更新 |

运行 `zlib <command> --help` 可查看所有可用参数。分页、下载目录及过滤选项见[使用指南](docs/usage.md)。

## Agent 接入

CLI 支持通过 `--json` 进行无交互自动化。仓库内置的通用技能位于 [`.agents/skills/`](.agents/skills/)。

### Claude Code

在 Claude Code 中执行：

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

使用 `/zlib "the pragmatic programmer"` 搜书，或用 `/zlib:kindle ./book.epub` 推送文件。

### Codex

在终端执行：

```bash
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

向 Codex 提问：“用 zlib 搜索《程序员修炼之道》的 EPUB 版本，列出结果后再下载。”

非交互式调用和 JSON 数据结构见 [Agent 与脚本](docs/agents.md)。

## 配置

| 环境变量 | 说明 | 默认值 |
| --- | --- | --- |
| `ZLIB_DOWNLOAD_DIR` | 默认图书下载目录 | 当前运行目录 |
| `ZLIB_THEME` | 强制指定当前进程的主题配色 | `catppuccin-mocha` |
| `ZLIB_DOMAIN` | 指定镜像站点域名 | 登录时保存的可用域名 |
| `HTTP_PROXY` / `HTTPS_PROXY` | HTTP 或 SOCKS5 代理地址 | 无 |

配置文件存储路径：
- macOS / Linux：`~/.config/zlib/`
- Windows：`%APPDATA%\zlib\`

详细代理配置和文件说明见[配置指南](docs/configuration.md)。

## 文档

| 指南 | 说明 |
| --- | --- |
| [安装指南](docs/installation.md) | 包管理器、Go 安装、源码编译、版本更新 |
| [使用指南](docs/usage.md) | 搜索过滤、文件下载、收藏、下载历史 |
| [终端界面](docs/tui.md) | 终端 Shell 命令、界面排版、快捷键 |
| [主题指南](docs/themes.md) | 配色列表与选择器操作 |
| [配置指南](docs/configuration.md) | 镜像域名、代理、下载目录、配置文件 |
| [Agent 与脚本](docs/agents.md) | 插件、技能、无交互命令、JSON 格式 |
| [Kindle](docs/kindle.md) | SMTP 配置与文件推送 |
| [排障指南](docs/troubleshooting.md) | 登录、可用域名、网络故障处理 |
| [EAPI 笔记](docs/eapi.md) | 接口端点与限制说明（中文） |

## 贡献

遇到问题请在 [Issues](https://github.com/heartleo/zlib/issues) 提交反馈，并附上 `zlib version`、操作系统、所用命令和报错信息。请勿提交 Cookie、密码或会话文件。
```
