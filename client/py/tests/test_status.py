#  Copyright 2025 Synnax Labs, Inc.
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


@pytest.mark.status
class TestStatusClient:
    """Tests for the status client."""

    def test_set_single_status(self, client: sy.Synnax):
        """Should create a single status."""
        status = sy.Status(
            variant=sy.status.INFO_VARIANT,
            message="Test status message",
            name="Test Status",
        )
        created = client.statuses.set(status)

        assert created.key == status.key
        assert created.variant == sy.status.INFO_VARIANT
        assert created.message == "Test status message"
        assert created.name == "Test Status"
        assert created.time is not None

    def test_set_multiple_statuses(self, client: sy.Synnax):
        """Should create multiple statuses at once."""
        statuses = [
            sy.Status(variant=sy.status.SUCCESS_VARIANT, message="Task 1 complete"),
            sy.Status(variant=sy.status.ERROR_VARIANT, message="Task 2 failed"),
            sy.Status(variant=sy.status.WARNING_VARIANT, message="Task 3 warning"),
        ]
        created = client.statuses.set(statuses)

        assert len(created) == 3
        assert created[0].variant == sy.status.SUCCESS_VARIANT
        assert created[1].variant == sy.status.ERROR_VARIANT
        assert created[2].variant == sy.status.WARNING_VARIANT

    def test_update_existing_status(self, client: sy.Synnax):
        """Should update an existing status."""
        key = str(uuid4())
        original = sy.Status(
            key=key,
            variant=sy.status.INFO_VARIANT,
            message="Original message",
        )
        client.statuses.set(original)

        updated = sy.Status(
            key=key,
            variant=sy.status.SUCCESS_VARIANT,
            message="Updated message",
        )
        result = client.statuses.set(updated)

        assert result.key == key
        assert result.variant == sy.status.SUCCESS_VARIANT
        assert result.message == "Updated message"

    def test_retrieve_by_key(self, client: sy.Synnax):
        """Should retrieve a status by key."""
        status = sy.Status(variant=sy.status.INFO_VARIANT, message="Retrievable status")
        created = client.statuses.set(status)

        retrieved = client.statuses.retrieve(key=created.key)

        assert retrieved.key == created.key
        assert retrieved.variant == created.variant
        assert retrieved.message == created.message

    def test_retrieve_multiple_by_keys(self, client: sy.Synnax):
        """Should retrieve multiple statuses by keys."""
        statuses = [
            sy.Status(variant=sy.status.INFO_VARIANT, message="Status 1"),
            sy.Status(variant=sy.status.SUCCESS_VARIANT, message="Status 2"),
            sy.Status(variant=sy.status.WARNING_VARIANT, message="Status 3"),
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        retrieved = client.statuses.retrieve(keys=keys)

        assert len(retrieved) == 3
        retrieved_keys = {s.key for s in retrieved}
        assert all(k in retrieved_keys for k in keys)

    def test_retrieve_with_search_term(self, client: sy.Synnax):
        """Should search for statuses by message."""
        unique_term = f"unique_search_{uuid4()}"
        status = sy.Status(
            variant=sy.status.INFO_VARIANT,
            message=f"Status with {unique_term}",
            name=unique_term,
        )
        created = client.statuses.set(status)

        results = client.statuses.retrieve(search_term=unique_term)

        assert len(results) >= 1
        assert any(s.key == created.key for s in results)

    def test_retrieve_with_pagination(self, client: sy.Synnax):
        """Should paginate results."""
        statuses = [
            sy.Status(variant=sy.status.INFO_VARIANT, message=f"Paginated status {i}")
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

    def test_delete_single_status(self, client: sy.Synnax):
        """Should delete a status by key."""
        status = sy.Status(variant=sy.status.INFO_VARIANT, message="To be deleted")
        created = client.statuses.set(status)

        client.statuses.delete(created.key)

        with pytest.raises(sy.NotFoundError, match="not found"):
            client.statuses.retrieve(key=created.key)

    def test_delete_multiple_statuses(self, client: sy.Synnax):
        """Should delete multiple statuses."""
        statuses = [
            sy.Status(variant=sy.status.INFO_VARIANT, message=f"Delete me {i}")
            for i in range(3)
        ]
        created = client.statuses.set(statuses)
        keys = [s.key for s in created]

        client.statuses.delete(keys)

        with pytest.raises(sy.NotFoundError, match="not found"):
            client.statuses.retrieve(keys=keys)

    def test_delete_idempotent(self, client: sy.Synnax):
        """Should be idempotent - deleting non-existent status should not error."""
        non_existent_key = str(uuid4())

        # Should not raise an error
        client.statuses.delete(non_existent_key)

    def test_all_variants(self, client: sy.Synnax):
        """Should support all status variants."""
        variants = [
            sy.status.SUCCESS_VARIANT,
            sy.status.INFO_VARIANT,
            sy.status.WARNING_VARIANT,
            sy.status.ERROR_VARIANT,
            sy.status.LOADING_VARIANT,
        ]

        statuses = [
            sy.Status(variant=variant, message=f"Testing {variant}")
            for variant in variants
        ]
        created = client.statuses.set(statuses)

        assert len(created) == len(variants)
        for i, variant in enumerate(variants):
            assert created[i].variant == variant

    def test_status_with_description(self, client: sy.Synnax):
        """Should create status with description."""
        status = sy.Status(
            variant=sy.status.INFO_VARIANT,
            message="Main message",
            description="Detailed description",
        )
        created = client.statuses.set(status)

        assert created.description == "Detailed description"

    def test_set_with_parent(self, client: sy.Synnax):
        """Should create status with a parent ontology ID."""
        parent_group = client.ontology.groups.create(
            parent=sy.ontology.ROOT, name="Status Parent Group"
        )
        parent_id = {"type": "group", "key": str(parent_group.key)}

        status = sy.Status(variant=sy.status.INFO_VARIANT, message="Child status")
        created = client.statuses.set(status, parent=parent_id)

        assert created.key is not None

        parent_id_str = f"group:{str(parent_group.key)}"
        children = client.ontology.retrieve_children(parent_id_str)
        status_resource = next((r for r in children if r.id.key == created.key), None)
        assert status_resource is not None

    def test_ontology_id_helper(self):
        """Should create proper ontology ID."""
        key = "test-status-key"
        oid = ontology_id(key)

        assert oid["type"] == "status"
        assert oid["key"] == key

    def test_retrieve_nonexistent_key(self, client: sy.Synnax):
        """Should raise error when retrieving non-existent status."""
        with pytest.raises(sy.NotFoundError, match="not found"):
            client.statuses.retrieve(key="nonexistent-key")

    def test_status_persistence(self, client: sy.Synnax):
        """Should persist status across operations."""
        status = sy.Status(
            variant=sy.status.SUCCESS_VARIANT,
            message="Persistent status",
            name="Persistent",
        )
        created = client.statuses.set(status)
        retrieved = client.statuses.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.variant == created.variant
        assert retrieved.message == created.message
        assert retrieved.name == created.name
        updated_status = sy.Status(
            key=created.key,
            variant=sy.status.ERROR_VARIANT,
            message="Updated persistent status",
        )
        client.statuses.set(updated_status)
        final = client.statuses.retrieve(key=created.key)
        assert final.variant == sy.status.ERROR_VARIANT
        assert final.message == "Updated persistent status"
