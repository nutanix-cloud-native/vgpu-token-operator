# Copyright 2023 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

INCLUDE_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

include $(INCLUDE_DIR)make.mk
include $(INCLUDE_DIR)repo.mk
include $(INCLUDE_DIR)kind.mk
include $(INCLUDE_DIR)goreleaser.mk
include $(INCLUDE_DIR)helm.mk
include $(INCLUDE_DIR)go.mk
include $(INCLUDE_DIR)precommit.mk
