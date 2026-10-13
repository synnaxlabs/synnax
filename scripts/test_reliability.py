#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

import pytest

from reliability import (
    JOB_NAME,
    TC_JOB_NAME,
    Cases,
    enumerate_tests,
    integration_products,
    integration_sources,
    os_of,
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
    "integration/tests/console/plot.py": "import x\n\n\nclass Plot(TestCase):\n    pass\n",
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
        assert enumerate_tests(root, sha, "go", ["core"]) == [
            ("core/frame_test.go", 2, "reads a frame"),
            ("core/frame_test.go", 3, "wraps its name"),
            ("core/frame_test.go", 8, "one"),
        ]

    def test_should_name_ts_its_and_each_tables(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha, "ts", ["console"]) == [
            ("console/plot.spec.ts", 1, "renders"),
            ("console/plot.spec.ts", 2, "doubles %d"),
        ]

    def test_should_name_cpp_tests_and_fixtures(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha, "cpp", ["driver"]) == [
            ("driver/task_test.cpp", 1, "Task, Starts"),
            ("driver/task_test.cpp", 2, "TaskFixture, Stops"),
        ]

    def test_should_name_sync_and_async_py_tests(self, repo: Repo) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha, "py", ["client/py"]) == [
            ("client/py/test_frame.py", 1, "test_reads"),
            ("client/py/test_frame.py", 5, "test_streams"),
        ]

    def test_should_return_nothing_for_a_directory_without_tests(
        self, repo: Repo
    ) -> None:
        root, sha = repo
        assert enumerate_tests(root, sha, "go", ["console"]) == []


class TestIntegrationSources:
    """Integration sources read at a commit."""

    def test_should_key_python_files_by_their_path_in_integration(
        self, repo: Repo
    ) -> None:
        root, sha = repo
        assert integration_sources(root, sha) == {
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
        "        self.client.tasks.configure(self.tsk)\n"
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
        '            self.auto_pass(msg="Windows DAQmx drivers required")\n\n\n'
        "class NIMissing(TaskCase):\n"
        "    def setup(self):\n"
        '        if platform.system().lower() == "windows":\n'
        '            self.auto_pass(msg="DAQmx is installed")\n'
    ),
}


class TestCases:
    """Integration cases resolved to their classes and tagged."""

    def test_should_resolve_a_module_named_case_to_its_one_test_class(self) -> None:
        case = Cases(CASES).resolve("console/plot", "plot")
        assert case.source == "integration/tests/console/plot.py#L8"
        assert case.tags == ["playwright", "no-driver"]
        assert case.skipped == {}

    def test_should_tag_a_simulator_case_as_using_the_driver(self) -> None:
        case = Cases(CASES).resolve("console/modbus", "modbus")
        assert case.tags == ["playwright", "driver"]

    def test_should_skip_a_windows_only_case_on_other_platforms(self) -> None:
        case = Cases(CASES).resolve("driver/ni", "NIRead")
        assert case.source == "integration/tests/driver/ni.py#L6"
        assert case.tags == ["headless", "driver", "windows-only"]
        assert case.skipped == {
            "ubuntu": "Windows DAQmx drivers required",
            "macos": "Windows DAQmx drivers required",
        }

    def test_should_skip_a_not_windows_case_on_windows(self) -> None:
        case = Cases(CASES).resolve("driver/ni", "NIMissing")
        assert case.tags == ["headless", "driver", "not-windows"]
        assert case.skipped == {"windows": "DAQmx is installed"}

    def test_should_reject_a_name_matching_no_class(self) -> None:
        with pytest.raises(ValueError, match="no class for driver/ni › ni"):
            Cases(CASES).resolve("driver/ni", "ni")


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


class TestTcJobName:
    """Tests for matching integration test jobs."""

    @pytest.mark.parametrize(
        ("name", "runner", "job"),
        [
            ("integration / Test (windows) / console", "windows", "console"),
            ("integration / Test (windows) / arc + console", "windows", "arc"),
            (
                "integration / Test (ubuntu) / driver + arc + console",
                "ubuntu",
                "driver",
            ),
        ],
    )
    def test_should_match_the_runner_and_job(
        self, name: str, runner: str, job: str
    ) -> None:
        m = TC_JOB_NAME.match(name)
        assert m is not None
        assert m.groups() == (runner, job)

    def test_should_reject_other_jobs(self) -> None:
        assert (
            TC_JOB_NAME.match("integration / build / Build (windows-build-bot)") is None
        )


class TestIntegrationProducts:
    """Tests for reading the product of each integration test file."""

    def test_should_read_products_from_jobs(self) -> None:
        sources = {"jobs.py": 'X = 1\nPRODUCTS = {"control": "arc"}\n'}
        assert integration_products(sources) == {"control": "arc"}

    def test_should_raise_without_products(self) -> None:
        with pytest.raises(ValueError, match="integration/jobs.py has no PRODUCTS"):
            integration_products({"jobs.py": "X = 1\n"})
