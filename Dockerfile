# Multi-stage image for the goprecords daemon.
#
# Stages:
#   build   - compiles a static binary (CGO off, pure-Go SQLite).
#   test    - runs vet, the unit tests and the fixture integration tests on the
#             same toolchain; build it with --target test before the runtime.
#   runtime - minimal Alpine with only the binary and the CA bundle.
#
# Bases are pinned by tag and multi-arch index digest so a rebuild under the
# same image tag is reproducible. Refresh both pins (and the image tag in
# conf f3s/goprecords) together when the Go toolchain or Alpine patch moves.
ARG GO_IMAGE=docker.io/library/golang:1.27.3-alpine3.24@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18
ARG RUNTIME_IMAGE=docker.io/library/alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

FROM ${GO_IMAGE} AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/goprecords ./cmd/goprecords

FROM build AS test
# Unprivileged: several tests assert that writes into read-only directories
# fail, which root would silently bypass.
RUN adduser -D -u 10001 tester && chown -R tester /src
USER tester
ENV GOCACHE=/tmp/go-build
RUN CGO_ENABLED=0 go vet ./... \
    && CGO_ENABLED=0 go test ./... \
    && CGO_ENABLED=0 go run ./cmd/goprecords test

FROM ${RUNTIME_IMAGE} AS runtime
# apk upgrade pulls any security fix published after the pinned base was built.
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /out/goprecords /usr/local/bin/goprecords

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/goprecords"]
