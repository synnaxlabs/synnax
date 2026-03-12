#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI device reset and disconnect integration tests.

Tests that the driver:
1. Recovers after a DAQmxResetDevice mid-acquisition.
2. Stops producing samples when a simulated device is removed from NI MAX.
3. Recovers when the device is re-added and task is reconfigured.
4. Emits appropriate status feedback during disconnection.

Covers multiple device types: analog read (cDAQ module), digital read
(cDAQ module on a different chassis), and digital write.
"""

import ctypes
import os
import tempfile
from collections.abc import Callable

import nidaqmx.system
import nisyscfg
import synnax as sy
from synnax.task.payload import Status

from tests.driver.ni_read import NIDigitalRead, NIReadCurrentVoltage
from tests.driver.ni_write import NIDigitalWrite
from tests.driver.task import ReadTaskCase, WriteTaskCase, send_and_verify_commands

_STATUS_TIMEOUT = 10 * sy.TimeSpan.SECOND


def _export_ni_config() -> str:
    """Export the full NI MAX config to a temp file, return the path."""
    fd, path = tempfile.mkstemp(suffix=".ini")
    os.close(fd)
    with nisyscfg.Session() as session:
        session._library.ExportConfiguration(
            session._session,
            path.encode(),
            b"",
            1,  # overwriteIfExists = True
        )
    return path


def _import_ni_config(ini_path: str) -> None:
    """Import an NI MAX config from a file."""
    with nisyscfg.Session() as session:
        detailed_result = ctypes.POINTER(ctypes.c_char)()
        status = session._library.ImportConfiguration(
            session._session,
            ini_path.encode(),
            b"",
            nisyscfg.enums.ImportMode.MERGE_ITEMS,
            ctypes.byref(detailed_result),
        )
        if status != 0:
            raise RuntimeError(
                f"NISysCfgImportConfiguration failed with status {status}"
            )


def _delete_ni_device(device_name: str) -> None:
    """Delete a simulated device from NI MAX via NI System Configuration."""
    with nisyscfg.Session() as session:
        filt = session.create_filter()
        filt.is_simulated = True
        for resource in session.find_hardware(filt):
            if resource.expert_user_alias[0] == device_name:
                resource.delete(
                    nisyscfg.enums.DeleteValidationMode.DELETE_ITEM_AND_ANY_DEPENDENCIES
                )
                return
    raise RuntimeError(f"Simulated device '{device_name}' not found in NI MAX")


def _wait_for_device_status(
    streamer: sy.Streamer,
    message: str,
    timeout: sy.TimeSpan = _STATUS_TIMEOUT,
    *,
    device_key: str = "",
    device_name: str = "",
) -> list[Status]:
    """Wait until a device emits a status with the given message.

    Filter by ``device_key`` (exact match on ``device:<key>``) or
    ``device_name`` (match on status ``name`` field). At least one must
    be provided.

    Returns all matching device statuses collected up to (and including)
    the first status with the expected ``message``.
    """
    device_statuses: list[Status] = []
    timer = sy.Timer()
    while timer.elapsed() < timeout:
        remaining = timeout - timer.elapsed()
        frame = streamer.read(timeout=remaining)
        if frame is None:
            break
        if "sy_status_set" not in frame:
            continue
        for raw in frame["sy_status_set"]:
            s = Status.model_validate(raw)
            if device_key and s.key != f"device:{device_key}":
                continue
            if device_name and s.name != device_name:
                continue
            if not s.key.startswith("device:"):
                continue
            device_statuses.append(s)
            if s.message == message:
                return device_statuses
    label = device_key or device_name
    msgs = [f"  {s.variant}: {s.message}" for s in device_statuses]
    detail = "\n".join(msgs) if msgs else "  (no statuses)"
    raise AssertionError(
        f"Timed out waiting for '{message}' on {label}, got:\n{detail}"
    )


def _log_statuses(
    log: Callable[[str], None], statuses: list[Status], label: str
) -> None:
    """Log final status with count of skipped intermediate ones."""
    n = len(statuses)
    final = statuses[-1]
    if n > 1:
        log(f"  [{label}] {n - 1} intermediate, then {final.variant}: {final.message}")
    else:
        log(f"  [{label}] {final.variant}: {final.message}")


class _NIReadDisconnectMixin(ReadTaskCase):
    """Mixin providing reset + remove/re-add disconnect test logic for read tasks.

    Subclasses set ``disconnect_device`` to the NI MAX alias of the device
    to reset and remove (e.g. "E101Mod4").
    """

    disconnect_device: str

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

        tsk = self.tsk
        dev = self.disconnect_device
        device = self.client.devices.retrieve(location=dev)
        dev_key = device.key
        dev_name = device.name

        self.log(f"Test 1 - Assert samples ({dev})")
        self.assert_sample_count(task=tsk)

        self.log(f"Test 2 - Reset {dev} via DAQmxResetDevice")
        nidaqmx.system.Device(dev).reset_device()

        self.log("Test 3 - Assert samples resume after reset")
        self.assert_sample_count(task=tsk, strict=False)

        self.log("Test 4 - Export NI MAX config")
        saved_config = _export_ni_config()

        with self.client.open_streamer(["sy_status_set"]) as streamer:
            self.log(f"Test 5 - Remove {dev} from NI MAX")
            _delete_ni_device(dev)
            statuses = _wait_for_device_status(
                streamer, "Device disconnected", device_key=dev_key
            )
            _log_statuses(self.log, statuses, "removal")

        self.log("Test 6 - Assert no new samples after removal")
        channel_keys = self._channel_keys(tsk)
        start = sy.TimeStamp.now()
        sy.sleep(0.5)
        end = sy.TimeStamp.now()
        tr = sy.TimeRange(start, end)
        for key in channel_keys:
            ch = self.client.channels.retrieve(key)
            num_samples = len(ch.read(tr))
            if num_samples > 0:
                self.fail(
                    f"Channel '{ch.name}' has {num_samples} samples "
                    f"after device removal, expected 0"
                )

        with self.client.open_streamer(["sy_status_set"]) as streamer:
            self.log(f"Test 7 - Re-add {dev} to NI MAX")
            _import_ni_config(saved_config)
            statuses = _wait_for_device_status(
                streamer, "Device present", device_name=dev_name
            )
            _log_statuses(self.log, statuses, "re-add")

        self.log("Test 8 - Reconfigure and assert samples resume")
        self.client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)


class NIAnalogReadDisconnect(_NIReadDisconnectMixin, NIReadCurrentVoltage):
    """Disconnect test for analog read on cDAQ chassis E101.

    E101Mod3 (NI 9203) + E101Mod4 (NI 9205). Disconnects E101Mod4.
    """

    task_name = "NI Analog Read Disconnect"
    disconnect_device = "E101Mod4"


class NIDigitalReadDisconnect(_NIReadDisconnectMixin, NIDigitalRead):
    """Disconnect test for digital read on cDAQ chassis E102.

    E102Mod3 (NI 9375). Disconnects E102Mod3.
    """

    task_name = "NI Digital Read Disconnect"
    disconnect_device = "E102Mod3"


class _NIWriteDisconnectMixin(WriteTaskCase):
    """Mixin providing reset + remove/re-add disconnect test logic for write tasks.

    Subclasses set ``disconnect_device`` to the NI MAX alias of the device
    to reset and remove (e.g. "SYMod1").
    """

    disconnect_device: str

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

        tsk = self.tsk
        dev = self.disconnect_device
        device = self.client.devices.retrieve(location=dev)
        dev_key = device.key
        dev_name = device.name

        self.log(f"Test 1 - Send commands ({dev})")
        with tsk.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=self._channel_keys(tsk),
                writer_name=f"{self.task_name}_pre_reset",
                task_name=tsk.name,
                task_key=tsk.key,
                command_values=self.command_values,
            )

        self.log(f"Test 2 - Reset {dev} via DAQmxResetDevice")
        nidaqmx.system.Device(dev).reset_device()

        self.log("Test 3 - Send commands after reset")
        with tsk.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=self._channel_keys(tsk),
                writer_name=f"{self.task_name}_post_reset",
                task_name=tsk.name,
                task_key=tsk.key,
                command_values=self.command_values,
            )

        self.log("Test 4 - Export NI MAX config")
        saved_config = _export_ni_config()

        with self.client.open_streamer(["sy_status_set"]) as streamer:
            self.log(f"Test 5 - Remove {dev} from NI MAX")
            _delete_ni_device(dev)
            statuses = _wait_for_device_status(
                streamer, "Device disconnected", device_key=dev_key
            )
            _log_statuses(self.log, statuses, "removal")

        with self.client.open_streamer(["sy_status_set"]) as streamer:
            self.log(f"Test 6 - Re-add {dev} to NI MAX")
            _import_ni_config(saved_config)
            statuses = _wait_for_device_status(
                streamer, "Device present", device_name=dev_name
            )
            _log_statuses(self.log, statuses, "re-add")

        self.log("Test 7 - Reconfigure and send commands after re-add")
        self.client.tasks.configure(tsk)
        with tsk.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=self._channel_keys(tsk),
                writer_name=f"{self.task_name}_post_readd",
                task_name=tsk.name,
                task_key=tsk.key,
                command_values=self.command_values,
            )


class NIDigitalWriteDisconnect(_NIWriteDisconnectMixin, NIDigitalWrite):
    """Disconnect test for digital write on SYMod1.

    SYMod1 (NI 9375 on SYChassis). Disconnects SYMod1.
    """

    task_name = "NI Digital Write Disconnect"
    disconnect_device = "SYMod1"
