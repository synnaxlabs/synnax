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
from reliability import JOB_NAME, enumerate_tests, os_of, tc_source

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


class TestTcSource:
    """Source links for integration cases."""

    def test_should_link_the_class_line(self, repo: Repo) -> None:
        root, sha = repo
        case = {"case": "console/plot", "name": "Plot"}
        assert tc_source(root, sha, case) == "integration/tests/console/plot.py#L4"

    def test_should_link_the_file_when_the_class_is_missing(self, repo: Repo) -> None:
        root, sha = repo
        case = {"case": "console/plot", "name": "Missing"}
        assert tc_source(root, sha, case) == "integration/tests/console/plot.py"


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
