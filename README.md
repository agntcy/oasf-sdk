# OASF SDK

![GitHub Release (latest by date)](https://img.shields.io/github/v/release/agntcy/oasf-sdk)
[![CI](https://github.com/agntcy/oasf-sdk/actions/workflows/lint.yaml/badge.svg?branch=main)](https://github.com/agntcy/oasf-sdk/actions/workflows/lint.yaml)
[![Coverage](https://codecov.io/gh/agntcy/oasf-sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/agntcy/oasf-sdk)
[![License](https://img.shields.io/github/license/agntcy/oasf-sdk)](./LICENSE.md)

The OASF SDK contains SDKs related to the [OASF](https://github.com/agntcy/oasf) project.

## Usage

See the [USAGE.md](https://github.com/agntcy/oasf-sdk/blob/main/USAGE.md) file for more information on how to use the
SDKs.

## Proto Bindings

Check out the proto bindings available under [buf.build/agntcy/oasf-sdk](https://buf.build/agntcy/oasf-sdk).

## Third-Party Model

The published server image ships the extractor's default embedding model,
[`sentence-transformers/all-MiniLM-L6-v2`](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2),
so the server needs no HuggingFace access at startup. The model is
redistributed unmodified under the Apache 2.0 License; see the model card for
its authorship and citation.

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
