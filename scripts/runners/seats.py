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
by its instance tags. A seat stops itself once idle, when its login reaches a usage
limit, and when Claude rejects the token of its login.

The controller picks the login of each seat it starts (see logins.py): it tags the
instance `login` and adds the seat to the holders of the login, which lets the seat
read the token. Once the seat stops, the controller removes it from the holders and
deletes its `login` tag, so a seat started outside the controller holds no login. A seat
at a usage limit tags its own instance `limited-until`, and the controller moves the
limit to the login. A seat whose token Claude rejected tags its own instance
`login-broken`, and the controller marks the login broken."""

from collections.abc import Mapping, Sequence
from typing import Protocol

import synnax as sy
from mypy_boto3_ec2.type_defs import TagTypeDef

from common import RECOMMAND_DELAY
from log import Log
from logins import MAX_HOLDERS, Login, Logins

PROJECT = "synnax-agents"
# Labels every seat runner carries, which name no seat.
GENERIC = {"self-hosted", "linux", "x64"}
UP = ("pending", "running", "stopping")
# A seat with a free runner takes a queued job in seconds, and a started seat boots in
# less. A job that waits longer than this beside an up seat finds every runner busy.
FULL_AFTER = 5 * sy.TimeSpan.MINUTE
# The longest usage limit of a Claude subscription is a week.
MAX_RESET = 8 * 24 * 3600
# Agents whose seats run on a shared login. A seat of any other agent holds its own
# login, and its `limited-until` tag keeps it stopped until the limit resets.
SHARED = {"claude"}


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


def seat_limited(tags: dict[str, str], *, now: int) -> bool:
    """Whether a seat that holds its own login is at a usage limit that has not reset
    by now. Agents can write the tag, so a value that is not a whole number counts as
    no limit."""
    until = tags.get("limited-until", "")
    return until.isdecimal() and int(until) > now


class Seat(Protocol):
    def start_instances(self, *, InstanceIds: list[str]) -> object: ...

    def create_tags(
        self, *, Resources: list[str], Tags: list[TagTypeDef]
    ) -> object: ...

    def delete_tags(
        self, *, Resources: list[str], Tags: list[TagTypeDef]
    ) -> object: ...


class Fleet(Protocol):
    """The AWS calls Seats makes, as aws.AWS has them."""

    @property
    def ec2(self) -> Mapping[str, Seat]: ...


class Seats:
    """A queued job starts one stopped seat that carries its labels, on a usable
    shared login for a Claude seat, unless a seat that carries them is already up with
    usage left. A stuck job also starts one when every such seat launched FULL_AFTER
    ago or earlier. A seat that holds its own login starts once its limit resets.
    A seat that is stopping starts once it has stopped. While disabled, each decision
    is logged once, marked [DISABLED], and nothing is written."""

    def __init__(self, *, aws: Fleet, logins: Logins, log: Log) -> None:
        self.aws = aws
        self.logins = logins
        self.log = log
        self.started: dict[str, sy.TimeStamp] = {}
        self.held: set[str] = set()
        self.unserved: set[str] = set()

    def step(
        self,
        *,
        queued: list[set[str]],
        found: dict[str, dict],
        enabled: bool = True,
        stuck: Sequence[set[str]] = (),
        connected: bool = True,
    ) -> None:
        """Starts the seats that the queued jobs, given by their labels, wait for.
        stuck gives the labels of the queued jobs that have waited FULL_AFTER. While
        not connected, the runners cannot reach GitHub, so no job counts as stuck."""
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
            and (
                info["tags"]["agent"] in SHARED
                or not seat_limited(info["tags"], now=seconds)
            )
        }
        if enabled:
            # EC2 can report a seat stopped for some seconds after its start.
            recent = {
                iid
                for iid, last in self.started.items()
                if now.span(last) < RECOMMAND_DELAY
            }
            self.settle(seats, logins=logins, found=found, recent=recent, now=seconds)
        booting = {
            iid
            for iid, info in found.items()
            if info["state"] == "pending" or now.span(info["launched"]) < FULL_AFTER
        }
        starting: set[str] = set()
        waited = list(stuck) if connected else []
        jobs = [(job, False) for job in queued] + [(job, True) for job in waited]
        for job, full in jobs:
            wanted = {label.lower() for label in job} - GENERIC
            matching = [iid for iid, labels in seats.items() if wanted <= labels]
            if not wanted or any(
                iid in starting
                or (
                    found[iid]["state"] in ("pending", "running")
                    and self.served(found[iid], logins=logins, now=seconds)
                    and (not full or iid in booting)
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
            login = None
            if found[iid]["tags"]["agent"] in SHARED:
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
            on = f" on login {login.name}" if login else ""
            self.log(f"{name}: start requested{on}")

    def settle(
        self,
        seats: dict[str, set[str]],
        *,
        logins: dict[str, Login],
        found: dict,
        recent: set[str],
        now: int,
    ) -> None:
        """Moves the usage limit and the broken token each seat recorded to its login,
        and sets the holders of each login to the seats that are up with its `login`
        tag, up to MAX_HOLDERS with the current holders first. A seat started
        recently, given by recent, or whose move failed keeps its holder entries and
        `login` tag. Deletes the `login` tag of each other stopped seat and of each
        seat left out by the cap."""
        kept = set(recent)
        for iid in seats:
            info = found[iid]
            if info["tags"]["agent"] not in SHARED:
                continue
            login = logins.get(info["tags"].get("login", ""))
            until = info["tags"].get("limited-until")
            if until is not None:
                # Agents can write the seat's tag, so a value that is not a whole
                # number in the next reset window counts as no limit.
                if (
                    login is not None
                    and until.isdecimal()
                    and login.limited_until < int(until) <= now + MAX_RESET
                ):
                    try:
                        self.logins.limit(login, until=int(until))
                    except Exception as e:
                        self.log.error(f"login {login.name} limit", e)
                        kept.add(iid)
                        continue
                self.clear(iid, key="limited-until", found=found)
            at = info["tags"].get("login-broken")
            if at is not None:
                # A job that started before seat.sh stored the token ran on the old
                # one. Agents can write the seat's tag, so a value that is not a whole
                # number up to now counts as no rejection.
                if (
                    login is not None
                    and not login.broken
                    and at.isdecimal()
                    and login.stored < int(at) <= now
                ):
                    try:
                        self.logins.reject(login, at=int(at))
                    except Exception as e:
                        self.log.error(f"login {login.name} reject", e)
                        kept.add(iid)
                        continue
                    self.log(f"login {login.name}: token rejected, marked broken")
                self.clear(iid, key="login-broken", found=found)
        for login in logins.values():
            tagged = [
                self.logins.holder(iid)
                for iid, info in sorted(found.items())
                if info["state"] in UP and info["tags"].get("login") == login.name
            ]
            holders = [
                h for h in login.holders if h in tagged or h.rpartition(":")[2] in kept
            ]
            holders += [h for h in tagged if h not in holders]
            # The holders tag value has room for MAX_HOLDERS role sessions.
            released = [h.rpartition(":")[2] for h in holders[MAX_HOLDERS:]] + [
                iid
                for iid, info in sorted(found.items())
                if info["state"] == "stopped"
                and iid not in kept
                and info["tags"].get("login") == login.name
            ]
            holders = holders[:MAX_HOLDERS]
            if holders != login.holders:
                try:
                    self.logins.hold(login, holders=holders)
                except Exception as e:
                    self.log.error(f"login {login.name} hold", e)
            for iid in released:
                try:
                    self.aws.ec2[found[iid]["region"]].delete_tags(
                        Resources=[iid], Tags=[{"Key": "login"}]
                    )
                except Exception as e:
                    self.log.error(f"{self.name(iid, found=found)} login clear", e)

    def clear(self, iid: str, *, key: str, found: dict) -> None:
        """Deletes the tag key of the seat iid. A tag left on the seat would undo a new
        token's clear of the login."""
        try:
            self.aws.ec2[found[iid]["region"]].delete_tags(
                Resources=[iid], Tags=[{"Key": key}]
            )
        except Exception as e:
            self.log.error(f"{self.name(iid, found=found)} {key} clear", e)

    def start(self, iid: str, *, login: Login | None, info: dict) -> None:
        """Hands the login, if any, to the stopped seat iid, then starts it."""
        ec2 = self.aws.ec2[info["region"]]
        if login is not None:
            ec2.create_tags(
                Resources=[iid], Tags=[{"Key": "login", "Value": login.name}]
            )
            self.logins.hold(login, holders=[*login.holders, self.logins.holder(iid)])
        ec2.start_instances(InstanceIds=[iid])

    def served(self, info: dict, *, logins: dict[str, Login], now: int) -> bool:
        """Whether the login of a seat, if any, is neither broken nor at a usage limit
        that has not reset by now."""
        login = logins.get(info["tags"].get("login", ""))
        return login is None or not (login.broken or login.limited_until > now)

    def name(self, iid: str, *, found: dict[str, dict]) -> str:
        return f"seat {found[iid]['tags']['seat']}"
