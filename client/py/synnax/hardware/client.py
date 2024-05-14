#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.hardware.payload import Device, Rack, Task
from synnax.hardware.retrieve import Retriever
from synnax.hardware.writer import Writer


class Client:
    __writer: Writer
    __retriever: Retriever

    def __init__(self, writer: Writer, retriever: Retriever) -> None:
        self.__writer = writer
        self.__retriever = retriever

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

    def retrieve_rack(self, keys: list[int] | None = None) -> list[Rack]:
        return self.__retriever.retrieve_rack(keys)

    def retrieve_task(
        self, rack: int | None = None, keys: list[int] | None = None
    ) -> list[Task]:
        return self.__retriever.retrieve_task(rack, keys)

    def retrieve_device(self, keys: list[str] | None = None) -> list[Device]:
        return self.__retriever.retrieve_device(keys)
