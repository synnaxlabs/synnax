#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import urllib.parse
from email.message import Message
from pathlib import Path
from typing import Any
from urllib.error import HTTPError

import pytest
import yaml

from changed_jobs import WORKFLOW
from queue_skip import Unproven, conclusions, main, passed, skip
from wait_for_turn import Transient

REPO = "synnaxlabs/synnax"
BASE = "b" * 40
GROUP = {
    "head_ref": f"refs/heads/gh-readonly-queue/main/pr-7-{BASE}",
    "head_sha": "group",
    "base_sha": BASE,
}

CI = """\
jobs:
  changes:
    runs-on: ubuntu-latest
  core:
    needs: changes
    if: needs.changes.outputs.core == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'core')
    with:
      cached: ${{ github.event_name == 'pull_request' }}
  core-image:
    needs: changes
    if: needs.changes.outputs.core_image == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'core-image')
  client-py:
    needs: [changes, core-image]
    if: needs.changes.outputs.client_py == 'true' ||
      needs.changes.outputs.core == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'client-py')
  driver:
    needs: [changes, core-image]
    if: needs.changes.outputs.driver == 'true' ||
      needs.changes.outputs.core == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'driver')
    with:
      clang_tidy: ${{ needs.changes.outputs.driver == 'true' }}
  docs:
    needs: changes
    if: needs.changes.outputs.docs == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'docs')
  integration:
    needs: changes
    if: github.event_name == 'merge_group' &&
      (needs.changes.outputs.integration == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'integration'))
  ok:
    name: OK
    needs: [changes, client-py, core, core-image, docs, driver, integration]
"""
JOBS = yaml.safe_load(CI)["jobs"]

CORE_CHANGE = {
    "core": "true",
    "core_image": "true",
    "client_py": "false",
    "driver": "false",
    "docs": "false",
    "integration": "true",
}
# The jobs of a pull request run for CORE_CHANGE.
PASSED = {
    "changes": "success",
    "core / Test (ubuntu-latest)": "success",
    "core-image / Build - Docker": "success",
    "client-py / Test (ubuntu-latest)": "success",
    "driver / Test (ubuntu-build-bot)": "success",
    "driver / Lint": "skipped",
    "docs": "skipped",
    "integration": "skipped",
    "OK": "success",
}


def run(head: str = "head", **overrides: Any) -> dict[str, Any]:
    return {
        "id": 11,
        "head_sha": head,
        "event": "pull_request",
        "path": WORKFLOW,
        "head_repository": {"full_name": REPO},
        "status": "completed",
        "conclusion": "success",
        **overrides,
    }


class FakeAPI:
    """Answers GitHub API requests for one pull request and its CI runs."""

    def __init__(self) -> None:
        self.head = "head"
        self.trees = {"group": "tree", "head": "tree"}
        self.behind_by = 0
        self.pulls = [{"number": 7, "head": {"sha": "head"}}]
        self.runs = [run()]
        self.jobs = dict(PASSED)
        self.failure: Exception | None = None
        self.urls: list[str] = []

    def __call__(self, url: str, body: dict[str, Any] | None) -> Any:
        self.urls.append(url)
        if self.failure is not None:
            raise self.failure
        parsed = urllib.parse.urlparse(url)
        params = urllib.parse.parse_qs(parsed.query)
        path = parsed.path.removeprefix(f"/repos/{REPO}/")
        page = int(params.get("page", ["1"])[0])
        if path == "pulls/7":
            return {"head": {"sha": self.head}}
        if path.startswith("git/commits/"):
            return {"tree": {"sha": self.trees[path.split("/")[-1]]}}
        if path == f"compare/{BASE}...{self.head}":
            return {"behind_by": self.behind_by}
        if path == f"commits/{self.head}/pulls":
            return self.pulls
        if path == "actions/workflows/ci.yaml/runs":
            assert params["head_sha"] == [self.head]
            assert params["event"] == ["pull_request"]
            return {"workflow_runs": self.runs[(page - 1) * 100 : page * 100]}
        if path == "actions/runs/11/jobs":
            jobs = [{"name": n, "conclusion": c} for n, c in self.jobs.items()]
            return {"jobs": jobs[(page - 1) * 100 : page * 100]}
        raise AssertionError(f"unexpected request: {url}")


def find(api: FakeAPI, group: dict[str, Any] = GROUP) -> dict[str, str]:
    return conclusions(group, repository=REPO, request=api)


class TestConclusions:
    """Tests for the lookup of the pull request run that tested the group's tree."""

    def test_should_return_the_jobs_of_the_run_on_the_head(self) -> None:
        assert find(FakeAPI()) == PASSED

    def test_should_read_every_page_of_jobs(self) -> None:
        api = FakeAPI()
        api.jobs = {f"job-{i}": "success" for i in range(250)}
        assert len(find(api)) == 250

    def test_should_reject_a_ref_that_names_no_pull_request(self) -> None:
        group = {**GROUP, "head_ref": "refs/heads/main"}
        with pytest.raises(Unproven, match="names no pull request"):
            find(FakeAPI(), group)

    def test_should_reject_a_group_with_another_tree(self) -> None:
        api = FakeAPI()
        api.trees["group"] = "other"
        with pytest.raises(Unproven, match="tree"):
            find(api)

    def test_should_reject_a_head_behind_the_base(self) -> None:
        # The base of a group behind another group is not on the head either.
        api = FakeAPI()
        api.behind_by = 1
        with pytest.raises(Unproven, match="behind"):
            find(api)

    def test_should_reject_a_head_that_another_pull_request_has(self) -> None:
        api = FakeAPI()
        api.pulls.append({"number": 8, "head": {"sha": "head"}})
        with pytest.raises(Unproven, match="pull requests"):
            find(api)

    def test_should_ignore_a_pull_request_that_only_contains_the_head(self) -> None:
        api = FakeAPI()
        api.pulls.append({"number": 8, "head": {"sha": "child"}})
        assert find(api) == PASSED

    def test_should_reject_a_head_with_no_run(self) -> None:
        api = FakeAPI()
        api.runs = []
        with pytest.raises(Unproven, match="no pull request run"):
            find(api)

    @pytest.mark.parametrize(
        "overrides",
        [
            {"head_sha": "older"},
            {"event": "pull_request_target"},
            {"event": "merge_group"},
            {"path": ".github/workflows/forged.yaml"},
            {"head_repository": {"full_name": "fork/synnax"}},
            {"status": "in_progress", "conclusion": None},
            {"conclusion": "failure"},
            {"conclusion": "cancelled"},
        ],
    )
    def test_should_reject_a_run_that_is_no_proof(
        self, overrides: dict[str, Any]
    ) -> None:
        api = FakeAPI()
        api.runs = [run(**overrides)]
        with pytest.raises(Unproven, match="no pull request run"):
            find(api)

    def test_should_take_the_run_that_passed_among_others(self) -> None:
        api = FakeAPI()
        api.runs = [run(id=12, conclusion="failure"), run()]
        assert find(api) == PASSED


class TestPassed:
    """Tests for whether one job passed in the pull request run."""

    def test_should_pass_a_job_that_succeeded(self) -> None:
        assert passed("knip", {"knip": "success"})

    def test_should_not_pass_a_job_that_was_skipped(self) -> None:
        assert not passed("docs", PASSED)

    def test_should_not_pass_a_job_that_did_not_run(self) -> None:
        assert not passed("knip", PASSED)

    def test_should_pass_a_call_with_a_skipped_part(self) -> None:
        assert passed("driver", PASSED)

    @pytest.mark.parametrize("conclusion", ["failure", "cancelled", "timed_out"])
    def test_should_not_pass_a_call_with_a_part_that_did_not_pass(
        self, conclusion: str
    ) -> None:
        jobs = {"core / Lint (linux)": "success", "core / Test": conclusion}
        assert not passed("core", jobs)

    def test_should_not_match_a_job_with_the_same_prefix(self) -> None:
        assert not passed("core", {"core-image / Build - Docker": "success"})


def skipped(
    filters: dict[str, str], ci: list[str], conclusions: dict[str, str] = PASSED
) -> tuple[dict[str, str], list[str]]:
    return skip(filters, ci, jobs=JOBS, conclusions=conclusions)


NONE = dict.fromkeys(CORE_CHANGE, "false")
# The path filter also lists the names of the filters that matched.
TOUCHED = {**CORE_CHANGE, "changes": '["core","core_image","integration"]'}


class TestSkip:
    """Tests for the change outputs of a merge group that the pull request proved."""

    def test_should_skip_the_jobs_that_passed(self) -> None:
        assert skipped(CORE_CHANGE, []) == (NONE, ["core", "integration"])

    def test_should_run_a_job_that_reads_the_event(self) -> None:
        assert "core" in skipped(CORE_CHANGE, [])[1]

    def test_should_run_a_job_that_the_pull_request_skipped(self) -> None:
        filters = {**CORE_CHANGE, "docs": "true"}
        assert skipped(filters, [])[1] == ["core", "docs", "integration"]

    def test_should_run_a_job_that_a_remaining_job_needs(self) -> None:
        jobs = {**PASSED, "client-py / Test (ubuntu-latest)": "skipped"}
        ci = skipped(CORE_CHANGE, [], jobs)[1]
        assert ci == ["client-py", "core", "core-image", "integration"]

    def test_should_skip_a_job_that_an_edit_to_ci_selects(self) -> None:
        assert skipped(NONE, ["client-py", "core-image"]) == (NONE, [])

    def test_should_keep_a_filter_that_a_remaining_job_reads_outside_its_condition(
        self,
    ) -> None:
        filters = {**NONE, "driver": "true", "core_image": "true"}
        jobs = {**PASSED, "driver / Test (ubuntu-build-bot)": "skipped"}
        got, ci = skipped(filters, ["client-py"], jobs)
        assert got == {**NONE, "driver": "true"}
        assert ci == ["core-image", "driver"]

    def test_should_run_a_job_that_reads_a_kept_filter(self) -> None:
        filters = {"driver": "true", "core_image": "true"}
        jobs = {
            **JOBS,
            "driver-tsan": {"if": "needs.changes.outputs.driver == 'true'"},
        }
        found = {
            "driver / Test": "skipped",
            "driver-tsan / Test": "success",
            "core-image / Build - Docker": "success",
        }
        assert skip(filters, [], jobs=jobs, conclusions=found) == (filters, [])

    def test_should_not_let_an_ungated_job_keep_the_jobs_it_needs(self) -> None:
        got, ci = skipped(CORE_CHANGE, sorted(JOBS))
        assert got == NONE
        assert ci == ["core", "docs", "integration"]

    def test_should_change_nothing_when_no_job_passed(self) -> None:
        filters = {**NONE, "docs": "true"}
        assert skipped(filters, ["integration"]) == (filters, ["integration"])

    def test_should_match_a_job_by_its_name(self) -> None:
        jobs = {"knip": {"name": "Knip", "if": "needs.changes.outputs.knip == 'true'"}}
        got = skip({"knip": "true"}, [], jobs=jobs, conclusions={"Knip": "success"})
        assert got == ({"knip": "false"}, [])


class TestMain:
    """Tests for the outputs that the changes job gets."""

    @pytest.fixture(autouse=True)
    def workspace(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        (tmp_path / WORKFLOW).parent.mkdir(parents=True)
        (tmp_path / WORKFLOW).write_text(CI)
        (tmp_path / "event.json").write_text(json.dumps({"merge_group": GROUP}))
        monkeypatch.chdir(tmp_path)
        monkeypatch.setenv("TOUCHED", json.dumps(TOUCHED))
        monkeypatch.setenv("CI_JOBS", "[]")
        monkeypatch.setenv("GITHUB_EVENT_NAME", "merge_group")
        monkeypatch.setenv("GITHUB_EVENT_PATH", str(tmp_path / "event.json"))
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)

    def outputs(self, capsys: pytest.CaptureFixture[str]) -> dict[str, str]:
        lines = capsys.readouterr().out.splitlines()
        return dict(line.split("=", 1) for line in lines)

    def test_should_print_the_outputs_with_the_proven_jobs_left_out(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        main(FakeAPI())
        assert self.outputs(capsys) == {**NONE, "ci": '["core", "integration"]'}

    def test_should_not_ask_the_api_outside_a_merge_group(
        self, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("GITHUB_EVENT_NAME", "pull_request")
        monkeypatch.setenv("CI_JOBS", '["docs"]')
        api = FakeAPI()
        main(api)
        assert self.outputs(capsys) == {**CORE_CHANGE, "ci": '["docs"]'}
        assert api.urls == []

    def test_should_change_nothing_without_a_proof(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        api = FakeAPI()
        api.trees["group"] = "other"
        main(api)
        assert self.outputs(capsys) == {**CORE_CHANGE, "ci": "[]"}

    @pytest.mark.parametrize(
        "failure",
        [
            Transient("server error (502)"),
            HTTPError("https://api.github.com", 404, "Not Found", Message(), None),
            ConnectionResetError("reset"),
            KeyError("head"),
        ],
    )
    def test_should_change_nothing_when_the_api_fails(
        self, failure: Exception, capsys: pytest.CaptureFixture[str]
    ) -> None:
        api = FakeAPI()
        api.failure = failure
        main(api)
        captured = capsys.readouterr()
        assert "::warning::" in captured.err
        outputs = dict(line.split("=", 1) for line in captured.out.splitlines())
        assert outputs == {**CORE_CHANGE, "ci": "[]"}
