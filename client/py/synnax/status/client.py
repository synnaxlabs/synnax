#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, Literal, TypeVar, overload
from uuid import UUID

from pydantic import BaseModel, JsonValue

from freighter import Empty, UnaryClient
from synnax import util
from synnax.ontology import ID
from synnax.status import builder_gen
from synnax.status.builder_gen import Condition
from synnax.status.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Status,
    Variant,
    Where,
)
from x import filter
from x.lists import normalize

D = TypeVar("D")


class _SetRequest(BaseModel, Generic[D]):
    parent: ID | None = None
    statuses: list[Status[D]]


class _SetResponse(BaseModel):
    statuses: list[Status[JsonValue]]


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[str]


_SET_ENDPOINT = "/status/set"
_RETRIEVE_ENDPOINT = "/status/retrieve"
_DELETE_ENDPOINT = "/status/delete"

SET_CHANNEL = "sy_status_set"
DELETE_CHANNEL = "sy_status_delete"


class Client:
    """Client for managing statuses in a Synnax cluster.

    Statuses are standardized messages used to communicate state across the Synnax
    platform. They support different variants (success, info, warning, error, loading,
    disabled) and can be organized hierarchically using the ontology system.
    """

    client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        """Initialize the status client.

        Args:
            transport: The unary transport client to use for communication.
        """
        self.client = transport

    @overload
    def set(
        self,
        status: Status[D],
        *,
        parent: ID | None = None,
    ) -> Status[JsonValue]: ...

    @overload
    def set(
        self,
        status: list[Status[D]],
        *,
        parent: ID | None = None,
    ) -> list[Status[JsonValue]]: ...

    def set(
        self,
        status: Status[D] | list[Status[D]] | None = None,
        *,
        parent: ID | None = None,
    ) -> Status[JsonValue] | list[Status[JsonValue]]:
        """Create or update a status.

        Args:
            status: A single status or list of statuses to create or update.
            parent: Optional parent ontology ID for organizing the status.

        Returns:
            The created/updated status or list of statuses.

        Examples:
            Create a single status:
                >>> status = client.statuses.set(
                ...     Status(
                ...         variant="warning",
                ...         name="Tank pressure",
                ...         message="Above 600 psi",
                ...         description="Vent the tank before you continue.",
                ...     )
                ... )

            Create multiple statuses:
                >>> statuses = client.statuses.set([
                ...     Status(
                ...         variant="success", name="Pump 1", message="Started pump"
                ...     ),
                ...     Status(
                ...         variant="error",
                ...         name="Pump 2",
                ...         message="Failed to start pump",
                ...         description="The motor controller did not respond.",
                ...     ),
                ... ])

            Create with parent:
                >>> from synnax.status import ontology_id
                >>> parent_id = ontology_id("parent-key")
                >>> status = client.statuses.set(
                ...     Status(variant="info", name="Valve 3", message="Opened valve"),
                ...     parent=parent_id
                ... )
        """
        if status is None:
            raise ValueError("status parameter is required")

        single = not isinstance(status, list)
        statuses = status if isinstance(status, list) else [status]

        res = self.client.send(
            _SET_ENDPOINT, _SetRequest(statuses=statuses, parent=parent), _SetResponse
        ).statuses

        if single:
            return res[0]
        return res

    @overload
    def retrieve(
        self, key: str, /, *, include_labels: bool = False
    ) -> Status[JsonValue]: ...

    @overload
    def retrieve(
        self, keys: list[str], /, *, include_labels: bool = False
    ) -> list[Status[JsonValue]]: ...

    @overload
    def retrieve(
        self, *, key: str, include_labels: bool = False
    ) -> Status[JsonValue]: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        keys: list[str] | None = None,
        search_term: str = "",
        offset: int = 0,
        limit: int | None = None,
        include_labels: bool = False,
        has_labels: list[UUID] | None = None,
        variants: list[Variant] | None = None,
    ) -> Status[JsonValue]: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        keys: list[str] | None = None,
        search_term: str = "",
        offset: int = 0,
        limit: int | None = None,
        include_labels: bool = False,
        has_labels: list[UUID] | None = None,
        variants: list[Variant] | None = None,
    ) -> list[Status[JsonValue]]: ...

    def retrieve(
        self,
        *args: str | list[str] | Condition,
        key: str | None = None,
        one: bool = False,
        keys: list[str] | None = None,
        search_term: str = "",
        offset: int = 0,
        limit: int | None = None,
        include_labels: bool = False,
        has_labels: list[UUID] | None = None,
        variants: list[Variant] | None = None,
    ) -> Status[JsonValue] | list[Status[JsonValue]]:
        """Retrieves the statuses that match every condition and filter.

        :param args: Conditions from the status builder, such as
            ``status.variant.eq("error")``, or one key or list of keys.
        :param key: The key of one status to retrieve.
        :param one: Returns the one status that matches, instead of a list.
        :param keys: Keeps statuses with one of these keys.
        :param search_term: A fuzzy search term matched against status names and
            messages.
        :param offset: The number of statuses to skip.
        :param limit: The maximum number of statuses to return.
        :param include_labels: Whether to populate each status's labels.
        :param has_labels: Keeps statuses that have any of these labels.
        :param variants: Keeps statuses with one of these variants.
        :returns: One status for a key or one, and a list otherwise.
        :raises NotFoundError: If no status matches a key or one.
        :raises MultipleFoundError: If one is True and more than one status matches.
        :raises TypeError: If args are neither status conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "Status", args, Condition, str, builder_gen.key
        )
        if key is not None:
            keys = [key]
            query = key
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            variant=None if variants is None else filter.Set[Variant](in_=variants),
            labels=has_labels,
        )
        where = filter.merge(filter.join(Where, conditions), flat)
        res = self.client.send(
            _RETRIEVE_ENDPOINT,
            RetrieveRequest(
                where=where,
                search_term=search_term,
                offset=offset,
                limit=limit,
                include_labels=include_labels,
            ),
            RetrieveResponse,
        ).statuses

        if query is None and not one:
            return res
        return util.one("Status", where if query is None else query, res)

    def rename(self, key: Key, name: str) -> None:
        """Renames a status.

        :param key: The key of the status to rename.
        :param name: The new name for the status.
        :raises NotFoundError: If no status has the given key.
        """
        self.client.send("/status/rename", _RenameRequest(key=key, name=name), Empty)

    def delete(self, keys: str | list[str]) -> None:
        """Delete statuses by their keys.

        This operation is idempotent - deleting a non-existent status will not
        raise an error.

        Args:
            keys: A single key or list of keys to delete.

        Examples:
            Delete a single status:
                >>> client.statuses.delete("my-status")

            Delete multiple statuses:
                >>> client.statuses.delete(["status-1", "status-2", "status-3"])
        """
        self.client.send(_DELETE_ENDPOINT, _DeleteRequest(keys=normalize(keys)), Empty)


ONTOLOGY_TYPE = ID(type="status")


def ontology_id(key: str) -> ID:
    return ID(type="status", key=str(key))
