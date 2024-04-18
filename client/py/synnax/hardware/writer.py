#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload, UnaryClient

from synnax.hardware.payload import Device, Rack, Task


class _CreateRackRequest(Payload):
    racks: list[Rack]


class _CreateRackResponse(Payload):
    racks: list[Rack]


class _DeleteRackRequest(Payload):
    keys: list[int]


class _DeleteRackResponse(Payload):
    ...


class _CreateTaskRequest(Payload):
    tasks: list[Task]


class _CreateTaskResponse(Payload):
    tasks: list[Task]


class _DeleteTaskRequest(Payload):
    keys: list[int]


class _DeleteTaskResponse(Payload):
    ...


class _CreateDeviceRequest(Payload):
    devices: list[Device]


class _CreateDeviceResponse(Payload):
    devices: list[Device]


class _DeleteDeviceRequest(Payload):
    keys: list[str]


class _DeleteDeviceResponse(Payload):
    ...


_CREATE_RACK_ENDPOINT = "/hardware/rack/create"
_DELETE_RACK_ENDPOINT = "/hardware/rack/delete"
_CREATE_TASK_ENDPOINT = "/hardware/task/create"
_DELETE_TASK_ENDPOINT = "/hardware/task/delete"
_CREATE_DEVICE_ENDPOINT = "/hardware/device/create"
_DELETE_DEVICE_ENDPOINT = "/hardware/device/delete"


class Writer:
    client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self.client = client

    def create_rack(self, racks: list[Rack]) -> list[Rack]:
        req = _CreateRackRequest(racks=racks)
        res, exc = self.client.send(_CREATE_RACK_ENDPOINT, req, _CreateRackResponse)
        if exc is not None:
            raise exc
        return res.racks

    def delete_rack(self, keys: list[int]):
        req = _DeleteRackRequest(keys=keys)
        res, exc = self.client.send(_DELETE_RACK_ENDPOINT, req, _DeleteRackResponse)
        if exc is not None:
            raise exc
        return res

    def create_task(self, tasks: list[Task]) -> list[Task]:
        req = _CreateTaskRequest(tasks=tasks)
        res, exc = self.client.send(_CREATE_TASK_ENDPOINT, req, _CreateTaskResponse)
        if exc is not None:
            raise exc
        return res.tasks

    def delete_task(self, keys: list[int]):
        req = _DeleteTaskRequest(keys=keys)
        res, exc = self.client.send(_DELETE_TASK_ENDPOINT, req, _DeleteTaskResponse)
        if exc is not None:
            raise exc
        return res

    def create_device(self, devices: list[Device]) -> list[Device]:
        req = _CreateDeviceRequest(devices=devices)
        res, exc = self.client.send(_CREATE_DEVICE_ENDPOINT, req, _CreateDeviceResponse)
        if exc is not None:
            raise exc
        return res.devices

    def delete_device(self, keys: list[str]):
        req = _DeleteDeviceRequest(keys=keys)
        res, exc = self.client.send(_DELETE_DEVICE_ENDPOINT, req, _DeleteDeviceResponse)
        if exc is not None:
            raise exc
        return res
