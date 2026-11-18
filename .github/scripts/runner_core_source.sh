#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the Url and Version variables of core.ps1 for a release version or a run URL.

set -euo pipefail

core=$1
repo=${GITHUB_REPOSITORY:?}

if [[ $core =~ /actions/runs/([0-9]+) ]]; then
    run=${BASH_REMATCH[1]}
    ids=$(gh api --paginate "repos/$repo/actions/runs/$run/artifacts" --jq \
        '.artifacts[] | select(.expired | not)
        | select(.name | test("^synnax-core-(v.+-)?windows$")) | .id')
    count=$(wc -w <<< "$ids" | tr -d ' ')
    if [[ $count != 1 ]]; then
        echo "::error::Run $run has $count Windows Core artifacts, not 1." >&2
        exit 1
    fi
    # The API answers with a redirect to a link that is valid for one minute.
    url=$(curl -sS -o /dev/null -w '%{redirect_url}' \
        -H "Authorization: Bearer ${GH_TOKEN:?}" \
        "https://api.github.com/repos/$repo/actions/artifacts/$ids/zip")
    if [[ -z $url ]]; then
        echo "::error::GitHub gave no download link for artifact $ids." >&2
        exit 1
    fi
    version=
elif [[ $core =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
    url=https://github.com/$repo/releases/download/core/v$core/synnax-v$core-windows.exe
    version=$core
else
    echo "::error::'$core' is not a release version or the URL of a run or job." >&2
    exit 1
fi

printf 'Url=%s\nVersion=%s\n' "$url" "$version"
