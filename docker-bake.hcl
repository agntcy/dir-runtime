// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

variable "IMAGE_REPO" { default = "ghcr.io/agntcy" }
variable "IMAGE_TAG" { default = "latest" }
variable "BUILD_LDFLAGS" { default = "-s -w -extldflags -static" }
variable "IMAGE_NAME_SUFFIX" { default = "" }

function "get_tag" {
  params = [tags, name]
  result = coalescelist(tags, ["${IMAGE_REPO}/${name}${IMAGE_NAME_SUFFIX}:${IMAGE_TAG}"])
}

group "default" {
  targets = [
    "dir-runtime-discovery",
    "dir-runtime-server",
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
  args = {
    BUILD_LDFLAGS = "${BUILD_LDFLAGS}"
  }
}

target "docker-metadata-action" {
  tags = []
}

target "dir-runtime-discovery" {
  context = "."
  dockerfile = "./discovery/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-runtime-discovery.name}")
}

target "dir-runtime-server" {
  context = "."
  dockerfile = "./server/Dockerfile"
  target = "production"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-runtime-server.name}")
}
