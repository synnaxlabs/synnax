#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from uuid import uuid4

import pytest

import synnax as sy


@pytest.mark.arc
class TestArc:
    """Tests for Arc CRUD operations."""

    def test_create_from_kwargs(self, client: sy.Synnax):
        """Should create an Arc from keyword arguments."""
        name = f"test-arc-{uuid4()}"
        arc = client.arcs.create(
            name=name,
            mode="text",
            text=sy.Text(raw="x := 1"),
        )
        assert arc.name == name
        assert arc.key != sy.Arc().key
        assert arc.mode == "text"
        assert arc.text.raw == "x := 1"

    def test_create_from_instance(self, client: sy.Synnax):
        """Should create an Arc from an existing instance."""
        name = f"test-arc-{uuid4()}"
        arc = sy.Arc(
            name=name,
            mode="text",
            text=sy.Text(raw="y := 2"),
        )
        created = client.arcs.create(arc)
        assert created.name == name
        assert created.key != sy.Arc().key
        assert created.text.raw == "y := 2"

    def test_create_multiple(self, client: sy.Synnax):
        """Should create multiple Arcs at once."""
        arcs = [
            sy.Arc(name=f"test-arc-multi-{uuid4()}", mode="text"),
            sy.Arc(name=f"test-arc-multi-{uuid4()}", mode="text"),
        ]
        created = client.arcs.create(arcs)
        assert len(created) == 2
        for arc in created:
            assert arc.name.startswith("test-arc-multi-")
            assert arc.key != sy.Arc().key

    def test_retrieve_by_key(self, client: sy.Synnax):
        """Should retrieve an Arc by key."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        retrieved = client.arcs.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == name

    def test_retrieve_by_name(self, client: sy.Synnax):
        """Should retrieve an Arc by name."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        retrieved = client.arcs.retrieve(name=name)
        assert retrieved.key == created.key
        assert retrieved.name == name

    def test_retrieve_multiple_by_keys(self, client: sy.Synnax):
        """Should retrieve multiple Arcs by keys."""
        arcs = [
            sy.Arc(name=f"test-arc-keys-{uuid4()}", mode="text"),
            sy.Arc(name=f"test-arc-keys-{uuid4()}", mode="text"),
        ]
        created = client.arcs.create(arcs)
        keys = [arc.key for arc in created]
        retrieved = client.arcs.retrieve(keys=keys)
        assert len(retrieved) == 2
        retrieved_keys = {arc.key for arc in retrieved}
        assert retrieved_keys == set(keys)

    def test_retrieve_not_found(self, client: sy.Synnax):
        """Should raise NotFoundError when Arc is not found."""
        with pytest.raises(sy.NotFoundError):
            client.arcs.retrieve(name=f"nonexistent-arc-{uuid4()}")

    def test_delete_single(self, client: sy.Synnax):
        """Should delete a single Arc."""
        name = f"test-arc-delete-{uuid4()}"
        arc = client.arcs.create(name=name, mode="text")
        client.arcs.delete(arc.key)
        with pytest.raises(sy.NotFoundError):
            client.arcs.retrieve(key=arc.key)

    def test_delete_multiple(self, client: sy.Synnax):
        """Should delete multiple Arcs."""
        arcs = [
            sy.Arc(name=f"test-arc-delete-multi-{uuid4()}", mode="text"),
            sy.Arc(name=f"test-arc-delete-multi-{uuid4()}", mode="text"),
        ]
        created = client.arcs.create(arcs)
        keys = [arc.key for arc in created]
        client.arcs.delete(keys)
        for key in keys:
            with pytest.raises(sy.NotFoundError):
                client.arcs.retrieve(key=key)

    def test_search(self, client: sy.Synnax):
        """Should search for Arcs by name."""
        unique_prefix = f"searchable-arc-{uuid4()}"
        client.arcs.create(name=f"{unique_prefix}-1", mode="text")
        client.arcs.create(name=f"{unique_prefix}-2", mode="text")
        time.sleep(0.2)
        results = client.arcs.retrieve(search_term=unique_prefix)
        assert len(results) >= 2
        for arc in results:
            assert unique_prefix in arc.name


@pytest.mark.arc
class TestArcGraph:
    """Tests for Arc graph mode functionality."""

    def test_graph_with_edges(self, client: sy.Synnax):
        """Should create an Arc with graph nodes and edges."""
        name = f"test-arc-graph-{uuid4()}"
        graph = sy.Graph(
            nodes=[
                sy.GraphNode(
                    key="node1",
                    type="constant",
                    config={"value": 1},
                    position=sy.Position(x=0, y=0),
                ),
                sy.GraphNode(
                    key="node2",
                    type="add",
                    config={},
                    position=sy.Position(x=100, y=0),
                ),
            ],
            edges=[
                sy.Edge(
                    source=sy.Handle(param="output", node="node1"),
                    target=sy.Handle(param="input1", node="node2"),
                ),
            ],
        )
        arc = client.arcs.create(name=name, mode="graph", graph=graph)
        assert arc.mode == "graph"
        assert len(arc.graph.nodes) == 2
        assert len(arc.graph.edges) == 1
        assert arc.graph.nodes[0].key == "node1"
        assert arc.graph.edges[0].source.node == "node1"
        assert arc.graph.edges[0].target.node == "node2"


@pytest.mark.arc
class TestArcTask:
    """Tests for Arc task functionality."""

    def test_create_arc_task(self, client: sy.Synnax):
        """Should create an Arc task."""
        arc = client.arcs.create(
            name=f"test-arc-task-{uuid4()}",
            mode="text",
            text=sy.Text(raw="x := 1"),
        )
        task = sy.arc.Task(name="Test Arc Task", arc_key=arc.key, auto_start=False)
        assert task.config.arc_key == str(arc.key)
        assert task.config.auto_start is False
        assert task.TYPE == "arc"

    def test_arc_task_to_payload(self, client: sy.Synnax):
        """Should convert Arc task to payload."""
        arc = client.arcs.create(
            name=f"test-arc-task-{uuid4()}",
            mode="text",
        )
        task = sy.arc.Task(name="Test Arc Task", arc_key=arc.key)
        payload = task.to_payload()
        assert payload.type == "arc"
        assert payload.name == "Test Arc Task"
        assert payload.config["arc_key"] == str(arc.key)

    def test_arc_task_requires_arc_key(self):
        """Should raise error when arc_key is not provided."""
        with pytest.raises(ValueError, match="arc_key is required"):
            sy.arc.Task(name="Test")

    def test_create_arc_task_on_cluster(self, client: sy.Synnax):
        """Should create an Arc task on the cluster."""
        arc = client.arcs.create(
            name=f"test-arc-task-cluster-{uuid4()}",
            mode="text",
            text=sy.Text(raw="x := 1"),
        )
        task_name = f"Test Arc Task {uuid4()}"
        task = sy.arc.Task(name=task_name, arc_key=arc.key, auto_start=False)
        # Create task using to_payload pattern (matching modbus/opcua tests)
        created = client.tasks.create(
            name=task_name,
            type="arc",
            config=task.to_payload().config,
        )
        assert created.key != 0
        assert created.name == task_name
        assert created.type == "arc"
        assert created.config["arc_key"] == str(arc.key)

    def test_retrieve_arc_task(self, client: sy.Synnax):
        """Should retrieve an Arc task from the cluster."""
        arc = client.arcs.create(
            name=f"test-arc-task-retrieve-{uuid4()}",
            mode="text",
        )
        task_name = f"Test Arc Task {uuid4()}"
        task = sy.arc.Task(name=task_name, arc_key=arc.key)
        created = client.tasks.create(
            name=task_name,
            type="arc",
            config=task.to_payload().config,
        )
        retrieved = client.tasks.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.type == "arc"

    def test_wrap_retrieved_task_as_arc_task(self, client: sy.Synnax):
        """Should wrap a retrieved task as an ArcTask."""
        arc = client.arcs.create(
            name=f"test-arc-task-wrap-{uuid4()}",
            mode="text",
        )
        task_name = f"Test Arc Task {uuid4()}"
        task = sy.arc.Task(name=task_name, arc_key=arc.key)
        created = client.tasks.create(
            name=task_name,
            type="arc",
            config=task.to_payload().config,
        )
        retrieved = client.tasks.retrieve(key=created.key)
        # Wrap the generic Task as an ArcTask
        arc_task = sy.arc.Task(internal=retrieved)
        assert arc_task.config.arc_key == str(arc.key)
        assert arc_task.config.auto_start is False
