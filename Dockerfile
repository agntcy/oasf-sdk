# syntax=docker/dockerfile:1

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

# The extractor's default embedding model, fetched at build time so the server
# never contacts huggingface.co at startup. These are exactly the files
# cybertron's downloader looks for: config.json plus the "bert" entry in its
# supportedModelsFiles. Pinned by revision and checksum, so an upstream change
# fails the build instead of silently shipping different weights. Runs on
# $BUILDPLATFORM so the ~91 MB is fetched once and shared by every target
# platform. Overriding MODEL or MODEL_REVISION means overriding all four
# checksums along with it.
FROM --platform=$BUILDPLATFORM scratch AS model

ARG MODEL=sentence-transformers/all-MiniLM-L6-v2
ARG MODEL_REVISION=1110a243fdf4706b3f48f1d95db1a4f5529b4d41
ARG HF_URL=https://huggingface.co/${MODEL}/resolve/${MODEL_REVISION}

# Left without --chmod deliberately: that flag applies to the directories ADD
# creates as well as to the files, stripping their traverse bit. The defaults
# (0755 directories, 0600 files) already match how the extractor writes its own
# assets — see permAssetDir/permAssetFile in pkg/extractor/manifest.go.
ADD --checksum=sha256:953f9c0d463486b10a6871cc2fd59f223b2c70184f49815e7efbcab5d8908b41 \
  ${HF_URL}/config.json /models/${MODEL}/
ADD --checksum=sha256:c3a85f238711653950f6a79ece63eb0ea93d76f6a6284be04019c53733baf256 \
  ${HF_URL}/pytorch_model.bin /models/${MODEL}/
ADD --checksum=sha256:07eced375cec144d27c900241f3e339478dec958f92fddbc551f295c992038a3 \
  ${HF_URL}/vocab.txt /models/${MODEL}/
ADD --checksum=sha256:acb92769e8195aabd29b7b2137a9e6d6e25c476a4f15aa4355c233426c61576b \
  ${HF_URL}/tokenizer_config.json /models/${MODEL}/

FROM gcr.io/distroless/static:nonroot@sha256:627d6c5a23ad24e6bdff827f16c7b60e0289029b0c79e9f7ccd54ae3279fb45f

WORKDIR /

# The base image leaves HOME unset, so os.UserHomeDir() — and with it the
# extractor's default asset directory — would resolve per runtime: containerd
# reads /home/nonroot from /etc/passwd, `docker run` injects "/" for a non-root
# user, and with HOME absent Go falls back to the temp dir. Pinning it makes the
# path below the asset directory everywhere.
ENV HOME=/home/nonroot

# Provisioning finds these already present and skips the download. It still
# converts the model to spaGO format and fetches the taxonomy from
# OASF_SDK_EXTRACTOR_OASF_URL on every start, so this directory must stay
# writable. A non-default OASF_SDK_EXTRACTOR_MODEL_NAME resolves to a sibling
# directory, finds nothing baked, and downloads as before.
COPY --from=model --chown=65532:65532 /models \
  ${HOME}/.agntcy/oasf-sdk/extractor/models

COPY --from=builder /bin/oasf-sdk ./oasf-sdk

ENTRYPOINT ["./oasf-sdk"]
