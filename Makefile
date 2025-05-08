# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

REPO_ROOT := $(CURDIR)
include make/all.mk
