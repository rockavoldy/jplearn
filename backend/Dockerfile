# syntax=docker/dockerfile:1

# ---- Build stage ----------------------------------------------------------
# The Kagome tokenizer is pure Go, so a fully static binary can be produced
# with CGO disabled, which lets us ship a tiny distroless runtime image.
FROM golang:1.23-alpine AS build

WORKDIR /src

# Cache dependency downloads separately from the source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# Build the API server. Migrations are embedded via //go:embed, so no extra
# files need to be copied into the runtime image.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- Runtime stage --------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/api /app/api

# Distroless "nonroot" already runs as UID 65532.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]
