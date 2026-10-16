#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel, Field, JsonValue
from pydantic_core import to_jsonable_python

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import rack as rack_
from synnax.device.types_gen import Device
from synnax.exceptions import NotFoundError
from synnax.ontology.payload import ID as OntologyID
from x.lists import check_for_none, normalize, override


class _CreateRequest(BaseModel):
    devices: list[Device]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[str]


class _RetrieveRequest(BaseModel):
    keys: list[str] | None = None
    names: list[str] | None = None
    makes: list[str] | None = None
    models: list[str] | None = None
    locations: list[str] | None = None
    racks: list[int] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None
    ignore_not_found: bool = False
    include_status: bool = False
    include_parent: bool = False


class _RetrieveResponse(BaseModel):
    devices: list[Device] = Field(default_factory=list)


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
        rack: rack_.Key = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        configured: bool = False,
        parent: OntologyID | None = None,
        properties: dict[str, Any] | None = None,
    ) -> Device: ...

    @overload
    def create(
        self,
        devices: Device,
    ) -> Device: ...

    @overload
    def create(
        self,
        devices: list[Device],
    ) -> list[Device]: ...

    def create(
        self,
        devices: list[Device] | Device | None = None,
        *,
        key: str = "",
        location: str = "",
        rack: rack_.Key = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        configured: bool = False,
        parent: OntologyID | None = None,
        properties: dict[str, Any] | None = None,
    ) -> Device | list[Device]:
        is_single = not isinstance(devices, list)
        json_properties: dict[str, JsonValue] = (
            to_jsonable_python(properties) if properties is not None else {}
        )
        if devices is None:
            devices = [
                Device(
                    key=key,
                    location=location,
                    rack=rack,
                    name=name,
                    make=make,
                    model=model,
                    configured=configured,
                    parent=parent,
                    properties=json_properties,
                )
            ]
        req = _CreateRequest(devices=normalize(devices))
        res = self._client.send("/device/create", req, _CreateResponse)
        return res.devices[0] if is_single else res.devices

    def delete(self, keys: list[str]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/device/delete", req, Empty)

    # Each list overload requires one filter that makes the result a list. The last
    # overload covers calls that pass those filters as possibly None.
    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str],
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str],
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str],
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str],
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str],
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int],
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        make: str | None = None,
        name: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        names: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> Device | list[Device] | None: ...

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
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device] | Device | None:
        """Retrieves devices matching the given filters. Returns one device unless a
        plural filter, a search term, a limit, or an offset is given.

        :param racks: Only return devices on the racks with these keys.
        :param search_term: A fuzzy search term matched against device names.
        :param limit: The maximum number of devices to return.
        :param offset: The number of devices to skip.
        :param include_status: Whether to populate each device's status.
        :param include_parent: Whether to populate each device's parent ontology ID.
        :param ignore_not_found: Whether to return None instead of raising when a
            single device is requested and none matches.
        :raises NotFoundError: If a single device is requested and none matches.
        """
        is_single = check_for_none(
            keys, makes, models, locations, names, racks, search_term, limit, offset
        )
        res = self._client.send(
            "/device/retrieve",
            _RetrieveRequest(
                keys=override(key, keys),
                makes=override(make, makes),
                models=override(model, models),
                locations=override(location, locations),
                names=override(name, names),
                racks=racks,
                search_term=search_term,
                limit=limit,
                offset=offset,
                ignore_not_found=ignore_not_found,
                include_status=include_status,
                include_parent=include_parent,
            ),
            _RetrieveResponse,
        )
        if is_single:
            if len(res.devices) > 0:
                return res.devices[0]
            if ignore_not_found:
                return None
            raise NotFoundError("Device not found")
        return res.devices
