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
from synnax.exceptions import ValidationError
from synnax.label.types_gen import (
    Cursor,
    Key,
    Label,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.resource import ID, CrudeID
from synnax.util import request_where, returns_one, sole
from x import color as color_, filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    labels: list[Label]


class _CreateResponse(BaseModel):
    labels: list[Label]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _SetRequest(BaseModel):
    id: ID
    labels: list[Key]
    replace: bool = False


class _RemoveRequest(BaseModel):
    id: ID
    labels: list[Key]


class Client:
    """Client for creating, retrieving, deleting, and attaching labels in a Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, color: color_.Crude) -> Label: ...

    @overload
    def create(self, labels: Label) -> Label: ...

    @overload
    def create(self, labels: list[Label]) -> list[Label]: ...

    def create(
        self,
        labels: Label | list[Label] | None = None,
        *,
        name: str | None = None,
        color: color_.Crude | None = None,
    ) -> Label | list[Label]:
        """Creates one or more labels.

        :param labels: A single label or a list of them. When omitted, a label is
            created from the ``name`` and ``color`` keyword arguments.
        :param name: The name of the label to create when ``labels`` is omitted.
        :param color: The color of the label to create when ``labels`` is omitted.
        :returns: The created label, or a list of labels when ``labels`` is a list.
        :raises ValidationError: If ``labels`` is omitted and ``name`` or ``color`` is
            too, or if a label with a given key already exists.
        """
        is_single = not isinstance(labels, list)
        if labels is None:
            if name is None or color is None:
                raise ValidationError("A label needs a name and a color.")
            labels = Label(name=name, color=color_.Color(color))
        req = _CreateRequest(labels=normalize(labels))
        res = self._client.send("/label/create", req, _CreateResponse)
        return res.labels[0] if is_single else res.labels

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Label: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Label: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Label: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Label]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Label | list[Label]:
        """Retrieves labels from the Core. With no arguments, retrieves every label.

        Each filter field takes a value, a list of values, or an operator, and a label
        must match every field. A plain ``key`` or ``name`` returns one label.

        :param key: Keeps labels with this key.
        :param name: Keeps labels with this name.
        :param and_: Keeps labels that match every filter in the list.
        :param or_: Keeps labels that match any filter in the list.
        :param not_: Keeps labels that do not match the filter.
        :param where: A filter tree that the labels must also match.
        :param keys: Keeps labels with one of these keys.
        :param names: Keeps labels with one of these names.
        :param search_term: A fuzzy search term matched against label names.
        :param limit: The maximum number of labels to retrieve.
        :param order: The order of the labels. Defaults to key order.
        :param after: Retrieves the labels past this position in the order: the key of
            the last label of the page before, and its value of the order field.
        :param one: Returns one label for any filter.
        :returns: One label for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of labels. The list leaves out keys that do not exist.
        :raises NotFoundError: If one label is wanted and none matches.
        :raises MultipleFoundError: If one label is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one label is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/label/retrieve", req, RetrieveResponse)
        return sole("label", req, res.labels) if single else res.labels

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more labels and removes them from every range and status.
        Deleting a label that does not exist is not an error.

        :param keys: A single label key or a list of keys to delete.
        """
        self._client.send("/label/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def label(
        self, id: CrudeID, labels: Key | list[Key], *, replace: bool = False
    ) -> None:
        """Adds labels to a range or a status.

        :param id: The resource ID of the range or status to label.
        :param labels: A single label key or a list of keys to add.
        :param replace: If true, removes every other label from the range or status.
        :raises ValidationError: If ``id`` names a resource that is not a range or a
            status.
        """
        req = _SetRequest(id=ID(id), labels=normalize(labels), replace=replace)
        self._client.send("/label/set", req, Empty)

    def remove(self, id: CrudeID, labels: Key | list[Key]) -> None:
        """Removes labels from a range or a status.

        :param id: The resource ID of the range or status.
        :param labels: A single label key or a list of keys to remove.
        :raises ValidationError: If ``id`` names a resource that is not a range or a
            status.
        """
        req = _RemoveRequest(id=ID(id), labels=normalize(labels))
        self._client.send("/label/remove", req, Empty)
