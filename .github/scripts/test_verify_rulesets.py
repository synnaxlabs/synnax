#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import subprocess
from pathlib import Path
from typing import Any

import pytest

SCRIPT = Path(__file__).resolve().parent / "verify_rulesets.sh"

# Stands in for gh: serves the canned rulesets from FAKE_GH_DIR and applies the
# requested jq expression.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
ENDPOINT=""
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        api | --paginate) shift ;;
        --jq)
            EXPR=$2
            shift 2
            ;;
        *)
            ENDPOINT=$1
            shift
            ;;
    esac
done
case "$ENDPOINT" in
    */rulesets\\?*) FILE="$FAKE_GH_DIR/rulesets.json" ;;
    */rulesets/*) FILE="$FAKE_GH_DIR/ruleset-${ENDPOINT##*/}.json" ;;
    *)
        echo "unexpected endpoint: $ENDPOINT" >&2
        exit 1
        ;;
esac
jq -r "$EXPR" "$FILE"
"""


def ruleset(name: str, count: int = 0) -> dict[str, Any]:
    return {
        "name": name,
        "target": "branch",
        "enforcement": "active",
        "conditions": {"ref_name": {"include": ["~DEFAULT_BRANCH"], "exclude": []}},
        "bypass_actors": [],
        "rules": [
            {"type": "deletion"},
            {
                "type": "pull_request",
                "parameters": {"required_approving_review_count": count},
            },
        ],
    }


class Harness:
    """A fake gh serving live rulesets, and a .github/rulesets directory of files."""

    def __init__(self, path: Path) -> None:
        self.root = path
        self.data = path / "gh"
        self.data.mkdir()
        self.files = path / ".github" / "rulesets"
        self.files.mkdir(parents=True)
        bin_dir = path / "bin"
        bin_dir.mkdir()
        gh = bin_dir / "gh"
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "FAKE_GH_DIR": str(self.data),
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def file(self, rules: dict[str, Any]) -> None:
        (self.files / f"{rules['name']}.json").write_text(json.dumps(rules))

    def live(self, *rulesets: dict[str, Any]) -> None:
        """Sets the rulesets on GitHub, with the fields GitHub adds to each one."""
        listing = []
        for key, rules in enumerate(rulesets, start=1):
            listing.append({"id": key, "name": rules["name"]})
            served = {
                "id": key,
                "source_type": "Repository",
                "updated_at": "2026-09-28T22:15:18Z",
                **rules,
                "rules": list(reversed(rules["rules"])),
            }
            (self.data / f"ruleset-{key}.json").write_text(json.dumps(served))
        (self.data / "rulesets.json").write_text(json.dumps(listing))

    def run(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT)],
            cwd=self.root,
            env=self.env,
            capture_output=True,
            text=True,
        )


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    harness = Harness(tmp_path)
    harness.file(ruleset("main"))
    harness.live(ruleset("main"))
    return harness


class TestRulesets:
    """Tests for comparing the live rulesets against the files."""

    def test_passes(self, harness: Harness) -> None:
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "1 rulesets match .github/rulesets" in result.stdout

    def test_fails_on_a_changed_parameter(self, harness: Harness) -> None:
        harness.live(ruleset("main", count=1))
        result = harness.run()
        assert result.returncode != 0
        lines = [line.replace(" ", "") for line in result.stderr.splitlines()]
        assert '-"required_approving_review_count":0' in lines
        assert '+"required_approving_review_count":1' in lines

    def test_fails_on_a_ruleset_missing_from_github(self, harness: Harness) -> None:
        harness.file(ruleset("release"))
        result = harness.run()
        assert result.returncode != 0
        assert "rulesets not on GitHub:\n  release" in result.stderr

    def test_fails_on_a_ruleset_with_no_file(self, harness: Harness) -> None:
        harness.live(ruleset("main"), ruleset("release"))
        result = harness.run()
        assert result.returncode != 0
        assert "with no file in .github/rulesets:\n  release" in result.stderr
