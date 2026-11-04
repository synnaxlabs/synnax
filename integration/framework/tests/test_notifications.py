#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

import pytest
from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
from console.notifications import NotificationsClient

CLEAR = "clear"
CLICK = "click"
ATTEMPTS_TO_DEADLINE = 10


class FakeTimer(sy.Timer):
    def __init__(self) -> None:
        self.now = sy.TimeSpan(0)

    def elapsed(self) -> sy.TimeSpan:
        return self.now


class FakeLocator:
    """Times out for the first `failures` clicks, then lands. Each click takes 1 s."""

    def __init__(self, timer: FakeTimer, events: list[str], *, failures: int) -> None:
        self.timer = timer
        self.events = events
        self.failures = failures
        self.errors: list[PlaywrightTimeoutError] = []

    def click(self, *, timeout: int) -> None:
        self.events.append(CLICK)
        self.timer.now = self.timer.now + sy.TimeSpan.SECOND
        if len(self.errors) < self.failures:
            self.errors.append(PlaywrightTimeoutError(f"Timeout {timeout}ms exceeded"))
            raise self.errors[-1]


class FakeNotificationsClient(NotificationsClient):
    def __init__(self, timer: FakeTimer, events: list[str]) -> None:
        super().__init__(cast(Page, None), new_timer=lambda: timer)
        self.events = events

    def close_all(self) -> None:
        self.events.append(CLEAR)


def click_events(*, failures: int) -> list[str]:
    timer = FakeTimer()
    events: list[str] = []
    locator = FakeLocator(timer, events, failures=failures)
    FakeNotificationsClient(timer, events).clear_and_click(cast(Locator, locator))
    return events


class TestClearAndClick:
    def test_should_clear_then_click_once_when_the_click_lands(self) -> None:
        assert click_events(failures=0) == [CLEAR, CLICK]

    def test_should_clear_again_before_each_retry(self) -> None:
        assert click_events(failures=2) == [CLEAR, CLICK] * 3

    def test_should_raise_the_last_timeout_at_the_deadline(self) -> None:
        timer = FakeTimer()
        events: list[str] = []
        locator = FakeLocator(timer, events, failures=ATTEMPTS_TO_DEADLINE + 1)
        notifications = FakeNotificationsClient(timer, events)
        with pytest.raises(PlaywrightTimeoutError) as raised:
            notifications.clear_and_click(cast(Locator, locator))
        assert raised.value is locator.errors[-1]
        assert events == [CLEAR, CLICK] * ATTEMPTS_TO_DEADLINE
