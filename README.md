# Telegram Drive Bot

**简体中文** | [English](doc/README_EN.md)

一个基于 Telegram Bot 的个人媒体元信息管理与检索工具. 把发给 Bot 的文件、图片、视频等媒体自动入库, 提供浏览、关键词检索、语义搜索、随机抽取、Telegraph 文章抓取等能力

---

## 重要声明

> 1. **本项目不存储也不提供任何违反版权的内容. ** 项目本身不内置、不分发任何资源. 
> 2. **本项目只提供元信息管理与检索功能. ** 文件实体始终保留在 Telegram 平台, 数据库内仅保存文件元信息(文件名、caption、类型、大小、引用 ID 等). 
> 3. **本项目只服务于私人需求, 不面向公众提供开放服务. ** 默认启用白名单机制, 仅 Owner 与其授权的用户可调用 Bot, 不存在面向公众的注册/开放接入入口. 
> 4. **本项目仅用作学习与研究, 不得商用. ** 任何形式的商业化部署、对外营利性服务均不在本项目授权与支持范围之内. 
> 5. **任何商业、违法用途与本项目无关. ** 部署者与使用者需自行对其行为负责, 因滥用本项目产生的一切后果由行为人自行承担, 与本项目作者及贡献者无关. 

---

## 目录

- [功能特性](#功能特性)
- [命令一览](#命令一览)
- [部署](#部署)
  - [方式一:Docker Compose(推荐)](#方式一docker-compose推荐)
  - [方式二:预编译二进制](#方式二预编译二进制)
  - [方式三:从源码编译](#方式三从源码编译)
  - [systemd 守护(可选)](#systemd-守护可选)
- [数据库准备](#数据库准备)
- [存储模式说明](#存储模式说明)
- [向量搜索(可选)](#向量搜索可选)
- [环境变量](#环境变量)
- [许可证](#许可证)

---

## 功能特性

- **媒体自动入库**:接收 document / photo / video / audio / animation / voice / video_note 七类媒体, 落库后即可检索;
- **双存储模式**:
  - `channel` 模式:文件转发到部署者自有的存储频道/超级群组保存, 支持多频道冗余写入, 取回时按顺序尝试每个副本, 全失败时降级为 `file_id` 直发. 
  - `direct` 模式:仅记录 Telegram `file_id`, 无需任何存储频道, 部署最轻量. 
  - 为防止账号被ban导致炸库的情况, 推荐使用channel模式并且每个冗余存储频道由不同的telegram账号创建, 并在频道中将白名单用户设为管理员.
- **频道健康追踪**:单个频道连续失败达到阈值后自动进入冷却期, 期间跳过该频道, 冷却结束自动恢复;成功一次即重置失败计数. 
- **可选异步频道操作**:存入/删除时频道 I/O 放到后台 goroutine 执行, 用户响应不受频道延迟影响. 
- **三级权限模型**:Owner(超级管理员)→ Admin → User, 默认全局白名单, 非白名单用户不被响应. 
- **检索能力**:
  - `/list` 分页浏览(支持按类型筛选)
  - `/search` 全链检索:向量语义搜索 → 全文检索(FTS)→ 模糊匹配(ILIKE), 逐级降级
  - `/ss` 快速检索:跳过向量层, 只走 FTS + ILIKE, 适合精确关键词
- **随机抽取**:`/rand`、`/randv`、`/randp` 用于回顾历史媒体, 默认 5 条, 上限 10 条. 
- **可选向量搜索**:支持 OpenAI 兼容 API 与 Google Gemini 原生 API, 文件入库后异步生成 embedding, 基于 pgvector 进行余弦距离检索. 
- **Telegraph 抓取**:`/tph <url>` 把 telegra.ph 文章抓取为 Markdown 文件保存, 便于后续检索. 
- **后台维护任务**:
  - `cap_sync` —— 媒体组 caption 回填, 可定时自动跑或手动触发
  - `emb_sync` / `emb_re` —— 增量补齐 / 全量重做 embedding 索引
- **数据库自迁移**:启动时自动执行列重命名、`AutoMigrate`、生成列与索引创建, 无需手工执行 SQL. 
- **代理支持**:支持 SOCKS5 与 HTTP/HTTPS 代理直连 Telegram. 

---

## 命令一览

### 普通用户

| 命令 | 说明 |
|---|---|
| `/start` | 欢迎提示;若携带 `file_<unique_id>` 参数(deep link), 返回归属于自己的对应文件 |
| `/list` | 分页浏览自己的媒体库, 支持按类型筛选 |
| `/search <关键词>` | 全链检索(向量 → FTS → 模糊匹配) |
| `/ss <关键词>` | 快速关键词检索, 跳过向量层 |
| `/stats` | 查看个人统计 |
| `/rand [数量]` | 随机返回媒体(默认 5, 上限 10) |
| `/randv [数量]` | 随机返回视频/动图 |
| `/randp [数量]` | 随机返回图片 |
| `/tph <url>` | 抓取 telegra.ph 文章为 Markdown 并保存 |

### Admin 命令

| 命令 | 说明 |
|---|---|
| `/adduser <user_id>` | 添加白名单用户 |
| `/removeuser <user_id>` | 移除白名单用户 |
| `/listuser` | 用户列表(支持提升/降级管理员) |

### Owner 命令

| 命令 | 说明 |
|---|---|
| `/emb_re` | 全量重做 embedding 索引 |
| `/emb_sync` | 增量补齐缺失的 embedding |
| `/cap_sync` | 媒体组 caption 回填(同时与定时任务共享互斥锁) |

---

## 部署

### 方式一:Docker Compose(推荐)

预先准备好 Telegram Bot Token、自己的 Telegram `user_id`(推荐使用[https://t.me/userinfobot](https://t.me/userinfobot) 获取)、PostgreSQL 连接串(推荐使用 [Supabase](https://supabase.com) 免费套餐, 自带 pgvector). 

```bash
# 1. 拉取仓库或仅下载 docker-compose.yml + .env.example
git clone https://github.com/Merack/telegram-drive-bot.git
cd telegram-drive-bot

# 2. 复制并填写环境变量
cp .env.example .env
# 编辑 .env,  至少填好 BOT_TOKEN / OWNER_ID / DATABASE_URL
#   若使用 channel 存储模式,  还需填写 STORAGE_CHAT_IDS

# 3. 启动
docker compose up -d

# 4. 查看日志
docker compose logs -f bot
```

镜像默认从 `ghcr.io/merack/telegram-drive-bot:latest` 拉取. 如需使用 Docker Hub 镜像或本地源码构建, 可按 `docker-compose.yml` 注释切换. 

启用 `channel` 存储模式时, 需要先创建一个或多个 Telegram 频道/超级群组, 并把 Bot 加为管理员(具备发消息与删除消息权限), 再把对应 `chat_id`(以 `-100` 开头)填入 `STORAGE_CHAT_IDS`. 

### 方式二:预编译二进制

从 [Releases](https://github.com/Merack/telegram-drive-bot/releases) 下载对应平台的 `tar.gz`, 解压后:

```bash
cp .env.example .env
# 编辑 .env 填好必填项
./telegram-drive-bot
```

### 方式三:从源码编译

需要 Go 1.26+. 

```bash
git clone https://github.com/Merack/telegram-drive-bot.git
cd telegram-drive-bot
go build -trimpath -ldflags="-s -w" -o telegram-drive-bot ./cmd/bot
cp .env.example .env
# 编辑 .env
./telegram-drive-bot
```

### systemd 守护(可选)

仓库根目录提供了 `tg-drive-bot.service` 模板. 把二进制和 `.env` 放到一处工作目录后, 修改服务文件中的 `WorkingDirectory` 与 `ExecStart` 路径并安装:

```bash
sudo cp tg-drive-bot.service /etc/systemd/system/
# 修改 /etc/systemd/system/tg-drive-bot.service 中的路径
sudo systemctl daemon-reload
sudo systemctl enable --now tg-drive-bot
sudo systemctl status tg-drive-bot
```

---

## 数据库准备

支持任何 PostgreSQL 兼容数据库. 启动时会自动创建/迁移所需的表与索引, **无需手工执行 SQL**. 

- **推荐:Supabase**. 免运维、自带 pgvector、免费套餐约 15 个并发连接, 默认连接池配置已对此适配;在控制台 `Database → Extensions` 启用 `vector` 扩展即可同时使用向量搜索. 
- **本地 PostgreSQL**:`docker-compose.yml` 中预留了 `pgvector/pgvector:pg17` 服务定义, 取消相关注释即可在docker中启用本地数据库, 镜像自带 pgvector, 向量搜索开箱即用. 或者通过PostgreSQL通用链接连接其他位置任何 PostgreSQL 兼容数据库.

---

## 存储模式说明

通过 `USE_CHANNEL_STORAGE` 切换:

| 维度 | `channel` 模式(默认) | `direct` 模式 |
|---|---|---|
| 文件保存方式 | 转发到自建存储频道, 数据库记录每个频道的副本消息 ID | 仅记录 Telegram 返回的 `file_id` |
| 取回流程 | 按顺序尝试每个副本, 全失败再降级为 `file_id` 直发 | 直接用 `file_id` 发送 |
| 部署复杂度 | 需要至少一个存储频道, Bot 需为管理员 | 无需任何额外资源 |
| 可靠性 | 高(可多频道冗余;`file_id` 失效时仍能从频道读取) | 受 Telegram `file_id` 生命周期影响 |
| 适用场景 | 长期归档、对可用性要求高 | 短期轻量试用, 对文件可靠性要求不高 |

`channel` 模式可叠加以下增强:

- **多频道冗余**:`STORAGE_CHAT_IDS` 填入多个频道 ID, 以逗号分隔, 每个新文件会同时写入所有可用频道, 任一成功即视为保存成功. 
- **健康追踪**:`STORAGE_FAILURE_LIMIT` / `STORAGE_COOLDOWN_PERIOD` 控制故障频道的冷却策略. 
- **异步操作**:`ASYNC_FORWARD` / `ASYNC_DELETE` 启用后, 频道 I/O 放到后台执行, 用户响应更快;失败仅记录日志, 代价是可能产生孤儿副本(可由后续手工清理). 
- 为防止账号被ban导致炸库的情况, 推荐使用channel模式并且每个冗余存储频道由不同的telegram账号创建, 并在频道中将白名单用户设为管理员.
---

## 向量搜索(可选)

启用方式:`VECTOR_SEARCH_ENABLED=true`, 并配置 Embedding API 的 base URL、API key、模型与维度. 

- **OpenAI 兼容 API**:`EMBEDDING_API_TYPE=openai`, 使用 `dimensions` 参数. 常用模型:`text-embedding-3-small`, 推荐维度 `256`. 
- **Google Gemini 原生 API**:`EMBEDDING_API_TYPE=gemini`, 使用 `outputDimensionality` 参数. 常用模型:`gemini-embedding-2`, 推荐维度 `768`. 

文件入库后会异步调用 Embedding API 生成向量, 不影响用户交互延迟. 检索时由 `/search` 优先使用语义匹配, 失败再降级到 FTS / ILIKE. `EMBEDDING_THRESHOLD` 控制余弦距离上限, 值越小越严格. 

数据库需要安装 `pgvector` 扩展(Supabase 在控制台启用即可, 本地 PG 推荐使用 `pgvector/pgvector:pg17` 镜像). 

---

## 环境变量

> 必填项放在最前. `channel` 模式下 `STORAGE_CHAT_IDS` 也是必填, 启用向量搜索时 `EMBEDDING_*` 一组也变为必填. 

### 必填

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `BOT_TOKEN` | Telegram Bot Token, 从 [@BotFather](https://t.me/BotFather) 获取 | - | 是 |
| `OWNER_ID` | 超级管理员的 Telegram `user_id`(整数) | - | 是 |
| `DATABASE_URL` | PostgreSQL 连接串, 例:`postgres://user:pass@host:5432/db?sslmode=require` | - | 是 |

### 存储模式

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `USE_CHANNEL_STORAGE` | `true` 启用频道存储模式;`false` 使用 `file_id` 直发 | `false` | 否 |
| `STORAGE_CHAT_IDS` | 存储频道/超级群组 `chat_id` 列表(以 `-100` 开头), 逗号分隔. 第一个为主副本, 后续为冗余副本 | - | `USE_CHANNEL_STORAGE=true` 时必填 |
| `DELETE_CHANNEL_COPIES` | 删除文件时是否同步清理存储频道里的副本(仅 `channel` 模式生效) | `true` | 否 |
| `STORAGE_FAILURE_LIMIT` | 单频道连续失败多少次后进入冷却 | `3` | 否 |
| `STORAGE_COOLDOWN_PERIOD` | 冷却时长(Go duration, 如 `30s` / `5m` / `1h`) | `5m` | 否 |
| `ASYNC_FORWARD` | 异步转发:写库后立即返回, 频道转发放后台执行(仅 `channel` 模式生效) | `false` | 否 |
| `ASYNC_DELETE` | 异步删除:删库后立即返回, 频道副本删除放后台执行(需 `channel` 且 `DELETE_CHANNEL_COPIES=true`) | `false` | 否 |

### 数据库连接池

> 默认值对 Supabase 免费层(~15 conn 配额)友好, 预留给 `psql` 手动会话的空间. 

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `DB_MAX_OPEN_CONNS` | 最大并发连接数 | `10` | 否 |
| `DB_MAX_IDLE_CONNS` | 保持温热的空闲连接数(不得大于 `DB_MAX_OPEN_CONNS`) | `5` | 否 |
| `DB_CONN_MAX_LIFETIME` | 连接强制回收年龄(Go duration) | `30m` | 否 |
| `DB_CONN_MAX_IDLE_TIME` | 连接空闲多久后关闭 | `5m` | 否 |

### 日志与代理

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `LOG_FORMAT` | `text`(本地开发)或 `json`(日志采集) | `text` | 否 |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | 否 |
| `PROXY_URL` | 代理地址, 支持 `socks5://` 与 `http(s)://`, 留空不使用代理 | 空 | 否 |

### 向量搜索

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `VECTOR_SEARCH_ENABLED` | 是否启用向量语义搜索 | `false` | 否 |
| `EMBEDDING_API_TYPE` | `openai`(OpenAI 兼容)或 `gemini`(Google 原生) | `openai` | 启用向量搜索时必填 |
| `EMBEDDING_BASE_URL` | Embedding API base URL | - | 启用向量搜索时必填 |
| `EMBEDDING_API_KEY` | Embedding API key | - | 启用向量搜索时必填 |
| `EMBEDDING_MODEL` | 模型名, 如 `text-embedding-3-small` / `gemini-embedding-2` | - | 启用向量搜索时必填 |
| `EMBEDDING_DIMENSIONS` | 向量维度, 需与模型输出匹配 | `256` | 否 |
| `EMBEDDING_THRESHOLD` | 余弦距离上限, 范围 `0.0 ~ 2.0`, 越小越严格 | `0.4` | 否 |

### 后台维护任务

| Key | 说明 | 默认值 | 是否必填 |
|---|---|---|---|
| `CAP_SYNC_INTERVAL` | 定时 caption 同步周期(Go duration);留空/不设置即关闭后台任务, 仍可手动 `/cap_sync` 触发. 推荐值 `30m ~ 1h` | 关闭 | 否 |

---

## 许可证

本项目以仓库根目录的 [LICENSE](LICENSE) 为准. 请在使用前阅读并遵守相应条款. 

> 再次提醒:本项目仅供学习与个人使用, 不得用于任何商业或违法用途. 请确保你对部署的实例及其使用方式负全责. 
