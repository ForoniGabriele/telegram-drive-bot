# TG Drive Bot

[简体中文](../README.md) | **English**

A personal media metadata management and retrieval tool built on top of a Telegram Bot. Files, photos, videos and other media sent to the Bot are automatically indexed; the Bot offers browsing, keyword search, semantic search, random sampling and Telegraph article scraping.

---

## Important Notice

> 1. **This project does not store or distribute any copyright-infringing content.** All media is uploaded by the operator through Telegram; the project itself ships no built-in resources and distributes none.
> 2. **This project only provides metadata management and retrieval.** The actual media stays on Telegram's platform (referenced by `file_id` or forwarded to storage channels owned by the operator). The database holds only metadata such as file name, caption, type, size and reference IDs.
> 3. **This project is intended for personal use only and is not designed to provide any open service to the public.** A whitelist is enabled by default — only the Owner and explicitly authorized users can interact with the Bot. There is no public sign-up or open access entry point.
> 4. **This project is for learning and research purposes only and must not be used commercially.** Any commercial deployment or for-profit service is outside the scope of this project's authorization and support.
> 5. **Any commercial or unlawful use is unrelated to this project.** Operators and users are solely responsible for their own conduct, and any consequences arising from misuse are borne entirely by the actor — not by the project's authors or contributors.

---

## Table of Contents

- [Features](#features)
- [Commands](#commands)
- [Deployment](#deployment)
  - [Option 1: Docker Compose (recommended)](#option-1-docker-compose-recommended)
  - [Option 2: Pre-built binary](#option-2-pre-built-binary)
  - [Option 3: Build from source](#option-3-build-from-source)
  - [systemd unit (optional)](#systemd-unit-optional)
- [Database](#database)
- [Storage Modes](#storage-modes)
- [Vector Search (optional)](#vector-search-optional)
- [Environment Variables](#environment-variables)
- [License](#license)

---

## Features

- **Automatic media ingestion** — handles document / photo / video / audio / animation / voice / video_note. The caption on the first message of a Telegram media group is automatically backfilled to the rest of the group.
- **Two storage modes**:
  - `channel` mode: files are forwarded to one or more storage channels/supergroups owned by the operator, with optional multi-channel redundancy. On retrieval the Bot tries each copy in order, falling back to a direct `file_id` send if all copies fail.
  - `direct` mode: only the Telegram `file_id` is recorded — no storage channel required, lightest deployment possible.
- **Channel health tracking** — a channel that fails consecutively past a threshold enters a cooldown window during which it is skipped; it is automatically restored afterwards. A single success resets the failure counter.
- **Optional async channel I/O** — forward/delete operations against storage channels can run in background goroutines so user-facing latency is unaffected by Telegram's API.
- **Three-tier permission model** — Owner → Admin → User, with a global whitelist enabled by default. Users not on the whitelist are silently ignored.
- **Retrieval**:
  - `/list` — paginated browsing with type filter
  - `/search` — full chain: vector semantic search → full-text search (FTS) → fuzzy match (ILIKE), degrading layer by layer
  - `/ss` — quick search that skips the vector layer (FTS + ILIKE only), for precise keyword lookups
- **Random sampling** — `/rand`, `/randv`, `/randp` for revisiting past media (default 5, max 10).
- **Optional vector search** — supports OpenAI-compatible APIs and the Google Gemini native API. Embeddings are generated asynchronously after ingestion; retrieval uses cosine distance via pgvector.
- **Telegraph scraping** — `/tph <url>` fetches a telegra.ph article and saves it as a Markdown file for later retrieval.
- **Background maintenance**:
  - `cap_sync` — backfill captions for media groups, scheduled or manual
  - `emb_sync` / `emb_re` — incrementally fill / fully rebuild embedding indices
- **Auto schema migration** — column renames, `AutoMigrate`, generated columns and indexes are applied at boot. No manual SQL required.
- **Proxy support** — SOCKS5 and HTTP/HTTPS proxies for reaching the Telegram API.

---

## Commands

### User commands

| Command | Description |
|---|---|
| `/start` | Welcome message; with payload `file_<unique_id>` (deep link), returns the corresponding file if it belongs to the requesting user |
| `/list` | Paginated browsing with type filter |
| `/search <keywords>` | Full chain search (vector → FTS → fuzzy) |
| `/ss <keywords>` | Quick keyword search, skips the vector layer |
| `/stats` | Personal statistics |
| `/rand [n]` | Random media (default 5, max 10) |
| `/randv [n]` | Random video / animation |
| `/randp [n]` | Random photo |
| `/tph <url>` | Scrape a telegra.ph article into Markdown and save it |

### Admin commands

| Command | Description |
|---|---|
| `/adduser <user_id>` | Add a user to the whitelist |
| `/removeuser <user_id>` | Remove a user from the whitelist |
| `/listuser` | List users (with promote/demote actions) |

### Owner commands

| Command | Description |
|---|---|
| `/emb_re` | Rebuild all embeddings from scratch |
| `/emb_sync` | Fill in missing embeddings only |
| `/cap_sync` | Backfill media-group captions (shares a mutex with the scheduled task) |

---

## Deployment

### Option 1: Docker Compose (recommended)

Have the following ready: a Telegram Bot Token, your own Telegram `user_id`, and a PostgreSQL connection string ([Supabase](https://supabase.com) free tier is recommended — pgvector is built in).

```bash
# 1. Clone the repo (or just grab docker-compose.yml + .env.example)
git clone https://github.com/Merack/tg-drive-bot.git
cd tg-drive-bot

# 2. Copy and edit env file
cp .env.example .env
# Fill in at least BOT_TOKEN / OWNER_ID / DATABASE_URL.
# Add STORAGE_CHAT_IDS as well if using channel storage mode.

# 3. Start
docker compose up -d

# 4. Tail logs
docker compose logs -f bot
```

The default image is pulled from `ghcr.io/merack/tg-drive-bot:latest` (built automatically on tag push). To use the Docker Hub mirror or a local source build, follow the comments in `docker-compose.yml`.

If using `channel` storage mode, create one or more Telegram channels/supergroups, add the Bot as an administrator (with permission to send and delete messages), then put the `chat_id`s (starting with `-100`) into `STORAGE_CHAT_IDS`.

### Option 2: Pre-built binary

Each tag triggers a GitHub Actions build that produces 6 archives across linux / windows / darwin × amd64 / arm64. Download the appropriate `tar.gz` from [Releases](https://github.com/Merack/tg-drive-bot/releases), extract it, then:

```bash
cp .env.example .env
# Edit .env to fill in required keys
./tg-drive-bot
```

### Option 3: Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/Merack/tg-drive-bot.git
cd tg-drive-bot
go build -trimpath -ldflags="-s -w" -o tg-drive-bot ./cmd/bot
cp .env.example .env
# Edit .env
./tg-drive-bot
```

### systemd unit (optional)

A `tg-drive-bot.service` template is provided at the repo root. After placing the binary and `.env` in a working directory, edit `WorkingDirectory` and `ExecStart` in the unit file and install:

```bash
sudo cp tg-drive-bot.service /etc/systemd/system/
# Edit the paths in /etc/systemd/system/tg-drive-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now tg-drive-bot
sudo systemctl status tg-drive-bot
```

---

## Database

Any PostgreSQL-compatible database works. The required tables and indexes are created/migrated on startup — **no manual SQL needed**.

- **Recommended: Supabase.** No ops, pgvector built in, ~15 concurrent connections on the free tier (the default connection pool is tuned for this). Enable the `vector` extension under `Database → Extensions` in the dashboard to also use vector search.
- **Self-hosted PostgreSQL.** A `pgvector/pgvector:pg17` service stub is preset in `docker-compose.yml`; uncomment it to bring up a local database with pgvector ready to go.

---

## Storage Modes

Toggle via `USE_CHANNEL_STORAGE`:

| Aspect | `channel` mode (default) | `direct` mode |
|---|---|---|
| How files are stored | Forwarded to operator-owned storage channels; copy message IDs are persisted | Only the Telegram `file_id` is persisted |
| Retrieval flow | Try each copy in order, fall back to direct `file_id` send if all fail | Send directly via `file_id` |
| Deployment effort | Requires at least one storage channel with the Bot as admin | None |
| Reliability | High — multi-channel redundancy and fallback even when `file_id` expires | Bound to Telegram `file_id` lifetime |
| Use case | Long-term archival, high availability | Lightweight personal use, trial setups |

`channel` mode supports the following enhancements:

- **Multi-channel redundancy** — pass multiple channel IDs to `STORAGE_CHAT_IDS` (comma-separated). Every new file is written to all healthy channels in parallel; one success counts as saved.
- **Health tracking** — `STORAGE_FAILURE_LIMIT` / `STORAGE_COOLDOWN_PERIOD` control how unhealthy channels are temporarily disabled.
- **Async I/O** — `ASYNC_FORWARD` / `ASYNC_DELETE` move channel operations to background goroutines for snappier user responses; failures are logged only, at the cost of potential orphan copies (cleanable later).

---

## Vector Search (optional)

Enable via `VECTOR_SEARCH_ENABLED=true`, then configure the Embedding API base URL, key, model and dimensions.

- **OpenAI-compatible API** — `EMBEDDING_API_TYPE=openai`, uses the `dimensions` parameter. Common model: `text-embedding-3-small` at dimension `256`.
- **Google Gemini native API** — `EMBEDDING_API_TYPE=gemini`, uses `outputDimensionality`. Common model: `gemini-embedding-2` at dimension `768`.

After ingestion the Bot calls the Embedding API in the background; user-facing latency is unaffected. `/search` will prefer semantic match first and degrade to FTS / ILIKE on failure. `EMBEDDING_THRESHOLD` caps the cosine distance — smaller is stricter.

The database needs the `pgvector` extension (enable it in the Supabase dashboard, or use `pgvector/pgvector:pg17` for a self-hosted setup).

---

## Environment Variables

> Required keys come first. `STORAGE_CHAT_IDS` is required in `channel` mode; the `EMBEDDING_*` group becomes required when vector search is enabled.

### Required

| Key | Description | Default | Required |
|---|---|---|---|
| `BOT_TOKEN` | Telegram Bot Token from [@BotFather](https://t.me/BotFather) | - | Yes |
| `OWNER_ID` | Owner's Telegram `user_id` (integer) | - | Yes |
| `DATABASE_URL` | PostgreSQL connection string, e.g. `postgres://user:pass@host:5432/db?sslmode=require` | - | Yes |
| `STORAGE_CHAT_IDS` | Storage channel/supergroup `chat_id` list (each starting with `-100`), comma-separated. The first is primary, the rest are redundant copies | - | Required when `USE_CHANNEL_STORAGE=true` |

### Storage mode

| Key | Description | Default | Required |
|---|---|---|---|
| `USE_CHANNEL_STORAGE` | `true` for channel storage mode; `false` for direct `file_id` send | `true` | No |
| `DELETE_CHANNEL_COPIES` | Whether deleting a file also removes its channel copies (channel mode only) | `true` | No |
| `STORAGE_FAILURE_LIMIT` | Consecutive failures before a channel enters cooldown | `3` | No |
| `STORAGE_COOLDOWN_PERIOD` | Cooldown duration (Go duration, e.g. `30s` / `5m` / `1h`) | `5m` | No |
| `ASYNC_FORWARD` | Async forward: return on DB write, forward to channels in background (channel mode only) | `false` | No |
| `ASYNC_DELETE` | Async delete: return on DB delete, remove channel copies in background (requires channel mode and `DELETE_CHANNEL_COPIES=true`) | `false` | No |

### Database connection pool

> Defaults are tuned for the Supabase free tier (~15 conn quota), leaving headroom for ad-hoc `psql` sessions.

| Key | Description | Default | Required |
|---|---|---|---|
| `DB_MAX_OPEN_CONNS` | Hard cap on open connections | `10` | No |
| `DB_MAX_IDLE_CONNS` | Idle connections kept warm (must not exceed `DB_MAX_OPEN_CONNS`) | `5` | No |
| `DB_CONN_MAX_LIFETIME` | Hard recycle age (Go duration) | `30m` | No |
| `DB_CONN_MAX_IDLE_TIME` | Idle timeout before a connection is closed | `5m` | No |

### Logging & proxy

| Key | Description | Default | Required |
|---|---|---|---|
| `LOG_FORMAT` | `text` (local dev) or `json` (log shipping) | `text` | No |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | No |
| `PROXY_URL` | Proxy URL — supports `socks5://` and `http(s)://`. Empty disables the proxy | empty | No |

### Vector search

| Key | Description | Default | Required |
|---|---|---|---|
| `VECTOR_SEARCH_ENABLED` | Enable semantic vector search | `false` | No |
| `EMBEDDING_API_TYPE` | `openai` (OpenAI-compatible) or `gemini` (Google native) | `openai` | Required when vector search is enabled |
| `EMBEDDING_BASE_URL` | Embedding API base URL | - | Required when vector search is enabled |
| `EMBEDDING_API_KEY` | Embedding API key | - | Required when vector search is enabled |
| `EMBEDDING_MODEL` | Model name, e.g. `text-embedding-3-small` / `gemini-embedding-2` | - | Required when vector search is enabled |
| `EMBEDDING_DIMENSIONS` | Vector dimensionality — must match the model output | `256` | No |
| `EMBEDDING_THRESHOLD` | Cosine distance cap, range `0.0 ~ 2.0`. Smaller is stricter | `0.4` | No |

### Background maintenance

| Key | Description | Default | Required |
|---|---|---|---|
| `CAP_SYNC_INTERVAL` | Period of the scheduled caption-sync task (Go duration). Empty/unset disables the scheduler — `/cap_sync` can still be triggered manually. Recommended: `30m ~ 1h` | disabled | No |

---

## License

This project is licensed under the terms in [LICENSE](../LICENSE) at the repo root. Please read and comply with the applicable terms before use.

> Reminder: this project is for learning and personal use only and must not be used for any commercial or unlawful purpose. The operator is fully responsible for their deployment and how it is used.
