# Z-Library Android EAPI 深入调查

调查日期：2026-10-12。目标是用户提供的 [官方 APK 下载地址](https://s3proxy-alp.cdn-zlib.sk/swfs_second_public_files/soft/android/zlibrary-app-latest.apk) 当日返回的文件，而不是对今后所有 `latest.apk` 的承诺。两次独立下载得到相同的 33,076,250 字节与 SHA-256：`159a142595ab2f6be62a4b231d5d6193739607bfe1d816271bf52f217de32120`。APK 包名为 `com.positron_it.zlib`、`versionName=1.11.4`、`versionCode=21`、7,163 个类、1,179 个资源。没有验证 APK 签名，也没有运行 Android 应用，因此“最新”仅指此时该下载地址提供的文件。

## 证据来源和边界

| 标记 | 来源 | 能证明什么 |
| --- | --- | --- |
| A | 上述 APK 的 `classes.dex`；使用 [JADX 1.5.6](https://github.com/skylot/jadx/releases/tag/v1.5.6) 导出同一 SHA-256 文件中的 Retrofit 注解及模型 | 该 APK **声明**的路径、HTTP 方法、字段、部分调用链。静态代码不证明服务端今天可用，也不证明运行时一定调用。 |
| L | 2026-10-12 在 `https://z-lib.gd` 以用户明确授权的本地已保存会话发起的只读 EAPI 请求 | 该账户、域名和时间点的响应。未记录或展示凭证、账号内容、书名、下载链接和原始响应；只保留成功码、键名、计数和排序指纹比较。不是 Android 抓包。 |
| R | 本分支的 [客户端实现](../eapi.go)、[CLI 命令注册](../internal/cli/root.go) 与 [EAPI 文档](eapi.md) | 当前仓库的能力；工作树未提交，后续变化须重新核对。 |

`ZLibApi` 包含 37 个方法，路径和字段以同一 APK 的 JADX 导出为准。复核命令如下，临时导出文件不属于仓库：

```powershell
& 'D:\TEMP\jadx-1.5.6\bin\jadx.bat' --single-class 'com.positron_it.zlib.data.ZLibApi' --single-class-output 'D:\TEMP\ZLibApi.java' 'D:\TEMP\zlibrary-official-20261012-recheck.apk'
```

请先自行下载并校验 SHA-256 再复核；上述本机临时路径只是本次调查记录。下表的 A 证据可由 APK 中 `com.positron_it.zlib.data.ZLibApi` 及同包 `data.models` 类逐项复查。为了避免扩散 APK 内的固定授权值，文档不记录常量、原始请求头或可复用凭证。

## Retrofit 接口矩阵

表中“实测”只代表 L。`—` 表示未测，不能解释为不可用。`success=1` 也不保证期望的数据字段存在或排序确实生效。`{id}:{hash}` 是本仓库使用的书籍引用写法，不是 API 路径中的字面冒号。

| APK 方法、HTTP 路径 | 参数及 APK 响应模型 | 2026-10-12 只读实测 | 仓库现状或建议 |
| --- | --- | --- | --- |
| `login` · `POST /eapi/user/login` | 表单 `email,password,auth_token`；`ZLibUserListing` | —；既有 [EAPI 文档](eapi.md)记录部分账号被拒 | `login` 已接入；保留 Cookie 导入。拒绝时报告服务端错误。 |
| `searchBooks` · `POST eapi/book/search` | 表单 `message,yearFrom,yearTo,languages[],extensions[],e,page,limit,order`；`ZLibBooksListing` | 八个 APK 排序值返回 `success=1`，且同一查询的前 10 个 ID 顺序各异；见下节 | `search` 已接入；排序值需要纠正并开放 `--order`。 |
| `singleBook` · `GET /eapi/book/{id}/{hash}` | 路径 ID/hash；`ZLibBookSingle` | 一个已收藏书籍的详情 `success=1`，有 `book` 对象 | `FetchBook` 已接入。 |
| `fileFormats` · `GET /eapi/book/{id}/{hash}/formats` | 路径 ID/hash；`ZLibBookFormats` | 6 本不同的已收藏书均为 `success=1, books=[]` | 可增加 `formats` 查询，但空列表必须如实显示；不要承诺总有替代格式。 |
| `singleBookRecommendations` · `GET /eapi/book/{id}/{hash}/similar` | 路径 ID/hash；`ZLibMinBookListing` | 一次返回 `success=1` 和 24 本简版书籍 | 值得增加 `similar <id:hash>`。 |
| `getDownloadLink` · `GET /eapi/book/{id}/{hash}/file` | 路径 ID/hash；`ZLibFileListing` | —；会生成下载链接，本轮未触发 | `download` 已接入，仍需校验跳转目标。 |
| `downloadFile` · 动态 `GET @Url` | `@Streaming`，响应二进制文件 | — | 是文件流，不是另一个固定 EAPI 路径。 |
| `booksMostPopular` · `GET /eapi/book/most-popular` | 凭据头；`ZLibMinBookListing` | `success=1`，100 本 | `most-popular` 已接入。 |
| `booksRecentlyAdded` · `GET /eapi/book/recently` | 凭据头；`ZLibMinBookListing` | **只返回 `success=1`，没有 `books`** | 暂不应把 `recently` 当成可用书单；先确认响应契约。 |
| `booksRecommendations` · `GET /eapi/user/book/recommended` | 凭据头；`ZLibMinBookListing` | `success=1`，100 本 | `recommended` 已接入。 |
| `getDownloadedBooks` · `GET /eapi/user/book/downloaded` | `order,page,limit`；`ZLibTypedBooksListing` | `page=1,limit=2` 返回 2 本及分页；另取前 10 本时七个 APK 排序值各有不同 ID 顺序 | `history` 已接入；可增加排序。 |
| `getSavedBooks` · `GET /eapi/user/book/saved` | `order,page,limit`；`ZLibTypedBooksListing` | `page=1,limit=2` 返回 2 本及分页；现有 8 本在七个 APK 排序值下各有不同 ID 顺序 | `saved` 与 `favorites` 已接入；可增加排序。 |
| `userProfile` · `GET /eapi/user/profile` | 凭据头；`ZLibUserListing` | `success=1`，含账号和配额字段；响应含敏感字段，未记录其值 | `profile` 已接入；JSON 应继续省略凭证。 |
| `getHosts` · `GET /eapi/info/domains` | 凭据头；`ZLibHostsListing` | `success=1`，`domains` 有 5 项 | `doctor` 已使用；列表不等于全球全部可用域名，需逐个探测。 |
| `getServiceInfo` · `GET /eapi/info/service` | `ZLibServiceInfo` | `success=1`，`service` 含 `desktopApps,supportEmail` | 可作为只读 `info service`，不应与域名健康混淆。 |
| `getSiteInfo` · `GET /eapi/info` | 凭据头；`ZLibInfo` | `success=1`；顶层含 `articles,banner,books,extensions,languages,plans,service` | 可提供汇总 `info`，注意实际字段比 APK 模型显式字段多。 |
| `getLanguages` · `GET /eapi/info/languages` | `ZLibBooksLanguages` | `success=1`，`languages` 是名称映射 | 可给搜索筛选提供服务端值。 |
| `getExtensions` · `GET /eapi/info/extensions` | `ZLibBooksExtensions` | `success=1`，`extensions` 有 12 项 | 可给搜索筛选提供服务端值。 |
| `getPlanLimitations` · `GET /eapi/info/plans` | `ZLibPlanLimitListing` | `success=1`，`plans` 含 `basic,limits,premium` | 可增加 `info plans`，勿混同个人剩余额度。 |
| `getDonations` · `GET /eapi/user/donations` | 凭据头；`ZLibDonationsListing` | `success=1`，顶层为 `success,donations`；未读取记录值 | 含个人捐赠记录，单独评估 CLI 需求与隐私。 |
| `getHostsByUrl` · 动态 `GET @Url` | 凭据头；`List<ZLibHost>` | — | APK 用于额外域名来源；用户输入 URL 需要严格校验。 |
| `testConnection` / `testConnectionPersonal` · 动态 `GET @Url` | 测试目标 URL | — | 是 APK 内部网络探测，不应直接开放任意 URL。 |
| `loginPersonal` · 动态 `POST @Url` | 表单 `action,password,auth_token`；`ZLibUserListing` | — | 个人入口流程，先研究 URL 获取与认证关系。 |
| `tokenSignIn` · `POST /eapi/user/token-sign-in` | 表单 `id_token,name`；`ZLibUserListing` | — | 第三方 token 登录，不能从声明推断可用。 |
| `registration` · `POST /eapi/user/registration` | 表单 `email,password,name`；`ZLibUserListing` | — | 注册会创建外部状态；本轮不测。 |
| `registrationConfirm` · `POST /eapi/user/email/confirmation` | 表单 `email,password,name,code`；`ZLibUserListing` | — | 邮箱确认流程；本轮不测。 |
| `registrationResend` · `POST /eapi/user/email/confirmation/resend` | 表单 `email,password,name`；`ZLibUserListing` | — | 与下一个同路径但请求编码不同。 |
| `resendConfMail` · `POST /eapi/user/email/confirmation/resend` | 凭据头；`ZLibStatus` | — | 两个 Retrofit 方法不可仅按路径合并。 |
| `sendConfirmationCode` · `POST /eapi/user/send-code` | 表单 `verifyAction,email,name,verifyCode`；`ZLibUserListing` | — | 邮件发送可能发生，未测。 |
| `recoverPassword` · `POST /eapi/user/password-recovery` | 表单 `email,verifyCode`；`ZLibUserListing` | — | 密码找回流程，未测。 |
| `updateProfile` · `POST /eapi/user/update` | 凭据头；表单 `email,password,name,kindle_email,verifyCode` | — | 修改账户信息，未测。 |
| `saveBook` · `GET /eapi/user/book/{bookId}/save` | 数字书籍 ID；`ZLibStatus` | — | 虽用 GET，语义是收藏变更，不能按只读请求处理。 |
| `deleteSavedBook` · `GET /eapi/user/book/{bookId}/unsave` | 数字书籍 ID；`ZLibStatus` | — | 取消收藏，未测。 |
| `deleteDownloadedBook` · `GET /eapi/user/book/{id}/delete` | 数字书籍 ID；`ZLibStatus` | — | 删除历史记录，未测。 |
| `sendToMail` · `GET /eapi/book/{id}/{hash}/send-to-{type}` | 路径 ID/hash/type；`ZLibStatus` | — | 服务器发信，不等同当前 CLI 的本地 SMTP Kindle 投递。 |
| `onBannerHide` · `GET /eapi/user/hide-banner` | 凭据头；`ZLibStatus` | — | 修改远端显示状态，不适合作为诊断请求。 |

上述固定路径、字段和模型均来自 A；实测摘要来自 L。多个路径即使用 GET 也会改变账户或向外发送邮件，因而没有把它们纳入只读探针。APK 中 `DataModule` 还含打包时的 clearnet 与 onion 初始入口；打包值不能替代当前 `doctor` 的探测结果。

## 数据模型与调用行为

| 模型 | APK JSON 字段或行为 | CLI 影响 |
| --- | --- | --- |
| `ZLibBooksListing` | `success,books,exactBooksCount,pagination`；搜索每项是 `ZLibBook` | `exactBooksCount` 与分页总数不同，展示前应明确含义。 |
| `ZLibBook` | 含 `id,hash,title,author,year,extension,filesize,description`，另有 `kindleAvailable,_isUserSavedBook,sendToEmailAvailable,readOnlineAvailable,readOnlineUrl` 等 | 书籍详情可展示服务端能力标志；字段可能缺失。 |
| `ZLibMinBookListing` / `ZLibMinBook` | 顶层 `success,books`；每项在 APK 模型中只有 `id,title,author,cover,hash` | 热门、推荐、相似结果不可假定有格式、大小、年份；需再查详情。实测相似书籍还有模型未显式声明的字段。 |
| `ZLibTypedBooksListing` / `ZLibListBook` | 已收藏和已下载共用 `success,books,pagination`；书籍项字段较简版丰富 | 可复用分页和展示，但要保留各自的排序语义。 |
| `ZLibPagination` | `before,next,limit,current,total_items,total_pages` | 页码以服务端 `current/total_pages` 为准。 |
| `ZLibBookFormats` / `ZLibFormat` | 顶层 `success,books`；格式项有 `id,hash,extension,filesize,filesizeString` | 不同格式可以有不同 ID/hash，不能仅改变下载文件扩展名。空 `books` 是合法实测结果。 |
| `ZLibFileListing` / `ZLibFile` | `file.downloadLink,description,author,extension,allowDownload` | 下载前应检查 `allowDownload`，再对动态 URL 做安全验证。 |
| `ZLibHostsListing` / `ZLibHost` | `domains` 每项有 `domain,contentAvailable,premium,isRedirector` | 返回的域名需结合当前网络、可用性和重定向检查；`contentAvailable` 不等于本机实测健康。 |
| `ZLibUserListing` / `ZLibUser` | 用户名、邮箱、Kindle 邮箱、捐赠和下载配额、确认状态、Premium 等 | `profile` 应仅输出公开给本人的资料，避免写日志或 JSON 暴露凭证。 |
| `ZLibPlanLimitListing` / `ZLibPlans` | `plans.basic,plans.premium,plans.limits` | 计划规则是公共配置；与个人 `downloads_today/downloads_limit` 分开显示。 |

`ZLibBooksRepo` 对 `ZLibApi.searchBooks` 的调用传入 `BooksRepo.Query` 中的筛选、页码和 `order`，证明排序值会沿 Android 客户端调用链传到表单。客户端对热门、最近、推荐建立本地缓存表，对收藏和下载历史分别传 `order,page,limit`。因此 APK 内的缓存存在不代表服务端会返回相同内容，尤其是当前 `/recently` 只有成功标记。以上调用关系来自 A 中 `ZLibBooksRepo`、`BooksRepo.Query` 与 Room 模型。

认证方面，`ZLibApi` 多数账户相关方法显式声明 `remix-userid` / `remix-userkey` 请求头；`HeaderInterceptor` 还构造同名 Cookie，并加入应用版本、语言和一个固定授权头。这里不提供固定授权值，也不建议复制 APK 凭证。当前仓库已有自己的会话和 Cookie 导入路径；是否需要某个额外请求头，应由隔离测试与服务端响应证明，而非静态反编译推断。`DataModule` 的初始 base URL 是打包配置，不能证明其今日可用。[APK](https://s3proxy-alp.cdn-zlib.sk/swfs_second_public_files/soft/android/zlibrary-app-latest.apk) 的 `HeaderInterceptor`、`DataModule`、`ZLibApi` 是本段来源。

## 排序：APK 声明、调用点与服务端效果

Android 搜索排序弹窗把选项索引交给 `com.positron_it.zlib.ui.library.viewModel.a` 的数组，然后 `ZLibBooksRepo` 经 `BooksRepo.Query.order` 传到 `searchBooks` 的 `order` 表单字段。APK 中八种值依次为：

| UI 含义 | `order` 值 |
| --- | --- |
| 热门 | `popular` |
| 最佳匹配 | `bestmatch` |
| 最近添加 | `date` |
| 书名 A–Z | `titleA` |
| 书名 Z–A | `title` |
| 年份 | `year` |
| 文件从大到小 | `filesize` |
| 文件从小到大 | `filesizeA` |

同一会话、域名、查询词与 `limit=10` 下，八种值分别得到八个不同的 ID 顺序指纹；`date_created`、`date_updated` 和刻意无效的排序值则都与 `popular` 相同。此测试能证明服务器对八种值产生了可观察的不同排列，却不单凭十本样本证明年份和文件大小严格单调，也不证明其他查询恒定如此。当前 [OrderOption](../constants.go) 只有 `popular,date_created,date_updated`，后两者与 APK 的 UI 值不一致；[SearchOptions](../model.go) 甚至尚无 `order` 字段，CLI 也没有 `--order`。因此应先把 APK 值和对应标签贯通，并为同一测试数据上的排序结果加回归测试。成功码本身不足以验证排序。

收藏和下载历史使用另一组值。APK `com.positron_it.zlib.ui.mybooks.viewModel.g` 的收藏排序数组为 `saved_date,saved_dateA,titleA,title,year,filesize,filesizeA`；下载历史排序数组为 `downloaded_date,downloaded_dateA,titleA,title,year,filesize,filesizeA`。同一账户下，8 本收藏书籍在七个值下各有不同 ID 顺序；前 10 本下载历史也得到七种不同顺序。两类列表的 `date_created` 和刻意无效值各自同序，说明“服务器接受”仍不足以证明值有效。8 本和 10 本的比较只证明样本中的可观察差异，不证明字段排序的数学正确性，也不应与搜索共用同一排序枚举。

## 对当前 zlib 的命令建议

优先级按已观察的服务端行为、用户价值与实现风险排序；现有 `recommended`、`most-popular`、`saved`/`favorites`、`profile`、`history`、`doctor`、`search`、`download` 已在 R 中，不重复建议造同义命令。

| 优先级 | 命令或改进 | 依据与具体实现边界 |
| --- | --- | --- |
| 1 | `search --order`，交互搜索加入排序选择 | APK 八值有调用链证据，L 已观察八种不同排列。保持默认排序不变，显式值只接受经过验证的八种；不要继续把 `date_created/date_updated` 当作 Android 排序值。 |
| 1 | `similar <id:hash>` | `/similar` 本次返回 24 本，适合复用当前书籍列表与选择下载界面；简版图书只保证 ID/hash/标题/作者/封面。 |
| 1 | `saved --order` / `favorites --order` / `history --order` | APK 有各自七个值，现有 EAPI 方法只有分页；L 的书籍 ID 顺序比较支持这些值确实改变结果。收藏和历史不能误用搜索排序数组。 |
| 2 | `info languages`、`info extensions` | L 观察到语言映射和 12 个格式，可将搜索筛选值从本地固定枚举更新为服务端列表，仍需兼容离线与缓存失效。 |
| 2 | `info service`、`info plans` 或合并的 `info` | 三个只读端点当前返回结构化 JSON。`plans` 展示计划规则，`profile` 展示本人配额；服务信息可只展示有意义的字段。 |
| 3 | `formats <id:hash>` 或下载前的可选格式选择 | APK 有完整模型，但六次实测均为空数组；先作为如实报告的查询能力，避免承诺可转换或可选。 |
| 暂缓 | `recently` | 当前仅有 `success=1` 而无 `books`；先确认是否需要别的上下文或服务器已停止提供列表。 |
| 另案评估 | `save`、`unsave`、`history delete`、服务器 `send-to-*`、账户更新 | 都会改变服务端状态或发信，即使其中几项使用 GET。需要明确用户操作和错误恢复设计，不能混入健康检查。 |

现有 [EAPI 文档](eapi.md)对登录拒绝、域名和 Cookie 的限制仍适用。特别是 `/eapi/info/domains` 返回 5 项只能说明该账户在该时间点得到的候选域名；判断“当前可用”需要 `doctor` 对每个候选域名实际请求并验证 JSON、重定向和鉴权边界。所有只读实测均未覆盖匿名访问、其他账号、其他网络地区、移动端运行时路径或后续服务端更新。
