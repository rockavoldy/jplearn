.PHONY: help infra-up infra-down backend frontend extension extension-zip \
	import-dict build-backend build-frontend docker-build docker-push \
	pages-deploy

# Image coordinates for GHCR. REPO is auto-detected from the git remote and
# lower-cased (GHCR requires lowercase repository names).
REGISTRY     ?= ghcr.io
REPO         ?= $(shell git config --get remote.origin.url 2>/dev/null | sed -E 's#.+github\.com[:/](.+)\.git#\1#; s#\.git$$##' | tr '[:upper:]' '[:lower:]')
IMAGE_PREFIX ?= $(REGISTRY)/$(REPO)
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PLATFORMS    ?= linux/amd64,linux/arm64
PAGES_PROJECT ?= jp-reader

help:
	@echo "Targets:"
	@echo "  infra-up       Start Postgres + MinIO for local dev via docker compose"
	@echo "  infra-down     Stop local infra"
	@echo "  backend        Run Go API server"
	@echo "  frontend       Run Vite dev server"
	@echo "  import-dict    Import jmdict-simplified JSON into Postgres"
	@echo "                 (uses JMDICT_PATH from .env, or override: JMDICT_PATH=... make import-dict)"
	@echo "  build-backend  Build Go binary"
	@echo "  build-frontend Build Vue app"
	@echo "  extension      Build the MV3 extension into extension/dist"
	@echo "  extension-zip  Build and package the extension into a loadable zip"
	@echo "  docker-build   Build the backend image locally"
	@echo "  docker-push    Build and push the backend image to GHCR"
	@echo "  pages-deploy   Build and deploy the frontend to Cloudflare Pages (wrangler)"

infra-up:
	docker compose up -d

infra-down:
	docker compose down

backend:
	cd backend && go run ./cmd/api

frontend:
	cd frontend && npm run dev

import-dict:
	cd backend && go run ./cmd/import-jmdict $(if $(JMDICT_PATH),-file "$(JMDICT_PATH)",)

build-backend:
	cd backend && go build -o bin/api ./cmd/api

build-frontend:
	cd frontend && npm run build

extension:
	cd extension && npm run build

# Zip the *contents* of dist/ so manifest.json is at the archive root; the
# resulting file can be loaded directly via chrome://extensions > Load unpacked.
extension-zip: extension
	cd extension/dist && zip -r ../jp-reader-extension.zip . -x "*.DS_Store"
	@echo "Packaged extension/dist -> extension/jp-reader-extension.zip"

docker-build:
	docker build -t $(IMAGE_PREFIX)-backend:$(VERSION) ./backend

docker-push:
	docker buildx build --platform $(PLATFORMS) \
		-t $(IMAGE_PREFIX)-backend:$(VERSION) --push ./backend

# Deploy the SPA to Cloudflare Pages. Requires wrangler auth (CLOUDFLARE_API_TOKEN
# + CLOUDFLARE_ACCOUNT_ID) and VITE_API_BASE_URL set in the environment.
pages-deploy: build-frontend
	cd frontend && npx wrangler pages deploy dist --project-name=$(PAGES_PROJECT)
