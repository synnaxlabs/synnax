#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import pytest

from x.testutil import assert_eventually


class TestAssertEventually:
    def test_returns_immediately_when_condition_passes(self) -> None:
        """Should return on the first attempt if condition does not raise."""
        calls = 0

        def condition() -> None:
            nonlocal calls
            calls += 1

        assert_eventually(condition)
        assert calls == 1

    def test_retries_on_assertion_error_until_success(self) -> None:
        """Should retry on AssertionError and return once condition passes."""
        calls = 0

        def condition() -> None:
            nonlocal calls
            calls += 1
            if calls < 3:
                raise AssertionError("not yet")

        assert_eventually(condition, timeout=1.0, interval=0.01)
        assert calls == 3

    def test_retries_on_arbitrary_exception_until_success(self) -> None:
        """Should retry on any Exception subclass, not just AssertionError."""
        calls = 0

        def condition() -> None:
            nonlocal calls
            calls += 1
            if calls < 2:
                raise RuntimeError("not ready")

        assert_eventually(condition, timeout=1.0, interval=0.01)
        assert calls == 2

    def test_propagates_assertion_error_after_timeout(self) -> None:
        """Should re-raise the condition's AssertionError after the deadline."""

        def condition() -> None:
            raise AssertionError("never satisfied")

        with pytest.raises(AssertionError, match="never satisfied"):
            assert_eventually(condition, timeout=0.05, interval=0.01)

    def test_propagates_non_assertion_exception_after_timeout(self) -> None:
        """Should re-raise non-AssertionError exceptions after the deadline."""

        def condition() -> None:
            raise ValueError("custom failure")

        with pytest.raises(ValueError, match="custom failure"):
            assert_eventually(condition, timeout=0.05, interval=0.01)

    def test_does_not_swallow_keyboard_interrupt(self) -> None:
        """Should let KeyboardInterrupt and other BaseExceptions propagate."""

        def condition() -> None:
            raise KeyboardInterrupt

        with pytest.raises(KeyboardInterrupt):
            assert_eventually(condition, timeout=1.0, interval=0.01)

    def test_respects_timeout(self) -> None:
        """Should stop retrying within roughly the configured timeout."""

        def condition() -> None:
            raise AssertionError("nope")

        start = time.monotonic()
        with pytest.raises(AssertionError):
            assert_eventually(condition, timeout=0.1, interval=0.01)
        elapsed = time.monotonic() - start
        assert elapsed < 1.0

    def test_ignores_condition_return_value(self) -> None:
        """Should accept callables returning any value (return type is ignored)."""

        def returns_int() -> int:
            return 42

        def returns_list() -> list[str]:
            return ["a", "b"]

        assert_eventually(returns_int)
        assert_eventually(returns_list)
