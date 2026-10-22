# syntax=docker/dockerfile:1

FROM node:26-alpine AS web-admin
WORKDIR /src/web/admin
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci
COPY web/admin/ ./
RUN npm run build

FROM node:26-alpine AS web-player
WORKDIR /src/web/player
COPY web/player/package.json web/player/package-lock.json ./
RUN npm ci
COPY web/player/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-admin /src/web/admin/dist ./web/admin/dist
COPY --from=web-player /src/web/player/dist ./web/player/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/podcast .
# Blob storage mount point, created here because distroless has no shell.
RUN mkdir -p /out/data/blobs

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/podcast /podcast
# Owned by the nonroot user (65532) so a fresh named volume mounted here
# inherits a writable directory.
COPY --from=build --chown=65532:65532 /out/data /data
EXPOSE 8080
ENTRYPOINT ["/podcast"]
CMD ["serve"]
