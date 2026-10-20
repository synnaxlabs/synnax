#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.modbus import ModbusSim

import synnax as sy
from console.task.modbus import ModbusTask
from tests.console.task.task_case import ConsoleTaskCase


class ModbusCase(ConsoleTaskCase):
    """ConsoleTaskCase against the Modbus TCP simulator."""

    sim_classes = [ModbusSim]

    @classmethod
    def registers(cls, task: sy.Task) -> set[tuple[str, int]]:
        """Return the (type, address) pairs a task config points at."""
        pairs: set[tuple[str, int]] = set()
        for ch in cls.config_channels(task):
            type_, address = ch["type"], ch["address"]
            if not isinstance(type_, str) or not isinstance(address, int):
                raise TypeError(f"register ({type_!r}, {address!r}) is not (str, int)")
            pairs.add((type_, address))
        return pairs

    @staticmethod
    def assert_rows(page: ModbusTask, expected: list[tuple[str, int]]) -> None:
        """Assert the listed (type name, address) rows are ``expected``, in order."""
        listed = page.channels()
        assert listed == expected, f"Channel rows should be {expected}, got {listed}"
