#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import hashlib
import os
import subprocess
import sys
from collections.abc import Iterator
from pathlib import Path

import pytest
from scripts_common import FakeBlob, FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "publish_qa_report.py"
SCRIPTS = Path(__file__).resolve().parents[3] / "scripts"

REPORT = """# Console QA: 1 finding

## High: plot legend overlaps the axis

![legend](images/legend.png)
"""

DIGEST = hashlib.sha256(b"png").hexdigest()


@pytest.fixture
def gh(tmp_path: Path) -> FakeGH:
    gh = FakeGH(tmp_path)
    gh.serve("issue create", "https://github.com/synnaxlabs/synnax/issues/1")
    return gh


@pytest.fixture
def blob() -> Iterator[FakeBlob]:
    blob = FakeBlob()
    yield blob
    blob.close()


@pytest.fixture
def env(gh: FakeGH, blob: FakeBlob) -> dict[str, str]:
    base = {k: v for k, v in os.environ.items() if not k.startswith("GITHUB_")}
    return {
        **base,
        "PATH": f"{gh.bin_dir}{os.pathsep}{base['PATH']}",
        "PYTHONPATH": str(SCRIPTS),
        "BLOB_READ_WRITE_TOKEN": "token",
        "VERCEL_BLOB_API_URL": blob.url,
        "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        "GITHUB_RUN_ID": "42",
    }


@pytest.fixture
def report(tmp_path: Path) -> Path:
    qa = tmp_path / "qa"
    (qa / "images").mkdir(parents=True)
    (qa / "report.md").write_text(REPORT)
    (qa / "images" / "legend.png").write_bytes(b"png")
    return qa


def publish(
    env: dict[str, str], qa: Path, *args: str
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), str(qa), *args],
        cwd=qa.parent,
        env=env,
        capture_output=True,
        text=True,
    )


class TestPublish:
    def test_should_open_an_issue_with_hosted_images(
        self, env: dict[str, str], report: Path, gh: FakeGH, blob: FakeBlob
    ) -> None:
        done = publish(env, report)
        assert done.returncode == 0, done.stderr
        assert done.stdout.strip() == "https://github.com/synnaxlabs/synnax/issues/1"
        call = gh.calls()[-1]
        args = " ".join(call)
        assert "--title Console QA: 1 finding" in args
        assert "--label console-qa" in args
        body = Path(call[call.index("--body-file") + 1]).read_text()
        assert f"](https://blob.test/console-qa/{DIGEST}.png)" in body
        assert "actions/runs/42" in body

    def test_should_store_each_image_as_a_public_blob(
        self, env: dict[str, str], report: Path, blob: FakeBlob
    ) -> None:
        assert publish(env, report).returncode == 0
        [put] = blob.puts
        assert put.pathname == f"console-qa/{DIGEST}.png"
        assert put.body == b"png"
        assert put.headers["authorization"] == "Bearer token"
        assert put.headers["x-vercel-blob-access"] == "public"
        assert put.headers["x-content-type"] == "image/png"
        assert put.headers["x-allow-overwrite"] == "1"

    def test_should_fail_when_the_blob_api_rejects_an_upload(
        self, env: dict[str, str], report: Path, gh: FakeGH, blob: FakeBlob
    ) -> None:
        blob.failures = [403]
        done = publish(env, report)
        assert done.returncode != 0
        assert "403" in done.stderr
        assert gh.calls() == []

    def test_should_fail_without_a_blob_token(
        self, env: dict[str, str], report: Path, gh: FakeGH, blob: FakeBlob
    ) -> None:
        del env["BLOB_READ_WRITE_TOKEN"]
        done = publish(env, report)
        assert done.returncode != 0
        assert "BLOB_READ_WRITE_TOKEN" in done.stderr
        assert blob.puts == []
        assert gh.calls() == []

    def test_should_write_the_issue_to_a_file_in_a_dry_run(
        self, env: dict[str, str], report: Path, gh: FakeGH, blob: FakeBlob
    ) -> None:
        done = publish(env, report, "--dry-run")
        assert done.returncode == 0, done.stderr
        issue = (report / "issue.md").read_text()
        assert issue.startswith("# Console QA: 1 finding\n")
        assert "](images/legend.png)" in issue
        assert gh.calls() == []
        assert blob.puts == []

    def test_should_reject_a_report_without_a_title(
        self, env: dict[str, str], report: Path
    ) -> None:
        (report / "report.md").write_text("No title\n")
        done = publish(env, report, "--dry-run")
        assert done.returncode != 0
        assert "# <title>" in done.stderr

    def test_should_reject_a_link_to_a_missing_image(
        self, env: dict[str, str], report: Path
    ) -> None:
        (report / "images" / "legend.png").unlink()
        done = publish(env, report, "--dry-run")
        assert done.returncode != 0
        assert "images/legend.png" in done.stderr
