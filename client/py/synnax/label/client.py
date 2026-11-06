#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.exceptions import ValidationError
from synnax.label.types_gen import Key, Label, RetrieveRequest, RetrieveResponse, Where
from synnax.ontology.payload import ID, CrudeID
from synnax.util import sole
from x import color as color_
from x import filter
from x.lists import normalize

ONTOLOGY_TYPE = ID(type="label")


def ontology_id(key: Key) -> ID:
    """Returns the ontology ID of the label with the given key."""
    return ID(key=str(key), type="label")


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
        """Creates one or more labels. A label with the key of an existing label
        replaces it.

        :param labels: A single label or a list of them. When omitted, a label is
            created from the ``name`` and ``color`` keyword arguments.
        :param name: The name of the label to create when ``labels`` is omitted.
        :param color: The color of the label to create when ``labels`` is omitted.
        :returns: The created label, or a list of labels when ``labels`` is a list.
        :raises ValidationError: If ``labels`` is omitted and ``name`` or ``color`` is
            too.
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
    def retrieve(self, *, key: Key) -> Label: ...

    @overload
    def retrieve(self, *, name: str) -> Label: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        for_: CrudeID | None = None,
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Label]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        name: str | None = None,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        for_: CrudeID | None = None,
        limit: int | None = None,
        offset: int = 0,
    ) -> Label | list[Label]:
        """Retrieves labels from the Core. With no arguments, retrieves every label.

        :param key: The key of a single label to retrieve.
        :param name: The name of a single label to retrieve.
        :param where: A filter that the labels must also match.
        :param keys: The keys of the labels to retrieve.
        :param names: The names of the labels to retrieve.
        :param search_term: A fuzzy search term matched against label names.
        :param for_: The ontology ID of a resource. Retrieves only the labels on that
            resource.
        :param limit: The maximum number of labels to retrieve.
        :param offset: The number of labels to skip.
        :returns: The label when ``key`` or ``name`` is provided, otherwise a list of
            labels. The list leaves out keys that do not exist.
        :raises NotFoundError: If no label has the given ``key`` or ``name``.
        :raises MultipleFoundError: If more than one label has the given ``name``.
        """
        is_single = key is not None or name is not None
        keys = normalize(key, keys) or None
        names = normalize(name, names) or None
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            for_=None if for_ is None else ID(for_),
        )
        req = RetrieveRequest(
            where=filter.merge(where, flat),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self._client.send("/label/retrieve", req, RetrieveResponse)
        if not is_single:
            return res.labels
        return sole("Label", key if key is not None else name, res.labels)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more labels and removes them from every resource. Deleting
        a label that does not exist is not an error.

        :param keys: A single label key or a list of keys to delete.
        """
        self._client.send("/label/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def label(
        self, id: CrudeID, labels: Key | list[Key], *, replace: bool = False
    ) -> None:
        """Adds labels to a resource.

        :param id: The ontology ID of the resource to label.
        :param labels: A single label key or a list of keys to add.
        :param replace: If true, removes every other label from the resource.
        """
        req = _SetRequest(id=ID(id), labels=normalize(labels), replace=replace)
        self._client.send("/label/set", req, Empty)

    def remove(self, id: CrudeID, labels: Key | list[Key]) -> None:
        """Removes labels from a resource.

        :param id: The ontology ID of the resource.
        :param labels: A single label key or a list of keys to remove.
        """
        req = _RemoveRequest(id=ID(id), labels=normalize(labels))
        self._client.send("/label/remove", req, Empty)
