# Telegram Drive Bot

[简体中文](../README.md) | **English**

A personal media metadata management and retrieval tool built on top of a Telegram Bot. Files, photos, videos and other media sent to the Bot are automatically indexed; the Bot offers browsing, keyword search, semantic search, random sampling, Telegraph article scraping, etc.

---

## Important Notice

> 1. This project does not store or distribute any copyright-infringing content. The project itself does not include or distribute any resources.
> 2. This project only provides metadata management and retrieval. The actual media files always remain on the Telegram platform. The database holds only file metadata (file name, caption, type, size, reference ID, etc.).
> 3. This project is intended for personal use only and is not designed to provide public services. A whitelist is enabled by default — only the Owner and authorized users can interact with the Bot. There is no public registration or open access entry point.
> 4. This project is for learning and research purposes only and must not be used commercially. Any form of commercial deployment or for-profit service is outside the scope of this project's authorization and support.
> 5. Any commercial or unlawful use is unrelated to this project. Deployers and users are solely responsible for their own conduct, and any consequences arising from the misuse of this project are borne entirely by the actor — not by the project's authors or contributors.

---

## Table of Contents

- [Features](#features)
- [Commands](#commands)
- [Deployment](#deployment)
  - [Option 1: Docker Compose (recommended)](#option-1-docker-compose-recommended)
  - [Option 2: Pre-built binary](#option-2-pre-built-binary)
  - [Option 3: Build from source](#option-3-build-from-source)
  - [systemd service (optional)](#systemd-service-optional)
- [Database](#database)
- [Storage Modes](#storage-modes)
- [Vector Search (optional)](#vector-search-optional)
- [Environment Variables](#environment-variables)
- [License](#license)

---

## Features

- **Automatic media ingestion**: Receives 7 types of media (`document` / `photo` / `video` / `audio` / `animation` / `voice` / `video_note`) and indexes them for retrieval immediately;
- **Two storage modes**:
  - `channel` mode: Files are forwarded to the deployer's own storage channels/supergroups, supporting redundant writing to multiple channels. On retrieval, the bot tries each copy in sequence, falling back to direct `file_id` sending if all copies fail.
  - `direct` mode: Only the Telegram `file_id` is recorded. No storage channel is required, making the deployment extremely lightweight.
  - *To prevent database loss caused by account bans, it is recommended to use `channel` mode, create each redundant storage channel using different Telegram accounts, and set the whitelisted users as administrators in those channels.*
- **Channel health tracking**: If a single channel fails consecutively up to a threshold, it automatically enters a cooldown period during which it is skipped. It automatically recovers after the cooldown ends, and any successful operation resets the failure counter.
- **Optional async channel I/O**: Channel forward/delete operations are pushed to background goroutines so user responsiveness is not affected by channel latency.
- **Three-tier permission model**: Owner (Super Admin) → Admin → User. A global whitelist is enabled by default; non-whitelisted users are ignored.
- **Retrieval capabilities**:
  - `/list` - Paginated browsing (supports filtering by type)
  - `/search` - Full chain retrieval: Semantic vector search → Full-text search (FTS) → Fuzzy match (ILIKE), degrading step-by-step
  - `/ss` - Quick retrieval: Skips the vector layer, only runs FTS + ILIKE, suitable for precise keywords
- **Random sampling**: `/rand`, `/randv`, `/randp` for reviewing history media, default is 5 items, max 10 items.
- **Optional vector search**: Supports OpenAI-compatible APIs and Google Gemini native API. Embeddings are generated asynchronously after ingestion, and cosine distance retrieval is performed via `pgvector`.
- **Telegraph scraping**: `/tph <url>` scrapes a telegra.ph article into a Markdown file and saves it for later retrieval.
- **Background maintenance**:
  - `cap_sync` — Media group caption backfill, runs automatically on schedule or can be triggered manually.
  - `emb_sync` / `emb_re` — Incremental backfill / full rebuild of embedding indices.
- **Auto schema migration**: Columns are renamed, `AutoMigrate` is executed, and generated columns and indexes are created automatically on startup. No manual SQL execution is required.
- **Proxy support**: Supports SOCKS5 and HTTP/HTTPS proxies to connect to Telegram directly.

---

## Commands

### Regular Users

| Command | Description |
|---|---|
| `/start` | Welcome message; if carrying a `file_<unique_id>` parameter (deep link), returns the corresponding file belonging to the user |
| `/list` | Browse personal media library with pagination, supports filtering by type |
| `/search <keyword>` | Full chain retrieval (Vector → FTS → Fuzzy match) |
| `/ss <keyword>` | (simple search) FTS → Fuzzy match, skipping vector search. Try this if you are not satisfied with the results of /search |
| `/stats` | View personal statistics |
| `/rand [count]` | Return random media (default 5, max 10) |
| `/randv [count]` | Return random videos / animations |
| `/randp [count]` | Return random photos |
| `/tph <url>` | Scrape a telegra.ph article into Markdown and save it |

### Admin Commands

| Command | Description |
|---|---|
| `/adduser <user_id>` | Add a whitelisted user |
| `/removeuser <user_id>` | Remove a whitelisted user |
| `/listuser` | User list (supports promoting/demoting admins) |

### Owner Commands

| Command | Description |
|---|---|
| `/emb_re` | Full rebuild of the embedding index |
| `/emb_sync` | Incrementally sync missing embeddings |
| `/cap_sync` | Backfill media-group captions (shares a mutex with the scheduled task) |

---

## Deployment

### Option 1: Docker Compose (recommended)

Prepare your Telegram Bot Token, your Telegram `user_id` (recommended to obtain via [https://t.me/userinfobot](https://t.me/userinfobot)), and a PostgreSQL connection string (recommended to use [Supabase](https://supabase.com) free tier, which comes with pgvector).

```bash
# 1. Clone the repository or download docker-compose.yml + .env.example
git clone https://github.com/Merack/telegram-drive-bot.git
cd telegram-drive-bot

# 2. Copy and fill in the environment variables
cp .env.example .env
# Edit .env, fill in at least BOT_TOKEN / OWNER_ID / DATABASE_URL
# If using channel storage mode, also fill in STORAGE_CHAT_IDS

# 3. Start
docker compose up -d

# 4. View logs
docker compose logs -f bot
```

The image is pulled from `ghcr.io/merack/telegram-drive-bot:latest` by default. To use a Docker Hub mirror or build from local source, toggle the comments in `docker-compose.yml`.

When enabling `channel` storage mode, you need to create one or more Telegram channels/supergroups, add the Bot as an administrator (with send and delete message permissions), and then fill the corresponding `chat_id`s (starting with `-100`) into `STORAGE_CHAT_IDS`.

### Option 2: Pre-built binary

Download the corresponding platform's `tar.gz` from [Releases](https://github.com/Merack/telegram-drive-bot/releases), extract it, and then:

```bash
cp .env.example .env
# Edit .env and fill in required fields
./telegram-drive-bot
```

### Option 3: Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/Merack/telegram-drive-bot.git
cd telegram-drive-bot
go build -trimpath -ldflags="-s -w" -o telegram-drive-bot ./cmd/bot
cp .env.example .env
# Edit .env
./telegram-drive-bot
```

### systemd service (optional)

A `tg-drive-bot.service` template is provided in the repository root. After placing the binary and `.env` in a working directory, modify the `WorkingDirectory` and `ExecStart` paths in the service file, then install:

```bash
sudo cp tg-drive-bot.service /etc/systemd/system/
# Edit the paths in /etc/systemd/system/tg-drive-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now tg-drive-bot
sudo systemctl status tg-drive-bot
```

---

## Database

Supports any PostgreSQL-compatible database. Required tables and indexes are automatically created/migrated at startup — **no manual SQL execution required**.

- **Recommended: Supabase**. Zero ops, comes with pgvector, free tier supports ~15 concurrent connections (default connection pool is tuned for this). Simply enable the `vector` extension in the `Database → Extensions` console to use vector search.
- **Local PostgreSQL**: A `pgvector/pgvector:pg17` service definition is reserved in `docker-compose.yml`. Uncomment the relevant lines to enable a local database in Docker, which has pgvector built-in and ready for vector search. Or connect to any other PostgreSQL-compatible database at any location using a standard PostgreSQL connection string.

---

## Storage Modes

Toggle via `USE_CHANNEL_STORAGE`:

| Aspect | `channel` mode (default) | `direct` mode |
|---|---|---|
| File Storage Method | Forwarded to self-built storage channels; database records the message ID of each channel copy | Only records the Telegram `file_id` |
| Retrieval Flow | Tries each copy in order; falls back to direct `file_id` sending if all fail | Sends directly using `file_id` |
| Deployment Complexity | Requires at least one storage channel with the Bot as admin | No extra resources required |
| Reliability | High (supports multi-channel redundancy; still retrievable even if `file_id` expires) | Bound to the Telegram `file_id` lifetime |
| Use Case | Long-term archiving, high availability requirements | Short-term lightweight trials, lower file reliability requirements |

Enhancements available for `channel` mode:

- **Multi-channel redundancy**: Fill multiple channel IDs into `STORAGE_CHAT_IDS` separated by commas. Each new file is written to all available channels simultaneously, and any single success is treated as saved.
- **Health tracking**: `STORAGE_FAILURE_LIMIT` / `STORAGE_COOLDOWN_PERIOD` control the cooldown strategy for failing channels.
- **Async operations**: With `ASYNC_FORWARD` / `ASYNC_DELETE` enabled, channel I/O is offloaded to background goroutines, making user response faster. Failures are only logged, at the cost of potential orphan copies (can be cleaned up manually later).
- **Security Recommendation**: To prevent account bans from causing database loss, it is recommended to use `channel` mode, create each redundant storage channel using different Telegram accounts, and set whitelisted users as administrators in those channels.

---

## Vector Search (optional)

Enable via `VECTOR_SEARCH_ENABLED=true`, and configure the Embedding API base URL, API key, model and dimensions.

- **OpenAI-compatible API**: `EMBEDDING_API_TYPE=openai`, using the `dimensions` parameter. Common model: `text-embedding-3-small`, recommended dimension `256`.
- **Google Gemini native API**: `EMBEDDING_API_TYPE=gemini`, using the `outputDimensionality` parameter. Common model: `gemini-embedding-2`, recommended dimension `768`.

After files are indexed, the Bot calls the Embedding API asynchronously to generate vectors, which does not affect user interaction latency. `/search` prioritizes semantic matching, and degrades to FTS / ILIKE on failure. `EMBEDDING_THRESHOLD` controls the upper limit of cosine distance — smaller values are stricter.

The database must have the `pgvector` extension installed (enable it in the Supabase console, or use the `pgvector/pgvector:pg17` image for local PG).

---

## Environment Variables

> Required keys come first. `STORAGE_CHAT_IDS` is also required in `channel` mode; the `EMBEDDING_*` group becomes required when vector search is enabled.

### Required

| Key | Description | Default | Required |
|---|---|---|---|
| `BOT_TOKEN` | Telegram Bot Token, obtained from [@BotFather](https://t.me/BotFather) | - | Yes |
| `OWNER_ID` | Telegram `user_id` of the super administrator (integer) | - | Yes |
| `DATABASE_URL` | PostgreSQL connection string, e.g. `postgres://user:pass@host:5432/db?sslmode=require` | - | Yes |

### Storage mode

| Key | Description | Default | Required |
|---|---|---|---|
| `USE_CHANNEL_STORAGE` | `true` to enable channel storage mode; `false` to use `file_id` direct sending | `false` | No |
| `STORAGE_CHAT_IDS` | List of storage channel/supergroup `chat_id`s (starting with `-100`), comma-separated. The first is primary, the rest are redundant copies | - | Required when `USE_CHANNEL_STORAGE=true` |
| `DELETE_CHANNEL_COPIES` | Whether to clean up copies in storage channels when deleting files (only takes effect in `channel` mode) | `true` | No |
| `STORAGE_FAILURE_LIMIT` | Number of consecutive failures before a channel enters cooldown | `3` | No |
| `STORAGE_COOLDOWN_PERIOD` | Cooldown duration (Go duration, e.g., `30s` / `5m` / `1h`) | `5m` | No |
| `ASYNC_FORWARD` | Async forward: returns immediately after DB write, channel forwarding runs in the background (only takes effect in `channel` mode) | `false` | No |
| `ASYNC_DELETE` | Async delete: returns immediately after DB delete, channel copy deletion runs in the background (requires `channel` mode and `DELETE_CHANNEL_COPIES=true`) | `false` | No |

### Database connection pool

> Defaults are friendly to the Supabase free tier (~15 conn quota), leaving headroom for manual `psql` sessions.

| Key | Description | Default | Required |
|---|---|---|---|
| `DB_MAX_OPEN_CONNS` | Maximum concurrent connections | `10` | No |
| `DB_MAX_IDLE_CONNS` | Idle connections kept warm (must not exceed `DB_MAX_OPEN_CONNS`) | `5` | No |
| `DB_CONN_MAX_LIFETIME` | Hard recycle age (Go duration) | `30m` | No |
| `DB_CONN_MAX_IDLE_TIME` | Idle timeout before a connection is closed | `5m` | No |

### Logging & proxy

| Key | Description | Default | Required |
|---|---|---|---|
| `LOG_FORMAT` | `text` (local dev) or `json` (log shipping) | `text` | No |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | No |
| `PROXY_URL` | Proxy URL, supports `socks5://` and `http(s)://`, empty to disable proxy | empty | No |

### Vector search

| Key | Description | Default | Required |
|---|---|---|---|
| `VECTOR_SEARCH_ENABLED` | Enable semantic vector search | `false` | No |
| `EMBEDDING_API_TYPE` | `openai` (OpenAI-compatible) or `gemini` (Google native) | `openai` | Required when vector search is enabled |
| `EMBEDDING_BASE_URL` | Embedding API base URL | - | Required when vector search is enabled |
| `EMBEDDING_API_KEY` | Embedding API key | - | Required when vector search is enabled |
| `EMBEDDING_MODEL` | Model name, e.g., `text-embedding-3-small` / `gemini-embedding-2` | - | Required when vector search is enabled |
| `EMBEDDING_DIMENSIONS` | Vector dimensionality, must match model output | `256` | No |
| `EMBEDDING_THRESHOLD` | Cosine distance cap, range `0.0 ~ 2.0`, smaller is stricter | `0.4` | No |

### Background maintenance

| Key | Description | Default | Required |
|---|---|---|---|
| `CAP_SYNC_INTERVAL` | Period of the scheduled caption-sync task (Go duration). Empty/unset disables the scheduler — `/cap_sync` can still be triggered manually. Recommended: `30m ~ 1h` | disabled | No |

---

## License

This project is subject to the [LICENSE](LICENSE) at the repository root. Please read and comply with the applicable terms before use.

> Once again: this project is for learning and personal use only and must not be used for any commercial or unlawful purpose. Please ensure that you are fully responsible for your deployed instance and how it is used.
