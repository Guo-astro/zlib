# Crush TUI 对 zlib 的改进建议

分析日期：2026-10-13。Crush 源码固定在 `b7b5522585e7302d2ee19b1a74a7003ac29ea562`；zlib 对照当前工作树。本文是源码分析与设计建议，未实现或运行新的界面。

## 先做什么

优先修复列表高度和窄屏布局，再实现详情预览、错误重试与下载后返回列表。zlib 已有 spinner、textinput、progress、huh 表单及多主题，主要缺口是这些组件之间的交互连续性。

| 优先级 | Crush 的实现 | zlib 当前问题 | 建议与成本 |
| --- | --- | --- | --- |
| P0 | [List](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/list/list.go) 管理宽高、可见偏移、选择位置与渲染缓存 | 搜索模型只记录宽度，逐行渲染全部结果；推荐列表可达 100 项 | 记录终端高度，固定标题和帮助区，按可用行数裁剪列表并保持选中项可见。中等成本 |
| P0 | [主界面](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/model/ui.go) 按宽度和高度切换 compact 布局 | 固定列宽在窄窗口中换行；ID 列只有 10 字符，截断完整引用 | 宽屏显示更多列，窄屏保留标题、作者、格式；完整引用放在详情中。根据 zlib 内容决定断点，不照搬 Crush 的 120×30。中等成本 |
| P1 | [Dialog/Overlay](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/dialog/dialog.go) 隔离弹层和主界面输入 | Enter 直接下载，没有详情预览 | 增加详情动作：宽屏右栏、窄屏独立详情页，Esc 返回并保留光标。可继续保留 Enter 下载，避免无提示改变现有键位。中等成本 |
| P1 | [Status](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/model/status.go) 组合 help 与有时限的消息 | 获取失败直接退出；选择图书后列表退出，再启动下载程序 | 用父模型保留查询、页码和光标；下载结束回到列表并提示保存路径。错误留在界面，提供重试和返回。中到高成本 |
| P1 | Status 使用 `help.Model`、`help.KeyMap` | 帮助文字手工拼接；空结果提前返回，隐藏了仍可用的搜索入口 | 用 `key.Binding` 驱动实际按键和提示，短帮助常驻，`?` 展开。空结果保留搜索、重试、退出操作。低成本 |
| P2 | [FilterableList](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/list/filterable.go) 对已加载项目模糊筛选 | 推荐、收藏列表没有筛选入口；搜索中的 `/` 只能修改关键词 | 对推荐列表提供本地筛选；收藏只筛当前已加载页时必须标明范围。远程搜索仍由确认操作发请求，避免每次按键请求 EAPI。中等成本 |
| P2 | [动作对话框](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/dialog/actions.go) 与 [命令对话框](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/internal/ui/dialog/commands.go) | 搜索、收藏、历史是独立入口 | 等父模型稳定后，提供切换列表和图书操作菜单。不要为了少数动作先引入整套命令系统。中等成本 |

## 本地落点

- [search_interactive.go](../internal/cli/search_interactive.go)：`searchSelectModel` 共用于搜索、推荐、热门和收藏。`WindowSizeMsg` 只保存 Width；`View` 遍历所有图书。优先在这里统一高度、滚动和响应式行布局。
- [history_interactive.go](../internal/cli/history_interactive.go)：历史有独立模型，同样逐行渲染且错误直接退出。复用列表呈现和键位逻辑，保留历史自己的数据加载方式。
- [download.go](../internal/cli/download.go)：已有进度条、取消和非终端输出。交互改进应放在父模型的状态转换，保留脚本/Agent 的直接下载路径。
- [eapi_book.go](../eapi_book.go)：`FetchBook` 的详情请求读取 `/eapi/book/{id}/{hash}`。映射只构造下载路径，不访问 `/file`；可用于详情预览。
- [theme.go](../internal/cli/theme.go)：已有语义色和多主题。继续统一使用这些颜色，无需先复制 Crush 的调色板。

详情按选中项加载并缓存。为异步响应携带图书引用或请求序号，忽略已过期的返回，防止快速移动光标时旧详情覆盖新选择。避免一次请求推荐列表中全部 100 本书的详情；预览也不应请求下载链接。

可视区域滚动与 EAPI 分页是两个层次：上下键移动当前页中的项目，左右键请求服务端上一页/下一页。推荐和热门没有服务端分页。本地模糊过滤不应显示为全库搜索。

## 依赖与复用边界

Crush 当前 [go.mod](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/go.mod) 使用 Go 1.27、Bubble Tea v2、Bubbles v2、Lip Gloss v2。zlib 的 [go.mod](../go.mod) 使用 Go 1.25、Bubble Tea v1.3.10、Bubbles v1.0.0、Lip Gloss v1.1.0 和 Huh v1.0.0。

Crush 的 `internal/ui` 受 Go internal 导入规则限制，不能直接从 zlib 导入。其当前 [许可证](https://github.com/charmbracelet/crush/blob/b7b5522585e7302d2ee19b1a74a7003ac29ea562/LICENSE.md) 标为 FSL-1.1-MIT，不应当作普通 MIT 代码直接复制。

第一轮用已有 Bubbles v1 的 `list`、`viewport`、`help`、`key` 即可。已核对本地模块缓存中的 `list.SetSize`、`SetFilteringEnabled`、`help.ShortHelpView` 和 `FullHelpView`。视口裁剪不等同于 Crush 的带缓存惰性列表；zlib 的规模目前不需要复制后者的全部复杂度。

## 实现顺序和验收

1. 列表高度、窄屏和帮助：在 80×24、120×30、较窄窗口下，标题、当前项和帮助始终可见；100 项列表能滚动到底；中文与长标题不破坏布局。
2. 详情和错误恢复：Esc 返回原页原位置；快速切换不串详情；超时后能重试；空结果能修改查询。
3. 下载后继续浏览：成功和取消均能回原列表；Ctrl+C 仍能可靠取消；连续下载不会重复启动同一请求。
4. 最后加入本地筛选和动作菜单，按实际使用决定是否需要框架 v2 迁移。

验证应覆盖窗口变化、页边界、空结果、网络错误与取消；同时保留 `--json`、重定向输出、Agent 非交互路径的原有行为。当前仅完成源码分析，没有执行这些界面验收。
