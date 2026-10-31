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
from synnax.device.types_gen import Device, Key
from synnax.exceptions import NotFoundError
from synnax.ontology.payload import ID as OntologyID
from x.lists import all_none, normalize, override


class _CreateRequest(BaseModel):
    devices: list[Device]


_CreateResponse = _CreateRequest


class _RenameRequest(BaseModel):
    key: Key
    name: str


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
        *,
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

    def rename(self, key: Key, name: str) -> None:
        """Renames a device.

        :param key: The key of the device to rename.
        :param name: The new name for the device.
        :raises NotFoundError: If no device has the given key.
        """
        self._client.send("/device/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(
        self,
        *,
        key: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        name: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        name: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        make: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        make: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        model: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        model: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        location: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[False] = False,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        location: str,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device]: ...

    def retrieve(
        self,
        *,
        key: str | None = None,
        name: str | None = None,
        make: str | None = None,
        model: str | None = None,
        location: str | None = None,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[int] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        include_status: bool = False,
        include_parent: bool = False,
        ignore_not_found: bool = False,
    ) -> list[Device] | Device | None:
        """Retrieves devices matching the given filters. Returns one device when a key,
        name, make, model, or location is given, and a list otherwise.

        :param racks: Only return devices on the racks with these keys.
        :param search_term: A fuzzy search term matched against device names.
        :param limit: The maximum number of devices to return.
        :param offset: The number of devices to skip.
        :param include_status: Whether to populate each device's status.
        :param include_parent: Whether to populate each device's parent ontology ID.
        :param ignore_not_found: Whether to return None instead of raising when a
            single device is requested and none matches.
        :raises ValueError: If two singular selectors are given, or one is
            combined with a list filter.
        :raises NotFoundError: If a single device is requested and none matches.
        """
        is_single = not all_none(key, name, make, model, location)
        if sum(v is not None for v in (key, name, make, model, location)) > 1:
            raise ValueError(
                "only one of key, name, make, model, and location can be given"
            )
        if is_single and not all_none(
            keys, names, makes, models, locations, racks, search_term, limit, offset
        ):
            raise ValueError(
                "key, name, make, model, and location cannot be combined with list "
                "filters; use their plural forms instead"
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
