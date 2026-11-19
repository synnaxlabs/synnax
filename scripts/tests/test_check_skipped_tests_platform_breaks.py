#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Mapping
from pathlib import Path

from conftest import GitRepo

from check_skipped_tests import platforms

TS = 'it("a", () => {});\n'
PY = "def test_a() -> None:\n    pass\n"
GO = 'package a\n\nimport "testing"\n\nfunc TestA(t *testing.T) {}\n'
EXCLUDES = "export default { test: { exclude: [...platformExcludes()] } };\n"
# The line that the check tells a Python package to add.
PY_IGNORES = (
    "import sys\n\n"
    'collect_ignore_glob = [f"*_{p}.py" for p in {"darwin", "linux", "win32"} - '
    '{sys.platform}] + (["*_posix.py"] if sys.platform == "win32" else [])\n'
)
CI_YAML = (
    Path(__file__).resolve().parents[2] / ".github/workflows/ci.yaml"
).read_text()


def _ci(workflow: str, directory: str, **inputs: object) -> str:
    lines = [f"      {k}: {str(v).lower()}" for k, v in inputs.items()]
    return "\n".join(
        [
            "jobs:",
            "  pkg:",
            f"    uses: ./.github/workflows/{workflow}",
            "    with:",
            f"      directory: {directory}",
            *lines,
            "",
        ]
    )


def _paths(repo: GitRepo, ci: str, files: Mapping[str, str]) -> list[str]:
    repo.commit({".github/workflows/ci.yaml": ci, **files})
    return [f.path for f in platforms(root=repo.path)]


class TestPlatformBreaks:
    def test_nearer_conftest_glob_shadows_the_platform_glob(
        self, git_repo: GitRepo
    ) -> None:
        # pytest reads collect_ignore_glob from the nearest conftest that defines it,
        # so x/conftest.py hides the platform glob and Linux collects the darwin file.
        files = {
            "lib/tests/conftest.py": PY_IGNORES,
            "lib/tests/x/conftest.py": 'collect_ignore_glob = ["*_fixture.py"]\n',
            "lib/tests/x/test_a_darwin.py": PY,
        }
        ci = _ci("test.py.yaml", "lib", macos=True)
        assert _paths(git_repo, ci, files) == ["lib/tests/x/test_a_darwin.py"]

    def test_suggested_glob_drops_a_file_the_check_does_not_see(
        self, git_repo: GitRepo
    ) -> None:
        # `*_darwin.py` matches test_darwin.py, so Linux skips it, and no job runs
        # macOS: the file runs nowhere.
        files = {
            "lib/tests/conftest.py": PY_IGNORES,
            "lib/tests/test_a_linux.py": PY,
            "lib/tests/test_darwin.py": PY,
        }
        ci = _ci("test.py.yaml", "lib")
        assert _paths(git_repo, ci, files) == ["lib/tests/test_darwin.py"]

    def test_vitest_reads_vitest_config_before_vite_config(
        self, git_repo: GitRepo
    ) -> None:
        # Vitest loads vitest.config.ts when both exist, so the excludes in
        # vite.config.ts do not apply and every platform runs the darwin file.
        files = {
            "web/vite.config.ts": EXCLUDES,
            "web/vitest.config.ts": "export default { test: {} };\n",
            "web/src/a.darwin.spec.ts": TS,
        }
        ci = _ci("test.ts.yaml", "web", macos=True)
        assert _paths(git_repo, ci, files) == ["web/src/a.darwin.spec.ts"]

    def test_client_ts_macos_runs_no_test_ts_file(self, git_repo: GitRepo) -> None:
        # With a core_image, test.ts.yaml gives macOS and Windows only the *.spec.ts
        # files, so the real CI runs this darwin file on no platform.
        files = {
            "client/ts/vite.config.ts": EXCLUDES,
            "client/ts/src/a.darwin.test.ts": TS,
        }
        assert _paths(git_repo, CI_YAML, files) == ["client/ts/src/a.darwin.test.ts"]

    def test_ts_job_with_ubuntu_false_still_tests_on_ubuntu(
        self, git_repo: GitRepo
    ) -> None:
        # The matrix include of test.ts.yaml adds the ubuntu-latest lint entry back
        # after the exclude, and that entry runs the Test step.
        files = {"web/vite.config.ts": EXCLUDES, "web/src/a.linux.spec.ts": TS}
        ci = _ci("test.ts.yaml", "web", ubuntu=False, macos=True)
        assert _paths(git_repo, ci, files) == []

    def test_conftest_that_excludes_only_darwin_passes_a_win32_file(
        self, git_repo: GitRepo
    ) -> None:
        conftest = (
            "import sys\n\n"
            'collect_ignore_glob = [] if sys.platform == "darwin" else ["*_darwin.py"]\n'
        )
        files = {
            "lib/tests/conftest.py": conftest,
            "lib/tests/test_a_darwin.py": PY,
            "lib/tests/test_b_win32.py": PY,
        }
        ci = _ci("test.py.yaml", "lib", macos=True, windows=True)
        assert _paths(git_repo, ci, files) == ["lib/tests/test_b_win32.py"]

    def test_go_platform_file_that_no_ci_platform_runs(self, git_repo: GitRepo) -> None:
        # No Go job of ci.yaml runs FreeBSD, so the test runs nowhere.
        files = {"x/go/a_freebsd_test.go": GO}
        assert _paths(git_repo, CI_YAML, files) == ["x/go/a_freebsd_test.go"]

    def test_job_that_runs_no_tests(self, git_repo: GitRepo) -> None:
        files = {"web/vite.config.ts": EXCLUDES, "web/src/a.darwin.spec.ts": TS}
        ci = _ci("test.ts.yaml", "web", macos=True, test=False)
        assert _paths(git_repo, ci, files) == ["web/src/a.darwin.spec.ts"]
