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
from dataclasses import dataclass, field

import nidaqmx.system
import nisyscfg
import synnax as sy
from synnax.task.payload import Status

from tests.driver.ni_read import NIDigitalRead, NIReadCurrentVoltage
from tests.driver.ni_write import NIDigitalWrite
from tests.driver.task import ReadTaskCase, WriteTaskCase, send_and_verify_commands

# The scan task polls every 5s. 30s gives ~5 cycles of margin under CI load.
_STATUS_TIMEOUT = 30 * sy.TimeSpan.SECOND



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



@dataclass
class _StatusWaitDiag:
    """Diagnostics collected during _wait_for_device_status for debugging
    timeout failures."""

    frames_received: int = 0
    frames_with_status: int = 0
    all_keys: list[str] = field(default_factory=list)

    def summary(self) -> str:
        unique = sorted(set(self.all_keys))
        return (
            f"frames={self.frames_received}, "
            f"frames_with_status={self.frames_with_status}, "
            f"total_statuses={len(self.all_keys)}, "
            f"unique_keys={unique}"
        )


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
    matched: list[Status] = []
    diag = _StatusWaitDiag()
    timer = sy.Timer()

    while timer.elapsed() < timeout:
        frame = streamer.read(timeout=timeout - timer.elapsed())
        if frame is None:
            break
        diag.frames_received += 1
        if "sy_status_set" not in frame:
            continue
        diag.frames_with_status += 1
        for raw in frame["sy_status_set"]:
            s = Status.model_validate(raw)
            diag.all_keys.append(s.key)
            if device_key and s.key != f"device:{device_key}":
                continue
            if device_name and s.name != device_name:
                continue
            if not s.key.startswith("device:"):
                continue
            matched.append(s)
            if s.message == message:
                return matched

    label = device_key or device_name
    lines = [f"  {s.variant}: {s.message}" for s in matched]
    detail = "\n".join(lines) if lines else "  (no matching statuses)"
    raise AssertionError(
        f"Timed out waiting for '{message}' on {label}:\n"
        f"{detail}\n"
        f"  {diag.summary()}"
    )


def _log_statuses(
    log: Callable[[str], None], statuses: list[Status], label: str
) -> None:
    """Log final status with count of skipped intermediate ones."""
    final = statuses[-1]
    n = len(statuses)
    if n > 1:
        log(f"  [{label}] {n - 1} intermediate, final: {final.variant}: {final.message}")
    else:
        log(f"  [{label}] {final.variant}: {final.message}")



def _remove_device_and_wait(
    client: sy.Synnax,
    log: Callable[[str], None],
    device_name: str,
    device_key: str,
) -> None:
    """Delete a device from NI MAX and wait for the driver to report it gone."""
    with client.open_streamer(["sy_status_set"]) as streamer:
        log(f"Remove {device_name} from NI MAX")
        _delete_ni_device(device_name)
        statuses = _wait_for_device_status(
            streamer, "Device disconnected", device_key=device_key
        )
        _log_statuses(log, statuses, "removal")


def _restore_device_and_wait(
    client: sy.Synnax,
    log: Callable[[str], None],
    saved_config: str,
    device_name: str,
) -> None:
    """Re-import the NI MAX config and wait for the driver to see the device."""
    with client.open_streamer(["sy_status_set"]) as streamer:
        log(f"Restore {device_name} to NI MAX")
        _import_ni_config(saved_config)
        try:
            statuses = _wait_for_device_status(
                streamer, "Device present", device_name=device_name
            )
            _log_statuses(log, statuses, "re-add")
        except Exception as e:
            log(f"Warning: could not confirm device re-add: {e}")



class _NIReadDisconnectMixin(ReadTaskCase):
    """Reset + remove/re-add disconnect test logic for read tasks.

    Subclasses set ``disconnect_device`` to the NI MAX alias of the device
    to reset and remove (e.g. "E101Mod4").

    The device deletion is wrapped in try/finally so that the NI MAX config
    is always restored, even if the status wait times out.
    """

    disconnect_device: str

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

        tsk = self.tsk
        dev = self.disconnect_device
        device = self.client.devices.retrieve(location=dev)

        # Phase 1: Verify normal operation and reset recovery
        self.log(f"Test 1 - Assert samples ({dev})")
        self.assert_sample_count(task=tsk)

        self.log(f"Test 2 - Reset {dev} via DAQmxResetDevice")
        nidaqmx.system.Device(dev).reset_device()

        self.log("Test 3 - Assert samples resume after reset")
        self.assert_sample_count(task=tsk, strict=False)

        # Phase 2: Device removal and recovery (always restores config)
        saved_config = _export_ni_config()
        try:
            _remove_device_and_wait(
                self.client, self.log, dev, device.key,
            )
            self._assert_no_samples_after_removal(tsk)
        finally:
            _restore_device_and_wait(
                self.client, self.log, saved_config, device.name,
            )

        # Phase 3: Verify task recovers after re-add
        self.log("Test 4 - Reconfigure and assert samples resume")
        self.client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)

    def _assert_no_samples_after_removal(self, tsk: sy.Task) -> None:
        self.log("Assert no new samples after removal")
        start = sy.TimeStamp.now()
        sy.sleep(0.5)
        end = sy.TimeStamp.now()
        tr = sy.TimeRange(start, end)
        for key in self._channel_keys(tsk):
            ch = self.client.channels.retrieve(key)
            n = len(ch.read(tr))
            if n > 0:
                self.fail(
                    f"Channel '{ch.name}' has {n} samples "
                    f"after device removal, expected 0"
                )


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
    """Reset + remove/re-add disconnect test logic for write tasks.

    Subclasses set ``disconnect_device`` to the NI MAX alias of the device
    to reset and remove (e.g. "SYMod1").

    The device deletion is wrapped in try/finally so that the NI MAX config
    is always restored, even if the status wait times out.
    """

    disconnect_device: str

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

        tsk = self.tsk
        dev = self.disconnect_device
        device = self.client.devices.retrieve(location=dev)

        # Phase 1: Verify normal operation and reset recovery
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

        # Phase 2: Device removal and recovery (always restores config)
        saved_config = _export_ni_config()
        try:
            _remove_device_and_wait(
                self.client, self.log, dev, device.key,
            )
        finally:
            _restore_device_and_wait(
                self.client, self.log, saved_config, device.name,
            )

        # Phase 3: Verify task recovers after re-add
        self.log("Test 4 - Reconfigure and send commands after re-add")
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
