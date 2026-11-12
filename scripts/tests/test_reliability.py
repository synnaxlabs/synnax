#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ast
import re
import subprocess
from pathlib import Path
from typing import Any

import pytest

from reliability import (
    JOB_NAME,
    TC_ARTIFACT,
    TC_JOB_NAME,
    Cases,
    case_rows,
    ci_jobs,
    enumerate_tests,
    hardware_suites,
    integration_sources,
    os_of,
    platform_skip,
    run_id_of,
    shipped_changes,
    spaces_key,
    suite_jobs,
)

FILES = {
    "core/frame_test.go": (
        'var _ = Describe("Frame", func() {\n'
        '\tIt("reads a frame", func() {})\n'
        "\tIt(\n"
        '\t\t"wraps its name",\n'
        "\t\tfunc() {},\n"
        "\t)\n"
        '\tDescribeTable("sizes", func(n int) {},\n'
        '\t\tEntry("one", 1),\n'
        "\t)\n"
        "})\n"
    ),
    "core/frame.go": 'It("is not in a test file")\n',
    "console/plot.spec.ts": (
        'it("renders", () => {});\ntest.each([1, 2])("doubles %d", (n) => {});\n'
    ),
    "driver/task_test.cpp": ("TEST(Task, Starts) {}\nTEST_F(TaskFixture, Stops) {}\n"),
    "client/py/test_frame.py": (
        "def test_reads():\n    pass\n\n\nasync def test_streams():\n    pass\n"
    ),
    "integration/tests/console/plot.py": (
        "import x\n\n\nclass Plot(TestCase):\n    pass\n"
    ),
    "scripts/test_check.py": "def test_checks():\n    pass\n",
    "scripts/runners/tests/test_scaler.py": "def test_scales():\n    pass\n",
}

# A scratch repo holding FILES, and the SHA of its only commit.
Repo = tuple[Path, str]


def git(root: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True, text=True
    ).stdout.strip()


@pytest.fixture(scope="module")
def repo(tmp_path_factory: pytest.TempPathFactory) -> Repo:
    root = tmp_path_factory.mktemp("repo")
    git(root, "init", "-q", "-b", "main")
    for path, text in FILES.items():
        (root / path).parent.mkdir(parents=True, exist_ok=True)
        (root / path).write_text(text)
    git(root, "add", ".")
    git(
        root,
        "-c",
        "user.email=ci@synnaxlabs.com",
        "-c",
        "user.name=ci",
        "commit",
        "-q",
        "-m",
        "tests",
    )
    return root, git(root, "rev-parse", "HEAD")


class TestEnumerateTests:
    """Test names and lines found in each language's test files."""

    def test_should_name_go_its_and_entries_including_wrapped_ones(
        self, repo: Repo
    ) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="go", dirs=["core"]) == [
            ("core/frame_test.go", 2, "reads a frame"),
            ("core/frame_test.go", 3, "wraps its name"),
            ("core/frame_test.go", 8, "one"),
        ]

    def test_should_name_ts_its_and_each_tables(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="ts", dirs=["console"]) == [
            ("console/plot.spec.ts", 1, "renders"),
            ("console/plot.spec.ts", 2, "doubles %d"),
        ]

    def test_should_name_cpp_tests_and_fixtures(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="cpp", dirs=["driver"]) == [
            ("driver/task_test.cpp", 1, "Task, Starts"),
            ("driver/task_test.cpp", 2, "TaskFixture, Stops"),
        ]

    def test_should_name_sync_and_async_py_tests(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="py", dirs=["client/py"]) == [
            ("client/py/test_frame.py", 1, "test_reads"),
            ("client/py/test_frame.py", 5, "test_streams"),
        ]

    def test_should_return_nothing_for_a_directory_without_tests(
        self, repo: Repo
    ) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="go", dirs=["console"]) == []

    def test_should_leave_out_the_tests_of_a_nested_job(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha=sha, lang="py", dirs=["scripts"]) == [
            ("scripts/test_check.py", 1, "test_checks"),
        ]


class TestIntegrationSources:
    """Integration sources read at a commit."""

    def test_should_key_python_files_by_their_path_in_integration(
        self, repo: Repo
    ) -> None:
        root, sha = repo
        assert integration_sources(root, sha=sha) == {
            "tests/console/plot.py": FILES["integration/tests/console/plot.py"],
        }


CASES = {
    "framework/test_case.py": "class TestCase:\n    pass\n",
    "console/case.py": (
        "from framework.test_case import TestCase\n\n\n"
        "class ConsoleCase(TestCase):\n    pass\n"
    ),
    "tests/driver/simulator_case.py": (
        "from framework.test_case import TestCase\n\n\n"
        "class SimulatorCase(TestCase):\n    pass\n"
    ),
    "tests/driver/task.py": (
        "from framework.test_case import TestCase\n\n\n"
        "class TaskCase(TestCase):\n"
        "    def run(self):\n"
        "        self.client.tasks.configure(self.tsk)\n\n\n"
        "class ReadTaskCase(TestCase, Generic[T]):\n    pass\n"
    ),
    "tests/hitl/crio_case.py": (
        "from framework.test_case import TestCase\n\n\n"
        "class CRIOCase(TestCase):\n    pass\n"
    ),
    "tests/hitl/crio_driver.py": (
        "from tests.hitl.crio_case import CRIOCase\n\n\n"
        "class CRIODriver(CRIOCase):\n    pass\n"
    ),
    "tests/hitl/ethercat.py": (
        "from tests.driver.task import ReadTaskCase\n\n\n"
        "class EtherCATRead(ReadTaskCase[sy.ethercat.ReadTask]):\n    pass\n"
    ),
    "tests/hitl/ethercat_perf.py": (
        "from tests.hitl.ethercat import EtherCATRead\n\n\n"
        "class EtherCATStartTime(EtherCATRead):\n    pass\n"
    ),
    "tests/console/plot.py": (
        "from console.case import ConsoleCase\n\n\n"
        "class _Helper:\n    pass\n\n\n"
        "class Plot(ConsoleCase):\n    pass\n"
    ),
    "tests/console/modbus.py": (
        "from console.case import ConsoleCase\n"
        "from tests.driver.simulator_case import SimulatorCase\n\n\n"
        "class ModbusRead(SimulatorCase, ConsoleCase):\n    pass\n"
    ),
    "tests/driver/ni.py": (
        "import platform\n\n"
        "from tests.driver.task import TaskCase\n\n\n"
        "class NIRead(TaskCase):\n"
        "    def setup(self):\n"
        '        if platform.system().lower() != "windows":\n'
        '            self.skip("Windows DAQmx drivers required")\n\n\n'
        "class NIMissing(TaskCase):\n"
        "    def setup(self):\n"
        '        if platform.system().lower() == "windows":\n'
        '            self.skip("DAQmx is installed")\n'
    ),
}


class TestCases:
    """Integration cases resolved to their classes and tagged."""

    def test_should_resolve_a_module_named_case_to_its_one_test_class(self) -> None:
        case = Cases(CASES).resolve("console/plot", name="plot")
        assert case.source == "integration/tests/console/plot.py#L8"
        assert case.tags == ["playwright", "no-driver"]
        assert case.skipped == {}

    def test_should_tag_a_simulator_case_as_using_the_driver(self) -> None:
        case = Cases(CASES).resolve("console/modbus", name="modbus")
        assert case.tags == ["playwright", "driver"]

    def test_should_skip_a_windows_only_case_on_other_platforms(self) -> None:
        case = Cases(CASES).resolve("driver/ni", name="NIRead")
        assert case.source == "integration/tests/driver/ni.py#L6"
        assert case.tags == ["headless", "driver", "windows-only"]
        assert case.skipped == {
            "ubuntu": "Windows DAQmx drivers required",
            "macos": "Windows DAQmx drivers required",
        }

    def test_should_resolve_a_matrix_name_to_its_class(self) -> None:
        case = Cases(CASES).resolve("driver/ni", name="NIRead_65537")
        assert case.source == "integration/tests/driver/ni.py#L6"

    def test_should_skip_a_not_windows_case_on_windows(self) -> None:
        case = Cases(CASES).resolve("driver/ni", name="NIMissing")
        assert case.tags == ["headless", "driver", "not-windows"]
        assert case.skipped == {"windows": "DAQmx is installed"}

    def test_should_tag_a_crio_case_as_using_the_driver(self) -> None:
        case = Cases(CASES).resolve("hitl/crio_driver", name="crio_driver")
        assert case.tags == ["headless", "driver"]

    def test_should_resolve_a_case_through_a_generic_base(self) -> None:
        case = Cases(CASES).resolve("hitl/ethercat_perf", name="ethercat_perf")
        assert case.source == "integration/tests/hitl/ethercat_perf.py#L4"

    def test_should_reject_a_name_matching_no_class(self) -> None:
        with pytest.raises(ValueError, match="no class for driver/ni › ni"):
            Cases(CASES).resolve("driver/ni", name="ni")


def tc(case: str, name: str, status: str, **fields: Any) -> dict[str, Any]:
    return {"case": case, "name": name, "status": status, "duration_s": 1.5, **fields}


class TestCaseRows:
    """Test rows read from a test conductor summary."""

    def test_should_give_a_passed_case_its_suite_source_and_tags(self) -> None:
        summary = {"tests": [tc("driver/ni", "NIRead", "PASSED")]}
        assert case_rows(summary, suite=3, runner_os="windows", cases=Cases(CASES)) == [
            [
                3,
                "driver/ni › NIRead",
                "passed",
                None,
                1.5,
                "integration/tests/driver/ni.py#L6",
                ["headless", "driver", "windows-only"],
            ]
        ]

    def test_should_read_the_skip_reason_and_the_error(self) -> None:
        summary = {
            "tests": [
                tc("driver/ni", "NIRead", "SKIPPED", skip_reason="No DAQmx"),
                tc("driver/ni", "NIMissing", "TIMEOUT", error_message="Timed out"),
            ]
        }
        rows = case_rows(summary, suite=0, runner_os="ubuntu", cases=Cases(CASES))
        assert [r[2:4] for r in rows] == [
            ["skipped", "No DAQmx"],
            ["failed", "Timed out"],
        ]

    def test_should_skip_a_case_an_older_run_passed_on_a_platform_it_skips(
        self,
    ) -> None:
        summary = {"tests": [tc("driver/ni", "NIRead", "PASSED")]}
        [row] = case_rows(summary, suite=0, runner_os="ubuntu", cases=Cases(CASES))
        assert row[2:4] == ["skipped", "Windows DAQmx drivers required"]


HITL_JOB = {
    "conclusion": "success",
    "html_url": "https://github.com/synnaxlabs/synnax/actions/runs/2/job/3",
}


def summary(name: str, *tests: dict[str, Any]) -> dict[str, Any]:
    return {"run": {"name": name}, "tests": list(tests)}


class TestHardwareSuites:
    """Suites and test rows read from the summaries of a hardware run."""

    def test_should_make_a_hardware_suite_for_each_conductor_run(self) -> None:
        suites, rows = hardware_suites(
            [
                summary("tc_hitl", tc("driver/ni", "NIRead", "PASSED")),
                summary("tc_hitl_perf", tc("driver/ni", "NIMissing", "FAILED")),
            ],
            job=HITL_JOB,
            cases=Cases(CASES),
            first=7,
        )
        assert [(s["job"], s["count"]) for s in suites] == [
            ("hitl", 1),
            ("hitl-perf", 1),
        ]
        assert suites[0] == {
            "job": "hitl",
            "product": "driver",
            "lang": "py",
            "kind": "system",
            "tags": ["e2e", "hardware"],
            "os": "windows",
            "conclusion": "success",
            "url": HITL_JOB["html_url"],
            "count": 1,
        }
        assert [(r[0], r[1], r[2]) for r in rows] == [
            (7, "driver/ni › NIRead", "passed"),
            (8, "driver/ni › NIMissing", "failed"),
        ]

    def test_should_keep_the_last_conductor_run_of_a_name(self) -> None:
        suites, rows = hardware_suites(
            [
                summary("tc_hitl", tc("driver/ni", "NIRead", "FAILED")),
                summary("tc_hitl", tc("driver/ni", "NIRead", "PASSED")),
                summary("tc_hitl_perf"),
            ],
            job=HITL_JOB,
            cases=Cases(CASES),
            first=0,
        )
        assert len(suites) == 2
        assert [r[2] for r in rows] == ["passed"]

    def test_should_exit_when_a_conductor_run_has_no_summary(self) -> None:
        with pytest.raises(
            SystemExit, match=r"hardware run has no summary for \['tc_hitl'\]"
        ):
            hardware_suites(
                [summary("tc_hitl_perf")], job=HITL_JOB, cases=Cases(CASES), first=0
            )


class TestShippedChanges:
    """Shipped files that differ between two commits."""

    def test_should_list_every_change_outside_unshipped_folders(
        self, tmp_path: Path
    ) -> None:
        git(tmp_path, "init", "-q", "-b", "main")

        def commit(files: dict[str, str]) -> str:
            for path, text in files.items():
                (tmp_path / path).parent.mkdir(parents=True, exist_ok=True)
                (tmp_path / path).write_text(text)
            git(tmp_path, "add", ".")
            git(
                tmp_path,
                "-c",
                "user.email=ci@synnaxlabs.com",
                "-c",
                "user.name=ci",
                "commit",
                "-q",
                "-m",
                "change",
            )
            return git(tmp_path, "rev-parse", "HEAD")

        files = [
            ".python-version",
            "core/a.go",
            "integration/tests/a.py",
            "scripts/a.py",
            "site/docs/a.md",
            "studio/a.ts",
            "vendor/open62541/a.c",
        ]
        base = commit(dict.fromkeys(files, "a"))
        docs = commit(
            dict.fromkeys(["scripts/a.py", "site/docs/a.md", "studio/a.ts"], "b")
        )
        assert shipped_changes(tmp_path, base=base, head=docs) == []
        code = commit(dict.fromkeys(files, "c"))
        assert shipped_changes(tmp_path, base=base, head=code) == [
            ".python-version",
            "core/a.go",
            "integration/tests/a.py",
            "vendor/open62541/a.c",
        ]


class TestSpacesKey:
    """Data keys read from release versions."""

    def test_should_key_the_data_by_the_release_train(self) -> None:
        assert spaces_key("0.61.0") == "docs/reliability/0.61/run.json"

    def test_should_exit_on_a_version_that_is_not_a_release(self) -> None:
        with pytest.raises(
            SystemExit,
            match=re.escape("--version must be a release such as 0.61.0, not 0.61"),
        ):
            spaces_key("0.61")


class TestRunIdOf:
    """Run IDs read from workflow run URLs."""

    def test_should_read_the_id_of_a_run_or_a_job_url(self) -> None:
        url = "https://github.com/synnaxlabs/synnax/actions/runs/43167272303/job/1"
        assert run_id_of(url) == "43167272303"

    def test_should_exit_on_a_url_that_names_no_run(self) -> None:
        with pytest.raises(SystemExit, match="not a workflow run URL: x"):
            run_id_of("x")


class TestPlatformSkip:
    """Skip reasons read from a platform gate."""

    @pytest.mark.parametrize(
        ("msg", "reason"),
        [
            ('"Windows only"', "Windows only"),
            ("5", "Skipped"),
            ("None", "Skipped"),
        ],
    )
    def test_should_read_a_string_message_and_fall_back_otherwise(
        self, msg: str, reason: str
    ) -> None:
        node = ast.parse(
            f'if platform.system().lower() != "windows":\n    self.skip(reason={msg})\n'
        ).body[0]
        assert isinstance(node, ast.If)
        assert platform_skip(node, system="linux") == reason

    def test_should_read_an_auto_pass_from_an_older_source(self) -> None:
        node = ast.parse(
            'if platform.system().lower() != "windows":\n'
            '    self.auto_pass(msg="Windows only")\n'
        ).body[0]
        assert isinstance(node, ast.If)
        assert platform_skip(node, system="linux") == "Windows only"


class TestOsOf:
    """Runner labels to operating systems."""

    @pytest.mark.parametrize(
        ("runner", "os"),
        [
            ("ubuntu-24.04", "ubuntu"),
            ("windows-build-bot", "windows"),
            ("macos-15", "macos"),
        ],
    )
    def test_should_map_a_runner_to_its_os(self, runner: str, os: str) -> None:
        assert os_of(runner) == os

    def test_should_reject_an_unknown_runner(self) -> None:
        with pytest.raises(ValueError, match="unknown runner: freebsd"):
            os_of("freebsd")


class TestJobName:
    """CI job names to job keys."""

    @pytest.mark.parametrize(
        ("name", "key"),
        [
            ("core / Test (ubuntu-latest)", "core"),
            ("x-go / Test (windows-build-bot)", "x-go"),
            ("scripts / Test", "scripts"),
        ],
    )
    def test_should_read_the_key_with_or_without_a_runner(
        self, name: str, key: str
    ) -> None:
        m = JOB_NAME.match(name)
        assert m is not None
        assert m.group(1) == key

    @pytest.mark.parametrize(
        "name", ["integration / Test (arc) / ubuntu", "oracle / Lint"]
    )
    def test_should_reject_other_jobs(self, name: str) -> None:
        assert JOB_NAME.match(name) is None


class TestIntegrationNames:
    """Integration job and artifact names to their runner, job, and suite."""

    @pytest.mark.parametrize(
        ("name", "os", "job"),
        [
            ("integration / Test (windows) / arc", "windows", "arc"),
            ("integration / Test (ubuntu) / console-2", "ubuntu", "console-2"),
        ],
    )
    def test_should_read_a_job_and_a_shard(self, name: str, os: str, job: str) -> None:
        m = TC_JOB_NAME.match(name)
        assert m is not None
        assert (m.group(1), m.group(2)) == (os, job)

    @pytest.mark.parametrize(
        ("name", "parts"),
        [
            ("test-results-ubuntu-build-bot-arc", ("ubuntu-build-bot", "arc", "arc")),
            (
                "test-results-windows-test-bot-console-2",
                ("windows-test-bot", "console-2", "console"),
            ),
        ],
    )
    def test_should_read_an_artifact_and_its_suite(
        self, name: str, parts: tuple[str, str, str]
    ) -> None:
        m = TC_ARTIFACT.match(name)
        assert m is not None
        assert (m["runner"], m["job"], m["target"]) == parts


class TestCiJobs:
    """CI jobs read from the run of a workflow."""

    def test_should_keep_every_job_of_a_ci_run(self) -> None:
        jobs = [{"name": "core / Test (ubuntu-latest)"}, {"name": "changes"}]
        assert ci_jobs(".github/workflows/ci.yaml", jobs=jobs) == jobs

    def test_should_keep_only_called_ci_jobs_without_their_prefix(self) -> None:
        jobs = [
            {"name": "CI / core / Test (ubuntu-latest)", "conclusion": "success"},
            {"name": "CI / integration / Test (windows) / arc"},
            {"name": "Publish Core"},
        ]
        assert ci_jobs(".github/workflows/release.yaml", jobs=jobs) == [
            {"name": "core / Test (ubuntu-latest)", "conclusion": "success"},
            {"name": "integration / Test (windows) / arc"},
        ]

    def test_should_return_nothing_for_a_workflow_that_does_not_call_ci(
        self,
    ) -> None:
        jobs = [{"name": "Deploy"}]
        assert ci_jobs(".github/workflows/deploy.docs.yaml", jobs=jobs) == []


def job(name: str, *, runner: str, conclusion: str = "success") -> dict[str, Any]:
    return {"name": name, "labels": [runner], "conclusion": conclusion}


class TestSuiteJobs:
    """Unit suites read from CI jobs."""

    def test_should_key_a_suite_by_its_job_and_os(self) -> None:
        core = job("core / Test (macos-latest)", runner="macos-latest")
        assert suite_jobs([core, job("changes", runner="ubuntu-latest")]) == {
            ("core", "macos"): core
        }

    def test_should_report_a_sharded_suite_through_a_shard_that_failed(self) -> None:
        failed = job(
            "console / Test (ubuntu-latest, 2/3)",
            runner="ubuntu-latest",
            conclusion="failure",
        )
        assert suite_jobs(
            [
                job("console / Test (ubuntu-latest, 1/3)", runner="ubuntu-latest"),
                failed,
                job("console / Test (ubuntu-latest, 3/3)", runner="ubuntu-latest"),
            ]
        ) == {("console", "ubuntu"): failed}

    def test_should_leave_out_unlisted_jobs(self) -> None:
        assert (
            suite_jobs([job("www / Test (ubuntu-latest)", runner="ubuntu-latest")])
            == {}
        )

    @pytest.mark.parametrize("key", ["eslint-config", "stylelint-config"])
    def test_should_include_the_lint_rule_suites(self, key: str) -> None:
        lint = job(f"{key} / Test (ubuntu-latest)", runner="ubuntu-latest")
        assert suite_jobs([lint]) == {(key, "ubuntu"): lint}

    def test_should_leave_out_the_thread_sanitizer_rerun_of_driver_tests(self) -> None:
        tsan = job("driver-tsan / Test (ubuntu-build-bot)", runner="ubuntu-build-bot")
        assert suite_jobs([tsan]) == {}

    def test_should_leave_out_the_oldest_dependency_rerun_of_client_tests(
        self,
    ) -> None:
        lowest = job(
            "client-py-lowest / Test (ubuntu-latest, Python oldest)",
            runner="ubuntu-latest",
        )
        assert suite_jobs([lowest]) == {}

    def test_should_reject_a_job_with_no_entry(self) -> None:
        with pytest.raises(
            ValueError, match="no JOBS or UNLISTED entry for new / Test"
        ):
            suite_jobs([job("new / Test", runner="ubuntu-latest")])
