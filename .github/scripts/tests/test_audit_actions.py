#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import shutil
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "audit_actions.sh"
HASH = "0123456789abcdef0123456789abcdef01234567"


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    scripts = tmp_path / ".github" / "scripts"
    scripts.mkdir(parents=True)
    shutil.copy(SCRIPT, scripts)
    (tmp_path / ".github" / "workflows").mkdir()
    return tmp_path


def audit(repo: Path, *uses: str) -> str:
    steps = "".join(f"      - uses: {use}\n" for use in uses)
    workflow = repo / ".github" / "workflows" / "a.yaml"
    workflow.write_text(f"jobs:\n  a:\n    steps:\n{steps}")
    return subprocess.run(
        ["sh", str(repo / ".github" / "scripts" / SCRIPT.name)],
        capture_output=True,
        text=True,
        check=True,
    ).stdout


class TestAuditActions:
    """Tests audit_actions.sh."""

    def test_should_read_the_version_from_the_comment_after_a_hash(
        self, repo: Path
    ) -> None:
        output = audit(repo, f"actions/checkout@{HASH} # v7")
        assert "v7 (1)" in output
        assert HASH not in output

    def test_should_read_the_ref_without_a_comment(self, repo: Path) -> None:
        assert "v6 (1)" in audit(repo, "actions/cache@v6")

    def test_should_skip_local_actions(self, repo: Path) -> None:
        output = audit(repo, "actions/cache@v6", "./.github/actions/setup-pnpm")
        assert "setup-pnpm" not in output

    def test_should_list_an_action_pinned_to_two_versions(self, repo: Path) -> None:
        output = audit(repo, f"a/b@{HASH} # v1", f"a/b@{HASH} # v2")
        assert "Actions with multiple pinned versions" in output
        assert "a.yaml:" in output
