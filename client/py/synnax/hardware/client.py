#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.hardware.writer import Writer
from synnax.hardware.payload import Rack, Task, Device

class Client:
    __writer: Writer

    def __init__(self, writer: Writer) -> None:
        self.__writer = writer

    def create_rack(self, racks: list[Rack]) -> list[Rack]:
        return self.__writer.create_rack(racks)

    def delete_rack(self, keys: list[int]):
        self.__writer.delete_rack(keys)

    def create_task(self, tasks: list[Task]) -> list[Task]:
        return self.__writer.create_task(tasks)

    def delete_task(self, keys: list[int]):
        self.__writer.delete_task(keys)

    def create_device(self, devices: list[Device]) -> list[Device]:
        return self.__writer.create_device(devices)

    def delete_device(self, keys: list[str]):
        self.__writer.delete_device(keys)


