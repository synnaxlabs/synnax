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

# A notice time, as the host writes it.
AT = "2026-11-22T10:00:00Z"

# Logs each call with the license it sees. A Core answers its readiness check only once
# the file "ready" exists, and ps lists the file "containers". lego writes a new
# certificate when the file "renews" exists.
FAKE_DOCKER = """#!/usr/bin/env bash
echo "docker $* | license=${SYNNAX_LICENSE_KEY:-}" >> "$FAKE_DIR/calls"
case "$1 $2" in
    "exec synnax-"*) [[ -f "$FAKE_DIR/ready" ]] || exit 7 ;;
    "pull "*) [[ ! -f "$FAKE_DIR/pull_fails" ]] || exit 1 ;;
    "inspect "*) cat "$FAKE_DIR/caddy_mounts" 2> /dev/null || exit 1 ;;
    "run --rm")
        if [[ $* != *goacme/lego* ]]; then
            [[ ! -f "$FAKE_DIR/sample_fails" ]] || exit 1
            exit 0
        fi
        [[ ! -f "$FAKE_DIR/lego_fails" ]] || exit 1
        if [[ -f "$FAKE_DIR/renews" ]]; then
            mkdir -p "$FAKE_DIR/acme/certificates"
            date +%s%N > "$FAKE_DIR/acme/certificates/_.preview.example.com.crt"
        fi
        ;;
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
            "PREVIEW_ZONE_ID=Z123\n"
            "PREVIEW_LOGIN_PARAMETERS=/synnax-preview/oauth2-proxy\n"
            "PREVIEW_GITHUB_ORG=synnaxlabs\n"
            f"PREVIEW_ACME={path / 'acme'}\n"
            f"PREVIEW_SITES={self.sites}\n"
            "PREVIEW_TTL_DAYS=7\n"
            "PREVIEW_LICENSE_PARAMETER=/synnax-preview/license-key\n"
            "PREVIEW_REGION=us-east-1\n"
            "PREVIEW_MAX=2\n"
            "PREVIEW_CPUS=0.25\n"
            "PREVIEW_MEMORY=1g\n"
            "PREVIEW_GO_MEMORY=800MiB\n"
            f"PREVIEW_LOGS={self.logs}\n"
        )
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
            "FAKE_DIR": str(path),
            "PREVIEW_ENV": str(env_file),
        }

    def notices(self) -> list[str]:
        path = self.sites / "notices"
        lines = path.read_text().splitlines() if path.exists() else []
        for ln in lines:
            assert re.fullmatch(
                r"\d+ \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ (evicted \d+|expired)", ln
            )
        # The time varies, so it is left out.
        return [" ".join([ln.split()[0], *ln.split()[2:]]) for ln in lines]

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
        result = host.run(
            "up", "12", "registry/synnax-preview:pr-12", "registry/sample:pr-12"
        )
        assert result.returncode == 0, result.stderr
        docker = [c for c in host.docker() if not c.startswith("docker ps")]
        assert docker[0] == "docker pull --quiet registry/synnax-preview:pr-12"
        assert docker[1] == "docker rm --force synnax-pr-12"
        run = docker[2]
        assert run.startswith("docker run --detach --name synnax-pr-12")
        assert "--label synnax.preview=12" in run
        assert "--env SYNNAX_LICENSE_KEY" in run
        assert "--memory 1g --memory-swap 1g --cpus 0.25" in run
        assert "--env GOMEMLIMIT=800MiB" in run
        assert run.endswith(
            "registry/synnax-preview:pr-12 --listen 0.0.0.0:9090 --insecure --mem "
            "--no-driver"
        )
        assert docker[3] == "docker image prune --all --force"
        assert docker[-2:] == [
            "docker exec caddy caddy reload --config /etc/caddy/Caddyfile",
            "docker run --rm --network preview --memory 1g --cpus 0.25 "
            "registry/sample:pr-12 http://synnax-pr-12:9090",
        ]
        cert = host.path / "acme" / "certificates" / "_.preview.example.com"
        assert host.site(12).read_text() == (
            "pr-12.preview.example.com {\n"
            f"    tls {cert}.crt {cert}.key\n"
            "    log {\n"
            f"        output file {host.logs}/pr-12.log\n"
            "    }\n"
            "    forward_auth login:4180 {\n"
            "        uri /oauth2/auth\n"
            "        @signed_out status 401\n"
            "        handle_response @signed_out {\n"
            "            redir * https://auth.preview.example.com/oauth2/start"
            "?rd={scheme}://{host}{uri}\n"
            "        }\n"
            "    }\n"
            "    reverse_proxy synnax-pr-12:9090 {\n"
            "        stream_close_delay 1h\n"
            "    }\n"
            "}\n"
        )

    def test_should_pass_the_license_to_the_core(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image", "sample")
        run = next(c for c in host.calls() if c.startswith("docker run"))
        assert run.endswith("| license=license-key")
        get = next(c for c in host.calls() if c.startswith("aws "))
        assert "--name /synnax-preview/license-key" in get
        assert "--region us-east-1" in get

    def test_should_keep_the_running_core_when_the_license_fails(
        self, host: Host
    ) -> None:
        (host.path / "aws_fails").touch()
        result = host.run("up", "12", "image", "sample")
        assert result.returncode != 0
        assert host.docker() == []

    def test_should_take_the_lock_first(self, host: Host) -> None:
        host.ready()
        host.run("up", "12", "image", "sample")
        assert host.calls()[0] == "flock 9"

    def test_should_release_the_lock_before_the_sample(self, host: Host) -> None:
        host.ready()
        assert host.run("up", "12", "image", "sample").returncode == 0
        calls = [c.split(" | ")[0] for c in host.calls()]
        sample = next(
            i
            for i, c in enumerate(calls)
            if c.endswith("sample http://synnax-pr-12:9090")
        )
        assert calls[sample - 1] == "flock --unlock 9"

    def test_should_fail_when_the_sample_fails(self, host: Host) -> None:
        host.ready()
        (host.path / "sample_fails").touch()
        result = host.run("up", "12", "image", "sample")
        assert result.returncode != 0
        assert host.site(12).exists()

    def test_should_fail_when_the_core_never_starts(self, host: Host) -> None:
        result = host.run("up", "12", "image", "sample")
        assert result.returncode == 1
        assert "did not start" in result.stderr
        assert "docker logs --tail 100 synnax-pr-12" in host.docker()
        assert not host.site(12).exists()


class TestCap:
    """Tests for the limit on previews the host runs at once."""

    def evicted(self, host: Host) -> list[str]:
        return [f"evicted {n.split()[0]}" for n in host.notices() if " evicted " in n]

    def test_should_remove_the_oldest_preview_at_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((13, 200), (12, 100))
        host.site(12).write_text("route")
        result = host.run("up", "14", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == ["evicted 12"]
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
        result = host.run("up", "14", "image", "sample")
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
        result = host.run("up", "14", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == ["evicted 13"]

    def test_should_keep_a_recently_viewed_preview(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        host.viewed(12, 300)
        host.site(13).write_text("route")
        host.viewed(13, 150)
        result = host.run("up", "14", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == ["evicted 13"]
        assert not (host.logs / "pr-13.log").exists()

    def test_should_mark_a_human_review(self, host: Host) -> None:
        host.ready()
        assert host.run("up", "12", "image", "sample", "human", "100").returncode == 0
        assert host.human(12)

    def test_should_keep_a_newer_mark(self, host: Host) -> None:
        host.ready()
        host.mark(12).write_text("200 human\n")
        assert host.run("up", "12", "image", "sample", "none", "100").returncode == 0
        assert host.human(12)

    def test_should_remove_nothing_when_the_pull_fails(self, host: Host) -> None:
        (host.path / "pull_fails").touch()
        host.containers((12, 100), (13, 200))
        result = host.run("up", "14", "image", "sample")
        assert result.returncode != 0
        assert self.evicted(host) == []
        assert not [c for c in host.docker() if c.startswith("docker rm")]

    def test_should_start_nothing_when_listing_previews_fails(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        (host.path / "ps_fails").touch()
        result = host.run("up", "14", "image", "sample")
        assert result.returncode != 0
        assert self.evicted(host) == []
        docker = host.docker()
        assert not [c for c in docker if c.startswith(("docker rm", "docker run"))]

    def test_should_remove_nothing_when_replacing_a_preview(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100), (13, 200))
        result = host.run("up", "12", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == []

    def test_should_remove_nothing_below_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((12, 100))
        result = host.run("up", "14", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == []

    def test_should_remove_enough_to_make_room_over_the_cap(self, host: Host) -> None:
        host.ready()
        host.containers((12, 300), (13, 100), (14, 200))
        result = host.run("up", "15", "image", "sample")
        assert result.returncode == 0, result.stderr
        assert self.evicted(host) == ["evicted 13", "evicted 14"]


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


LEGO = (
    "goacme/lego:v5.7.0"
    "@sha256:9fe6151508c0e1ddc51a6d3e474353520f4b008782bf0c6d31676da72e6d4766"
)


def lego_runs(host: Host) -> list[str]:
    return [c for c in host.docker() if "goacme/lego" in c]


OAUTH2_PROXY = (
    "quay.io/oauth2-proxy/oauth2-proxy:v7.15.6"
    "@sha256:c3415e8660af553f5e18f5d90db050614c02493c0ac6260ac59442f6d9b12048"
)

FORCED_RELOAD = "docker exec caddy caddy reload --config /etc/caddy/Caddyfile --force"


class TestUpgrade:
    """Tests for bringing Caddy and the running previews to the settings."""

    def caddy_runs(self, host: Host) -> list[str]:
        return [
            c for c in host.docker() if c.startswith("docker run --detach --name caddy")
        ]

    def test_should_start_the_login_with_the_secrets_in_its_env(
        self, host: Host
    ) -> None:
        assert host.run("upgrade").returncode == 0
        gets = [c for c in host.calls() if c.startswith("aws ssm get-parameter")]
        assert [g.split("--name ")[1].split()[0] for g in gets] == [
            f"/synnax-preview/oauth2-proxy/{name}"
            for name in ("client-id", "client-secret", "cookie-secret")
        ]
        docker = host.docker()
        run = next(
            c for c in docker if c.startswith("docker run --detach --name login")
        )
        pull = docker.index(f"docker pull --quiet {OAUTH2_PROXY}")
        assert pull < docker.index("docker rm --force login") < docker.index(run)
        for name in ("CLIENT_ID", "CLIENT_SECRET", "COOKIE_SECRET"):
            assert f"--env OAUTH2_PROXY_{name} " in run
        assert "license-key" not in run
        for flag in (
            "--provider=github",
            "--github-org=synnaxlabs",
            "--redirect-url=https://auth.preview.example.com/oauth2/callback",
            "--cookie-domain=.preview.example.com",
            "--whitelist-domain=.preview.example.com",
        ):
            assert f" {flag} " in f"{run} "

    def test_should_route_the_auth_site_to_the_login(self, host: Host) -> None:
        assert host.run("upgrade").returncode == 0
        cert = host.path / "acme" / "certificates" / "_.preview.example.com"
        assert (host.sites / "auth.caddy").read_text() == (
            "auth.preview.example.com {\n"
            f"    tls {cert}.crt {cert}.key\n"
            "    reverse_proxy login:4180\n"
            "}\n"
        )

    def test_should_keep_the_login_when_its_pull_fails(self, host: Host) -> None:
        (host.path / "pull_fails").touch()
        assert host.run("upgrade").returncode != 0
        assert "docker rm --force login" not in host.docker()

    def test_should_put_running_previews_behind_the_login(self, host: Host) -> None:
        host.containers((12, 100), (13, 200))
        host.site(12).write_text("route")
        assert host.run("upgrade").returncode == 0
        for pr in (12, 13):
            assert "forward_auth login:4180" in host.site(pr).read_text()
            assert f"reverse_proxy synnax-pr-{pr}:9090" in host.site(pr).read_text()

    def test_should_keep_caddy_when_a_login_secret_fails(self, host: Host) -> None:
        (host.path / "aws_fails").touch()
        assert host.run("upgrade").returncode != 0
        assert self.caddy_runs(host) == []
        assert "docker rm --force login" not in host.docker()

    def test_should_get_the_wildcard_certificate_first(self, host: Host) -> None:
        assert host.run("upgrade").returncode == 0
        assert host.docker()[0] == (
            f"docker run --rm --volume {host.path}/acme:/acme"
            " --env AWS_REGION=us-east-1 --env AWS_HOSTED_ZONE_ID=Z123"
            f" {LEGO} run --accept-tos --no-random-sleep --dns route53"
            " --dns.resolvers 169.254.169.253:53 --dns.propagation.disable-ans"
            " --domains *.preview.example.com --path /acme"
        )

    def test_should_change_nothing_when_the_certificate_fails(self, host: Host) -> None:
        (host.path / "lego_fails").touch()
        assert host.run("upgrade").returncode != 0
        assert self.caddy_runs(host) == []

    def test_should_reload_a_running_caddy(self, host: Host) -> None:
        (host.path / "caddy_mounts").write_text(f"/etc/caddy {host.logs} /data\n")
        assert host.run("upgrade").returncode == 0
        assert FORCED_RELOAD in host.docker()

    def test_should_start_caddy_with_the_access_log_mount(self, host: Host) -> None:
        assert host.run("upgrade").returncode == 0
        runs = self.caddy_runs(host)
        assert len(runs) == 1
        assert runs[0].startswith("docker run --detach --name caddy")
        assert f"--volume {host.logs}:{host.logs}" in runs[0]
        assert "--publish 443:443" in runs[0]
        assert "80:80" not in runs[0]

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

    def test_should_apply_the_cpu_limit_to_running_previews(self, host: Host) -> None:
        (host.path / "caddy_mounts").write_text(f"{host.logs}\n")
        host.containers((12, 100), (13, 200))
        assert host.run("upgrade").returncode == 0
        updates = [c for c in host.docker() if c.startswith("docker update")]
        assert updates == [
            f"docker update --cpus 0.25 synnax-pr-{pr}" for pr in (12, 13)
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
        host.mark(12).write_text("0 none\n")
        host.mark(13).write_text("0 none\n")
        host.site(12).write_text("route")
        host.site(13).write_text("route")
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        removed = [c for c in host.docker() if c.startswith("docker rm")]
        assert removed == ["docker rm --force synnax-pr-12"]
        assert not host.site(12).exists()
        assert host.site(13).exists()

    def test_should_keep_an_old_preview_viewed_within_the_ttl(self, host: Host) -> None:
        now = int(time.time())
        host.containers((12, now - 8 * 86400))
        host.mark(12).write_text("0 none\n")
        host.viewed(12, now - 3600)
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert not [c for c in host.docker() if c.startswith("docker rm")]

    def test_should_keep_an_expired_preview_marked_human(self, host: Host) -> None:
        host.containers((12, int(time.time()) - 8 * 86400))
        host.mark(12).write_text("50 human\n")
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert not [c for c in host.docker() if c.startswith("docker rm")]
        assert host.notices() == []

    def test_should_keep_an_expired_preview_without_a_mark(self, host: Host) -> None:
        host.containers((12, int(time.time()) - 8 * 86400))
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert not [c for c in host.docker() if c.startswith("docker rm")]

    def test_should_write_a_notice_for_each_removed_preview(self, host: Host) -> None:
        now = int(time.time())
        host.containers((12, now - 8 * 86400), (13, now - 86400))
        host.mark(12).write_text("0 none\n")
        host.mark(13).write_text("0 none\n")
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert host.notices() == ["12 expired"]

    def test_prunes_images_without_expired(self, host: Host) -> None:
        host.containers((13, int(time.time())))
        result = host.run("sweep")
        assert result.returncode == 0, result.stderr
        assert "docker image prune --all --force" in host.docker()

    def test_should_check_the_certificate(self, host: Host) -> None:
        assert host.run("sweep").returncode == 0
        assert len(lego_runs(host)) == 1
        assert FORCED_RELOAD not in host.docker()

    def test_should_reload_caddy_on_a_renewal(self, host: Host) -> None:
        (host.path / "renews").touch()
        assert host.run("sweep").returncode == 0
        assert host.docker()[-1] == FORCED_RELOAD

    def test_should_fail_when_the_renewal_fails(self, host: Host) -> None:
        (host.path / "lego_fails").touch()
        assert host.run("sweep").returncode != 0
        assert FORCED_RELOAD not in host.docker()


class TestNotices:
    """Tests for reading and deleting the notices of removed previews."""

    def write(self, host: Host, *lines: str) -> None:
        (host.sites / "notices").write_text("".join(f"{ln}\n" for ln in lines))

    def test_should_print_nothing_without_notices(self, host: Host) -> None:
        result = host.run("notices")
        assert result.returncode == 0, result.stderr
        assert result.stdout == ""

    def test_should_print_each_notice(self, host: Host) -> None:
        self.write(host, f"12 {AT} evicted 14", f"13 {AT} expired")
        result = host.run("notices")
        assert result.stdout.splitlines() == [f"12 {AT} evicted 14", f"13 {AT} expired"]

    def test_should_delete_only_the_acknowledged_notices(self, host: Host) -> None:
        later = "2026-11-22T11:00:00Z"
        self.write(
            host, f"12 {AT} evicted 14", f"12 {later} expired", f"13 {AT} expired"
        )
        result = host.run("ack", "12", AT, "13", AT)
        assert result.returncode == 0, result.stderr
        assert host.notices() == ["12 expired"]

    def test_should_delete_the_file_with_the_last_notice(self, host: Host) -> None:
        self.write(host, f"12 {AT} expired")
        assert host.run("ack", "12", AT).returncode == 0
        assert not (host.sites / "notices").exists()

    @pytest.mark.parametrize("args", [("12",), ("12", "yesterday"), ("12;rm", AT)])
    def test_should_reject_an_invalid_ack(
        self, host: Host, args: tuple[str, ...]
    ) -> None:
        self.write(host, f"12 {AT} expired")
        assert host.run("ack", *args).returncode != 0
        assert host.notices() == ["12 expired"]


POST = Path(__file__).resolve().parents[1] / "post_preview_notices.sh"

# Prints the notices in the file "notices" the way SSM does, with a tab before standard
# error, and logs every command.
FAKE_RUN = """#!/usr/bin/env bash
echo "run $COMMAND" >> "$FAKE_DIR/calls"
if [[ $COMMAND == *" notices" ]]; then
    cat "$FAKE_DIR/notices" 2> /dev/null || true
    printf '\\t\\n'
fi
echo "Status: Success"
"""

# Lists the preview comment of pull request N from the file "comment-N" as
# "<id> <updated_at>", and fails a PATCH when the file "patch_fails" exists. Each call
# is logged on one line.
FAKE_GH = """#!/usr/bin/env bash
args=$*
echo "gh ${args//$'\\n'/ }" >> "$FAKE_DIR/calls"
if [[ $* == *PATCH* ]]; then
    [[ ! -f "$FAKE_DIR/patch_fails" ]] || exit 1
    exit 0
fi
pr=$(sed -n 's|.*/issues/\\([0-9]*\\)/comments.*|\\1|p' <<< "$*")
cat "$FAKE_DIR/comment-$pr" 2> /dev/null || true
"""


class Poster:
    """The notice poster with fake gh and SSM commands."""

    def __init__(self, path: Path) -> None:
        self.path = path
        scripts = path / "scripts"
        scripts.mkdir()
        (scripts / POST.name).write_text(POST.read_text())
        bin_dir = path / "bin"
        bin_dir.mkdir()
        for fake, body in (
            (scripts / "run_preview_command.sh", FAKE_RUN),
            (bin_dir / "gh", FAKE_GH),
        ):
            fake.write_text(body)
        for fake in (*scripts.iterdir(), bin_dir / "gh"):
            fake.chmod(0o755)
        (path / "calls").touch()
        self.script = scripts / POST.name
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
            "FAKE_DIR": str(path),
            "INSTANCE": "i-1",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def run(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(self.script)], env=self.env, capture_output=True, text=True
        )

    def calls(self) -> list[str]:
        return (self.path / "calls").read_text().splitlines()

    def patches(self) -> list[str]:
        return [c for c in self.calls() if c.startswith("gh api -X PATCH")]

    def acks(self) -> list[str]:
        return [
            c for c in self.calls() if c.startswith("run /usr/local/bin/preview ack")
        ]


@pytest.fixture
def poster(tmp_path: Path) -> Poster:
    return Poster(tmp_path)


class TestPostNotices:
    """Tests for posting the notices of removed previews to their comments."""

    def test_should_post_an_eviction_and_acknowledge_it(self, poster: Poster) -> None:
        (poster.path / "notices").write_text(f"12 {AT} evicted 14\n")
        (poster.path / "comment-12").write_text("900 2026-11-22T09:00:00Z\n")
        result = poster.run()
        assert result.returncode == 0, result.stderr
        [patch] = poster.patches()
        assert "issues/comments/900" in patch
        assert "preview of #14" in patch
        assert poster.acks() == [f"run /usr/local/bin/preview ack 12 {AT}"]

    def test_should_post_an_expiry(self, poster: Poster) -> None:
        (poster.path / "notices").write_text(f"13 {AT} expired\n")
        (poster.path / "comment-13").write_text("901 2026-11-22T09:00:00Z\n")
        assert poster.run().returncode == 0
        [patch] = poster.patches()
        assert "no deploy or view" in patch

    def test_should_keep_a_newer_comment_and_acknowledge(self, poster: Poster) -> None:
        (poster.path / "notices").write_text(f"12 {AT} expired\n")
        (poster.path / "comment-12").write_text("900 2026-11-22T11:00:00Z\n")
        assert poster.run().returncode == 0
        assert poster.patches() == []
        assert poster.acks() == [f"run /usr/local/bin/preview ack 12 {AT}"]

    def test_should_acknowledge_nothing_when_a_post_fails(self, poster: Poster) -> None:
        (poster.path / "notices").write_text(f"12 {AT} expired\n")
        (poster.path / "comment-12").write_text("900 2026-11-22T09:00:00Z\n")
        (poster.path / "patch_fails").touch()
        assert poster.run().returncode != 0
        assert poster.acks() == []

    def test_should_acknowledge_nothing_without_notices(self, poster: Poster) -> None:
        assert poster.run().returncode == 0
        assert poster.acks() == []


class TestArguments:
    """Tests for rejected arguments."""

    @pytest.mark.parametrize(
        "args", [("up", "abc", "image", "sample"), ("down", "12;rm"), ("down", "")]
    )
    def test_should_reject_an_invalid_pull_request(
        self, host: Host, args: tuple[str, ...]
    ) -> None:
        result = host.run(*args)
        assert result.returncode == 1
        assert "invalid pull request number" in result.stderr
        assert host.docker() == []

    def test_should_reject_an_invalid_review_mark(self, host: Host) -> None:
        result = host.run("up", "12", "image", "sample", "bot", "100")
        assert result.returncode == 1
        assert "invalid review mark" in result.stderr
        assert host.docker() == []

    def test_should_reject_an_unknown_command(self, host: Host) -> None:
        result = host.run("restart")
        assert result.returncode == 2
        assert "usage:" in result.stderr
