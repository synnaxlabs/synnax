#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.hardware.payload import Device, Rack, Task


class _RetrieveTaskRequest(Payload):
    rack: int | None = None
    keys: list[int] | None = None


class _RetrieveTaskResponse(Payload):
    tasks: list[Task] | None = None


class _RetrieveDeviceRequest(Payload):
    keys: list[str] | None = None


class _RetrieveDeviceResponse(Payload):
    devices: list[Device] | None = None


class _RetrieveRackRequest(Payload):
    keys: list[int] | None = None


class _RetrieveRackResponse(Payload):
    racks: list[Rack] | None = None


RETRIEVE_TASK_ENDPOINT = "/hardware/task/retrieve"
RETRIEVE_DEVICE_ENDPOINT = "/hardware/device/retrieve"
RETRIEVE_RACK_ENDPOINT = "/hardware/rack/retrieve"


class Retriever:
    __client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve_task(
        self,
        rack: int | None = None,
        keys: list[int] | None = None,
    ) -> list[Task]:
        res, exc = self.__client.send(
            RETRIEVE_TASK_ENDPOINT,
            _RetrieveTaskRequest(rack=rack, keys=keys),
            _RetrieveTaskResponse,
        )
        if exc is not None:
            raise exc
        if res.tasks is None:
            return list()
        return res.tasks

    @trace("debug")
    def retrieve_device(self, keys: list[str] | None = None) -> list[Device]:
        res, exc = self.__client.send(
            RETRIEVE_DEVICE_ENDPOINT,
            _RetrieveDeviceRequest(keys=keys),
            _RetrieveDeviceResponse,
        )
        if exc is not None:
            raise exc
        if res.devices is None:
            return list()
        return res.devices

    @trace("debug")
    def retrieve_rack(self, keys: list[int] | None = None) -> list[Rack]:
        res, exc = self.__client.send(
            RETRIEVE_RACK_ENDPOINT,
            _RetrieveRackRequest(keys=keys),
            _RetrieveRackResponse,
        )
        if exc is not None:
            raise exc
        if res.racks is None:
            return list()
        return res.racks
