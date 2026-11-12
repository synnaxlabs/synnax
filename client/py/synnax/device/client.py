#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel, JsonValue
from pydantic_core import to_jsonable_python

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import group as group_, rack as rack_
from synnax.device import actions_gen
from synnax.device.types_gen import (
    Cursor,
    Device,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.transport import send_batch
from synnax.util import request_where, returns_one, sole, sole_or_none
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    devices: list[Device]


_CreateResponse = _CreateRequest


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


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
        key: Key = "",
        location: str = "",
        rack: rack_.Key = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        configured: bool = False,
        parent: Key | None = None,
        group: group_.Key | None = None,
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
        key: Key = "",
        location: str = "",
        rack: rack_.Key = 0,
        name: str = "",
        make: str = "",
        model: str = "",
        configured: bool = False,
        parent: Key | None = None,
        group: group_.Key | None = None,
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
                    group=group,
                    properties=json_properties,
                )
            ]
        req = _CreateRequest(devices=normalize(devices))
        res = self._client.send("/device/create", req, _CreateResponse)
        return res.devices[0] if is_single else res.devices

    def delete(self, keys: list[Key]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/device/delete", req, Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a device.

        :param key: The key of the device to rename.
        :param name: The new name for the device.
        :raises NotFoundError: If no device has the given key.
        """
        self.dispatch(
            key, actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        )

    def set_group(self, keys: list[Key], group: group_.Key | None = None) -> None:
        """Moves devices into a group, or out of their group when group is None.

        :param keys: The keys of the devices to move.
        :param group: The key of the group to move the devices into. The group must
            sit under the rack of each device.
        """
        value = None if group is None else str(group)
        action = actions_gen.set(actions_gen.SetPayload(path=["group"], value=value))
        send_batch(
            self._client,
            "/device/dispatch",
            [_DispatchRequest(key=k, actions=[action]) for k in keys],
        )

    def dispatch(
        self, key: Key, actions: actions_gen.Action | list[actions_gen.Action]
    ) -> None:
        """Applies actions to a device, in order, in one transaction.

        :param key: The key of the device.
        :param actions: The actions to apply, such as ``device.set`` and
            ``device.set_rack``.
        :raises NotFoundError: If no device has the given key.
        :raises ValidationError: If an action is invalid.
        """
        req = _DispatchRequest(key=key, actions=normalize(actions))
        self._client.send("/device/dispatch", req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
        ignore_not_found: Literal[False] = ...,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
        ignore_not_found: Literal[False] = ...,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        makes: list[str] | None = ...,
        models: list[str] | None = ...,
        locations: list[str] | None = ...,
        racks: list[rack_.Key] | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
        ignore_not_found: Literal[False] = ...,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        makes: list[str] | None = ...,
        models: list[str] | None = ...,
        locations: list[str] | None = ...,
        racks: list[rack_.Key] | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
        ignore_not_found: Literal[True],
    ) -> Device | None: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        make: filter.StringInput = ...,
        model: filter.StringInput = ...,
        location: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        makes: list[str] | None = ...,
        models: list[str] | None = ...,
        locations: list[str] | None = ...,
        racks: list[rack_.Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Device]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        makes: list[str] | None = None,
        models: list[str] | None = None,
        locations: list[str] | None = None,
        racks: list[rack_.Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        include_status: bool = False,
        one: bool = False,
        ignore_not_found: bool = False,
        **fields: Any,
    ) -> list[Device] | Device | None:
        """Retrieves devices from the Core. With no arguments, retrieves every device.

        Each filter field takes a value, a list of values, or an operator, and a device
        must match every field. A plain ``key`` or ``name`` returns one device.

        :param key: Keeps devices with this key.
        :param name: Keeps devices with this name.
        :param make: Keeps devices with this make.
        :param model: Keeps devices with this model.
        :param location: Keeps devices at this location.
        :param rack: Keeps devices on this rack.
        :param and_: Keeps devices that match every filter in the list.
        :param or_: Keeps devices that match any filter in the list.
        :param not_: Keeps devices that do not match the filter.
        :param where: A filter tree that the devices must also match.
        :param keys: Keeps devices with one of these keys.
        :param names: Keeps devices with one of these names.
        :param makes: Keeps devices with one of these makes.
        :param models: Keeps devices with one of these models.
        :param locations: Keeps devices at one of these locations.
        :param racks: Keeps devices on one of these racks.
        :param parents: Keeps devices that any of the devices with these keys hold.
        :param nested: Keeps only devices that another device holds when True, and
            only devices that no device holds when False.
        :param groups: Keeps devices in any of these groups.
        :param grouped: Keeps only devices in a group when True, and only devices in no
            group when False.
        :param search_term: A fuzzy search term matched against device names.
        :param limit: The maximum number of devices to retrieve.
        :param order: The order of the devices. Defaults to key order.
        :param after: Retrieves the devices past this position in the order: the key of
            the last device of the page before, and its value of the order field.
        :param include_status: Whether to populate each device's status.
        :param one: Returns one device for any filter.
        :param ignore_not_found: Returns None instead of raising when one device is
            wanted and none matches.
        :returns: One device for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of devices. The list leaves out keys that do not exist.
        :raises NotFoundError: If one device is wanted, none matches, and
            ``ignore_not_found`` is not set.
        :raises MultipleFoundError: If one device is wanted and more than one matches,
            even when ``ignore_not_found`` is set.
        :raises ValidationError: If a filter field is None, or if one device is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            make=None if makes is None else filter.String(in_=makes),
            model=None if models is None else filter.String(in_=models),
            location=None if locations is None else filter.String(in_=locations),
            rack=None if racks is None else filter.Set[rack_.Key](in_=racks),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
            include_status=include_status,
        )
        res = self._client.send("/device/retrieve", req, RetrieveResponse)
        if not single:
            return res.devices
        if ignore_not_found:
            return sole_or_none("device", req, res.devices)
        return sole("device", req, res.devices)
