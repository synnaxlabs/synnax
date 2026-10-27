#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy
from x.testutil import assert_eventually


@pytest.mark.status
class TestStatusClient:
    """Tests for the status client."""

    def test_set_single_status(self, client: sy.Synnax) -> None:
        """Should create a single status."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO,
            message="Test status message",
            name="Test Status",
        )
        created = client.statuses.set(status)

        assert created.key == status.key
        assert created.variant == sy.status.VARIANT_INFO
        assert created.message == "Test status message"
        assert created.name == "Test Status"
        assert created.time is not None

    def test_set_multiple_statuses(self, client: sy.Synnax) -> None:
        """Should create multiple statuses at once."""
        statuses: list[sy.Status[None]] = [
            sy.Status(variant=sy.status.VARIANT_SUCCESS, message="Task 1 complete"),
            sy.Status(variant=sy.status.VARIANT_ERROR, message="Task 2 failed"),
            sy.Status(variant=sy.status.VARIANT_WARNING, message="Task 3 warning"),
        ]
        created = client.statuses.set(statuses)

        assert len(created) == 3
        assert created[0].variant == sy.status.VARIANT_SUCCESS
        assert created[1].variant == sy.status.VARIANT_ERROR
        assert created[2].variant == sy.status.VARIANT_WARNING

    def test_update_existing_status(self, client: sy.Synnax) -> None:
        """Should update an existing status."""
        key = str(uuid4())
        original: sy.Status[None] = sy.Status(
            key=key,
            variant=sy.status.VARIANT_INFO,
            message="Original message",
        )
        client.statuses.set(original)

        updated: sy.Status[None] = sy.Status(
            key=key,
            variant=sy.status.VARIANT_SUCCESS,
            message="Updated message",
        )
        result = client.statuses.set(updated)

        assert result.key == key
        assert result.variant == sy.status.VARIANT_SUCCESS
        assert result.message == "Updated message"

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a status by key."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO, message="Retrievable status"
        )
        created = client.statuses.set(status)

        retrieved = client.statuses.retrieve(key=created.key)

        assert retrieved.key == created.key
        assert retrieved.variant == created.variant
        assert retrieved.message == created.message

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a status and keep its message."""
        created = client.statuses.set(
            sy.Status(
                name="Before Rename",
                variant=sy.status.VARIANT_INFO,
                message="Renamed status",
            )
        )
        client.statuses.rename(created.key, "After Rename")
        renamed = client.statuses.retrieve(key=created.key)
        assert renamed.name == "After Rename"
        assert renamed.message == "Renamed status"

    def test_retrieve_multiple_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple statuses by keys."""
        statuses: list[sy.Status[None]] = [
            sy.Status(variant=sy.status.VARIANT_INFO, message="Status 1"),
            sy.Status(variant=sy.status.VARIANT_SUCCESS, message="Status 2"),
            sy.Status(variant=sy.status.VARIANT_WARNING, message="Status 3"),
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        retrieved = client.statuses.retrieve(keys=keys)

        assert len(retrieved) == 3
        retrieved_keys = {s.key for s in retrieved}
        assert all(k in retrieved_keys for k in keys)

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for statuses by message."""
        unique_term = f"unique_search_{uuid4()}"
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO,
            message=f"Status with {unique_term}",
            name=unique_term,
        )
        created = client.statuses.set(status)

        def check() -> None:
            results = client.statuses.retrieve(search_term=unique_term)
            assert len(results) >= 1
            assert any(s.key == created.key for s in results)

        assert_eventually(check)

    def test_retrieve_with_pagination(self, client: sy.Synnax) -> None:
        """Should paginate results."""
        statuses: list[sy.Status[None]] = [
            sy.Status(variant=sy.status.VARIANT_INFO, message=f"Paginated status {i}")
            for i in range(5)
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        page1 = client.statuses.retrieve(keys=keys, limit=2, offset=0)
        assert len(page1) == 2

        page2 = client.statuses.retrieve(keys=keys, limit=2, offset=2)
        assert len(page2) == 2

        page1_keys = {s.key for s in page1}
        page2_keys = {s.key for s in page2}
        assert page1_keys.isdisjoint(page2_keys)

    def test_delete_single_status(self, client: sy.Synnax) -> None:
        """Should delete a status by key."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO, message="To be deleted"
        )
        created = client.statuses.set(status)

        client.statuses.delete(created.key)

        with pytest.raises(sy.NotFoundError, match="not found"):
            client.statuses.retrieve(key=created.key)

    def test_delete_multiple_statuses(self, client: sy.Synnax) -> None:
        """Should delete multiple statuses."""
        statuses: list[sy.Status[None]] = [
            sy.Status(variant=sy.status.VARIANT_INFO, message=f"Delete me {i}")
            for i in range(3)
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        client.statuses.delete(keys)

        assert client.statuses.retrieve(keys=keys) == []

    def test_delete_idempotent(self, client: sy.Synnax) -> None:
        """Should be idempotent - deleting non-existent status should not error."""
        non_existent_key = str(uuid4())

        # Should not raise an error
        client.statuses.delete(non_existent_key)

    def test_all_variants(self, client: sy.Synnax) -> None:
        """Should support all status variants."""
        variants: list[sy.status.Variant] = [
            sy.status.VARIANT_SUCCESS,
            sy.status.VARIANT_INFO,
            sy.status.VARIANT_WARNING,
            sy.status.VARIANT_ERROR,
            sy.status.VARIANT_LOADING,
        ]

        statuses: list[sy.Status[None]] = [
            sy.Status(variant=variant, message=f"Testing {variant}")
            for variant in variants
        ]
        created = client.statuses.set(statuses)

        assert len(created) == len(variants)
        for i, variant in enumerate(variants):
            assert created[i].variant == variant

    def test_retrieve_with_single_variant_filter(self, client: sy.Synnax) -> None:
        """Should filter statuses by a single variant."""
        statuses: list[sy.Status[None]] = [
            sy.Status(
                variant=sy.status.VARIANT_SUCCESS,
                message="Success status",
            ),
            sy.Status(
                variant=sy.status.VARIANT_ERROR,
                message="Error status",
            ),
            sy.Status(
                variant=sy.status.VARIANT_WARNING,
                message="Warning status",
            ),
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        results = client.statuses.retrieve(keys=keys, variants=["error"])

        assert len(results) == 1
        assert results[0].variant == sy.status.VARIANT_ERROR
        assert results[0].message == "Error status"

    def test_retrieve_with_multiple_variant_filter(self, client: sy.Synnax) -> None:
        """Should filter statuses by multiple variants."""
        statuses: list[sy.Status[None]] = [
            sy.Status(
                variant=sy.status.VARIANT_INFO,
                message="Info status",
            ),
            sy.Status(
                variant=sy.status.VARIANT_ERROR,
                message="Error status",
            ),
            sy.Status(
                variant=sy.status.VARIANT_SUCCESS,
                message="Success status",
            ),
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        results = client.statuses.retrieve(keys=keys, variants=["info", "success"])

        assert len(results) == 2
        result_variants = {s.variant for s in results}
        assert "info" in result_variants
        assert "success" in result_variants
        assert "error" not in result_variants

    def test_where(self, client: sy.Synnax) -> None:
        """Should keep statuses that match every condition and the other filters."""
        created = client.statuses.set(
            [
                sy.Status(variant=sy.status.VARIANT_INFO, message="Where 0"),
                sy.Status(variant=sy.status.VARIANT_ERROR, message="Where 1"),
                sy.Status(variant=sy.status.VARIANT_SUCCESS, message="Where 2"),
            ]
        )
        keys = [s.key for s in created]
        results = client.statuses.retrieve(
            ~sy.status.variant.eq(sy.status.VARIANT_ERROR), keys=keys
        )
        assert {s.key for s in results} == {created[0].key, created[2].key}
        assert client.statuses.retrieve(created[0].key).key == created[0].key
        error = client.statuses.retrieve(
            sy.status.variant.eq(sy.status.VARIANT_ERROR), keys=keys, one=True
        )
        assert error.key == created[1].key

    def test_empty_keys(self, client: sy.Synnax) -> None:
        """Should retrieve no statuses for an empty list of keys."""
        assert client.statuses.retrieve(keys=[]) == []

    def test_retrieve_with_no_matching_variants(self, client: sy.Synnax) -> None:
        """Should return empty when no statuses match the variant filter."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO,
            message="Info only",
        )
        created = client.statuses.set(status)

        results = client.statuses.retrieve(keys=[created.key], variants=["error"])

        assert len(results) == 0

    def test_status_with_description(self, client: sy.Synnax) -> None:
        """Should create status with description."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO,
            message="Main message",
            description="Detailed description",
        )
        created = client.statuses.set(status)

        assert created.description == "Detailed description"

    def test_set_with_parent(self, client: sy.Synnax) -> None:
        """Should create status with a parent ontology ID."""
        parent_group = client.groups.create(
            parent=sy.ontology.ROOT_ID, name="Status Parent Group"
        )
        parent_id = sy.group.ontology_id(parent_group.key)

        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_INFO, message="Child status"
        )
        created = client.statuses.set(status, parent=parent_id)

        assert created.key is not None

        parent_id_str = f"group:{parent_group.key!s}"
        children = client.ontology.retrieve_children(parent_id_str)
        assert any(r.key == created.key for r in children)

    def test_ontology_id_helper(self) -> None:
        """Should create proper ontology ID."""
        key = "test-status-key"
        oid = sy.status.ontology_id(key)

        assert oid.type == "status"
        assert oid.key == key

    def test_retrieve_nonexistent_key(self, client: sy.Synnax) -> None:
        """Should raise error when retrieving non-existent status."""
        with pytest.raises(sy.NotFoundError, match="not found"):
            client.statuses.retrieve(key="nonexistent-key")

    def test_status_persistence(self, client: sy.Synnax) -> None:
        """Should persist status across operations."""
        status: sy.Status[None] = sy.Status(
            variant=sy.status.VARIANT_SUCCESS,
            message="Persistent status",
            name="Persistent",
        )
        created = client.statuses.set(status)
        retrieved = client.statuses.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.variant == created.variant
        assert retrieved.message == created.message
        assert retrieved.name == created.name
        updated_status: sy.Status[None] = sy.Status(
            key=created.key,
            variant=sy.status.VARIANT_ERROR,
            message="Updated persistent status",
        )
        client.statuses.set(updated_status)
        final = client.statuses.retrieve(key=created.key)
        assert final.variant == sy.status.VARIANT_ERROR
        assert final.message == "Updated persistent status"
