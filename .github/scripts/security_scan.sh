#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the security scanners from the repository root. For each scan, writes the SARIF
# results to <out>/<scan>.sarif, the scanner's other output to <out>/<scan>.log, and its
# exit code to <out>/<scan>.exit. Prints nothing a scanner finds: logs on a public
# repository are public. Needs govulncheck, gitleaks, zizmor, and checkov on PATH.
#
# Usage: security_scan.sh <out>

set -uo pipefail

out=$(mkdir -p "$1" && cd "$1" && pwd)

scan() {
    local name=$1
    shift
    "$@" > "$out/$name.sarif" 2> "$out/$name.log"
    echo $? > "$out/$name.exit"
}

for mod in $(git ls-files '*go.mod'); do
    dir=$(dirname "$mod")
    (cd "$dir" && scan "govulncheck-${dir//\//-}" govulncheck -format sarif ./...)
done

scan gitleaks gitleaks git --redact --no-banner --exit-code 0 \
    --report-format sarif --report-path - .

scan osv python3 "$(dirname "$0")/security_osv.py"

scan zizmor zizmor --format sarif --no-exit-codes --no-progress .github

terraform=()
for dir in deploy/preview infra/agent-runner; do
    if [ -d "$dir" ]; then terraform+=(--directory "$dir"); fi
done
# checkov writes SARIF only to a file.
checkov_sarif() {
    checkov "$@" --framework terraform --output sarif --output-file-path "$out/checkov" \
        --soft-fail --quiet >&2 && cat "$out/checkov/results_sarif.sarif"
}
if [ ${#terraform[@]} -gt 0 ]; then scan checkov checkov_sarif "${terraform[@]}"; fi
