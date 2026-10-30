#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import re
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
    "pull "*) [[ ! -f "$FAKE_DIR/pull_fails" ]] || exit 1 ;;
    "inspect "*) cat "$FAKE_DIR/caddy_mounts" 2> /dev/null || exit 1 ;;
    "ps "*)
        [[ ! -f "$FAKE_DIR/ps_fails" ]] || exit 1
        if [[ $* == *--quiet* ]]; then
            awk '{ print "synnax-pr-" $1 }' "$FAKE_DIR/containers"
        else
            cat "$FAKE_DIR/containers"
        fi
        ;;
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
        self.logs = path / "logs"
        self.logs.mkdir()
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
            "PREVIEW_MAX=2\n"
            "PREVIEW_CPUS=0.25\n"
            "PREVIEW_MEMORY=1g\n"
            f"PREVIEW_LOGS={self.logs}\n"
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

    def viewed(self, pr: int, at: int) -> None:
        log = self.logs / f"pr-{pr}.log"
        log.touch()
        os.utime(log, (at, at))

    def calls(self) -> list[str]:
        return (self.path / "calls").read_text().splitlines()

    def docker(self) -> list[str]:
        return [c.split(" | ")[0] for c in self.calls() if c.startswith("docker ")]

    def mark(self, pr: int) -> Path:
        return self.sites / f"pr-{pr}.human"

    def human(self, pr: int) -> bool:
        path = self.mark(pr)
        return path.exists() and path.read_text().split()[1] == "human"

    def site(self, pr: int) -> Path:
        return self.sites / f"pr-{pr}.caddy"


@pytest.fixture
def host(tmp_path: Path) -> Host:
    return Host(tmp_path)


class TestUp:
    """Tests for starting or replacing the Core for a pull request."""

    def test_should_start_the_core_and_route_to_it(self, host: Host) -> None:
        host.ready()
        result = host.run("up", "12", "registry/synnax-preview:pr-12")
        assert result.returncode == 0, result.stderr
        docker = [c for c in host.docker() if not c.startswith("docker ps")]
        assert docker[0] == "docker pull --quiet registry/synnax-preview:pr-12"
        assert docker[1] == "docker rm --force synnax-pr-12"
        run = docker[2]
        assert run.startswith("docker run --detach --name synnax-pr-12")
        assert "--label synnax.preview=12" in run
        assert "--env SYNNAX_LICENSE_KEY" in run
        assert "--memory 1g --memory-swap 1g --cpus 0.25" in run
        assert run.endswith(
            "registry/synnax-preview:pr-12 --listen 0.0.0.0:9090 --insecure --mem "
            "--no-driver"
        )
        assert docker[3] == "docker image prune --all --force"
        assert docker[-1] == (
            "docker exec caddy caddy reload --config /etc/caddy/Caddyfile"
        )
        assert host.site(12).read_text() == (
            "pr-12.preview.example.com {\n"
            "    log {\n"
            f"        output file {host.logs}/pr-12.log\n"
            "    }\n"
            "    reverse_proxy synnax-pr-12:9090\n"
            "}\n"
        )

    def test_should_pass_the_license_to_the_core(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image")
        run = next(c for c in host.calls() if c.startswith("docker run"))
        assert run.endswith("| license=license-key")
        get = next(c for c in host.calls() if c.startswith("aws "))
        assert "--name /synnax-preview/license-key" in get
        assert "--region us-east-1" in get

    def test_should_keep_the_running_core_when_the_license_fails(
        self, host: Host
    ) -> None:
        (host.path / "aws_fails").touch()
        result = host.run("up", "12", "image")
        assert result.returncode != 0
        assert host.docker() == []

    def test_should_take_the_lock_first(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image")
        assert host.calls()[0] == "flock 9"

    def test_should_fail_when_the_core_never_starts(self, host: Host) -> None:
        result = host.run("up", "12", "image")
        assert result.returncode == 1
        assert "did not start" in result.stderr
        assert "docker logs --tail 100 synnax-pr-12" in host.docker()
        assert not host.site(12).exists()


class TestCap:
    """Tests for the limit on previews the host runs at once."""

    def evicted(self, result: subprocess.CompletedProcess[str]) -> list[str]:
        lines = [ln for ln in result.stdout.splitlines() if ln.startswith("evicted")]
        for ln in lines:
            assert re.fullmatch(r"evicted \d+ \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", ln)
        return [ln.rsplit(" ", 1)[0] for ln in lines]

    def test_should_remove_the_oldest_preview_at_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((13, 200), (12, 100))
        host.site(12).write_text("route")
        result = host.run("up", "14", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == ["evicted 12"]
        assert not host.site(12).exists()
        removed = [c for c in host.docker() if c.startswith("docker rm")]
        assert removed == [
            "docker rm --force synnax-pr-12",
            "docker rm --force synnax-pr-14",
        ]

    def test_should_prune_images_only_after_the_new_core_starts(
        self, host: Host
    ) -> None:
        host.ready()
        host.containers((13, 200), (12, 100))
        result = host.run("up", "14", "image")
        assert result.returncode == 0, result.stderr
        docker = [c for c in host.docker() if not c.startswith("docker ps")]
        prune = docker.index("docker image prune --all --force")
        assert docker[prune - 1].startswith("docker run --detach --name synnax-pr-14")
        assert docker.count("docker image prune --all --force") == 1

    def test_should_remove_unmarked_previews_before_human_reviews(
        self, host: Host
    ) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        host.mark(12).write_text("50 human\n")
        result = host.run("up", "14", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == ["evicted 13"]

    def test_should_keep_a_recently_viewed_preview(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        host.viewed(12, 300)
        host.site(13).write_text("route")
        host.viewed(13, 150)
        result = host.run("up", "14", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == ["evicted 13"]
        assert not (host.logs / "pr-13.log").exists()

    def test_should_mark_a_human_review(self, host: Host) -> None:
        host.ready()
        assert host.run("up", "12", "image", "human", "100").returncode == 0
        assert host.human(12)

    def test_should_keep_a_newer_mark(self, host: Host) -> None:
        host.ready()
        host.mark(12).write_text("200 human\n")
        assert host.run("up", "12", "image", "none", "100").returncode == 0
        assert host.human(12)

    def test_should_remove_nothing_when_the_pull_fails(self, host: Host) -> None:
        (host.path / "pull_fails").touch()
        host.containers((12, 100), (13, 200))
        result = host.run("up", "14", "image")
        assert result.returncode != 0
        assert self.evicted(result) == []
        assert not [c for c in host.docker() if c.startswith("docker rm")]

    def test_should_start_nothing_when_listing_previews_fails(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        (host.path / "ps_fails").touch()
        result = host.run("up", "14", "image")
        assert result.returncode != 0
        assert self.evicted(result) == []
        docker = host.docker()
        assert not [c for c in docker if c.startswith(("docker rm", "docker run"))]

    def test_should_remove_nothing_when_replacing_a_preview(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        result = host.run("up", "12", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == []

    def test_should_remove_nothing_below_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100))
        result = host.run("up", "14", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == []

    def test_should_remove_enough_to_make_room_over_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((12, 300), (13, 100), (14, 200))
        result = host.run("up", "15", "image")
        assert result.returncode == 0, result.stderr
        assert self.evicted(result) == ["evicted 13", "evicted 14"]


class TestMark:
    """Tests for marking a pull request that waits on a human review."""

    def test_should_mark_a_human_review(self, host: Host) -> None:
        assert host.run("mark", "12", "human", "100").returncode == 0
        assert host.human(12)

    def test_should_clear_the_mark_with_a_newer_one(self, host: Host) -> None:
        host.mark(12).write_text("100 human\n")
        assert host.run("mark", "12", "none", "200").returncode == 0
        assert not host.human(12)

    def test_should_ignore_an_older_mark(self, host: Host) -> None:
        host.mark(12).write_text("200 human\n")
        assert host.run("mark", "12", "none", "100").returncode == 0
        assert host.human(12)

    @pytest.mark.parametrize(
        "args", [("12", "bot", "100"), ("12", "human", ""), ("12", "human", "now")]
    )
    def test_should_reject_an_invalid_mark(
        self, host: Host, args: tuple[str, ...]
    ) -> None:
        result = host.run("mark", *args)
        assert result.returncode == 1
        assert "invalid review mark" in result.stderr


class TestUpgrade:
    """Tests for bringing Caddy and the running previews to the settings."""

    def caddy_runs(self, host: Host) -> list[str]:
        return [c for c in host.docker() if c.startswith("docker run")]

    def test_should_start_caddy_with_the_access_log_mount(self, host: Host) -> None:
        assert host.run("upgrade").returncode == 0
        runs = self.caddy_runs(host)
        assert len(runs) == 1
        assert runs[0].startswith("docker run --detach --name caddy")
        assert f"--volume {host.logs}:{host.logs}" in runs[0]

    def test_should_recreate_caddy_without_the_mount(self, host: Host) -> None:
        (host.path / "caddy_mounts").write_text("/etc/caddy /data\n")
        assert host.run("upgrade").returncode == 0
        assert "docker rm --force caddy" in host.docker()
        assert len(self.caddy_runs(host)) == 1

    def test_should_keep_caddy_with_the_mount(self, host: Host) -> None:
        (host.path / "caddy_mounts").write_text(f"/etc/caddy {host.logs} /data\n")
        assert host.run("upgrade").returncode == 0
        assert self.caddy_runs(host) == []
        assert "docker rm --force caddy" not in host.docker()

    def test_should_apply_the_limits_to_running_previews(self, host: Host) -> None:
        (host.path / "caddy_mounts").write_text(f"{host.logs}\n")
        host.containers((12, 100), (13, 200))
        assert host.run("upgrade").returncode == 0
        updates = [c for c in host.docker() if c.startswith("docker update")]
        assert updates == [
            f"docker update --cpus 0.25 --memory 1g --memory-swap 1g synnax-pr-{pr}"
            for pr in (12, 13)
        ]


class TestDown:
    """Tests for removing the Core for a pull request."""

    def test_should_remove_the_core_route_and_mark(self, host: Host) -> None:
        host.site(12).write_text("route")
        host.mark(12).write_text("100 human\n")
        result = host.run("down", "12")
        assert result.returncode == 0, result.stderr
        assert host.docker() == [
            "docker rm --force synnax-pr-12",
            "docker exec caddy caddy reload --config /etc/caddy/Caddyfile",
            "docker image prune --all --force",
        ]
        assert not host.site(12).exists()
        assert not host.mark(12).exists()


class TestSweep:
    """Tests for removing previews past the TTL."""

    def test_should_remove_only_expired_previews(self, host: Host) -> None:
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
    def test_should_reject_an_invalid_pull_request(
        self, host: Host, args: tuple[str, ...]
    ) -> None:
        result = host.run(*args)
        assert result.returncode == 1
        assert "invalid pull request number" in result.stderr
        assert host.docker() == []

    def test_should_reject_an_invalid_review_mark(self, host: Host) -> None:
        result = host.run("up", "12", "image", "bot", "100")
        assert result.returncode == 1
        assert "invalid review mark" in result.stderr
        assert host.docker() == []

    def test_should_reject_an_unknown_command(self, host: Host) -> None:
        result = host.run("restart")
        assert result.returncode == 2
        assert "usage:" in result.stderr
