#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, TypeVar, overload

from pydantic import BaseModel, Field, JsonValue

from freighter import Empty, UnaryClient
from synnax import label
from synnax.resource import ID, CrudeID
from synnax.status.types_gen import Key, Status, Variant
from x.lists import normalize
from x.params import require_named_params

D = TypeVar("D")


class _SetRequest(BaseModel, Generic[D]):
    statuses: list[Status[D]]


class _SetResponse(BaseModel):
    statuses: list[Status[JsonValue]]


class _RetrieveRequest(BaseModel):
    keys: list[str] | None = None
    owners: list[ID] | None = None
    search_term: str | None = None
    offset: int | None = None
    limit: int | None = None
    has_labels: list[label.Key] | None = None
    variants: list[Variant] | None = None


class _RetrieveResponse(BaseModel):
    statuses: list[Status[JsonValue]] = Field(default_factory=list)


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
    disabled), and a rack, task, device, or channel can own one.
    """

    client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        """Initialize the status client.

        Args:
            transport: The unary transport client to use for communication.
        """
        self.client = transport

    @overload
    def set(self, status: Status[D]) -> Status[JsonValue]: ...

    @overload
    def set(self, status: list[Status[D]]) -> list[Status[JsonValue]]: ...

    def set(
        self, status: Status[D] | list[Status[D]] | None = None
    ) -> Status[JsonValue] | list[Status[JsonValue]]:
        """Creates or updates one or more statuses. A status with an owner updates the
        status that owner already holds, whatever its key. A create stores the labels a
        status carries. An update keeps the stored labels: change them with
        ``client.labels.label`` and ``client.labels.remove``.

        :param status: A single status or a list of statuses to set.
        :returns: Each status as stored, with its key and labels resolved. A single
            status when ``status`` is a single status.

        Create a status that a task owns:

        >>> status = client.statuses.set(
        ...     Status(
        ...         variant="warning",
        ...         name="Tank pressure",
        ...         message="Above 600 psi",
        ...         description="Vent the tank before you continue.",
        ...         owner=task.resource_id,
        ...     )
        ... )
        """
        if status is None:
            raise ValueError("status parameter is required")

        single = not isinstance(status, list)
        statuses = status if isinstance(status, list) else [status]

        res = self.client.send(
            _SET_ENDPOINT, _SetRequest(statuses=statuses), _SetResponse
        ).statuses

        if single:
            return res[0]
        return res

    @overload
    def retrieve(self, *, key: str) -> Status[JsonValue]: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        owners: list[CrudeID] | None = None,
        search_term: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        has_labels: list[label.Key] | None = None,
        variants: list[Variant] | None = None,
    ) -> list[Status[JsonValue]]: ...

    @require_named_params(example_params=("key", "'status-key-123'"))
    def retrieve(
        self,
        *,
        key: str | None = None,
        keys: list[str] | None = None,
        owners: list[CrudeID] | None = None,
        search_term: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        has_labels: list[label.Key] | None = None,
        variants: list[Variant] | None = None,
    ) -> Status[JsonValue] | list[Status[JsonValue]]:
        """Retrieves one status by key, or a list of the statuses that match the given
        filters. A filter left as None does not filter.

        :param key: The key of one status to retrieve.
        :param keys: Keeps statuses with one of these keys.
        :param owners: Keeps the statuses these resources own.
        :param search_term: Fuzzy matches status names.
        :param offset: The number of matching statuses to skip.
        :param limit: The maximum number of statuses to return.
        :param has_labels: Keeps statuses that have one of these labels.
        :param variants: Keeps statuses with one of these variants.
        :returns: The status for a key, else a list of the matching statuses.
        :raises NotFoundError: If no status has a requested key.

        Retrieve the status of a task:

        >>> statuses = client.statuses.retrieve(owners=[task.resource_id])
        """
        single = key is not None
        if single and key is not None:
            keys = [key]

        res = self.client.send(
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(
                keys=keys,
                owners=None if owners is None else [ID(o) for o in owners],
                search_term=search_term,
                offset=offset,
                limit=limit,
                has_labels=has_labels,
                variants=variants,
            ),
            _RetrieveResponse,
        ).statuses

        if single:
            if len(res) == 0:
                raise ValueError(f"Status with key '{key}' not found")
            return res[0]
        return res

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
