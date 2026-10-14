# Z-Library Android 1.11.4 API 静态调查

调查日期：2026-10-12。用户提供的 [APK 下载地址](https://s3proxy-alp.cdn-zlib.sk/swfs_second_public_files/soft/android/zlibrary-app-latest.apk) 下载到 33,076,250 字节；SHA-256 为 `159a142595ab2f6be62a4b231d5d6193739607bfe1d816271bf52f217de32120`。另一 CDN 路径下载所得文件的大小和 SHA-256 均相同。本文只针对这一份文件；`latest.apk` 是可变地址，包内版本不能证明调查日没有更新版本。未验证 APK 签名，也未运行应用或发送账号请求。

## 工具与证据边界

APK 包名为 `com.positron_it.zlib`，`versionName=1.11.4`，`versionCode=21`。接口定义位于 `com.positron_it.zlib.data.ZLibApi`。使用 [JADX 1.5.6](https://github.com/skylot/jadx/releases/tag/v1.5.6) 对同一 SHA-256 的 APK 导出 `ZLibApi`、`DataModule` 和 `HeaderInterceptor`，读取 HTTP 注解和客户端配置。复核命令如下，输出文件位于系统临时目录，未加入仓库：

```powershell
& 'D:\TEMP\jadx-1.5.6\bin\jadx.bat' --single-class 'com.positron_it.zlib.data.ZLibApi' --single-class-output 'D:\TEMP\ZLibApi.java' 'D:\TEMP\zlibrary-official-20261012.apk'
```

本次静态分析没有执行 APK、验证签名或捕获运行时 HTTP。反编译结果不能证明某个请求已经发出或服务器仍接受它。

## 接口定义

以下来自 APK `classes.dex` 中 `ZLibApi` 的 Retrofit 注解。需要登录态的方法声明使用 `remix-userid` 和 `remix-userkey` 请求头。`HeaderInterceptor` 还构造同名凭据的 Cookie，加入 Android 版本、语言等头，以及一个固定的授权头常量；这里不记录授权常量的值。下表列出对本仓库最有用的接口，不代表运行时逐项调用过。

| 功能 | 方法和路径 | 明确声明的参数 |
| --- | --- | --- |
| 登录 | `POST /eapi/user/login` | 表单 `email`, `password`, `auth_token` |
| 搜索 | `POST eapi/book/search` | 表单 `message`, `yearFrom`, `yearTo`, `languages[]`, `extensions[]`, `e`, `page`, `limit`, `order` |
| 书籍详情 | `GET /eapi/book/{id}/{hash}` | 路径 `id`, `hash` |
| 下载链接 | `GET /eapi/book/{id}/{hash}/file` | 路径 `id`, `hash`；实际文件另由动态 URL 流式 GET |
| 可选格式 | `GET /eapi/book/{id}/{hash}/formats` | 路径 `id`, `hash` |
| 相似书籍 | `GET /eapi/book/{id}/{hash}/similar` | 路径 `id`, `hash` |
| 已下载、已收藏 | `GET /eapi/user/book/downloaded`, `GET /eapi/user/book/saved` | 查询 `order`, `page`, `limit` |
| 热门、最近、推荐 | `GET /eapi/book/most-popular`, `GET /eapi/book/recently`, `GET /eapi/user/book/recommended` | 登录态请求头 |
| 资料 | `GET /eapi/user/profile` | 登录态请求头 |
| 域名、服务信息 | `GET /eapi/info/domains`, `GET /eapi/info/service`, `GET /eapi/info` | 部分接口带登录态请求头 |
| 语言、扩展名、计划 | `GET /eapi/info/languages`, `GET /eapi/info/extensions`, `GET /eapi/info/plans` | 无显式参数 |
| 收藏状态变更 | `GET /eapi/user/book/{bookId}/save`, `/unsave`, `GET /eapi/user/book/{id}/delete` | 书籍 ID；虽用 GET，语义为状态变更 |
| 邮件发送 | `GET /eapi/book/{id}/{hash}/send-to-{type}` | 路径 `id`, `hash`, `type` |

接口还声明注册、确认邮件、密码找回、`POST /eapi/user/token-sign-in`、`POST /eapi/user/update` 和动态 URL 的个人登录。这里只能确认声明存在；不能推断这些流程当前可用。`DataModule` 的 Retrofit 初始 base URL 是 `https://zlibrary.se/`，另有 onion 入口；这只是打包默认值，不是当前可用域名保证。

## 与本仓库的关系

本仓库 [EAPI 文档](eapi.md)中的登录、搜索、域名检查、资料、下载历史、书籍详情和下载链接七个接口，都能在这个 APK 的 `ZLibApi` 定义里找到对应路径。`eapi.go` 已按 `remix-userid` / `remix-userkey` 设置请求头。建立本分支时，CLI 对登录拒绝后的 `/rpc.php` 回退是本仓库的实测处理；本分支已移除该回退，改为保留 Cookie 文件导入。不能从这个 APK 的 `ZLibApi` 注解推出相同回退流程。APK 还包含上述尚未在 CLI 全面实现的格式、收藏、推荐等方法，但新增功能前应核对响应模型和真实服务器行为。

最重要的差异是证据层级：本次确认的是 **1.11.4 APK 内的接口声明**；`docs/eapi.md` 记录的是本仓库实现及较早的实测。没有运行时流量和当下服务器响应，就不能把“声明存在”写成“最新版服务端可用”。

## 当前 zlib 可参考的部分

以下对照的是本分支从 `main` 建立时的 Go 代码，并非其他开发分支。优先级依据现有 CLI 可用性、实现范围和验证成本；不是 APK 的功能排序。

| 优先级 | APK 证据 | 当前 zlib | 可采纳的方向 |
| --- | --- | --- | --- |
| 先做 | 搜索表单有 `order` 字段 | `SearchOptions` 和 `searchEAPI` 尚无排序，CLI 也没有 `--order` | 先确认服务端接受的排序值，再把可选排序贯通模型、EAPI 请求与 CLI；默认请求保持不变。 |
| 先做 | `GET /eapi/book/{id}/{hash}/formats`；APK 模型将响应 `books` 映射为格式列表，元素含 `id`、`hash`、`extension`、`filesize`、`filesizeString` | 当前详情和下载只以一个 `id:hash` 表示书籍版本 | 可增加“查看可下载格式”，让用户明确选择目标格式；不同格式可能有不同的 ID/hash，不能仅替换文件后缀。 |
| 后做 | `GET /eapi/user/book/saved` 与 `/save`、`/unsave` | 只有下载历史，没有收藏列表或修改命令 | 先做只读收藏列表，再评估修改操作；APK 用 GET 表示状态变更，客户端不应把它当成可缓存的只读请求。 |
| 后做 | 热门、最近、推荐和相似书籍接口 | `main` 基线没有对应命令 | 可复用 `Book` 展示流程，但 APK 的热门/推荐响应为简版 `books` 列表；需核对简版字段，不能直接假设与搜索结果相同。 |
| 有条件 | `GET /eapi/info/languages`、`/extensions`、`/plans` | 搜索筛选值来自本地枚举，配额读 `/eapi/user/profile` | 可用于展示服务端可选筛选值与计划信息；先核对响应结构、缓存时效和匿名访问行为。 |

现有核心链路不需要按 APK 重写：`eapi.go` 的搜索字段、表单编码和凭据请求头与接口注解对应；`eapi_book.go` 的详情和两步下载与 APK 的链接接口对应；`eapi_profile.go` 的资料与下载历史路径也对应。APK 的 `HeaderInterceptor` 额外发送固定授权常量和设备信息，但本仓库已有独立的登录及服务器响应证据；不能仅凭静态代码断定必须复制这些头，更不应把 APK 内的常量写入源码。`DataModule` 的打包默认域名也不应覆盖本仓库的域名验证和凭据重定向保护。

实施新接口前，最小验证是用隔离的测试响应确认解码和错误路径，再在明确授权的账户与域名上确认服务器实际响应。收藏修改和邮件发送属于外部状态变更，应分别征得授权。
