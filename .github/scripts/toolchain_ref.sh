#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the pinned ref of an action that a workflow's build job uses, directly or
# through one of its local composite actions. Prints nothing when the job does not use
# it. Run from the repository root.
# Usage: toolchain_ref.sh <workflow> <action>

set -euo pipefail

WORKFLOW=$1
ACTION=$2

{
    yq '.jobs.build.steps[].uses // ""' "${WORKFLOW}"
    for composite in $(yq '.jobs.build.steps[].uses // "" | select(test("^[.]/"))' \
        "${WORKFLOW}"); do
        yq '.runs.steps[].uses // ""' "${composite}/action.yaml"
    done
} | awk -v a="${ACTION}@" '!found && index($0, a) == 1 { print; found = 1 }'
