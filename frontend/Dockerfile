# syntax=docker/dockerfile:1

# ---- Build stage ----------------------------------------------------------
FROM node:20-alpine AS build

WORKDIR /app

# Install dependencies first so Docker can cache this layer.
COPY package.json package-lock.json ./
RUN npm ci

# The API base URL is inlined into the bundle at build time (see
# src/api/client.js -> import.meta.env.VITE_API_BASE_URL). Pass it in with
# --build-arg VITE_API_BASE_URL=https://api.example.com when building.
ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

COPY . .
RUN npm run build

# ---- Runtime stage --------------------------------------------------------
FROM nginx:1.27-alpine AS runtime

# SPA-aware config: history fallback + asset caching headers.
COPY nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
