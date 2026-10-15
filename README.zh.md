<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

Z-Library 命令行客户端。

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/release.yml)](https://github.com/heartleo/zlib/actions)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases)
[![Downloads](https://img.shields.io/github/downloads/heartleo/zlib/total)](https://github.com/heartleo/zlib/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[English](README.md) | 中文

[安装](#安装) · [快速开始](#快速开始) · [Agent 接入](#agent-接入) · [文档](#文档)

</div>

搜索和下载图书，浏览收藏与推荐，把文件发送到 Kindle。
可以在终端里选择图书，也可以让 Agent 通过 JSON 输出调用 CLI。

![交互式搜索演示](docs/demo-search.gif)

## 安装

macOS / Linux（Homebrew）：

```bash
brew install heartleo/tap/zlib
```

Windows（PowerShell）：

```powershell
winget install --id heartleo.zlib --exact
```

macOS / Linux（安装脚本）：

```bash
curl -fsSL https://raw.githubusercontent.com/heartleo/zlib/main/install.sh | sh
```

也可以[下载可执行文件](https://github.com/heartleo/zlib/releases/latest)，或
[通过 Go 安装 / 从源码构建](docs/installation.md)。

## 快速开始

你需要一个 Z-Library 账号。先查找当前网络可用的域名：

```sh
zlib doctor
```

选择标为 `Available` 的域名，替换下面的示例地址，然后登录并打开搜索界面。
这些命令在 Bash 和 PowerShell 中都可使用：

```sh
zlib login --domain https://z-lib.gd
zlib search
```

输入关键词，用 `↑/↓` 选择图书、`←/→` 翻页，按 `Enter` 下载。
文件默认保存到当前目录。

登录后会保存会话和域名，后续无需重复设置。如果登录返回 `Authorization failed`，
可[导入浏览器 Cookie](docs/troubleshooting.md#login-returns-authorization-failed)。
如果 `doctor` 没有找到可用域名，请看[连接排障](docs/troubleshooting.md#no-available-domains)。

## Agent 接入

先安装 CLI，并在自己的终端完成上面的登录。插件提供调用本机 `zlib` 的技能，
会复用已保存的会话。

### Claude Code

在 Claude Code 中执行：

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

用 `/zlib "the pragmatic programmer"` 搜书，或用 `/zlib:kindle` 发送本地文件。

### Codex

在终端执行：

```sh
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

然后说：“用 zlib 搜索《程序员修炼之道》的 EPUB 版本，先列出结果让我选。”

[Agent 使用流程与 JSON 示例](docs/agents.md) · [Kindle 配置](docs/kindle.md)

通用技能位于 [`.agents/skills/`](.agents/skills)，可在仓库内直接使用。

## 常用命令

| 用途 | 命令 |
| --- | --- |
| 交互式搜索 | `zlib search` |
| 以表格显示搜索结果 | `zlib search "dune" --ext epub` |
| 输出 JSON | `zlib search "dune" --json --count 10` |
| 浏览已收藏图书 | `zlib favorites` |
| 浏览推荐 / 热门图书 | `zlib recommended` / `zlib most-popular` |
| 浏览下载历史 | `zlib history` |
| 查看账号和每日配额 | `zlib profile` |
| 发送本地文件到 Kindle | `zlib kindle send ./book.epub` |
| 切换主题 | `zlib theme nord` |

下载指定图书时，复制结果中的完整 `id:hash` 引用，执行 `zlib download "id:hash"`。
分页、下载目录和键盘操作见[使用指南](docs/usage.md)。

完整参数请运行 `zlib --help` 或 `zlib <command> --help`。

## 文档

以下使用指南以英文维护；EAPI 笔记为中文。

| 指南 | 内容 |
| --- | --- |
| [安装](docs/installation.md) | 可执行文件、Go 安装、源码构建和升级 |
| [使用](docs/usage.md) | 搜索、下载、收藏、历史和主题 |
| [配置](docs/configuration.md) | 域名、代理、下载目录和配置文件 |
| [Agent](docs/agents.md) | 插件接入、非交互调用和 JSON 输出 |
| [Kindle](docs/kindle.md) | SMTP 配置和文件投递 |
| [排障](docs/troubleshooting.md) | 登录、连接和下载失败 |
| [EAPI 笔记](docs/eapi.md) | 已接入的端点和接口限制 |

遇到问题请提交 [Issue](https://github.com/heartleo/zlib/issues)，附上 `zlib version`、
操作系统、命令和错误信息。请勿附上密码、Cookie 或会话文件。
