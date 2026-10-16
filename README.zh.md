<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

在终端搜索和下载 Z-Library 图书。

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/ci.yml)](https://github.com/heartleo/zlib/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases/latest)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[English](README.md) | 简体中文

[安装](#安装) · [快速开始](#快速开始) · [常用命令](#常用命令) · [Agent 接入](#agent-接入) · [文档](#文档)

</div>

![终端搜书演示](docs/demo-search.gif)

- 浏览搜索结果、收藏、推荐、热门图书和下载历史。
- 查看书籍详情、选择下载目录，或[将文件发送到 Kindle](docs/kindle.md)。
- 在全屏终端界面中操作，或通过 JSON 命令接入脚本和 Agent。

## 安装

macOS / Linux，使用 Homebrew：

```bash
brew install heartleo/tap/zlib
```

Windows，使用 winget：

```powershell
winget install --id heartleo.zlib --exact
```

也可以从 [Releases](https://github.com/heartleo/zlib/releases/latest) 下载程序，
放入 `PATH` 目录。安装脚本、Go 安装、源码编译和更新方式见[安装指南](docs/installation.md)。

如果安装版本缺少本文中的命令，可按[源码编译说明](docs/installation.md#build-from-source)
构建当前分支。

## 快速开始

使用前需要一个 Z-Library 账号。先检测当前网络可用的域名：

```sh
zlib doctor
```

选择结果中的域名，将下方示例 URL 替换为检测结果中的地址。
运行后在表单中输入邮箱和密码：

```sh
zlib login --domain https://z-lib.gd
zlib tui
```

进入终端界面后：

- 输入书名、作者或 ISBN，按 `Enter` 搜索。
- 用 `↑/↓` 选书，`←/→` 翻页，`Enter` 打开下载确认框。
- 输入 `/` 打开命令菜单。`/config` 配置网络、下载和 Kindle，
  `/theme` 选择主题，`/quit` 退出。

默认下载到当前目录，也可以在下载前选择其他目录。
登录状态保存在本地，后续直接运行 `zlib tui` 即可。

没有可用域名或登录失败时，参阅[排障指南](docs/troubleshooting.md)。
已有浏览器会话也可以通过[导入 Cookie](docs/troubleshooting.md#login-returns-authorization-failed)登录。

## 常用命令

带查询词时输出表格，加上 `--json` 则输出供脚本使用的 JSON：

```sh
zlib search "dune" --ext epub
zlib search "dune" --ext epub --json --count 10
```

| 操作 | 命令 |
| --- | --- |
| 浏览收藏 | `zlib favorites` |
| 浏览推荐 | `zlib recommended` |
| 浏览热门图书 | `zlib most-popular` |
| 浏览下载历史 | `zlib history` |
| 查看账号与配额 | `zlib profile` |
| 完成 [Kindle 配置](docs/kindle.md)后发送文件 | `zlib kindle send ./book.epub` |

在脚本中下载时，将 `id:hash` 替换为 JSON 结果中书籍的完整 `id`：

```sh
zlib download "id:hash"
```

运行 `zlib <command> --help` 查看参数。
过滤条件、分页和下载选项见[使用指南](docs/usage.md)。

## Agent 接入

先在 Agent 使用的环境中安装 zlib，并完成上述登录步骤。
插件提供技能，运行时需要能从 `PATH` 找到 CLI，并读取本地登录状态。

### Claude Code

在 Claude Code 中执行：

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

### Codex

在终端执行：

```sh
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

然后告诉 Agent：

> 用 zlib 搜索 The Pragmatic Programmer 的 EPUB 版本，先列出结果，再决定下载。

技能源文件位于 [`.agents/skills/`](.agents/skills/)。
JSON 格式、本地技能发现和脚本示例见 [Agent 与脚本](docs/agents.md)。

## 文档

| 指南 | 内容 |
| --- | --- |
| [安装指南](docs/installation.md) | 安装方式、源码编译、`zlib update` |
| [使用指南](docs/usage.md) | 搜索、下载、收藏、历史 |
| [终端界面](docs/tui.md) | 命令和快捷键 |
| [配置指南](docs/configuration.md) | 域名、代理、目录、配置路径 |
| [主题指南](docs/themes.md) | 配色列表和预览操作 |
| [Kindle](docs/kindle.md) | SMTP 配置与发送 |
| [Agent 与脚本](docs/agents.md) | 插件、技能、JSON 输出 |
| [排障指南](docs/troubleshooting.md) | 登录和网络问题 |
| [EAPI 笔记](docs/eapi.md) | 接口行为和已知限制（中文） |

遇到问题请在 [Issues](https://github.com/heartleo/zlib/issues) 附上
`zlib version`、操作系统、所用命令和报错。请勿附带密码、Cookie 或会话文件。
