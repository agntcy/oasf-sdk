// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

group "default" {
  targets = [
    "translation",
    "validation",
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

target "_e2e" {
  output = [
    "type=image",
  ]
  platforms = [
    "linux/amd64",
  ]
}

target "translation" {
  context = "."
  dockerfile = "./translation/Dockerfile"
  inherits = [
    "_common",
  ]
  tags = ["oasf-sdk-translation"]
}

target "validation" {
  context = "."
  dockerfile = "./validation/Dockerfile"
  inherits = [
    "_common",
  ]
  tags = ["oasf-sdk-validation"]
}

target "validation-e2e" {
  context = "."
  dockerfile = "./validation/Dockerfile"
  inherits = [
    "_e2e",
  ]
  tags = ["oasf-sdk-validation"]
}

target "translation-e2e" {
  context = "."
  dockerfile = "./translation/Dockerfile"
  inherits = [
    "_e2e",
  ]
  tags = ["oasf-sdk-translation"]
}
