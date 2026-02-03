#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time

import pytest

import synnax as sy


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


@pytest.mark.control
class TestController:
    def test_valve_toggle(self, client: sy.Synnax):
        """A happy path test that asserts basic functionality on the controller"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence(ev: threading.Event):
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

        def daq(ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False

        ev = threading.Event()
        t1 = threading.Thread(target=sequence, kwargs={"ev": ev})
        t2 = threading.Thread(target=daq, kwargs={"ev": ev})

        t2.start()
        t1.start()

        t1.join()
        t2.join()

        assert assertions["seq_first_ack"]
        assert assertions["seq_second_ack"]

    def test_remains_true_for_false(self, client: sy.Synnax):
        """Test that the controller can wait for a condition to be true for a certain amount of time"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence(ev: threading.Event):
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
                remained_true = auto.remains_true_for(
                    lambda c: not c[press_en.key],
                    duration=100 * sy.TimeSpan.MILLISECOND,
                )
                assertions["remained_true"] = remained_true

        def daq(ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False
                auto[press_en.key] = False
                auto.sleep(50 * sy.TimeSpan.MILLISECOND)
                auto[press_en.key] = True

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

    def test_remains_true_for_true(self, client: sy.Synnax):
        """Test that the controller can wait for a condition to be true for a certain amount of time"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence(ev: threading.Event):
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
                remained_true = auto.remains_true_for(
                    lambda c: not c[press_en.key],
                    duration=100 * sy.TimeSpan.MILLISECOND,
                )
                assertions["remained_true"] = remained_true

        def daq(ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False
                auto[press_en.key] = False
                auto.sleep(50 * sy.TimeSpan.MILLISECOND)
                auto[press_en.key] = False

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

    def test_remains_true_for_target_percent(self, client: sy.Synnax):
        """Test that the controller can wait for a condition to be true for a certain amount of time"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence(ev: threading.Event):
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

                def is_closed(auto):
                    nonlocal c
                    c += 1
                    return not auto[press_en.key]

                remained_true = auto.remains_true_for(
                    is_closed,
                    duration=100 * sy.TimeSpan.MILLISECOND,
                    percentage=0.5,
                )
                assertions["remained_true"] = remained_true
                assertions["remained_true_count"] = c

        def daq(ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False
                auto[press_en.key] = False
                auto[press_en.key] = True
                auto[press_en.key] = False
                auto[press_en.key] = True
                auto[press_en.key] = True
                auto[press_en.key] = False
                auto[press_en.key] = False

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

    def test_wait_while(self, client: sy.Synnax):
        """Test that the controller can wait for a condition to be true for a certain amount of time"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence(ev: threading.Event):
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

                def is_closed(auto):
                    nonlocal c
                    c += 1
                    return not auto[press_en.key]

                remained_true = auto.wait_while(is_closed)
                assertions["remained_true"] = remained_true
                assertions["remained_true_count"] = c

        def daq(ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False
                auto.sleep(50 * sy.TimeSpan.MILLISECOND)
                auto[press_en.key] = False
                auto[press_en.key] = False
                auto[press_en.key] = False
                auto[press_en.key] = True

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
        assert assertions["remained_true_count"] == 4

    def test_controller_channel_not_found(self, client: sy.Synnax):
        """Test that the controller raises a KeyError when a channel is not found"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)
        with pytest.raises(KeyError, match="Channel .* not found in controller state"):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                v = auto[press_en.key]
                assert v is None

    def test_controller_set_authority_mechanisms(self, client: sy.Synnax):
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

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
            auto.set_authority(press_en_cmd.key, 50)

    def test_controller_authority_transfer(self, client: sy.Synnax):
        """Test that the controller can transfer authority to another controller"""
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        assertions = dict()

        def sequence_one(daq_ev: threading.Event, seq_two_ev: threading.Event):
            daq_ev.wait()
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write_authorities=[100],
                write=[press_en_cmd.key],
            ) as auto:
                seq_two_ev.wait()
                auto[press_en_cmd.key] = True
                assertions["seq_one_first_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=50 * sy.TimeSpan.MILLISECOND,
                )
                time.sleep(0.15)
                auto[press_en_cmd.key] = True
                assertions["seq_one_second_ack"] = auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=50 * sy.TimeSpan.MILLISECOND,
                )
                auto[press_en_cmd.key] = False

        def sequence_two(daq_ev: threading.Event, seq_two_ev: threading.Event):
            daq_ev.wait()
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
                write_authorities=[255],
            ) as auto:
                seq_two_ev.set()
                # We use a sleep here instead of an auto.wait to ensure python has
                # bandwidth to switch threads
                auto.sleep(0.1)
                auto.set_authority({press_en_cmd.key: 50})
                # We use a sleep here instead of an auto.wait to ensure python has
                # bandwidth to switch threads
                auto.sleep(0.2)

        def daq(daq_ev: threading.Event):
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                daq_ev.set()
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False

        daq_ev = threading.Event()
        seq_two_ev = threading.Event()
        t1 = threading.Thread(
            target=sequence_one, kwargs={"daq_ev": daq_ev, "seq_two_ev": seq_two_ev}
        )
        t2 = threading.Thread(
            target=sequence_two, kwargs={"daq_ev": daq_ev, "seq_two_ev": seq_two_ev}
        )
        t3 = threading.Thread(target=daq, kwargs={"daq_ev": daq_ev})

        t3.start()
        t1.start()
        t2.start()
        t1.join()
        t2.join()
        t3.join()

        assert assertions["seq_one_first_ack"] is False
        assert assertions["seq_one_second_ack"] is True
