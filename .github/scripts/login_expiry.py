#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fails when a shared Claude login needs a person: its token expires within WARN, has
expired, or is marked broken, or no login is stored.

Reads the tags of each login that `infra/agent-runner/seat.sh token` stored in Secrets
Manager, never a token. Prints one line for each login.

Usage: login_expiry.py. Needs AWS credentials.
"""

import json
import subprocess
import sys
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from typing import Any

PREFIX = "synnax-agents/logins/"
REGION = "us-east-1"
WARN = timedelta(days=30)

# Runs the AWS CLI with the given arguments and returns its decoded JSON output.
Aws = Callable[[list[str]], Any]


def cli(args: list[str]) -> Any:
    run = subprocess.run(
        ["aws", *args, "--region", REGION, "--output", "json"],
        capture_output=True,
        text=True,
    )
    sys.stderr.write(run.stderr)
    run.check_returncode()
    return json.loads(run.stdout) if run.stdout.strip() else None


def check(aws: Aws, *, now: datetime) -> tuple[bool, list[str]]:
    """Returns whether no login needs a person, and one line for each login."""
    listed = aws(
        ["secretsmanager", "list-secrets", "--filters", f"Key=name,Values={PREFIX}"]
    )
    lines = []
    ok = True
    for name in sorted(s["Name"] for s in listed["SecretList"]):
        # The list can lag a deletion by minutes. Describe is current.
        try:
            secret = aws(["secretsmanager", "describe-secret", "--secret-id", name])
        except subprocess.CalledProcessError as e:
            if "ResourceNotFoundException" in e.stderr:
                continue
            raise
        if "DeletedDate" in secret:
            continue
        tags = {t["Key"]: t["Value"] for t in secret.get("Tags", [])}
        login = name.removeprefix(PREFIX)
        expires = datetime.fromtimestamp(int(tags["expires"]), UTC)
        if "broken" in tags:
            ok = False
            lines.append(f"{login}: broken, store a new token")
        elif expires <= now:
            ok = False
            lines.append(f"{login}: expired {expires:%Y-%m-%d}, store a new token")
        elif expires - now <= WARN:
            ok = False
            lines.append(f"{login}: expires {expires:%Y-%m-%d}, store a new token")
        else:
            lines.append(f"{login}: expires {expires:%Y-%m-%d}")
    if not lines:
        return False, ["no login is stored"]
    return ok, lines


def main() -> None:
    ok, lines = check(cli, now=datetime.now(UTC))
    print("\n".join(lines))
    if not ok:
        sys.exit(1)


if __name__ == "__main__":
    main()
