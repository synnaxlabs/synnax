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
from synnax import channel
from synnax.calculation import actions_gen
from synnax.calculation.types_gen import (
    Calculation,
    Cursor,
    Key,
    Operation,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.exceptions import ValidationError
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    calculations: list[Calculation]


_CreateResponse = _CreateRequest


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Client for creating, retrieving, changing, and deleting calculations in a
    Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(
        self,
        *,
        output: channel.Key,
        expression: str,
        operations: list[Operation] | None = None,
        primary: channel.Key | None = None,
    ) -> Calculation: ...

    @overload
    def create(self, calculations: Calculation) -> Calculation: ...

    @overload
    def create(self, calculations: list[Calculation]) -> list[Calculation]: ...

    def create(
        self,
        calculations: Calculation | list[Calculation] | None = None,
        *,
        output: channel.Key | None = None,
        expression: str | None = None,
        operations: list[Operation] | None = None,
        primary: channel.Key | None = None,
    ) -> Calculation | list[Calculation]:
        """Creates one or more calculations.

        :param calculations: The calculation or list of calculations to create. When
            omitted, one calculation is created from the keyword arguments.
        :param output: The virtual channel that the calculation writes, when
            ``calculations`` is omitted.
        :param expression: The Arc expression that computes each output sample.
        :param operations: The operations applied to the computed samples, in order.
        :param primary: The channel that the expression reads whose samples time the
            output.
        :returns: The created calculation, or a list of them when ``calculations`` is
            a list.
        :raises ValidationError: If ``calculations`` and ``output`` are both omitted,
            if a calculation is not valid, or if its output stores its samples or has
            another writer.
        :raises NotFoundError: If an output does not exist.
        """
        is_single = not isinstance(calculations, list)
        if calculations is None:
            if output is None or expression is None:
                raise ValidationError(
                    "A calculation needs an output and an expression."
                )
            calculations = Calculation(
                output=output,
                expression=expression,
                operations=operations or [],
                primary=primary,
            )
        req = _CreateRequest(calculations=normalize(calculations))
        res = self._client.send("/calculation/create", req, _CreateResponse)
        return res.calculations[0] if is_single else res.calculations

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        output: filter.SetInput[channel.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Calculation: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        output: filter.SetInput[channel.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Calculation: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        output: filter.SetInput[channel.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        include_status: bool = ...,
        limit: int | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Calculation]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        include_status: bool = False,
        limit: int | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Calculation | list[Calculation]:
        """Retrieves calculations from the Core. With no arguments, retrieves every
        calculation that the client can retrieve, with its output and each channel
        that it names.

        :param key: Keeps calculations with this key.
        :param output: Keeps calculations that write this channel.
        :param and_: Keeps calculations that match every filter in the list.
        :param or_: Keeps calculations that match any filter in the list.
        :param not_: Keeps calculations that do not match the filter.
        :param where: A filter tree that the calculations must also match.
        :param keys: Keeps calculations with one of these keys.
        :param include_status: Includes the status of each calculation.
        :param limit: The maximum number of calculations to retrieve.
        :param after: Retrieves the calculations past this key.
        :param one: Returns one calculation for any filter.
        :returns: One calculation for a plain ``key`` or when ``one`` is set,
            otherwise a list of them.
        :raises NotFoundError: If one calculation is wanted and none matches.
        :raises MultipleFoundError: If one calculation is wanted and more than one
            matches.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            limit=limit,
            after=after,
            include_status=include_status,
        )
        res = self._client.send("/calculation/retrieve", req, RetrieveResponse)
        return (
            sole("calculation", req, res.calculations) if single else res.calculations
        )

    def dispatch(
        self, key: Key, actions: actions_gen.Action | list[actions_gen.Action]
    ) -> None:
        """Applies actions to a calculation, in order, in one transaction.

        :param key: The key of the calculation.
        :param actions: The actions to apply, such as a ``set`` of ``expression``.
        :raises NotFoundError: If no calculation has the given key.
        :raises ValidationError: If an action is invalid or the result is not valid.
        """
        req = _DispatchRequest(key=key, actions=normalize(actions))
        self._client.send("/calculation/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more calculations. Deleting a calculation that does not
        exist is not an error. The output of each one stays as a channel with no
        producer.

        :param keys: A single key or a list of keys to delete.
        """
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/calculation/delete", req, Empty)
