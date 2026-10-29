#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest
import yaml

from changed_jobs import FILTER_REF, WORKFLOW, main, select

CI = """\
name: CI
on:
  pull_request:
env:
  A: "1"
concurrency:
  group: ci
permissions:
  contents: read
jobs:
  changes:
    runs-on: ubuntu-latest
    outputs:
      go: x
  core-image:
    needs: changes
    if: needs.changes.outputs.core == 'true'
    runs-on: ubuntu-latest
  alamos-go:
    needs: changes
    if: needs.changes.outputs.alamos_go == 'true'
    with:
      macos: &full ${{ inputs.full == true }}
  client-py:
    needs: [changes, core-image]
    if: needs.changes.outputs.client_py == 'true' ||
      needs.changes.outputs.core == 'true'
    with:
      macos: *full
  # A comment.
  docs:
    needs: changes
    if: needs.changes.outputs.docs == 'true'
    runs-on: ubuntu-latest
  fail-fast:
    runs-on: ubuntu-latest
  ok:
    needs: [alamos-go, changes, client-py, core-image, docs]
    runs-on: ubuntu-latest
"""

FILTERS = """\
go: &go
  - .golangci.yaml
py: &py
  - uv.lock
alamos_go:
  - *go
  - alamos/go/**
core:
  - *go
  - core/**
client_py:
  - *py
  - client/py/**
docs:
  - site/docs/**
"""

ALL = ["alamos-go", "changes", "client-py", "core-image", "docs", "fail-fast", "ok"]


def run(ci: str = CI, *, filters: str = FILTERS) -> list[str]:
    return select(CI, ci, old_filters=FILTERS, new_filters=filters)


class TestWorkflowEdits:
    """Tests selecting jobs from an edit to the workflow."""

    def test_should_select_nothing_without_an_edit(self) -> None:
        assert run() == []

    def test_should_select_nothing_for_a_comment(self) -> None:
        assert run(CI.replace("# A comment.", "# Another comment.")) == []

    def test_should_select_a_job_whose_block_changed(self) -> None:
        ci = CI.replace("docs == 'true'", "docs != 'false'")
        assert run(ci) == ["docs"]

    def test_should_select_the_jobs_a_selected_job_needs(self) -> None:
        ci = CI.replace("macos: *full", "macos: false")
        assert run(ci) == ["client-py", "core-image"]

    def test_should_select_a_job_added_to_ok(self) -> None:
        ci = CI.replace(
            "  fail-fast:",
            "  lint:\n    needs: [changes, core-image]\n  fail-fast:",
        ).replace("needs: [alamos-go,", "needs: [lint, alamos-go,")
        assert run(ci) == ["core-image", "lint"]

    def test_should_select_an_existing_job_added_to_ok(self) -> None:
        ci = CI.replace(
            "    needs: [alamos-go, changes, client-py, core-image, docs]\n",
            "    needs: [changes, client-py, core-image, docs]\n",
        )
        assert select(ci, CI, old_filters=FILTERS, new_filters=FILTERS) == ["alamos-go"]

    def test_should_select_nothing_for_a_job_removed_from_ok(self) -> None:
        ci = CI.replace(
            "  docs:\n    needs: changes\n"
            "    if: needs.changes.outputs.docs == 'true'\n"
            "    runs-on: ubuntu-latest\n",
            "",
        ).replace(", docs]", "]")
        assert run(ci) == []

    def test_should_select_an_added_job(self) -> None:
        ci = CI.replace("  fail-fast:", "  lint:\n    needs: changes\n  fail-fast:")
        assert run(ci) == ["lint"]


class TestSharedEdits:
    """Tests that an edit to a shared part selects every job."""

    @pytest.mark.parametrize(
        ("old", "new"),
        [
            ("  pull_request:\n", "  pull_request:\n  merge_group:\n"),
            ('A: "1"', 'A: "2"'),
            ("group: ci", "group: ci-2"),
            ("contents: read", "contents: write"),
            ("      go: x", "      go: y"),
            ("  ok:\n", "  ok:\n    if: always()\n"),
            (
                "  fail-fast:\n    runs-on: ubuntu-latest",
                "  fail-fast:\n    runs-on: x",
            ),
            ("${{ inputs.full == true }}", "${{ inputs.full }}"),
        ],
        ids=[
            "on",
            "env",
            "concurrency",
            "permissions",
            "changes",
            "ok",
            "fail-fast",
            "anchor",
        ],
    )
    def test_should_select_every_job(self, old: str, new: str) -> None:
        assert old in CI
        assert run(CI.replace(old, new)) == ALL


class TestFilterEdits:
    """Tests selecting jobs from an edit to the change map."""

    def test_should_select_the_jobs_that_read_a_changed_filter(self) -> None:
        filters = FILTERS.replace("site/docs/**", "site/docs/**\n  - site/common/**")
        assert run(filters=filters) == ["docs"]

    def test_should_resolve_anchors(self) -> None:
        filters = FILTERS.replace(".golangci.yaml", ".golangci.yml")
        assert run(filters=filters) == ["alamos-go", "client-py", "core-image"]

    def test_should_ignore_the_order_of_paths(self) -> None:
        filters = FILTERS.replace("  - *go\n  - core/**", "  - core/**\n  - *go")
        assert run(filters=filters) == []


class TestMain:
    """Tests the command line."""

    def test_should_print_nothing_without_a_base(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        main(["changed_jobs.py", ""])
        assert capsys.readouterr().out == "[]\n"


def test_should_gate_every_filtered_job_on_its_ci_output() -> None:
    """Every job the change map gates must also run when the CI edit selects it."""
    root = Path(__file__).resolve().parents[3]
    jobs = yaml.safe_load((root / WORKFLOW).read_text())["jobs"]
    for job, block in jobs.items():
        condition = str(block.get("if", ""))
        if FILTER_REF.search(condition):
            assert f"contains(fromJSON(needs.changes.outputs.ci), '{job}')" in condition
