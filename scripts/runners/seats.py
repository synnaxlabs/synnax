#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Starts the agent and review seats, driven by controller.py's poll cycle. A seat is
one EC2 instance whose runners share an agent login. The runners belong to the
organization, so the repository's runner list holds none of them, and a seat is found
by its instance tags. A seat stops itself once idle, and when its login reaches a usage
limit.

The controller picks the login of each seat it starts (see logins.py): it tags the
instance `login` and adds the seat to the holders of the login, which lets the seat
read the token. A seat at a usage limit tags its own instance `limited-until`, and the
controller moves the limit to the login."""

import synnax as sy

from aws import AWS
from common import RECOMMAND_DELAY
from log import Log
from logins import Login, Logins

PROJECT = "synnax-agents"
# Labels every seat runner carries, which name no seat.
GENERIC = {"self-hosted", "linux", "x64"}
UP = ("pending", "running", "stopping")


def seat_labels(tags: dict[str, str]) -> set[str]:
    """The labels the runners of a seat register with, empty when the tags name no
    seat."""
    pool, agent, seat = tags.get("pool"), tags.get("agent"), tags.get("seat")
    if tags.get("project") != PROJECT or not agent or not seat:
        return set()
    if pool == "agent":
        return {"agent", f"agent-{agent}", f"seat-{seat}"}
    if pool == "review":
        return {"review", f"seat-{seat}"}
    return set()


def pick(logins: dict[str, Login], *, agent: str, now: int) -> Login | None:
    """The usable login of the agent with the fewest holders, if any."""
    usable = [
        login
        for login in logins.values()
        if login.agent == agent and login.usable(now=now)
    ]
    return min(usable, key=lambda login: (len(login.holders), login.name), default=None)


class Seats:
    """A queued job starts one stopped seat that carries its labels, on a usable
    login, unless a seat that carries them is already up on a login with usage left.
    A seat that is stopping starts once it has stopped. While disabled, each decision
    is logged once, marked [DISABLED], and nothing is written."""

    def __init__(self, *, aws: AWS, logins: Logins, log: Log) -> None:
        self.aws = aws
        self.logins = logins
        self.log = log
        self.started: dict[str, sy.TimeStamp] = {}
        self.held: set[str] = set()
        self.unserved: set[str] = set()

    def step(
        self, *, queued: list[set[str]], found: dict[str, dict], enabled: bool = True
    ) -> None:
        """Starts the seats that the queued jobs, given by their labels, wait for."""
        now = sy.TimeStamp.now()
        seconds = int(now) // int(sy.TimeSpan.SECOND)
        try:
            logins = self.logins.read()
        except Exception as e:
            self.log.error("login read", e)
            return
        seats = {
            iid: labels
            for iid, info in sorted(found.items())
            if (labels := seat_labels(info["tags"]))
        }
        if enabled:
            self.settle(seats, logins=logins, found=found)
        starting: set[str] = set()
        for job in queued:
            wanted = {label.lower() for label in job} - GENERIC
            matching = [iid for iid, labels in seats.items() if wanted <= labels]
            if not wanted or any(
                iid in starting
                or (
                    found[iid]["state"] in ("pending", "running")
                    and not self.limited(found[iid], logins=logins, now=seconds)
                )
                for iid in matching
            ):
                continue
            stopped = [iid for iid in matching if found[iid]["state"] == "stopped"]
            starting.update(stopped[:1])
        if not enabled:
            for iid in sorted(starting - self.held):
                self.log(f"[DISABLED] {self.name(iid, found=found)}: start")
            self.held = starting
            return
        self.held = set()
        for iid in sorted(starting):
            last = self.started.get(iid)
            if last is not None and now.span(last) < RECOMMAND_DELAY:
                continue
            name = self.name(iid, found=found)
            login = pick(logins, agent=found[iid]["tags"]["agent"], now=seconds)
            if login is None:
                if iid not in self.unserved:
                    self.log(f"{name}: no usable login")
                self.unserved.add(iid)
                continue
            self.unserved.discard(iid)
            # Stamped before the calls, so a failure waits out the delay.
            self.started[iid] = now
            try:
                self.start(iid, login=login, info=found[iid])
            except Exception as e:
                self.log.error(f"{name} start", e)
                continue
            self.log(f"{name}: start requested on login {login.name}")

    def settle(
        self, seats: dict[str, set[str]], *, logins: dict[str, Login], found: dict
    ) -> None:
        """Moves the usage limit each seat recorded to its login, and drops the seats
        that are not up from the holders of each login."""
        for iid in seats:
            tags = found[iid]["tags"]
            login = logins.get(tags.get("login", ""))
            # Agents can write the seat's tag, so a value that is not a whole number
            # counts as no limit.
            until = tags.get("limited-until", "")
            if login is None or not until.isdecimal():
                continue
            if int(until) > login.limited_until:
                try:
                    self.logins.limit(login, until=int(until))
                except Exception as e:
                    self.log.error(f"login {login.name} limit", e)
        for login in logins.values():
            kept = [
                h
                for h in login.holders
                if found.get(h.rpartition(":")[2], {}).get("state") in UP
            ]
            if kept == login.holders:
                continue
            try:
                self.logins.hold(login, holders=kept)
            except Exception as e:
                self.log.error(f"login {login.name} release", e)

    def start(self, iid: str, *, login: Login, info: dict) -> None:
        """Hands the login to the stopped seat iid, then starts it. The seat's limit
        belongs to its last login, so it goes."""
        ec2 = self.aws.ec2[info["region"]]
        ec2.delete_tags(Resources=[iid], Tags=[{"Key": "limited-until"}])
        ec2.create_tags(Resources=[iid], Tags=[{"Key": "login", "Value": login.name}])
        self.logins.hold(login, holders=[*login.holders, self.logins.holder(iid)])
        ec2.start_instances(InstanceIds=[iid])

    def limited(self, info: dict, *, logins: dict[str, Login], now: int) -> bool:
        """Whether the login of a seat is at a usage limit that has not reset by
        now."""
        login = logins.get(info["tags"].get("login", ""))
        return login is not None and login.limited_until > now

    def name(self, iid: str, *, found: dict[str, dict]) -> str:
        return f"seat {found[iid]['tags']['seat']}"
