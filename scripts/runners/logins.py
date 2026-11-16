#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The agent CLI logins that seats run on. A login is one subscription: a Secrets
Manager secret under PREFIX that holds a token from `claude setup-token`, stored by
`infra/agent-runner/seat.sh token`. Its state is in the tags of the secret, which only
the fleet controller and seat.sh write:

- agent: the CLI the token is for.
- expires: when the token expires, in Unix seconds.
- limited-until: when the usage limit of the login resets, in Unix seconds.
- broken: present when a job found the token rejected.
- holders: the role sessions of the seats that may read the token, separated by
  spaces. A session is `<role id>:<instance id>`, the `aws:userid` of the seat.

The controller never reads a token."""

from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from mypy_boto3_secretsmanager.client import SecretsManagerClient

PREFIX = "synnax-agents/logins/"
# The IAM role of every seat.
SEAT_ROLE = "synnax-agent-runner"
# A tag value holds at most 256 characters, and a holder takes 42 with its space.
MAX_HOLDERS = 6


@dataclass
class Login:
    name: str
    arn: str
    agent: str
    expires: int
    limited_until: int = 0
    broken: bool = False
    holders: list[str] = field(default_factory=list)

    def usable(self, *, now: int) -> bool:
        """Whether a seat may start on the login at now, in Unix seconds."""
        return (
            not self.broken
            and self.limited_until <= now < self.expires
            and len(self.holders) < MAX_HOLDERS
        )


class Logins:
    """Reads the logins and writes their tags. role is the unique id of the seats'
    IAM role, which starts the session of each seat."""

    def __init__(self, *, secrets: "SecretsManagerClient", role: str) -> None:
        self.secrets = secrets
        self.role = role

    def read(self) -> dict[str, Login]:
        """Every login by name, with its current tags, except deleted ones. A login
        stored in the last few minutes can be missing. Raises KeyError or ValueError
        on a login whose tags seat.sh did not write."""
        found: dict[str, Login] = {}
        pages = self.secrets.get_paginator("list_secrets").paginate(
            Filters=[{"Key": "name", "Values": [PREFIX]}]
        )
        for page in pages:
            for listed in page["SecretList"]:
                # ListSecrets can lag tag writes and deletions by minutes.
                # DescribeSecret is current.
                try:
                    secret = self.secrets.describe_secret(SecretId=listed["ARN"])
                except self.secrets.exceptions.ResourceNotFoundException:
                    continue
                if "DeletedDate" in secret:
                    continue
                name = secret["Name"].removeprefix(PREFIX)
                tags = {t["Key"]: t["Value"] for t in secret.get("Tags", [])}
                found[name] = Login(
                    name=name,
                    arn=secret["ARN"],
                    agent=tags["agent"],
                    expires=int(tags["expires"]),
                    limited_until=int(tags.get("limited-until", "0")),
                    broken="broken" in tags,
                    holders=tags.get("holders", "").split(),
                )
        return found

    def holder(self, iid: str) -> str:
        """The holder entry of the seat instance iid."""
        return f"{self.role}:{iid}"

    def hold(self, login: Login, *, holders: list[str]) -> None:
        """Sets the holders of the login."""
        if holders:
            self.secrets.tag_resource(
                SecretId=login.arn,
                Tags=[{"Key": "holders", "Value": " ".join(holders)}],
            )
        else:
            self.secrets.untag_resource(SecretId=login.arn, TagKeys=["holders"])
        login.holders = holders

    def limit(self, login: Login, *, until: int) -> None:
        """Records that the usage limit of the login resets at until, in Unix
        seconds."""
        self.secrets.tag_resource(
            SecretId=login.arn, Tags=[{"Key": "limited-until", "Value": str(until)}]
        )
        login.limited_until = until
