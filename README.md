# JP Reader — Japanese Learning Web App

Convert any Japanese article into a distraction-free reading environment. Kanji
are annotated with **furigana** automatically and every word becomes clickable
for **on-demand JMdict definitions** via a floating tooltip.

This repository implements the architecture described in [`project_plan.md`](project_plan.md).

## Deployment Overview

| Component      | Where it runs                          | Notes                                          |
| -------------- | -------------------------------------- | ---------------------------------------------- |
| Frontend (SPA) | **Cloudflare Pages**                   | Static Vue 3 build, deployed via Git integration |
| Backend (API)  | **Your own server via Docker**         | Distroless Go image, `:8080`                   |
| Object storage | **Cloudflare R2**                      | S3-compatible; offloads EPUB assets            |
| Database       | PostgreSQL (on your server)            | JSONB + GIN indexes                            |
| Extension      | Chrome/Edge (unpacked or from Release) | Click the icon to scrape the active page       |

```
┌───────────────────┐   POST /api/articles + X-API-Key   ┌──────────────────────────┐
│  Browser Extension │ ────────────────────────────────▶ │   Go API (Docker on      │
│  (Manifest V3)     │   Readability-extracted HTML       │   your server)           │
└───────────────────┘                                    │                          │
┌───────────────────┐   GET /api/*  (public, no key)     │  ┌────────────────────┐  │
│  Vue SPA on        │ ◀───────────────────────────────▶ │  │ Kagome NLP         │  │
│  Cloudflare Pages  │                                    │  │ (furigana + vocab) │  │
└───────────────────┘                                    │  └────────────────────┘  │
                                                          └───────┬──────────┬───────┘
                                                                  │          │
                                                        ┌─────────▼──┐  ┌────▼─────────┐
                                                        │ PostgreSQL │  │ Cloudflare R2│
                                                        └────────────┘  └──────────────┘
```

## Tech Stack

| Layer            | Technology                                                        |
| ---------------- | ----------------------------------------------------------------- |
| Backend          | Go 1.22 (`net/http` stdlib mux), `pgx/v5`, `aws-sdk-go-v2`        |
| NLP              | [`github.com/ikawaha/kagome/v2`](https://github.com/ikawaha/kagome) (IPA dict) |
| Frontend         | Vue 3.4 (Composition API, `<script setup>`), Pinia, `@floating-ui/dom` |
| Database         | PostgreSQL 15+ (JSONB + GIN indexes)                              |
| Object storage   | Cloudflare R2 (any S3-compatible store also works)                |
| Frontend hosting | Cloudflare Pages                                                  |
| Browser ext.     | Manifest V3, `@mozilla/readability`                               |

## Hard Constraints (implemented)

- **Dictionary lookup < 50ms** — served from PostgreSQL `jmdict_entries` using
  GIN indexes on the `kanji` / `readings` arrays; no runtime XML parsing.
- **Scrape payload ≤ 5MB** — enforced in the extension *and* on the API.
- **Manifest V3** — service worker + content script, no persistent background page.

## Feature Flags

| Flag                        | Default | Effect                                                       |
| --------------------------- | ------- | ------------------------------------------------------------ |
| `ENABLE_EPUB_UPLOAD` (API)  | `false` | Registers `POST /api/epubs/upload`; when off the route 404s. |
| `VITE_ENABLE_EPUB_UPLOAD`   | `false` | Shows the Upload nav link and `/upload` route in the SPA.    |
| `API_KEY` (API)             | empty   | When set, write endpoints require the `X-API-Key` header.    |

> **EPUB upload is turned off for now.** Both flags must be enabled to use it
> again. Reads (GET) are always public; only writes require the API key.

## Project Layout

```
backend/
  cmd/api/                 # API server entrypoint
  cmd/import-jmdict/       # jmdict-simplified JSON → PostgreSQL importer
  internal/config/         # env-based configuration
  internal/database/       # pgx pool + embedded SQL migrations
  internal/models/         # domain types
  internal/store/          # SQL data access
  internal/storage/        # S3-compatible client (R2)
  internal/nlp/            # Kagome furigana + vocab-span processor
  internal/epub/           # EPUB container/OPF/spine parser
  internal/service/        # EPUB ingestion orchestration
  internal/api/            # HTTP handlers + middleware (API key, CORS)
frontend/                  # Vue 3 SPA (deployed to Cloudflare Pages)
  public/_redirects        # SPA history fallback for Pages
extension/                 # Manifest V3 browser extension
docker-compose.yml         # Local dev infra (Postgres + MinIO)
docker-compose.prod.yml    # Production (Postgres + backend image, R2)
```

## Getting Started (local development)

### 1. Start infrastructure

```bash
docker compose up -d      # Postgres :5432, MinIO :9000 (console :9001)
```

### 2. Configure via `.env`

```bash
cp .env.example .env
# For local dev, keep the MinIO S3_* values (see comments in .env.example).
# Set API_KEY to any string, or leave empty to disable auth locally.
```

Configuration is loaded from a `.env` file **automatically** — no export step is
needed. The backend searches for `.env` starting in the working directory and
walking up to the filesystem root, so running from `backend/` still finds the
repo-root `.env`. Variables already present in the process environment always
take precedence over the file. Set `ENV_FILE=/path/to/file` to point at a
specific file.

The frontend reads the same root `.env` (via Vite's `envDir`), so a single file
configures the whole stack.

### 3. Import the JMdict dictionary

The dictionary is sourced from the
[`jmdict-simplified`](https://github.com/scriptin/jmdict-simplified) JSON
distribution (not the original XML). Download a release archive and point
`JMDICT_PATH` at it (this is already set in `.env`):

```bash
# e.g. the "eng-common" subset (~1.4MB) or the full "eng" set (~150MB)
curl -L -o data/jmdict-eng-common.json.tgz \
  https://github.com/scriptin/jmdict-simplified/releases/latest/download/jmdict-eng-common.json.tgz

make import-dict                 # uses JMDICT_PATH from .env
# or override for a one-off:
JMDICT_PATH=data/jmdict-eng-common.json.tgz make import-dict
```

### 4. Run the API

```bash
make backend              # http://localhost:8080
```

Migrations run automatically on startup.

### 5. Run the frontend

```bash
cd frontend && npm install && npm run dev   # http://localhost:5173
```

### 6. Build the browser extension

```bash
cd extension && npm install && npm run build
```

Then load `extension/dist/` as an unpacked extension in `chrome://extensions`
(enable *Developer mode*). **Clicking the toolbar icon scrapes the current page
immediately** and shows a ✓/! badge. Set the backend URL and API key from the
extension's options page (right-click the icon → *Options*).

## Production Deployment

### 1. Cloudflare R2 (object storage)

1. In the Cloudflare dashboard, create an R2 bucket (e.g. `jp-scrapper`).
2. Create an R2 API token with *Object Read & Write* for that bucket; note the
   **Access Key ID** and **Secret Access Key**.
3. (Optional) Attach a public domain (`r2.dev` or a custom domain) and set
   `S3_PUBLIC_BASE_URL` so images are served directly; otherwise images are
   proxied through `/api/images/{id}`.
4. Configure the backend `.env`:

   ```env
   S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
   S3_REGION=auto
   S3_BUCKET=jp-scrapper
   S3_ACCESS_KEY_ID=<R2_ACCESS_KEY_ID>
   S3_SECRET_ACCESS_KEY=<R2_SECRET_ACCESS_KEY>
   S3_USE_PATH_STYLE=false
   S3_PUBLIC_BASE_URL=
   ```

No code changes are needed — R2 is S3-compatible and the client already
supports custom endpoints.

### 2. Backend (Docker on your server)

The backend image is published to GHCR by
[`.github/workflows/docker-publish.yml`](.github/workflows/docker-publish.yml)
on every push to `main` and every `v*.*.*` tag:

| Image                            | Contents                                             |
| -------------------------------- | ---------------------------------------------------- |
| `ghcr.io/<owner>/<repo>-backend` | Static Go API server (distroless, non-root, `:8080`) |

Deploy on your server with the provided compose file:

```bash
# On the server:
git clone <repo> && cd <repo>
cp .env.example .env
# Edit .env: DATABASE_URL, API_KEY, CORS_ALLOWED_ORIGINS, R2 S3_* values.
# Set BACKEND_IMAGE to your published image tag.
docker compose -f docker-compose.prod.yml up -d
```

`docker-compose.prod.yml` runs PostgreSQL plus the backend image, loading all
runtime settings from `.env`. Key production variables:

```env
API_KEY=<long-random-string>            # required for writes
ENABLE_EPUB_UPLOAD=false                # uploads are off for now
CORS_ALLOWED_ORIGINS=https://jp-reader.pages.dev,chrome-extension://*
```

Write requests must include the key:

```bash
curl -X POST https://api.example.com/api/articles \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: <long-random-string>' \
  -d '{"url":"https://example.com/a","title":"テスト","html":"<p>私は本を読みます。</p>"}'
```

### 3. Frontend (Cloudflare Pages)

The SPA is deployed to Cloudflare Pages using **Git integration** — no wrangler
CLI, API token, or GitHub Actions workflow required. Cloudflare clones the repo,
builds the SPA, and redeploys automatically on every push to `main`.

**One-time dashboard setup:**

1. Cloudflare dashboard → **Workers & Pages** → **Create** → **Pages** →
   **Connect to Git**, then select the GitHub repository.
2. Configure the build settings:
   - Project name: `jp-reader`
   - Production branch: `main`
   - Build command: `npm run build`
   - Build output directory: `dist`
   - Root directory: `frontend`
3. Add environment variables for the **Production** (and **Preview**)
   environment:
   - `VITE_API_BASE_URL` → your backend URL (e.g. `https://api.example.com`)
   - `VITE_ENABLE_EPUB_UPLOAD` → `false`
   - `NODE_VERSION` → `20` (optional; recommended for Vite 5)
4. Click **Save and Deploy**.

[`frontend/vite.config.js`](frontend/vite.config.js) already reads `VITE_*`
values from the environment via `loadEnv`, so the dashboard variables are picked
up at build time with no committed `.env` file.

The SPA uses Vue Router history mode; [`frontend/public/_redirects`](frontend/public/_redirects)
provides the `/*  /index.html  200` fallback so deep links work on Pages. Vite
copies that file into `dist/` during the build.

> The old frontend nginx image is no longer published — the SPA is served by
> Cloudflare Pages.

## Browser Extension Releases

Pushing a `v*.*.*` tag triggers
[`.github/workflows/extension-release.yml`](.github/workflows/extension-release.yml),
which:

1. Syncs `extension/manifest.json`'s `version` with the git tag (`v1.2.3` → `1.2.3`).
2. Bundles the extension with esbuild (`npm run build`).
3. Zips the **contents** of `extension/dist/` so `manifest.json` sits at the
   archive root, then attaches `jp-reader-extension.zip` to the GitHub Release.

**Install the released extension:**

1. Download `jp-reader-extension.zip` from the repository's **Releases** page.
2. Unzip it to a permanent folder.
3. Open `chrome://extensions`, enable **Developer mode** (top-right).
4. Click **Load unpacked** and select the unzipped folder.
5. Open the extension's **Options** and set the backend URL and API key.

You can also build and package it locally with `make extension-zip`, which
produces the same loadable archive at `extension/jp-reader-extension.zip`.

## API Reference

Read endpoints are public. Write endpoints require the `X-API-Key` header when
`API_KEY` is configured on the server.

| Method | Path                       | Auth | Description                                                    |
| ------ | -------------------------- | ---- | -------------------------------------------------------------- |
| POST   | `/api/articles`            | Key  | Receive scraped article, run Kagome, upsert into `articles`.   |
| GET    | `/api/articles`            | —    | List recent articles.                                          |
| GET    | `/api/articles/{id}`       | —    | Fetch processed article HTML.                                  |
| POST   | `/api/epubs/upload`        | Key  | Upload EPUB (disabled unless `ENABLE_EPUB_UPLOAD=true`).       |
| GET    | `/api/books`               | —    | List books.                                                    |
| GET    | `/api/books/{id}/chapters` | —    | Table of contents for a book.                                  |
| GET    | `/api/chapters/{id}`       | —    | Rendered chapter HTML.                                         |
| GET    | `/api/images/{id}`         | —    | Stream image from R2 with immutable cache headers.             |
| GET    | `/api/dictionary?lemma={w}`| —    | GIN-indexed JMdict lookup; returns JSONB meanings.             |
| GET    | `/healthz`                 | —    | Liveness probe.                                                |

### Example: dictionary lookup

```bash
curl 'http://localhost:8080/api/dictionary?lemma=食べる'
```

## Data Model

The schema lives in [`backend/internal/database/migrations/0001_init.sql`](backend/internal/database/migrations/0001_init.sql):

- `jmdict_entries` — `kanji TEXT[]`, `readings TEXT[]`, `meanings JSONB` with GIN indexes.
- `articles` — unique `url`, processed `content_html`.
- `books` / `chapters` — EPUB metadata, ordered chapters, cascade delete.
- `epub_images` — R2 `s3_key` / `s3_url` references.

## Notes & Extensibility

- The NLP processor (`internal/nlp`) preserves `<script>`, `<style>` and existing
  `<ruby>` subtrees, so re-processing is idempotent-safe.
- The S3 client works against R2, AWS S3 and MinIO via `S3_ENDPOINT` +
  `S3_USE_PATH_STYLE`.
- The image proxy sets `Cache-Control: public, max-age=31536000, immutable` so
  browsers cache offloaded EPUB images aggressively.
