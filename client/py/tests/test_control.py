#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import threading
import time
from collections.abc import Callable
from typing import cast

import numpy as np
import pytest

import freighter
import synnax as sy
from synnax.control import ScheduledCommand
from synnax.control.controller import Controller, _RemainsTrueFor
from tests.channel import RetrieveCounter
from x.strings import random_name

# Seconds to wait for a cross-thread handshake. Generous, so a slow Core cannot fail the
# test, but bounded, so a broken run fails instead of hanging CI.
HANDSHAKE_TIMEOUT = 5
# A wait timeout long enough that only a stopped receiver can wake the wait in time.
BLOCKED_WAIT = 30 * sy.TimeSpan.SECOND


def create_valve_set(
    client: sy.Synnax,
) -> tuple[sy.Channel, sy.Channel, sy.Channel, sy.Channel]:
    press_end_cmd_time = client.channels.create(
        name="press_en_cmd_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    press_en_cmd = client.channels.create(
        name="press_en_cmd",
        data_type=sy.DataType.UINT8,
        index=press_end_cmd_time.key,
        retrieve_if_name_exists=True,
    )
    daq_time = client.channels.create(
        name="daq_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    press_en = client.channels.create(
        name="press_en",
        data_type=sy.DataType.UINT8,
        index=daq_time.key,
        retrieve_if_name_exists=True,
    )
    return press_end_cmd_time, press_en_cmd, press_en, daq_time


def acknowledge_valve_cycle(
    auto: Controller,
    cmd: sy.Channel,
    state: sy.Channel,
    ready: threading.Event,
) -> bool:
    """Acknowledges one open-then-close command cycle as a simulated DAQ would.

    Signals ready, then drives state high once the open command arrives and low once
    it clears.

    :param auto: A controller reading cmd and writing state.
    :param cmd: The command channel the sequence under test writes.
    :param state: The state channel this DAQ answers with.
    :param ready: Set once the DAQ is listening, so sequences can start commanding.
    :returns: False if either command failed to arrive within HANDSHAKE_TIMEOUT. The
        caller must stop, because answering a command the Core never delivered would
        acknowledge a command that was never seen.
    """
    ready.set()
    if not auto.wait_until(lambda c: c[cmd.key], timeout=HANDSHAKE_TIMEOUT):
        return False
    auto[state.key] = True
    if not auto.wait_until(lambda c: not c[cmd.key], timeout=HANDSHAKE_TIMEOUT):
        return False
    auto[state.key] = False
    return True


@pytest.mark.control
class TestController:
    def test_valve_toggle(self, client: sy.Synnax) -> None:
        """A happy path test that asserts basic functionality on the controller"""
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions = {}

        def sequence(ev: threading.Event) -> None:
            # Wait for the simulated DAQ to boot up
            ev.wait()
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
            ) as auto:
                auto[press_en_cmd.key] = True
                assertions["seq_first_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )
                auto[press_en_cmd.key] = False
                assertions["seq_second_ack"] = auto.wait_until(
                    lambda c: not c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )

        def daq(ev: threading.Event) -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                acknowledge_valve_cycle(auto, press_en_cmd, press_en, ev)

        ev = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()

        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]

    def test_remains_true_for_false(self, client: sy.Synnax) -> None:
        """Should return False when the condition stops holding during the window."""
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions = {}
        acked = threading.Event()
        stop = threading.Event()

        def sequence(ev: threading.Event) -> None:
            # Wait for the simulated DAQ to boot up
            ev.wait()
            try:
                with client.control.acquire(
                    name="Basic Valve Toggle",
                    read=[press_en.key],
                    write=[press_en_cmd.key],
                ) as auto:
                    auto[press_en_cmd.key] = True
                    assertions["seq_first_ack"] = auto.wait_until(
                        lambda c: c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    auto[press_en_cmd.key] = False
                    assertions["seq_second_ack"] = auto.wait_until(
                        lambda c: not c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    acked.set()
                    assertions["remained_true"] = auto.remains_true_for(
                        lambda c: not c[press_en.key],
                        duration=100 * sy.TimeSpan.MILLISECOND,
                    )
            finally:
                stop.set()

        def daq(ev: threading.Event) -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                if not acknowledge_valve_cycle(auto, press_en_cmd, press_en, ev):
                    return
                # Keep publishing closed-valve samples until the sequence has observed
                # the acknowledgment, so the second wait_until always fires, then keep
                # publishing open-valve samples until the sequence finishes, so the
                # observation window is guaranteed to see the condition fail no matter
                # when it starts.
                while not acked.is_set() and not stop.is_set():
                    auto[press_en.key] = False
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)
                while not stop.is_set():
                    auto[press_en.key] = True
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)

        ev = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()

        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]
        assert not assertions["remained_true"]

    def test_remains_true_for_true(self, client: sy.Synnax) -> None:
        """Should return True when the condition holds for the entire window."""
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions = {}
        stop = threading.Event()

        def sequence(ev: threading.Event) -> None:
            # Wait for the simulated DAQ to boot up
            ev.wait()
            try:
                with client.control.acquire(
                    name="Basic Valve Toggle",
                    read=[press_en.key],
                    write=[press_en_cmd.key],
                ) as auto:
                    auto[press_en_cmd.key] = True
                    assertions["seq_first_ack"] = auto.wait_until(
                        lambda c: c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    auto[press_en_cmd.key] = False
                    assertions["seq_second_ack"] = auto.wait_until(
                        lambda c: not c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    assertions["remained_true"] = auto.remains_true_for(
                        lambda c: not c[press_en.key],
                        duration=100 * sy.TimeSpan.MILLISECOND,
                    )
            finally:
                stop.set()

        def daq(ev: threading.Event) -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                if not acknowledge_valve_cycle(auto, press_en_cmd, press_en, ev):
                    return
                # Keep publishing closed-valve samples until the sequence finishes, so
                # the observation window always contains samples no matter when it
                # starts.
                while not stop.is_set():
                    auto[press_en.key] = False
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)

        ev = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()

        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]
        assert assertions["remained_true"]

    def test_remains_true_for_target_percent(self, client: sy.Synnax) -> None:
        """Should return True when the condition holds for more than the target
        percentage of the window."""
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions: dict[str, bool | int] = {}
        acked = threading.Event()
        stop = threading.Event()

        def sequence(ev: threading.Event) -> None:
            # Wait for the simulated DAQ to boot up
            ev.wait()
            try:
                with client.control.acquire(
                    name="Basic Valve Toggle",
                    read=[press_en.key],
                    write=[press_en_cmd.key],
                ) as auto:
                    auto[press_en_cmd.key] = True
                    assertions["seq_first_ack"] = auto.wait_until(
                        lambda c: c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    auto[press_en_cmd.key] = False
                    assertions["seq_second_ack"] = auto.wait_until(
                        lambda c: not c[press_en.key],
                        timeout=2 * sy.TimeSpan.SECOND,
                    )
                    acked.set()
                    c = 0

                    def is_closed(auto: Controller) -> bool:
                        nonlocal c
                        c += 1
                        return not auto[press_en.key]

                    assertions["remained_true"] = auto.remains_true_for(
                        is_closed,
                        duration=100 * sy.TimeSpan.MILLISECOND,
                        percentage=0.5,
                    )
                    assertions["remained_true_count"] = c
            finally:
                stop.set()

        def daq(ev: threading.Event) -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                if not acknowledge_valve_cycle(auto, press_en_cmd, press_en, ev):
                    return
                # Keep publishing closed-valve samples until the sequence has observed
                # the acknowledgment, so the second wait_until always fires. Then keep
                # the valve closed for three out of every four samples (75% closed,
                # comfortably above the 50% target) until the sequence finishes.
                # Starting the pattern with its closed run right as the observation
                # window opens means even a heavily-truncated window sees a
                # closed-dominated prefix of the pattern.
                while not acked.is_set() and not stop.is_set():
                    auto[press_en.key] = False
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)
                i = 0
                while not stop.is_set():
                    auto[press_en.key] = i % 4 == 3
                    i += 1
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)

        ev = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()

        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]
        assert assertions["remained_true"]
        assert assertions["remained_true_count"] > 0

    def test_remains_true_for_bare_number_is_seconds(self, client: sy.Synnax) -> None:
        """Should read a bare number duration as seconds."""
        _, _, press_en, _ = create_valve_set(client)
        with client.control.acquire(
            name="Bare duration", read=[press_en.key], write=None
        ) as auto:
            start = time.perf_counter()
            auto.remains_true_for(lambda _: True, duration=0.2)
            assert time.perf_counter() - start >= 0.2

    def test_wait_while(self, client: sy.Synnax) -> None:
        """Test that the controller can wait for a condition to be true for a certain
        amount of time
        """
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions: dict[str, bool | int] = {}

        def sequence(ev: threading.Event) -> None:
            # Wait for the simulated DAQ to boot up
            ev.wait()
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
            ) as auto:
                auto[press_en_cmd.key] = True
                assertions["seq_first_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )
                auto[press_en_cmd.key] = False
                assertions["seq_second_ack"] = auto.wait_until(
                    lambda c: not c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )
                c = 0

                def is_closed(auto: Controller) -> bool:
                    nonlocal c
                    c += 1
                    observing.set()
                    return not auto[press_en.key]

                remained_true = auto.wait_while(is_closed, timeout=HANDSHAKE_TIMEOUT)
                assertions["remained_true"] = remained_true
                assertions["remained_true_count"] = c

        def daq(ev: threading.Event) -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                if not acknowledge_valve_cycle(auto, press_en_cmd, press_en, ev):
                    return
                # Publish closed samples until the sequence's callback answers one, so
                # the opening sample can never arrive before wait_while registers.
                timer = sy.Timer()
                while not observing.is_set():
                    if timer.elapsed() > HANDSHAKE_TIMEOUT * sy.TimeSpan.SECOND:
                        return
                    auto[press_en.key] = False
                    auto.sleep(5 * sy.TimeSpan.MILLISECOND)
                auto[press_en.key] = True

        ev = threading.Event()
        observing = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()
        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]
        assert assertions["remained_true"]
        assert assertions["remained_true_count"] > 0

    def test_controller_channel_not_found(self, client: sy.Synnax) -> None:
        """Test that the controller raises a KeyError when a channel is not found"""
        _, press_en_cmd, press_en, _ = create_valve_set(client)
        with pytest.raises(KeyError, match=r"Channel .* not found in controller state"):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                v = auto[press_en.key]
                assert v is None

    def test_set_by_attribute(self, client: sy.Synnax) -> None:
        """Should command the channel named by an assigned attribute."""
        ch = client.channels.create(
            name="control_set_by_attribute",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.control.acquire(
            name="Set By Attribute", read=[ch.key], write=[ch.key]
        ) as auto:
            auto.control_set_by_attribute = 42
            assert auto.wait_until(
                lambda c: c.get(ch.key) == 42, timeout=HANDSHAKE_TIMEOUT
            )
            assert "control_set_by_attribute" not in vars(auto)

    def test_acquire_raises_when_streamer_fails_to_open(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should raise the streamer's open error instead of hanging."""
        ch = client.channels.create(
            name="control_streamer_open_fails",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
        )
        host, port, username, password = login_info
        subject = sy.Synnax(
            host=host, port=port, username=username, password=password, secure=False
        )

        async def delete_before_open(
            ctx: freighter.Context, next: freighter.AsyncNext
        ) -> freighter.Context:
            # The transports put the URL in protocol, so check both fields.
            if "/frame/stream" in ctx.target + ctx.protocol:
                client.channels.delete(ch.key)
            return await next(ctx)

        subject._transport.use_async(delete_before_open)
        with pytest.raises(freighter.StreamClosed):
            subject.control.acquire(name="Open Fails", read=[ch.key], write=None)

    @pytest.mark.parametrize(
        "wait",
        [
            lambda auto: auto.wait_until(lambda _: False, timeout=BLOCKED_WAIT),
            lambda auto: auto.remains_true_for(lambda _: True, duration=BLOCKED_WAIT),
        ],
        ids=["wait_until", "remains_true_for"],
    )
    def test_wait_raises_when_receiver_stops(
        self, client: sy.Synnax, wait: Callable[[Controller], bool]
    ) -> None:
        """Should wake a blocked wait with an error once the receiver stops."""
        ch = client.channels.create(
            name="control_receiver_stops",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.control.acquire(
            name="Receiver Stops", read=[ch.key], write=[ch.key]
        ) as auto:
            receiver = auto._receiver
            errors: list[Exception] = []

            def run() -> None:
                try:
                    wait(auto)
                except Exception as e:
                    errors.append(e)

            waiter = threading.Thread(target=run, daemon=True)
            waiter.start()
            deadline = time.monotonic() + HANDSHAKE_TIMEOUT
            while not receiver.processors:
                if errors:
                    raise errors[0]
                assert time.monotonic() < deadline, "waiter never registered"
                time.sleep(0.01)
            asyncio.run_coroutine_threadsafe(
                receiver.streamer.close_loop(), receiver.loop
            ).result(HANDSHAKE_TIMEOUT)
            waiter.join(HANDSHAKE_TIMEOUT)
            assert not waiter.is_alive(), "blocked wait was not woken"
            assert len(errors) == 1
            assert isinstance(errors[0], sy.UnexpectedError)

    def test_controller_set_authority_mechanisms(self, client: sy.Synnax) -> None:
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        with client.control.acquire(
            name="Basic Valve Toggle",
            read=[press_en.key],
            write=[press_en_cmd.key],
            write_authorities=[100],
        ) as auto:
            auto[press_en_cmd.key] = True
            auto.set_authority(
                {
                    press_en_cmd.key: 50,
                    press_en.key: 50,
                }
            )
            auto.set_authority(50)
            auto.set_authority(100)
            auto.set_authority(press_en_cmd.key, authority=50)

    def test_controller_authority_transfer(self, client: sy.Synnax) -> None:
        """Test that the controller can transfer authority to another controller"""
        _, press_en_cmd, press_en, _ = create_valve_set(client)

        assertions = {}
        daq_ready = threading.Event()
        seq_two_acquired = threading.Event()
        seq_one_blocked = threading.Event()
        seq_two_lowered = threading.Event()
        seq_one_done = threading.Event()

        def sequence_one() -> None:
            daq_ready.wait(HANDSHAKE_TIMEOUT)
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
                write_authorities=[100],
            ) as auto:
                seq_two_acquired.wait(HANDSHAKE_TIMEOUT)
                auto[press_en_cmd.key] = True
                assertions["seq_one_first_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=250 * sy.TimeSpan.MILLISECOND,
                )
                seq_one_blocked.set()
                seq_two_lowered.wait(HANDSHAKE_TIMEOUT)
                auto[press_en_cmd.key] = True
                assertions["seq_one_second_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=5 * sy.TimeSpan.SECOND,
                )
                seq_one_done.set()
                auto[press_en_cmd.key] = False

        def sequence_two() -> None:
            daq_ready.wait(HANDSHAKE_TIMEOUT)
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
                write_authorities=[255],
            ) as auto:
                seq_two_acquired.set()
                seq_one_blocked.wait(HANDSHAKE_TIMEOUT)
                # set_authority blocks until the Core acknowledges it, so sequence one
                # commands only after the lower authority is in effect. Sequence two
                # holds its writer open until sequence one is done, so the commands go
                # through because of the authority change alone.
                auto.set_authority({press_en_cmd.key: 50})
                seq_two_lowered.set()
                seq_one_done.wait(HANDSHAKE_TIMEOUT)

        def daq() -> None:
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                acknowledge_valve_cycle(auto, press_en_cmd, press_en, daq_ready)

        t1 = threading.Thread(target=sequence_one)
        t2 = threading.Thread(target=sequence_two)
        t3 = threading.Thread(target=daq)

        t3.start()
        t1.start()
        t2.start()
        t1.join()
        t2.join()
        t3.join()

        assert assertions["seq_one_first_ack"] is False
        assert assertions["seq_one_second_ack"] is True


@pytest.mark.control
class TestControllerChannelResolution:
    """Tests that a controller resolves its channels once, when it acquires them."""

    def test_access_by_name_sends_no_retrieves(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should read, command, and set authority by name without retrieving"""
        client, counter = counted_client
        read = client.channels.create(
            name=f"test_{random_name()}", virtual=True, data_type=sy.DataType.FLOAT64
        )
        cmd = client.channels.create(
            name=f"test_{random_name()}", virtual=True, data_type=sy.DataType.FLOAT64
        )
        with client.control.acquire(
            name="resolution", read=[read.name], write=[cmd.name]
        ) as ctrl:
            counter.count = 0
            for i in range(100):
                ctrl.get(read.name)
                ctrl.set(cmd.name, i)
                ctrl.set({cmd.name: i})
                ctrl.set_authority(cmd.name, authority=200)
            assert counter.count == 0

    def test_access_by_numeric_string_key(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should treat a numeric string as a channel key, as JSON object keys are"""
        client, counter = counted_client
        cmd = client.channels.create(
            name=f"test_{random_name()}", virtual=True, data_type=sy.DataType.FLOAT64
        )
        key = str(cmd.key)
        with client.control.acquire(
            name="resolution", read=[cmd.key], write=[cmd.key]
        ) as ctrl:
            counter.count = 0
            ctrl.set_authority({key: 200})
            ctrl.set({key: 7.0})
            assert ctrl.wait_until(
                lambda c: c.get(key) == 7.0, timeout=HANDSHAKE_TIMEOUT
            )
            assert counter.count == 0

    def test_unacquired_channel_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve a channel that was not acquired from the Core"""
        read = client.channels.create(
            name=f"test_{random_name()}", virtual=True, data_type=sy.DataType.FLOAT64
        )
        other = client.channels.create(
            name=f"test_{random_name()}", virtual=True, data_type=sy.DataType.FLOAT64
        )
        with client.control.acquire(
            name="resolution", read=[read.name], write=None
        ) as ctrl:
            assert ctrl.get(other.name, 42) == 42
            with pytest.raises(sy.NotFoundError):
                ctrl.get(f"test_{random_name()}")


@pytest.mark.control
class TestRemainsTrueForProcessor:
    """Unit tests for the _RemainsTrueFor percentage logic, independent of streaming
    timing."""

    # The callbacks under test never touch the controller, so a bare sentinel stands in
    # for it.
    _state = cast(Controller, object())

    def _run(self, results: list[bool], percentage: float) -> _RemainsTrueFor:
        it = iter(results)
        processor = _RemainsTrueFor(lambda _: next(it), percentage)
        for _ in results:
            processor.process(self._state)
        return processor

    def test_all_true_meets_full_target(self) -> None:
        """Should never set the exit event and track a 100% true rate."""
        processor = self._run([True, True, True], percentage=1)
        assert not processor.event.is_set()
        assert processor.count == 3
        assert processor.actual == 1

    def test_partial_true_meets_lower_target(self) -> None:
        """Should track the running percentage of true results."""
        processor = self._run([True, False, True, True], percentage=0.5)
        assert not processor.event.is_set()
        assert processor.count == 4
        assert processor.actual == pytest.approx(0.75)

    def test_partial_true_below_target(self) -> None:
        """Should report an actual percentage below the target."""
        processor = self._run([False, False, True, False], percentage=0.5)
        assert not processor.event.is_set()
        assert processor.count == 4
        assert processor.actual == pytest.approx(0.25)

    def test_exits_immediately_on_false_when_target_is_full(self) -> None:
        """Should set the exit event as soon as the condition fails when the target
        percentage is 1."""
        processor = self._run([True, False], percentage=1)
        assert processor.event.is_set()
        assert processor.count == 1

    def test_exits_immediately_on_numpy_false(self) -> None:
        """Should treat a NumPy false result as a failed condition."""
        processor = _RemainsTrueFor(lambda _: np.int64(3) > 5, percentage=1)
        processor.process(self._state)
        assert processor.event.is_set()
        assert processor.count == 0

    def test_counts_numeric_results_as_booleans(self) -> None:
        """Should count a truthy numeric result as one true result, not as its
        value."""
        it = iter([2, 0, 2, 0])
        processor = _RemainsTrueFor(lambda _: cast(bool, next(it)), percentage=0.75)
        for _ in range(4):
            processor.process(self._state)
        assert processor.count == 4
        assert processor.actual == pytest.approx(0.5)

    def test_captures_callback_exception(self) -> None:
        """Should capture an exception raised by the callback and exit."""

        def boom(_: Controller) -> bool:
            raise ValueError("boom")

        processor = _RemainsTrueFor(boom, percentage=1)
        processor.process(self._state)
        assert isinstance(processor.exc, ValueError)
        assert processor.event.is_set()


class TestScheduledCommand:
    def test_bare_number_delay_is_seconds(self) -> None:
        """Should read a bare number delay as seconds."""
        assert (
            ScheduledCommand(channel="ch", value=1, delay=2).delay
            == 2 * sy.TimeSpan.SECOND
        )

    def test_time_span_delay_passes_through(self) -> None:
        """Should keep a TimeSpan delay unchanged."""
        delay = 5 * sy.TimeSpan.MILLISECOND
        assert ScheduledCommand(channel="ch", value=1, delay=delay).delay == delay


@pytest.mark.control
class TestRetrieve:
    def test_retrieve_held_channel(self, client: sy.Synnax) -> None:
        """Should return the state of a channel a writer holds."""
        ch = client.channels.create(
            name="control_retrieve_held",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[ch.key], name="philadelphia"
        ):
            state = client.control.retrieve(key=ch.key)
            assert state is not None
            assert state.resource == ch.key
            assert state.subject.name == "philadelphia"

    def test_retrieve_uncontrolled_channel(self, client: sy.Synnax) -> None:
        """Should return None for a channel no subject controls."""
        ch = client.channels.create(
            name="control_retrieve_unheld",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        assert client.control.retrieve(key=ch.key) is None

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should resolve a channel name before reading its state."""
        ch = client.channels.create(
            name="control_retrieve_by_name",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[ch.key], name="denver"
        ):
            state = client.control.retrieve(name=ch.name)
            assert state is not None
            assert state.subject.name == "denver"

    def test_retrieve_all(self, client: sy.Synnax) -> None:
        """Should return every controlled channel when no params are given."""
        ch = client.channels.create(
            name="control_retrieve_all",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[ch.key], name="austin"
        ):
            states = client.control.retrieve()
            assert any(s.resource == ch.key for s in states)

    def test_retrieve_several(self, client: sy.Synnax) -> None:
        """Should return the state of each held channel and omit the rest."""
        held = client.channels.create(
            name="control_retrieve_several_held",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        free = client.channels.create(
            name="control_retrieve_several_free",
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[held.key], name="boulder"
        ):
            states = client.control.retrieve(keys=[held.key, free.key])
            assert [s.resource for s in states] == [held.key]
            states = client.control.retrieve(names=[held.name, free.name])
            assert [s.resource for s in states] == [held.key]

    @pytest.mark.parametrize("params", [{"keys": []}, {"names": []}])
    def test_retrieve_empty_params(
        self, client: sy.Synnax, params: dict[str, list[str]]
    ) -> None:
        """Should return no states when an empty set of channels is requested."""
        assert client.control.retrieve(**params) == []

    def test_retrieve_unknown_names(self, client: sy.Synnax) -> None:
        """Should return no states when no requested name resolves to a channel."""
        assert client.control.retrieve(names=["control_retrieve_does_not_exist"]) == []
