FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder

WORKDIR /build/server

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=bind,source=.,target=/build,ro \
  go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=bind,source=.,target=/build,ro \
  CGO_ENABLED=0 go build -ldflags="-s -w -extldflags -static" \
  -o /bin/oasf-sdk ./cmd/main.go

FROM gcr.io/distroless/static:nonroot@sha256:627d6c5a23ad24e6bdff827f16c7b60e0289029b0c79e9f7ccd54ae3279fb45f

WORKDIR /

COPY --from=builder /bin/oasf-sdk ./oasf-sdk

ENTRYPOINT ["./oasf-sdk"]
