// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

group "default" {
  targets = [
    "oasf-sdk",
  ]
}

target "oasf-sdk" {
  context = "."
  dockerfile = "Dockerfile"
  output = [
    "type=image",
  ]
  platforms = [
    "linux/arm64",
    "linux/amd64",
  ]
  tags = ["oasf-sdk"]
}

target "oasf-sdk-e2e" {
  context = "."
  dockerfile = "Dockerfile"
  output = [
    "type=image",
  ]
  platforms = [
    "linux/amd64",
  ]
  tags = ["oasf-sdk"]
}
