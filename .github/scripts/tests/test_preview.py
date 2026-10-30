#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
import time
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[3] / "deploy" / "preview" / "preview.sh"

# Logs each call with the license it sees. A Core answers its readiness check only once
# the file "ready" exists, and ps lists the file "containers".
FAKE_DOCKER = """#!/usr/bin/env bash
echo "docker $* | license=${SYNNAX_LICENSE_KEY:-}" >> "$FAKE_DIR/calls"
case "$1 $2" in
    "exec synnax-"*) [[ -f "$FAKE_DIR/ready" ]] || exit 7 ;;
    "ps "*) cat "$FAKE_DIR/containers" ;;
esac
"""

FAKE_AWS = """#!/usr/bin/env bash
echo "aws $*" >> "$FAKE_DIR/calls"
[[ -f "$FAKE_DIR/aws_fails" ]] && exit 254
echo license-key
"""

FAKE_FLOCK = """#!/usr/bin/env bash
echo "flock $*" >> "$FAKE_DIR/calls"
"""


class Host:
    """A fake preview host with Docker, the AWS CLI, and flock on PATH."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.sites = path / "sites"
        self.sites.mkdir()
        bin_dir = path / "bin"
        bin_dir.mkdir()
        for name, body in (
            ("docker", FAKE_DOCKER),
            ("aws", FAKE_AWS),
            ("flock", FAKE_FLOCK),
            ("sleep", "#!/bin/sh\n"),
        ):
            fake = bin_dir / name
            fake.write_text(body)
            fake.chmod(0o755)
        (path / "calls").touch()
        (path / "containers").touch()
        env_file = path / "preview.env"
        env_file.write_text(
            "PREVIEW_DOMAIN=preview.example.com\n"
            f"PREVIEW_SITES={self.sites}\n"
            "PREVIEW_TTL_DAYS=7\n"
            "PREVIEW_LICENSE_PARAMETER=/synnax-preview/license-key\n"
            "PREVIEW_REGION=us-east-1\n"
        )
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
            "FAKE_DIR": str(path),
            "PREVIEW_ENV": str(env_file),
        }

    def run(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), *args], env=self.env, capture_output=True, text=True
        )

    def ready(self) -> None:
        (self.path / "ready").touch()

    def containers(self, *rows: tuple[int, int]) -> None:
        text = "".join(f"{pr} {deployed}\n" for pr, deployed in rows)
        (self.path / "containers").write_text(text)

    def calls(self) -> list[str]:
        return (self.path / "calls").read_text().splitlines()

    def docker(self) -> list[str]:
        return [c.split(" | ")[0] for c in self.calls() if c.startswith("docker ")]

    def site(self, pr: int) -> Path:
        return self.sites / f"pr-{pr}.caddy"


@pytest.fixture
def host(tmp_path: Path) -> Host:
    return Host(tmp_path)


class TestUp:
    """Tests for starting or replacing the Core for a pull request."""

    def test_starts_core_and_routes_it(self, host: Host) -> None:
        host.ready()
        result = host.run("up", "12", "registry/synnax-preview:pr-12")
        assert result.returncode == 0, result.stderr
        docker = host.docker()
        assert docker[0] == "docker pull --quiet registry/synnax-preview:pr-12"
        assert docker[1] == "docker rm --force synnax-pr-12"
        run = docker[2]
        assert run.startswith("docker run --detach --name synnax-pr-12")
        assert "--label synnax.preview=12" in run
        assert "--env SYNNAX_LICENSE_KEY" in run
        assert run.endswith(
            "registry/synnax-preview:pr-12 --listen 0.0.0.0:9090 --insecure --mem "
            "--no-driver"
        )
        assert docker[3] == "docker image prune --all --force"
        assert docker[-1] == (
            "docker exec caddy caddy reload --config /etc/caddy/Caddyfile"
        )
        assert host.site(12).read_text() == (
            "pr-12.preview.example.com {\n\treverse_proxy synnax-pr-12:9090\n}\n"
        )

    def test_passes_license_to_core(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image")
        run = next(c for c in host.calls() if c.startswith("docker run"))
        assert run.endswith("| license=license-key")
        get = next(c for c in host.calls() if c.startswith("aws "))
        assert "--name /synnax-preview/license-key" in get
        assert "--region us-east-1" in get

    def test_license_failure_keeps_running_core(self, host: Host) -> None:
        (host.path / "aws_fails").touch()
        result = host.run("up", "12", "image")
        assert result.returncode != 0
        assert host.docker() == []

    def test_takes_lock_first(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image")
        assert host.calls()[0] == "flock 9"

    def test_core_never_ready(self, host: Host) -> None:
        result = host.run("up", "12", "image")
        assert result.returncode == 1
        assert "did not start" in result.stderr
        assert "docker logs --tail 100 synnax-pr-12" in host.docker()
        assert not host.site(12).exists()


class TestDown:
    """Tests for removing the Core for a pull request."""

    def test_removes_core_and_route(self, host: Host) -> None:
        host.site(12).write_text("route")
        result = host.run("down", "12")
        assert result.returncode == 0, result.stderr
        assert host.docker() == [
            "docker rm --force synnax-pr-12",
            "docker exec caddy caddy reload --config /etc/caddy/Caddyfile",
            "docker image prune --all --force",
        ]
        assert not host.site(12).exists()


class TestSweep:
    """Tests for removing previews past the TTL."""

    def test_removes_only_expired(self, host: Host) -> None:
        now = int(time.time())
        host.containers((12, now - 8 * 86400), (13, now - 86400))
        host.site(12).write_text("route")
        host.site(13).write_text("route")
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        removed = [c for c in host.docker() if c.startswith("docker rm")]
        assert removed == ["docker rm --force synnax-pr-12"]
        assert not host.site(12).exists()
        assert host.site(13).exists()

    def test_prunes_images_without_expired(self, host: Host) -> None:
        host.containers((13, int(time.time())))
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert host.docker()[-1] == "docker image prune --all --force"


class TestArguments:
    """Tests for rejected arguments."""

    @pytest.mark.parametrize(
        "args", [("up", "abc", "image"), ("down", "12;rm"), ("down", "")]
    )
    def test_invalid_pr(self, host: Host, args: tuple[str, ...]) -> None:
        result = host.run(*args)
        assert result.returncode == 1
        assert "invalid pull request number" in result.stderr
        assert host.docker() == []

    def test_unknown_command(self, host: Host) -> None:
        result = host.run("restart")
        assert result.returncode == 2
        assert "usage:" in result.stderr
