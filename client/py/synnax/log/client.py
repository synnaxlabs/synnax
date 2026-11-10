#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel, ConfigDict, Field

from freighter import Empty, UnaryClient
from synnax import channel as channel_, group as group_, project as project_, ranger
from synnax.log.types_gen import (
    ChannelEntry,
    Key,
    Log,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import color as color_, filter, notation as notation_, telem
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


class _ChannelPayload(BaseModel):
    channel: channel_.Key


class _ColorPayload(BaseModel):
    channel: channel_.Key
    color: color_.Color


class _AliasPayload(BaseModel):
    channel: channel_.Key
    alias: str


class _NotationPayload(BaseModel):
    channel: channel_.Key
    notation: notation_.Notation


class _PrecisionPayload(BaseModel):
    channel: channel_.Key
    precision: int | None = Field(ge=0, le=17)


class _EntryPayload(BaseModel):
    entry: ChannelEntry


class _FormatPayload(BaseModel):
    channel: channel_.Key
    format: telem.TimestampFormat


class _ChannelTimeZonePayload(BaseModel):
    channel: channel_.Key
    time_zone: telem.TimeZone


class _SwapPayload(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    from_: channel_.Key = Field(alias="from")
    to: channel_.Key


class _TimestampPrecisionPayload(BaseModel):
    timestamp_precision: int = Field(ge=0, le=3)


class _ChannelNamesHiddenPayload(BaseModel):
    channel_names_hidden: bool


class _TimestampsHiddenPayload(BaseModel):
    timestamps_hidden: bool


class _TimeZonePayload(BaseModel):
    time_zone: telem.TimeZone


# Holds its payload in an extra field named after its type.
class _Action(BaseModel):
    model_config = ConfigDict(extra="allow")

    type: str


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
        self._dispatch(key, "set", _SetPayload(path=["name"], value=name))

    def add_channel(self, key: Key, channel: channel_.Key) -> None:
        """Add a channel to a log with the default display configuration. Does
        nothing when the log already shows the channel.

        :param key: The key of the log.
        :param channel: The key of the channel to add.
        """
        self._dispatch(key, "add_channel", _ChannelPayload(channel=channel))

    def remove_channel(self, key: Key, channel: channel_.Key) -> None:
        """Remove a channel from a log. Does nothing when the log does not show the
        channel.

        :param key: The key of the log.
        :param channel: The key of the channel to remove.
        """
        self._dispatch(key, "remove_channel", _ChannelPayload(channel=channel))

    def set_channel_color(
        self, key: Key, channel: channel_.Key, color: color_.Color | str
    ) -> None:
        """Set the display color of a channel in a log. A zero color,
        ``Color(a=0)``, shows the channel in the text color of the log. Does nothing
        when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param color: The color, as a ``Color`` or a hex string such as ``"#FF0000"``.
        """
        payload = _ColorPayload(channel=channel, color=color_.Color(color))
        self._dispatch(key, "set_channel_color", payload)

    def set_channel_alias(self, key: Key, channel: channel_.Key, alias: str) -> None:
        """Set the name a log shows for a channel. An empty alias shows the channel
        name. Does nothing when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param alias: The name to show, at most 64 characters.
        """
        payload = _AliasPayload(channel=channel, alias=alias)
        self._dispatch(key, "set_channel_alias", payload)

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
        self._dispatch(key, "set_channel_notation", payload)

    def set_channel_precision(
        self, key: Key, channel: channel_.Key, precision: int | None
    ) -> None:
        """Set the number of digits after the decimal point of a channel in a log.
        Does nothing when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the channel.
        :param precision: A number of digits from 0 to 17, or None to show each sample
            as it arrives.
        :raises ValidationError: If the precision is above 17.
        """
        payload = _PrecisionPayload(channel=channel, precision=precision)
        self._dispatch(key, "set_channel_precision", payload)

    def set_channel_entry(self, key: Key, entry: ChannelEntry) -> None:
        """Set the full display configuration of a channel in a log. Adds the channel
        when the log does not show it.

        :param key: The key of the log.
        :param entry: The display configuration, with the key of its channel.
        """
        self._dispatch(key, "set_channel_entry", _EntryPayload(entry=entry))

    def set_channel_timestamp_format(
        self, key: Key, channel: channel_.Key, format: telem.TimestampFormat
    ) -> None:
        """Set the format of the values of a timestamp channel in a log. Does nothing
        when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the timestamp channel.
        :param format: The format of the values, such as ``"ISO"``.
        """
        payload = _FormatPayload(channel=channel, format=format)
        self._dispatch(key, "set_channel_timestamp_format", payload)

    def set_channel_timestamp_time_zone(
        self, key: Key, channel: channel_.Key, time_zone: telem.TimeZone
    ) -> None:
        """Set the time zone of the values of a timestamp channel in a log. Does
        nothing when the log does not show the channel.

        :param key: The key of the log.
        :param channel: The key of the timestamp channel.
        :param time_zone: ``"local"`` or ``"UTC"``.
        """
        payload = _ChannelTimeZonePayload(channel=channel, time_zone=time_zone)
        self._dispatch(key, "set_channel_timestamp_time_zone", payload)

    def swap_channel(self, key: Key, from_: channel_.Key, to: channel_.Key) -> None:
        """Replace a channel in a log with another channel, and keep its position and
        display configuration. Does nothing when the log does not show ``from_`` or
        already shows ``to``.

        :param key: The key of the log.
        :param from_: The key of the channel to replace.
        :param to: The key of the channel to show in its place.
        """
        self._dispatch(key, "swap_channel", _SwapPayload(from_=from_, to=to))

    def set_timestamp_precision(self, key: Key, precision: int) -> None:
        """Set the number of decimal places of the seconds in the line times of a log.

        :param key: The key of the log.
        :param precision: A number of decimal places from 0 to 3.
        :raises ValidationError: If the precision is above 3.
        """
        payload = _TimestampPrecisionPayload(timestamp_precision=precision)
        self._dispatch(key, "set_timestamp_precision", payload)

    def set_channel_names_hidden(self, key: Key, hidden: bool) -> None:
        """Hide or show the channel names in the lines of a log.

        :param key: The key of the log.
        :param hidden: True to hide the channel names.
        """
        payload = _ChannelNamesHiddenPayload(channel_names_hidden=hidden)
        self._dispatch(key, "set_channel_names_hidden", payload)

    def set_timestamps_hidden(self, key: Key, hidden: bool) -> None:
        """Hide or show the line times of a log.

        :param key: The key of the log.
        :param hidden: True to hide the line times.
        """
        payload = _TimestampsHiddenPayload(timestamps_hidden=hidden)
        self._dispatch(key, "set_timestamps_hidden", payload)

    def set_time_zone(self, key: Key, time_zone: telem.TimeZone) -> None:
        """Set the time zone of the line times and day rows of a log.

        :param key: The key of the log.
        :param time_zone: ``"local"`` or ``"UTC"``.
        """
        self._dispatch(key, "set_time_zone", _TimeZonePayload(time_zone=time_zone))

    def _dispatch(self, key: Key, type_: str, payload: BaseModel) -> None:
        action = _Action.model_validate({"type": type_, type_: payload})
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/log/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more logs by key.

        :param keys: A single log key or a list of keys to delete.
        """
        self._client.send("/log/delete", _DeleteRequest(keys=normalize(keys)), Empty)
