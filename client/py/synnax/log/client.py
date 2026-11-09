#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import channel as channel_, group as group_, project as project_, ranger
from synnax.log.types_gen import (
    Key,
    Log,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import color as color_, filter, notation as notation_
from x.lists import normalize


class _CreateRequest(BaseModel):
    logs: list[Log]


class _CreateResponse(BaseModel):
    logs: list[Log]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _SetPayload(BaseModel):
    path: list[str]
    value: Any


class _SetAction(BaseModel):
    type: Literal["set"] = "set"
    set: _SetPayload


class _ChannelPayload(BaseModel):
    channel: channel_.Key


class _AddChannelAction(BaseModel):
    type: Literal["add_channel"] = "add_channel"
    add_channel: _ChannelPayload


class _RemoveChannelAction(BaseModel):
    type: Literal["remove_channel"] = "remove_channel"
    remove_channel: _ChannelPayload


class _ColorPayload(BaseModel):
    channel: channel_.Key
    color: color_.Color


class _SetChannelColorAction(BaseModel):
    type: Literal["set_channel_color"] = "set_channel_color"
    set_channel_color: _ColorPayload


class _AliasPayload(BaseModel):
    channel: channel_.Key
    alias: str


class _SetChannelAliasAction(BaseModel):
    type: Literal["set_channel_alias"] = "set_channel_alias"
    set_channel_alias: _AliasPayload


class _NotationPayload(BaseModel):
    channel: channel_.Key
    notation: notation_.Notation


class _SetChannelNotationAction(BaseModel):
    type: Literal["set_channel_notation"] = "set_channel_notation"
    set_channel_notation: _NotationPayload


class _PrecisionPayload(BaseModel):
    channel: channel_.Key
    precision: int | None


class _SetChannelPrecisionAction(BaseModel):
    type: Literal["set_channel_precision"] = "set_channel_precision"
    set_channel_precision: _PrecisionPayload


_Action = (
    _SetAction
    | _AddChannelAction
    | _RemoveChannelAction
    | _SetChannelColorAction
    | _SetChannelAliasAction
    | _SetChannelNotationAction
    | _SetChannelPrecisionAction
)


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_Action]


class Client:
    """Client for creating, retrieving, editing, and deleting logs in a Synnax Core.

    A log is a Console visualization that displays the samples of a set of channels as
    timestamped rows.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, project: project_.Key) -> Log: ...

    @overload
    def create(self, logs: Log) -> Log: ...

    @overload
    def create(self, logs: list[Log]) -> list[Log]: ...

    def create(
        self,
        logs: Log | list[Log] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> Log | list[Log]:
        """Create one or more logs.

        :param logs: A single log or a list of them. Each one belongs to exactly one of
            a project and a range. When omitted, a log is created from the ``name`` and
            ``project`` keyword arguments.
        :param name: The name of the log to create when ``logs`` is omitted.
        :param project: The key of the project that owns the log when ``logs`` is
            omitted.
        :returns: The created log, or a list of created logs when ``logs`` is a list.
        :raises ValidationError: If a log does not belong to exactly one of a project
            and a range.
        """
        is_single = not isinstance(logs, list)
        if logs is None:
            logs = Log(name=name, project=project)
        req = _CreateRequest(logs=normalize(logs))
        res = self._client.send("/log/create", req, _CreateResponse)
        return res.logs[0] if is_single else res.logs

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        one: bool = ...,
    ) -> Log: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[True],
    ) -> Log: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[False] = ...,
    ) -> list[Log]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Log | list[Log]:
        """Retrieves logs. With no arguments, retrieves every log.

        Each filter field takes a value, a list of values, or an operator, and a log
        must match every field. A plain ``key`` returns one log.

        :param key: Keeps logs with this key.
        :param and_: Keeps logs that match every filter in the list.
        :param or_: Keeps logs that match any filter in the list.
        :param not_: Keeps logs that do not match the filter.
        :param where: A filter tree that the logs must also match.
        :param keys: Keeps logs with one of these keys.
        :param projects: Keeps the logs owned by any of these projects.
        :param ranges: Keeps the logs owned by any of these ranges.
        :param groups: Keeps logs in any of these groups.
        :param grouped: Keeps only logs in a group when True, and only logs in no group
            when False.
        :param one: Returns one log for any filter.
        :returns: One log for a plain ``key`` or when ``one`` is set, otherwise a list
            of logs. The list leaves out keys that do not exist.
        :raises NotFoundError: If one log is wanted and none matches.
        :raises MultipleFoundError: If one log is wanted and more than one matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields)
        )
        res = self._client.send("/log/retrieve", req, RetrieveResponse)
        return sole("log", req, res.logs) if single else res.logs

    def rename(self, key: Key, name: str) -> None:
        """Rename a log.

        :param key: The key of the log to rename.
        :param name: The new name for the log.
        """
        self._dispatch(key, _SetAction(set=_SetPayload(path=["name"], value=name)))

    def add_channel(self, key: Key, channel: channel_.Key) -> None:
        """Add a channel to a log with the default display configuration. Does
        nothing when the log already shows the channel.

        :param key: The key of the log.
        :param channel: The key of the channel to add.
        """
        payload = _ChannelPayload(channel=channel)
        self._dispatch(key, _AddChannelAction(add_channel=payload))

    def remove_channel(self, key: Key, channel: channel_.Key) -> None:
        """Remove a channel from a log. Does nothing when the log does not show the
        channel.

        :param key: The key of the log.
        :param channel: The key of the channel to remove.
        """
        payload = _ChannelPayload(channel=channel)
        self._dispatch(key, _RemoveChannelAction(remove_channel=payload))

    def set_channel_color(
        self, key: Key, channel: channel_.Key, color: color_.Color | str
    ) -> None:
        """Set the display color of a channel in a log. A zero color shows the
        channel in the text color of the log. Does nothing when the log does not show
        the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param color: The color, as a ``Color`` or a hex string such as ``"#FF0000"``.
        """
        payload = _ColorPayload(channel=channel, color=color_.Color(color))
        self._dispatch(key, _SetChannelColorAction(set_channel_color=payload))

    def set_channel_alias(self, key: Key, channel: channel_.Key, alias: str) -> None:
        """Set the name a log shows for a channel. An empty alias shows the channel
        name. Does nothing when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param alias: The name to show, at most 64 characters.
        """
        payload = _AliasPayload(channel=channel, alias=alias)
        self._dispatch(key, _SetChannelAliasAction(set_channel_alias=payload))

    def set_channel_notation(
        self, key: Key, channel: channel_.Key, notation: notation_.Notation
    ) -> None:
        """Set the numeric notation of a channel in a log. Does nothing when the log
        does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param notation: One of ``"standard"``, ``"scientific"``, or
            ``"engineering"``.
        """
        payload = _NotationPayload(channel=channel, notation=notation)
        self._dispatch(key, _SetChannelNotationAction(set_channel_notation=payload))

    def set_channel_precision(
        self, key: Key, channel: channel_.Key, precision: int | None
    ) -> None:
        """Set the number of digits after the decimal point of a channel in a log.
        Does nothing when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param precision: A number of digits from 0 to 17, or None to show each sample
            as it arrives.
        """
        payload = _PrecisionPayload(channel=channel, precision=precision)
        self._dispatch(key, _SetChannelPrecisionAction(set_channel_precision=payload))

    def _dispatch(self, key: Key, action: _Action) -> None:
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/log/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more logs by key.

        :param keys: A single log key or a list of keys to delete.
        """
        self._client.send("/log/delete", _DeleteRequest(keys=normalize(keys)), Empty)
