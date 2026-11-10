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
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "verify_rulesets.sh"


def ruleset(name: str, *, count: int = 0) -> dict[str, Any]:
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
        self.files = path / ".github" / "rulesets"
        self.files.mkdir(parents=True)
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
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
            self.gh.serve(f"api */rulesets/{key}", served)
        self.gh.serve("api */rulesets[?]*", listing)

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
