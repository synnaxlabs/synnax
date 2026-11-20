#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
from mypy_boto3_ec2.type_defs import TagTypeDef

from common import RECOMMAND_DELAY, Channels
from conftest import Client, Secrets
from log import Log
from logins import MAX_HOLDERS, PREFIX, Logins
from seats import FULL_AFTER, Seats, seat_labels

JOB = {"self-hosted", "agent-claude"}
CLAUDE = "i-0aaa"
CLAUDE_2 = "i-0bbb"
CODEX = "i-0ccc"
CLAUDE_3 = "i-0eee"
REVIEW = "i-0ddd"


ROLE = "AROAROLE"
# Long enough ago that no seat counts as booting.
LAUNCHED = sy.TimeStamp(0)
YEAR = 365 * 24 * sy.TimeSpan.HOUR


def seconds(ts: sy.TimeStamp) -> str:
    return str(int(ts / sy.TimeSpan.SECOND))


def seat(
    state: str,
    *,
    name: str = "claude-1",
    pool: str = "agent",
    login: str | None = None,
    until: sy.TimeStamp | None = None,
    rejected: sy.TimeStamp | None = None,
    launched: sy.TimeStamp = LAUNCHED,
) -> dict:
    # Review seats run Claude Code.
    agent = "claude" if pool == "review" else name.split("-")[0]
    tags = {"project": "synnax-agents", "pool": pool, "agent": agent, "seat": name}
    if login is not None:
        tags["login"] = login
    if until is not None:
        tags["limited-until"] = seconds(until)
    if rejected is not None:
        tags["login-broken"] = seconds(rejected)
    return {"state": state, "region": "us-east-1", "launched": launched, "tags": tags}


class Instances:
    """Stands in for AWS, recording what is started and the tags written. Starts and
    tag deletes fail while failing is set."""

    def __init__(self) -> None:
        self.started: list[str] = []
        self.tags: dict[str, dict[str, str]] = {}
        self.deleted: list[tuple[str, str]] = []
        self.failing = False
        self.ec2 = {"us-east-1": self}

    def start_instances(self, *, InstanceIds: list[str]) -> None:
        if self.failing:
            raise PermissionError("not authorized")
        self.started.extend(InstanceIds)

    def create_tags(self, *, Resources: list[str], Tags: list[TagTypeDef]) -> None:
        for iid in Resources:
            self.tags.setdefault(iid, {}).update({t["Key"]: t["Value"] for t in Tags})

    def delete_tags(self, *, Resources: list[str], Tags: list[TagTypeDef]) -> None:
        if self.failing:
            raise PermissionError("not authorized")
        for iid in Resources:
            for t in Tags:
                self.deleted.append((iid, t["Key"]))
                self.tags.setdefault(iid, {}).pop(t["Key"], None)


def secrets(*names: str, stored: sy.TimeStamp | None = None) -> Secrets:
    """A store with a usable login of each name, for the agent its name starts
    with, stored at stored or now."""
    store = Secrets()
    expires = seconds((stored or sy.TimeStamp.now()) + YEAR)
    for name in names:
        store.tags[f"{PREFIX}{name}"] = {
            "agent": name.split("-")[0],
            "expires": expires,
        }
    return store


def seats(client: Client, aws: Instances, store: Secrets | None = None) -> Seats:
    log = Log(
        client=client,
        channels=Channels(client=client, enabled=False),
        channel="log",
    )
    store = store or secrets("claude-1", "claude-2")
    logins = Logins(secrets=store, role=ROLE)
    return Seats(aws=aws, logins=logins, log=log)


class TestSeatLabels:
    """Tests for mapping instance tags to the labels of a seat's runners."""

    def test_should_map_an_agent_seat_to_its_runner_labels(self) -> None:
        labels = seat_labels(seat("stopped")["tags"])
        assert labels == {"agent", "agent-claude", "seat-claude-1"}

    def test_should_map_a_review_seat_to_its_runner_labels(self) -> None:
        labels = seat_labels(seat("stopped", name="review-1", pool="review")["tags"])
        assert labels == {"review", "seat-review-1"}

    def test_should_map_other_instances_to_no_labels(self) -> None:
        assert seat_labels(seat("stopped", pool="other")["tags"]) == set()
        assert seat_labels({"RunnerOS": "ubuntu"}) == set()

    def test_should_map_an_agent_instance_without_seat_tags_to_no_labels(
        self,
    ) -> None:
        assert seat_labels({"project": "synnax-agents", "pool": "agent"}) == set()


class TestSeats:
    """Tests for starting seats to meet queued jobs."""

    def test_should_start_a_stopped_seat_for_a_queued_job(self, client: Client) -> None:
        aws = Instances()
        seats(client, aws).step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.started == [CLAUDE]
        assert client.written("log") == [
            ["seat claude-1: start requested on login claude-1"]
        ]

    def test_should_start_one_seat_for_several_queued_jobs(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(queued=[JOB, {"self-hosted", "agent"}], found=found)
        assert aws.started == [CLAUDE]

    def test_should_start_only_the_seat_of_the_agent_a_job_asks_for(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CODEX: seat("stopped", name="codex-1")}
        seats(client, aws).step(queued=[{"self-hosted", "agent-codex"}], found=found)
        assert aws.started == [CODEX]

    def test_should_start_the_seat_a_job_names(self, client: Client) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(
            queued=[{"self-hosted", "Linux", "seat-claude-2"}], found=found
        )
        assert aws.started == [CLAUDE_2]

    def test_should_start_a_stopped_review_seat_for_a_queued_review(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {
            CLAUDE: seat("stopped"),
            REVIEW: seat("stopped", name="review-1", pool="review"),
        }
        seats(client, aws).step(queued=[{"self-hosted", "review"}], found=found)
        assert aws.started == [REVIEW]
        assert client.written("log") == [
            ["seat review-1: start requested on login claude-1"]
        ]

    def test_should_not_start_a_review_seat_for_an_agent_job(
        self, client: Client
    ) -> None:
        aws = Instances()
        seats(client, aws).step(
            queued=[JOB, {"self-hosted", "agent"}],
            found={REVIEW: seat("stopped", name="review-1", pool="review")},
        )
        assert aws.started == []

    def test_should_not_start_a_seat_for_jobs_of_other_pools(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {
            CLAUDE: seat("stopped"),
            REVIEW: seat("stopped", name="review-1", pool="review"),
        }
        seats(client, aws).step(
            queued=[
                {"self-hosted", "ubuntu-build-bot"},
                {"self-hosted", "linux"},
                {"ubuntu-latest"},
            ],
            found=found,
        )
        assert aws.started == []

    def test_should_not_start_a_seat_while_one_is_up(self, client: Client) -> None:
        aws = Instances()
        s = seats(client, aws)
        for state in ("pending", "running"):
            found = {CLAUDE: seat(state), CLAUDE_2: seat("stopped", name="claude-2")}
            s.step(queued=[JOB], found=found)
        assert aws.started == []

    def test_should_start_another_seat_for_a_job_stuck_beside_a_busy_one(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("running"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(queued=[JOB], found=found, stuck=[JOB])
        assert aws.started == [CLAUDE_2]

    def test_should_not_start_a_seat_for_a_stuck_job_while_disconnected(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("running"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(queued=[JOB], found=found, stuck=[JOB], connected=False)
        assert aws.started == []

    def test_should_not_start_another_seat_for_a_stuck_job_while_one_boots(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("pending"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(queued=[JOB], found=found, stuck=[JOB])
        assert aws.started == []

    def test_should_give_a_seat_that_just_launched_time_to_take_a_stuck_job(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        up = seat("running", launched=sy.TimeStamp.now())
        found = {CLAUDE: up, CLAUDE_2: seat("stopped", name="claude-2")}
        s.step(queued=[JOB], found=found, stuck=[JOB])
        assert aws.started == []
        client.elapsed += FULL_AFTER
        s.step(queued=[JOB], found=found, stuck=[JOB])
        assert aws.started == [CLAUDE_2]

    def test_should_start_one_seat_for_several_stuck_jobs(self, client: Client) -> None:
        aws = Instances()
        found = {
            CLAUDE: seat("running"),
            CLAUDE_2: seat("stopped", name="claude-2"),
            CLAUDE_3: seat("stopped", name="claude-3"),
        }
        seats(client, aws).step(queued=[JOB, JOB], found=found, stuck=[JOB, JOB])
        assert aws.started == [CLAUDE_2]

    def test_should_start_a_stopping_seat_once_it_has_stopped(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        s.step(queued=[JOB], found={CLAUDE: seat("stopping")})
        assert aws.started == []
        s.step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.started == [CLAUDE]

    def test_should_wait_out_the_delay_before_it_starts_a_seat_again(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]
        client.elapsed += RECOMMAND_DELAY
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE, CLAUDE]

    def test_should_log_a_failed_start_and_try_again_after_the_delay(
        self, client: Client
    ) -> None:
        aws = Instances()
        aws.failing = True
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        s.step(queued=[JOB], found=found)
        assert client.written("log") == [
            ["[ERROR] seat claude-1 start failed: PermissionError: not authorized"]
        ]
        aws.failing = False
        client.elapsed += RECOMMAND_DELAY + sy.TimeSpan.SECOND
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

    def test_should_log_each_start_once_while_disabled(self, client: Client) -> None:
        aws = Instances()
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found, enabled=False)
        s.step(queued=[JOB], found=found, enabled=False)
        assert aws.started == []
        assert client.written("log") == [["[DISABLED] seat claude-1: start"]]
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

    def test_should_hand_the_seat_a_login_before_it_starts(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        until = sy.TimeStamp.now() - sy.TimeSpan.HOUR
        seats(client, aws, store).step(
            queued=[JOB], found={CLAUDE: seat("stopped", login="old", until=until)}
        )
        assert aws.tags[CLAUDE] == {"login": "claude-1"}
        assert store.tags[f"{PREFIX}claude-1"]["holders"] == f"{ROLE}:{CLAUDE}"
        assert aws.started == [CLAUDE]

    def test_should_pick_the_login_with_the_fewest_holders(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1", "claude-2")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE_2}"
        found = {
            CLAUDE: seat("stopped"),
            CLAUDE_2: seat("running", name="claude-2", login="claude-1"),
        }
        seats(client, aws, store).step(queued=[{"seat-claude-1"}], found=found)
        assert aws.tags[CLAUDE] == {"login": "claude-2"}

    def test_should_start_a_codex_seat_on_its_own_login(self, client: Client) -> None:
        aws = Instances()
        seats(client, aws).step(
            queued=[{"self-hosted", "agent-codex"}],
            found={CODEX: seat("stopped", name="codex-1")},
        )
        assert aws.started == [CODEX]
        assert aws.tags == {}
        assert client.written("log") == [["seat codex-1: start requested"]]

    def test_should_keep_a_limited_codex_seat_stopped_until_the_reset(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {CODEX: seat("stopped", name="codex-1", until=until)}
        s.step(queued=[{"agent-codex"}], found=found)
        assert aws.started == []
        assert aws.deleted == []
        client.elapsed += sy.TimeSpan.HOUR + sy.TimeSpan.SECOND
        s.step(queued=[{"agent-codex"}], found=found)
        assert aws.started == [CODEX]

    def test_should_start_another_codex_seat_while_one_is_limited(
        self, client: Client
    ) -> None:
        aws = Instances()
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {
            CODEX: seat("running", name="codex-1", until=until),
            "i-0eee": seat("stopped", name="codex-2"),
        }
        seats(client, aws).step(queued=[{"agent-codex"}], found=found)
        assert aws.started == ["i-0eee"]

    def test_should_pick_only_a_login_of_the_seats_agent(self, client: Client) -> None:
        aws = Instances()
        seats(client, aws, secrets("codex-1")).step(
            queued=[JOB], found={CLAUDE: seat("stopped")}
        )
        assert aws.started == []
        assert client.written("log") == [["seat claude-1: no usable login"]]

    def test_should_log_a_seat_without_a_usable_login_once(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        store.tags[f"{PREFIX}claude-1"]["broken"] = ""
        s = seats(client, aws, store)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        s.step(queued=[JOB], found=found)
        assert aws.started == []
        assert client.written("log") == [["seat claude-1: no usable login"]]
        del store.tags[f"{PREFIX}claude-1"]["broken"]
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

    def test_should_skip_a_login_at_its_limit(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1", "claude-2")
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        store.tags[f"{PREFIX}claude-1"]["limited-until"] = seconds(until)
        seats(client, aws, store).step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.tags[CLAUDE] == {"login": "claude-2"}

    def test_should_move_the_limit_of_a_seat_to_its_login(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1")
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {CLAUDE: seat("running", login="claude-1", until=until)}
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[f"{PREFIX}claude-1"]["limited-until"] == seconds(until)
        assert aws.deleted == [(CLAUDE, "limited-until")]

    def test_should_keep_the_limit_on_the_seat_when_the_login_write_fails(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        store.failing = True
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {CLAUDE: seat("running", login="claude-1", until=until)}
        seats(client, aws, store).step(queued=[], found=found)
        assert aws.deleted == []
        assert client.written("log") == [
            ["[ERROR] login claude-1 limit failed: PermissionError: not authorized"]
        ]

    def test_should_keep_a_stopped_seat_on_its_login_until_its_limit_moves(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        store.tags[name]["holders"] = f"{ROLE}:{CLAUDE}"
        store.failing = True
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {CLAUDE: seat("stopped", login="claude-1", until=until)}
        s = seats(client, aws, store)
        s.step(queued=[], found=found)
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        assert aws.deleted == []
        store.failing = False
        s.step(queued=[], found=found)
        assert store.tags[name]["limited-until"] == seconds(until)
        assert "holders" not in store.tags[name]
        assert aws.deleted == [(CLAUDE, "limited-until"), (CLAUDE, "login")]

    def test_should_not_lower_the_limit_of_a_login(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1")
        later = seconds(sy.TimeStamp.now() + 2 * sy.TimeSpan.HOUR)
        store.tags[f"{PREFIX}claude-1"]["limited-until"] = later
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {CLAUDE: seat("running", login="claude-1", until=until)}
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[f"{PREFIX}claude-1"]["limited-until"] == later
        assert aws.deleted == [(CLAUDE, "limited-until")]

    def test_should_ignore_a_malformed_limit(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1")
        found = {CLAUDE: seat("running", login="claude-1")}
        found[CLAUDE]["tags"]["limited-until"] = "1795266000.0"
        seats(client, aws, store).step(queued=[], found=found)
        assert "limited-until" not in store.tags[f"{PREFIX}claude-1"]
        assert aws.deleted == [(CLAUDE, "limited-until")]

    def test_should_ignore_a_limit_past_the_longest_reset(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1")
        found = {CLAUDE: seat("running", login="claude-1")}
        for until in (sy.TimeStamp.now() + 9 * 24 * sy.TimeSpan.HOUR, "9" * 30):
            found[CLAUDE]["tags"]["limited-until"] = (
                until if isinstance(until, str) else seconds(until)
            )
            seats(client, aws, store).step(queued=[], found=found)
        assert "limited-until" not in store.tags[f"{PREFIX}claude-1"]
        assert aws.deleted == [(CLAUDE, "limited-until")] * 2

    def test_should_clear_the_limit_of_a_seat_without_a_login(
        self, client: Client
    ) -> None:
        aws = Instances()
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        seats(client, aws).step(queued=[], found={CLAUDE: seat("stopped", until=until)})
        assert aws.deleted == [(CLAUDE, "limited-until")]

    def test_should_mark_the_login_of_a_seat_with_a_rejected_token_broken(
        self, client: Client
    ) -> None:
        aws = Instances()
        now = sy.TimeStamp.now()
        store = secrets("claude-1", stored=now - sy.TimeSpan.HOUR)
        rejected = now - sy.TimeSpan.MINUTE
        found = {CLAUDE: seat("running", login="claude-1", rejected=rejected)}
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[f"{PREFIX}claude-1"]["broken"] == seconds(rejected)
        assert aws.deleted == [(CLAUDE, "login-broken")]
        assert client.written("log") == [
            ["login claude-1: token rejected, marked broken"]
        ]

    def test_should_ignore_a_rejection_of_a_job_started_before_the_token_was_stored(
        self, client: Client
    ) -> None:
        aws = Instances()
        now = sy.TimeStamp.now()
        store = secrets("claude-1", stored=now - sy.TimeSpan.MINUTE)
        rejected = now - sy.TimeSpan.HOUR
        found = {CLAUDE: seat("running", login="claude-1", rejected=rejected)}
        seats(client, aws, store).step(queued=[], found=found)
        assert "broken" not in store.tags[f"{PREFIX}claude-1"]
        assert aws.deleted == [(CLAUDE, "login-broken")]

    @pytest.mark.parametrize("value", ["soon", "-1", "99999999999"])
    def test_should_ignore_a_rejection_that_is_not_a_past_time(
        self, client: Client, value: str
    ) -> None:
        aws = Instances()
        store = secrets("claude-1", stored=sy.TimeStamp.now() - sy.TimeSpan.HOUR)
        found = {CLAUDE: seat("running", login="claude-1")}
        found[CLAUDE]["tags"]["login-broken"] = value
        seats(client, aws, store).step(queued=[], found=found)
        assert "broken" not in store.tags[f"{PREFIX}claude-1"]
        assert aws.deleted == [(CLAUDE, "login-broken")]

    def test_should_keep_the_rejected_token_on_the_seat_when_the_login_write_fails(
        self, client: Client
    ) -> None:
        aws = Instances()
        now = sy.TimeStamp.now()
        store = secrets("claude-1", stored=now - sy.TimeSpan.HOUR)
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        store.failing = True
        rejected = now - sy.TimeSpan.MINUTE
        found = {CLAUDE: seat("running", login="claude-1", rejected=rejected)}
        seats(client, aws, store).step(queued=[], found=found)
        assert aws.deleted == []
        assert client.written("log") == [
            ["[ERROR] login claude-1 reject failed: PermissionError: not authorized"]
        ]

    def test_should_clear_a_rejected_token_of_a_login_already_broken(
        self, client: Client
    ) -> None:
        aws = Instances()
        now = sy.TimeStamp.now()
        store = secrets("claude-1", stored=now - sy.TimeSpan.HOUR)
        store.tags[f"{PREFIX}claude-1"]["broken"] = "1"
        store.failing = True
        rejected = now - sy.TimeSpan.MINUTE
        found = {CLAUDE: seat("stopped", login="claude-1", rejected=rejected)}
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[f"{PREFIX}claude-1"]["broken"] == "1"
        assert (CLAUDE, "login-broken") in aws.deleted

    def test_should_start_another_seat_while_one_is_on_a_broken_login(
        self, client: Client
    ) -> None:
        aws = Instances()
        now = sy.TimeStamp.now()
        store = secrets("claude-1", "claude-2", stored=now - sy.TimeSpan.HOUR)
        found = {
            CLAUDE: seat(
                "running", login="claude-1", rejected=now - sy.TimeSpan.MINUTE
            ),
            CLAUDE_2: seat("stopped", name="claude-2"),
        }
        seats(client, aws, store).step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE_2]
        assert aws.tags[CLAUDE_2] == {"login": "claude-2"}

    def test_should_drop_a_seat_that_is_not_up_from_the_holders(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        store.tags[name]["holders"] = f"{ROLE}:{CLAUDE} {ROLE}:{CLAUDE_2} {ROLE}:i-gone"
        found = {
            CLAUDE: seat("stopping", login="claude-1"),
            CLAUDE_2: seat("stopped", name="claude-2", login="claude-1"),
        }
        s = seats(client, aws, store)
        s.step(queued=[], found=found)
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        found[CLAUDE]["state"] = "stopped"
        s.step(queued=[], found=found)
        assert "holders" not in store.tags[name]

    def test_should_keep_a_started_seat_that_ec2_still_reports_stopped(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        s = seats(client, aws, store)
        s.step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        client.elapsed += 3 * sy.TimeSpan.SECOND
        s.step(queued=[], found={CLAUDE: seat("stopped", login="claude-1")})
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        assert aws.tags[CLAUDE] == {"login": "claude-1"}

    def test_should_drop_a_started_seat_still_stopped_after_the_delay(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        s = seats(client, aws, store)
        s.step(queued=[JOB], found={CLAUDE: seat("stopped")})
        client.elapsed += RECOMMAND_DELAY
        s.step(queued=[], found={CLAUDE: seat("stopped", login="claude-1")})
        assert "holders" not in store.tags[name]
        assert aws.deleted == [(CLAUDE, "login")]

    def test_should_drop_a_stopped_seat_and_delete_its_login_tag(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        store.tags[name]["holders"] = f"{ROLE}:{CLAUDE}"
        found = {CLAUDE: seat("stopped", login="claude-1")}
        seats(client, aws, store).step(queued=[], found=found)
        assert "holders" not in store.tags[name]
        assert aws.deleted == [(CLAUDE, "login")]

    def test_should_log_a_failed_holders_write_and_still_delete_the_login_tag(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        store.tags[name]["holders"] = f"{ROLE}:{CLAUDE}"
        store.failing = True
        found = {CLAUDE: seat("stopped", login="claude-1")}
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        assert aws.deleted == [(CLAUDE, "login")]
        assert client.written("log") == [
            ["[ERROR] login claude-1 hold failed: PermissionError: not authorized"]
        ]

    def test_should_log_each_failed_login_tag_delete_and_go_on(
        self, client: Client
    ) -> None:
        aws = Instances()
        aws.failing = True
        store = secrets("claude-1", "claude-2")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        store.tags[f"{PREFIX}claude-2"]["holders"] = f"{ROLE}:{CLAUDE_2}"
        found = {
            CLAUDE: seat("stopped", login="claude-1"),
            CLAUDE_2: seat("stopped", name="claude-2", login="claude-2"),
        }
        seats(client, aws, store).step(queued=[], found=found)
        assert "holders" not in store.tags[f"{PREFIX}claude-1"]
        assert "holders" not in store.tags[f"{PREFIX}claude-2"]
        assert client.written("log") == [
            [
                "[ERROR] seat claude-1 login clear failed: "
                "PermissionError: not authorized"
            ],
            [
                "[ERROR] seat claude-2 login clear failed: "
                "PermissionError: not authorized"
            ],
        ]

    def test_should_not_add_a_seat_started_without_a_login_tag(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        store.tags[name]["holders"] = f"{ROLE}:{CLAUDE}"
        found = {
            CLAUDE: seat("running", login="claude-1"),
            CLAUDE_2: seat("running", name="claude-2"),
        }
        seats(client, aws, store).step(queued=[], found=found)
        assert store.tags[name]["holders"] == f"{ROLE}:{CLAUDE}"
        assert aws.deleted == []

    def test_should_cap_the_holders_and_delete_the_login_tag_of_the_rest(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        name = f"{PREFIX}claude-1"
        iids = [f"i-{n:04d}" for n in range(MAX_HOLDERS + 2)]
        store.tags[name]["holders"] = f"{ROLE}:{iids[-1]}"
        found = {
            iid: seat("running", name=f"claude-{n}", login="claude-1")
            for n, iid in enumerate(iids)
        }
        seats(client, aws, store).step(queued=[], found=found)
        kept = [iids[-1], *iids[: MAX_HOLDERS - 1]]
        assert store.tags[name]["holders"] == " ".join(f"{ROLE}:{i}" for i in kept)
        assert aws.deleted == [(iid, "login") for iid in iids[MAX_HOLDERS - 1 : -1]]

    def test_should_add_an_up_seat_missing_from_the_holders_of_its_login(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1", "claude-2")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        found = {
            CLAUDE: seat("running", login="claude-1"),
            CLAUDE_2: seat("pending", name="claude-2", login="claude-1"),
            REVIEW: seat("running", name="review-1", pool="review", login="claude-2"),
        }
        seats(client, aws, store).step(queued=[], found=found)
        assert (
            store.tags[f"{PREFIX}claude-1"]["holders"]
            == f"{ROLE}:{CLAUDE} {ROLE}:{CLAUDE_2}"
        )
        assert store.tags[f"{PREFIX}claude-2"]["holders"] == f"{ROLE}:{REVIEW}"

    def test_should_drop_a_seat_from_a_login_its_tag_does_not_name(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1", "claude-2")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        found = {CLAUDE: seat("running", login="claude-2")}
        seats(client, aws, store).step(queued=[], found=found)
        assert "holders" not in store.tags[f"{PREFIX}claude-1"]
        assert store.tags[f"{PREFIX}claude-2"]["holders"] == f"{ROLE}:{CLAUDE}"

    def test_should_start_another_seat_while_one_is_at_its_limit(
        self, client: Client
    ) -> None:
        aws = Instances()
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {
            CLAUDE: seat("running", login="claude-1", until=until),
            CLAUDE_2: seat("stopped", name="claude-2"),
        }
        seats(client, aws).step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE_2]
        assert aws.tags[CLAUDE_2] == {"login": "claude-2"}

    def test_should_start_a_seat_once_its_login_limit_resets(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        store.tags[f"{PREFIX}claude-1"]["limited-until"] = seconds(until)
        s = seats(client, aws, store)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        assert aws.started == []
        client.elapsed += sy.TimeSpan.HOUR + sy.TimeSpan.SECOND
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

    def test_should_log_a_failed_login_read_and_start_nothing(
        self, client: Client
    ) -> None:
        aws = Instances()
        store = secrets("claude-1")
        store.tags[f"{PREFIX}claude-1"] = {"agent": "claude"}
        seats(client, aws, store).step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.started == []
        assert client.written("log") == [
            ["[ERROR] login read failed: KeyError: 'expires'"]
        ]

    def test_should_write_nothing_while_disabled(self, client: Client) -> None:
        aws = Instances()
        store = secrets("claude-1")
        store.tags[f"{PREFIX}claude-1"]["holders"] = f"{ROLE}:{CLAUDE}"
        until = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        found = {
            CLAUDE: seat("stopped", login="claude-1", until=until),
            CLAUDE_2: seat("stopped", name="claude-2"),
        }
        seats(client, aws, store).step(
            queued=[{"seat-claude-2"}], found=found, enabled=False
        )
        assert store.tags[f"{PREFIX}claude-1"]["holders"] == f"{ROLE}:{CLAUDE}"
        assert "limited-until" not in store.tags[f"{PREFIX}claude-1"]
        assert aws.tags == {}
        assert aws.started == []
