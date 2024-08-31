#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient, send_required, Empty
from synnax.hardware.device.payload import Device
from typing import overload
from synnax.exceptions import NotFoundError
from synnax.util.normalize import normalize, check_for_none, override


class _CreateRequest(Payload):
    devices: list[Device]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[str]


class _RetrieveRequest(Payload):
    keys: list[str] | None = None
    names: list[str] | None = None
    makes: list[str] | None = None
    models: list[str] | None = None
    locations: list[str] | None = None


class _RetrieveResponse(Payload):
    devices: list[Device] | None = None


_CREATE_ENDPOINT = "/hardware/device/create"
_DELETE_ENDPOINT = "/hardware/device/delete"
_RETRIEVE_ENDPOINT = "/hardware/device/retrieve"


class Client:
    _client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        key: str = "",
        location: str = "",
        rack: int = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        properties: str = "",
    ):
        ...

    @overload
    def create(self, devices: Device):
        ...

    @overload
    def create(self, devices: list[Device]):
        ...

    def create(
        self,
        devices: list[Device] | Device | None = None,
        *,
        key: str = "",
        location: str = "",
        rack: int = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        properties: str = "",
    ):
        is_single = not isinstance(devices, list)
        if devices is None:
            devices = [
                Device(
                    key=key,
                    location=location,
                    rack=rack,
                    name=name,
                    make=make,
                    model=model,
                    properties=properties,
                )
            ]
        req = _CreateRequest(devices=normalize(devices))
        res = send_required(
            self._client,
            _CREATE_ENDPOINT,
            req,
            _CreateResponse,
        )
        return res.devices[0] if is_single else res.devices

    def delete(self, keys: list[str]) -> None:
        req = _DeleteRequest(keys=keys)
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
    ) -> Device:
        ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
    ) -> list[Device]:
        ...

    @trace("debug")
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        model: str | None = None,
        name: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
    ) -> list[Device] | Device:
        is_single = check_for_none(keys, makes, models, locations, names)
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(
                keys=override(key, keys),
                makes=override(make, makes),
                models=override(model, models),
                locations=override(location, locations),
                names=override(name, names),
            ),
            _RetrieveResponse,
        )
        if is_single:
            if len(res.devices) > 0:
                return res.devices[0]
            raise NotFoundError("Device not found")
        return res.devices
