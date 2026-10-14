# JP Reader — Japanese Learning Web App

Convert any Japanese article or EPUB book into a distraction-free reading
environment. Kanji are annotated with **furigana** automatically and every word
becomes clickable for **on-demand JMdict definitions** via a floating tooltip.

This repository implements the architecture described in [`project_plan.md`](project_plan.md).

## Architecture

```
┌───────────────────┐      POST /api/articles      ┌──────────────────────────┐
│  Browser Extension │ ───────────────────────────▶ │                          │
│  (Manifest V3 +    │  Readability-extracted HTML  │      Go API Server       │
│   @mozilla/        │                              │  (net/http + Chi-style   │
│   readability)     │                              │   stdlib mux)            │
└───────────────────┘                              │                          │
┌───────────────────┐      POST /api/epubs/upload   │  ┌────────────────────┐  │
│   Vue 3 Frontend  │ ───────────────────────────▶ │  │ Kagome NLP (furigana│ │
│  (Pinia + Floating │ ◀───── processed HTML ────── │  │  + vocab spans)     │ │
│   UI tooltip)      │                              │  └────────────────────┘  │
└───────────────────┘                              │  ┌────────────────────┐  │
                                                   │  │ EPUB parser (OPF   │  │
                                                   │  │ spine + images)     │ │
                                                   │  └────────────────────┘  │
                                                   └───────┬──────────┬───────┘
                                                           │          │
                                                  ┌────────▼───┐  ┌───▼─────────┐
                                                  │ PostgreSQL │  │ S3 / R2 /   │
                                                  │ (JMdict +  │  │ MinIO       │
                                                  │  content)  │  │ (EPUB imgs) │
                                                  └────────────┘  └─────────────┘
```

## Tech Stack

| Layer            | Technology                                                        |
| ---------------- | ----------------------------------------------------------------- |
| Backend          | Go 1.22 (`net/http` stdlib mux), `pgx/v5`, `aws-sdk-go-v2`        |
| NLP              | [`github.com/ikawaha/kagome/v2`](https://github.com/ikawaha/kagome) (IPA dict) |
| Frontend         | Vue 3.4 (Composition API, `<script setup>`), Pinia, `@floating-ui/dom` |
| Database         | PostgreSQL 15+ (JSONB + GIN indexes)                              |
| Object storage   | Any S3-compatible store (AWS S3, Cloudflare R2, MinIO)            |
| Browser ext.     | Manifest V3, `@mozilla/readability`                               |

## Hard Constraints (implemented)

- **Dictionary lookup < 50ms** — served from PostgreSQL `jmdict_entries` using
  GIN indexes on the `kanji` / `readings` arrays; no runtime XML parsing.
- **EPUB upload ≤ 50MB** — enforced via `http.MaxBytesReader` + size check.
- **Scrape payload ≤ 5MB** — enforced in the extension *and* on the API.
- **Manifest V3** — service worker + content script, no persistent background page.

## Project Layout

```
backend/
  cmd/api/                 # API server entrypoint
  cmd/import-jmdict/       # jmdict-simplified JSON → PostgreSQL importer
  internal/config/         # env-based configuration
  internal/database/       # pgx pool + embedded SQL migrations
  internal/models/         # domain types
  internal/store/          # SQL data access
  internal/storage/        # S3-compatible client
  internal/nlp/            # Kagome furigana + vocab-span processor
  internal/epub/           # EPUB container/OPF/spine parser
  internal/service/        # EPUB ingestion orchestration
  internal/api/            # HTTP handlers + middleware
frontend/                  # Vue 3 SPA
extension/                 # Manifest V3 browser extension
docker-compose.yml         # Postgres + MinIO for local dev
```

## Getting Started

### 1. Start infrastructure

```bash
docker compose up -d      # Postgres :5432, MinIO :9000 (console :9001)
```

### 2. Configure via `.env`

```bash
cp .env.example .env
# Edit .env if your ports/credentials differ.
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

The importer transparently handles the release formats `.json`, `.json.gz`,
`.tgz` and `.zip`, streams the `words` array one entry at a time (bounded
memory), and stores `kanji`, `kana` readings and English glosses into
`jmdict_entries` with GIN indexes.

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
(enable *Developer mode*). Use the popup to send the current page to your
library, or set a custom backend URL in *Backend settings*.

## Container Images (GHCR)

Every push to `main` and every `v*.*.*` tag builds and publishes two images to
the GitHub Container Registry via
[`.github/workflows/docker-publish.yml`](.github/workflows/docker-publish.yml):

| Image                                | Contents                                            |
| ------------------------------------ | --------------------------------------------------- |
| `ghcr.io/<owner>/<repo>-backend`     | Static Go API server (distroless, non-root, `:8080`)|
| `ghcr.io/<owner>/<repo>-frontend`    | Vue SPA served by nginx (`:80`, SPA history fallback)|

Both are built multi-arch (`linux/amd64`, `linux/arm64`). Tag a release to get
semver tags (`1.2.3`, `1.2`) plus `latest` on the default branch; every build
also gets a `sha-<commit>` tag.

Pull and run them:

```bash
docker pull ghcr.io/<owner>/<repo>-backend:latest
docker pull ghcr.io/<owner>/<repo>-frontend:latest

# Backend — point it at your Postgres + S3/MinIO using the same env vars as .env
docker run --rm -p 8080:8080 \
  -e DATABASE_URL='postgres://jpl:jpl@host.docker.internal:5432/jplearn?sslmode=disable' \
  -e S3_ENDPOINT='http://host.docker.internal:9000' \
  -e S3_ACCESS_KEY_ID=minioadmin -e S3_SECRET_ACCESS_KEY=minioadmin \
  -e S3_USE_PATH_STYLE=true \
  ghcr.io/<owner>/<repo>-backend:latest

# Frontend — API base URL is inlined at build time and defaults to same-origin,
# so put it behind a reverse proxy that also serves the API on /api.
docker run --rm -p 8081:80 ghcr.io/<owner>/<repo>-frontend:latest
```

> **Frontend API URL.** The SPA reads `VITE_API_BASE_URL` at build time (see
> [`frontend/src/api/client.js`](frontend/src/api/client.js)). The published
> image leaves it empty so the app calls `/api` on its own origin. To bake in a
> different backend, rebuild with
> `docker build --build-arg VITE_API_BASE_URL=https://api.example.com frontend`.

Build locally with `make docker-build`, or build and push with `make docker-push`
(override `REPO=owner/name` if the git remote cannot be detected).

## Browser Extension Releases

Pushing a `v*.*.*` tag also triggers
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

You can also build and package it locally with `make extension-zip`, which
produces the same loadable archive at `extension/jp-reader-extension.zip`.

## API Reference

| Method | Path                       | Description                                                    |
| ------ | -------------------------- | -------------------------------------------------------------- |
| POST   | `/api/articles`            | Receive scraped article, run Kagome, upsert into `articles`.   |
| GET    | `/api/articles`            | List recent articles.                                          |
| GET    | `/api/articles/{id}`       | Fetch processed article HTML.                                  |
| POST   | `/api/epubs/upload`        | Upload EPUB; parse spine, offload images, annotate chapters.   |
| GET    | `/api/books`               | List books.                                                    |
| GET    | `/api/books/{id}/chapters` | Table of contents for a book.                                  |
| GET    | `/api/chapters/{id}`       | Rendered chapter HTML.                                         |
| GET    | `/api/images/{id}`         | Stream image from S3 with immutable cache headers.             |
| GET    | `/api/dictionary?lemma={w}`| GIN-indexed JMdict lookup; returns JSONB meanings.             |
| GET    | `/healthz`                 | Liveness probe.                                                |

### Example: dictionary lookup

```bash
curl 'http://localhost:8080/api/dictionary?lemma=食べる'
```

### Example: article ingest

```bash
curl -X POST http://localhost:8080/api/articles \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/a","title":"テスト","html":"<p>私は本を読みます。</p>"}'
```

## Data Model

The schema lives in [`backend/internal/database/migrations/0001_init.sql`](backend/internal/database/migrations/0001_init.sql)
and matches the plan exactly:

- `jmdict_entries` — `kanji TEXT[]`, `readings TEXT[]`, `meanings JSONB` with GIN indexes.
- `articles` — unique `url`, processed `content_html`.
- `books` / `chapters` — EPUB metadata, ordered chapters, cascade delete.
- `epub_images` — S3 `s3_key` / `s3_url` references.

## Notes & Extensibility

- The NLP processor (`internal/nlp`) preserves `<script>`, `<style>` and existing
  `<ruby>` subtrees, so re-processing is idempotent-safe.
- S3 works against AWS, R2 and MinIO via `S3_ENDPOINT` + `S3_USE_PATH_STYLE`.
- The image proxy sets `Cache-Control: public, max-age=31536000, immutable` so
  browsers cache offloaded EPUB images aggressively.
