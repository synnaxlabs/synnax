#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation
from freighter import Payload, UnaryClient, send_required, Empty
from synnax.hardware.rack.payload import Rack
from typing import overload
from synnax.util.normalize import check_for_none, override


class _CreateRequest(Payload):
    racks: list[Rack]


class _CreateResponse(Payload):
    racks: list[Rack]


class _DeleteRequest(Payload):
    keys: list[int]


class _RetrieveRequest(Payload):
    keys: list[int] | None = None
    names: list[str] | None = None


class _RetrieveResponse(Payload):
    racks: list[Rack] | None = None


_CREATE_ENDPOINT = "/hardware/rack/create"
_DELETE_ENDPOINT = "/hardware/rack/delete"
_RETRIEVE_ENDPOINT = "/hardware/rack/retrieve"


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
    def create(self, *, key: int = 0, name: str = "") -> Rack:
        ...

    @overload
    def create(self, rack: Rack) -> Rack:
        ...

    @overload
    def create(self, racks: list[Rack]) -> list[Rack]:
        ...

    def create(
        self, racks: Rack | list[Rack] | None = None, *, key: int = 0, name: str = ""
    ) -> list[Rack]:
        is_single = True
        if racks is None:
            racks = [Rack(key=key, name=name)]
        elif isinstance(racks, Rack):
            racks = [racks]
        else:
            is_single = False
            racks = [r.to_payload() for r in racks]
        req = _CreateRequest(racks=racks)
        res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
        if is_single:
            return res.racks[0]
        return res.racks

    def delete(self, keys: list[int]):
        req = _DeleteRequest(keys=keys)
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
    ) -> Rack:
        ...

    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        keys: list[int] | None = None,
        names: list[str] | None = None,
    ) -> list[Rack]:
        is_single = check_for_none(keys, names)
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(keys=override(key, keys), names=override(name, names)),
            _RetrieveResponse,
        )
        return res.racks[0] if is_single else res.racks
