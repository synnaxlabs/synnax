#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI invalid configuration integration tests.

Each test attempts to configure a task with an invalid setting and verifies
the driver rejects it with a ConfigurationError.
"""

import platform

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import create_channel, create_index


def _cleanup_task(client: sy.Synnax, task: sy.Task) -> None:
    """Delete the task if it was assigned a key during configure."""
    if task.key and task.key != 0:
        try:
            client.tasks.delete(task.key)
        except sy.NotFoundError:
            pass


class NIInvalidConfig(TestCase):
    """Verify the driver rejects invalid NI task configurations.

    Tests (run sequentially):
        1. Invalid port — port number that doesn't exist on the device.
        2. Incorrect channel type — RTD channel on a voltage-only module.
        3. Out-of-range values — min/max outside hardware capabilities.
        4. Duplicate channel — two tasks using the same channel on the same port.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="Windows DAQmx drivers required")
            return
        self.devices = {
            loc: self.client.devices.retrieve(location=loc) for loc in ["E101Mod4"]
        }

    def run(self) -> None:
        self.test_invalid_port()
        self.test_incorrect_task_type()
        self.test_out_of_range_values()
        self.test_duplicate_channel()

    def test_invalid_port(self) -> None:
        """Configure an analog read task with a nonexistent port (99)."""
        self.log("Testing: Invalid port (port 99 on NI 9205)")
        idx = create_index(self.client, "ni_invalid_port_idx")
        task = sy.ni.AnalogReadTask(
            name="NI Invalid Port Test",
            device=self.devices["E101Mod4"].key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            channels=[
                sy.ni.AIVoltageChan(
                    port=99,
                    channel=create_channel(
                        self.client,
                        name="ni_invalid_port_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    terminal_config="Cfg_Default",
                    min_val=-10.0,
                    max_val=10.0,
                ),
            ],
        )
        self._assert_configure_fails(task, "invalid port")

    def test_incorrect_task_type(self) -> None:
        """Configure an RTD channel on a voltage-only module (NI 9205)."""
        self.log("Testing: Incorrect channel type (RTD on E101Mod4 / NI 9205)")
        idx = create_index(self.client, "ni_wrong_type_idx")
        task = sy.ni.AnalogReadTask(
            name="NI Wrong Channel Type Test",
            device=self.devices["E101Mod4"].key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            channels=[
                sy.ni.AIRTDChan(
                    port=0,
                    channel=create_channel(
                        self.client,
                        name="ni_wrong_type_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    units="DegC",
                    rtd_type="Pt3851",
                    resistance_config="3Wire",
                    current_excit_source="Internal",
                    current_excit_val=0.001,
                    r0=100.0,
                    min_val=0.0,
                    max_val=100.0,
                ),
            ],
        )
        self._assert_configure_fails(task, "incorrect channel type")

    def test_out_of_range_values(self) -> None:
        """Configure an analog read task with min/max far outside hardware limits."""
        self.log("Testing: Out-of-range values (±1000V on NI 9205)")
        idx = create_index(self.client, "ni_oor_idx")
        task = sy.ni.AnalogReadTask(
            name="NI Out-of-Range Test",
            device=self.devices["E101Mod4"].key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            channels=[
                sy.ni.AIVoltageChan(
                    port=0,
                    channel=create_channel(
                        self.client,
                        name="ni_oor_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    terminal_config="Cfg_Default",
                    min_val=-1000.0,
                    max_val=1000.0,
                ),
            ],
        )
        self._assert_configure_fails(task, "out-of-range values")

    def test_duplicate_channel(self) -> None:
        """Configure and run two tasks that use the same channel on the same port."""
        self.log("Testing: Duplicate channel (two tasks on E101Mod4 port 0)")
        idx = create_index(self.client, "ni_dup_ch_idx")
        shared_ch_key = create_channel(
            self.client,
            name="ni_dup_ch",
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )

        def _make_task(name: str) -> sy.ni.AnalogReadTask:
            return sy.ni.AnalogReadTask(
                name=name,
                device=self.devices["E101Mod4"].key,
                sample_rate=100 * sy.Rate.HZ,
                stream_rate=25 * sy.Rate.HZ,
                channels=[
                    sy.ni.AIVoltageChan(
                        port=0,
                        channel=shared_ch_key,
                        terminal_config="Cfg_Default",
                        min_val=-10.0,
                        max_val=10.0,
                    ),
                ],
            )

        task_a = _make_task("NI Dup Channel Task A")
        task_b = _make_task("NI Dup Channel Task B")
        self.client.tasks.configure(task_a)
        self.log("  Task A configured")

        rejected = False
        try:
            with task_a.run():
                self.log("  Task A running")
                rejected = self._try_configure_and_run(task_b)
        finally:
            _cleanup_task(self.client, task_a)
            _cleanup_task(self.client, task_b)

        if not rejected:
            self.fail(
                "Driver did not reject second task using the "
                "same channel — both tasks ran simultaneously"
            )

    def _try_configure_and_run(self, task: sy.Task) -> bool:
        """Try to configure and start a task. Return True if rejected."""
        try:
            self.client.tasks.configure(task)
        except (sy.ConfigurationError, TimeoutError) as e:
            self.log(f"  Correctly rejected on configure: {e}")
            return True

        self.log("  Task B configured (attempting run)")
        try:
            task.start()
        except (sy.ConfigurationError, TimeoutError, RuntimeError) as e:
            self.log(f"  Correctly rejected on run: {e}")
            return True

        task.stop()
        return False

    def _assert_configure_fails(self, task: sy.Task, label: str) -> None:
        """Attempt to configure a task and assert it raises ConfigurationError."""
        try:
            self.client.tasks.configure(task)
        except sy.ConfigurationError as e:
            self.log(f"  Correctly rejected ({label}): {e}")
            _cleanup_task(self.client, task)
            return
        except Exception as e:
            _cleanup_task(self.client, task)
            self.fail(
                f"Expected ConfigurationError for {label}, got {type(e).__name__}: {e}"
            )
        _cleanup_task(self.client, task)
        self.fail(f"Driver did not reject {label} — configure succeeded unexpectedly")


class NIMissingLibraries(TestCase):
    """Verify the driver rejects NI tasks when DAQmx libraries are not installed.

    Only runs on non-Windows platforms where NI-DAQmx is unavailable.
    """

    def setup(self) -> None:
        if platform.system().lower() == "windows":
            self.auto_pass(msg="NI-DAQmx is available on Windows")

    def run(self) -> None:
        self.log("Testing: NI task on machine without DAQmx libraries")
        idx = create_index(self.client, "ni_no_libs_idx")
        ch_key = create_channel(
            self.client,
            name="ni_no_libs_ch",
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        task = sy.ni.AnalogReadTask(
            name="NI Missing Libraries Test",
            device="nonexistent",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            channels=[
                sy.ni.AIVoltageChan(
                    port=0,
                    channel=ch_key,
                    terminal_config="Cfg_Default",
                    min_val=-10.0,
                    max_val=10.0,
                ),
            ],
        )
        try:
            self.client.tasks.configure(task)
        except sy.ConfigurationError as e:
            self.log(f"  Correctly rejected: {e}")
            msg = str(e).lower()
            assert "ni-daqmx" in msg and "libraries" in msg, (
                f"Expected error about missing libraries, got: {e}"
            )
            _cleanup_task(self.client, task)
            return
        _cleanup_task(self.client, task)
        self.fail("Driver did not reject NI task on machine without DAQmx libraries")
