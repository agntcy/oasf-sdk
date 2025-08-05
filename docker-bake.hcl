// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

group "default" {
  targets = [
    "translation",
  ]
}

target "_common" {
  output = [
    "type=image",
  ]
  platforms = [
    "linux/arm64",
    "linux/amd64",
  ]
}

target "translation" {
  context = "."
  dockerfile = "./translation/Dockerfile"
  inherits = [
    "_common",
  ]
  tags = ["${target.translation.name}"]
}

