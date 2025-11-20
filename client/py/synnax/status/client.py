#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload
from uuid import UUID

from freighter import Empty, Payload, UnaryClient, send_required

from synnax.exceptions import NotFoundError
from synnax.status.payload import Status
from synnax.util.normalize import normalize
from synnax.util.params import require_named_params


class _SetRequest(Payload):
    parent: dict[str, str] | None = None
    statuses: list[Status]


class _SetResponse(Payload):
    statuses: list[Status]


class _RetrieveRequest(Payload):
    keys: list[str] | None = None
    search_term: str | None = None
    offset: int | None = None
    limit: int | None = None
    include_labels: bool | None = None
    has_labels: list[UUID] | None = None


class _RetrieveResponse(Payload):
    statuses: list[Status] | None = None


class _DeleteRequest(Payload):
    keys: list[str]


_SET_ENDPOINT = "/status/set"
_RETRIEVE_ENDPOINT = "/status/retrieve"
_DELETE_ENDPOINT = "/status/delete"


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
        status: Status,
        *,
        parent: dict[str, str] | None = None,
    ) -> Status: ...

    @overload
    def set(
        self,
        statuses: list[Status],
        *,
        parent: dict[str, str] | None = None,
    ) -> list[Status]: ...

    def set(
        self,
        status: Status | list[Status] | None = None,
        *,
        parent: dict[str, str] | None = None,
    ) -> Status | list[Status]:
        """Create or update a status.

        Args:
            status: A single status or list of statuses to create or update.
            parent: Optional parent ontology ID for organizing the status.

        Returns:
            The created/updated status or list of statuses.

        Examples:
            Create a single status:
                >>> status = client.statuses.set(
                ...     Status(variant="info", message="Task started")
                ... )

            Create multiple statuses:
                >>> statuses = client.statuses.set([
                ...     Status(variant="success", message="Task 1 complete"),
                ...     Status(variant="error", message="Task 2 failed"),
                ... ])

            Create with parent:
                >>> from synnax.status import ontology_id
                >>> parent_id = ontology_id("parent-key")
                >>> status = client.statuses.set(
                ...     Status(variant="info", message="Child status"),
                ...     parent=parent_id
                ... )
        """
        if status is None:
            raise ValueError("status parameter is required")

        single = not isinstance(status, list)
        statuses = [status] if single else status

        res = send_required(
            self.client,
            _SET_ENDPOINT,
            _SetRequest(statuses=statuses, parent=parent),
            _SetResponse,
        ).statuses

        if single:
            return res[0]
        return res

    @overload
    def retrieve(self, *, key: str, include_labels: bool = False) -> Status: ...

    @overload
    def retrieve(
        self, *, keys: list[str], include_labels: bool = False
    ) -> list[Status]: ...

    @overload
    def retrieve(
        self,
        *,
        search_term: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        include_labels: bool = False,
        has_labels: list[UUID] | None = None,
    ) -> list[Status]: ...

    @require_named_params(example_params=("key", "'status-key-123'"))
    def retrieve(
        self,
        *,
        key: str | None = None,
        keys: list[str] | None = None,
        search_term: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        include_labels: bool = False,
        has_labels: list[UUID] | None = None,
    ) -> Status | list[Status]:
        """Retrieve statuses from the cluster.

        Args:
            key: Retrieve a single status by key.
            keys: Retrieve multiple statuses by keys.
            search_term: Search for statuses by name or message.
            offset: Pagination offset.
            limit: Pagination limit.
            include_labels: Whether to include labels in the response.
            has_labels: Filter statuses that have all of these labels.

        Returns:
            A single status, or a list of statuses.

        Examples:
            Retrieve by key:
                >>> status = client.statuses.retrieve(key="my-status")

            Retrieve multiple:
                >>> statuses = client.statuses.retrieve(
                ...     keys=["status-1", "status-2"]
                ... )

            Search with pagination:
                >>> statuses = client.statuses.retrieve(
                ...     search_term="error",
                ...     limit=10,
                ...     offset=0
                ... )

            Filter by labels:
                >>> from uuid import UUID
                >>> statuses = client.statuses.retrieve(
                ...     has_labels=[UUID("label-uuid")],
                ...     include_labels=True
                ... )
        """
        single = key is not None
        if single:
            keys = [key]

        res = send_required(
            self.client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(
                keys=keys,
                search_term=search_term,
                offset=offset,
                limit=limit,
                include_labels=include_labels,
                has_labels=has_labels,
            ),
            _RetrieveResponse,
        ).statuses

        if res is None:
            return [] if not single else None

        if single:
            if len(res) == 0:
                raise ValueError(f"Status with key '{key}' not found")
            return res[0]
        return res

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
        send_required(
            self.client,
            _DELETE_ENDPOINT,
            _DeleteRequest(keys=normalize(keys)),
            Empty,
        )
