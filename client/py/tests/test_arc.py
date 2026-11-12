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
from x import filter
from x.testutil import assert_eventually
from x.uuid import NIL


def text_of(arc: sy.arc.Arc) -> sy.Text:
    assert isinstance(arc.source, sy.arc.TextSource)
    return arc.source.text


def graph_of(arc: sy.arc.Arc) -> sy.Graph:
    assert isinstance(arc.source, sy.arc.GraphSource)
    return arc.source.graph


@pytest.mark.arc
class TestArc:
    """Tests for Arc CRUD operations."""

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create an Arc from keyword arguments."""
        name = f"test-arc-{uuid4()}"
        arc = client.arcs.create(
            name=name,
            mode="text",
            text=sy.Text(raw="x := 1"),
        )
        assert arc.name == name
        assert arc.key != NIL
        assert arc.source.mode == "text"
        assert text_of(arc).raw == "x := 1"

    def test_create_from_instance(self, client: sy.Synnax) -> None:
        """Should create an Arc from an existing instance."""
        name = f"test-arc-{uuid4()}"
        arc = sy.arc.Automation(
            name=name, source=sy.arc.TextSource(text=sy.Text(raw="y := 2"))
        )
        created = client.arcs.create(arc)
        assert created.name == name
        assert created.key != NIL
        assert text_of(created).raw == "y := 2"

    def test_create_rejects_the_body_of_the_other_mode(self, client: sy.Synnax) -> None:
        """Should reject a graph for a text Arc and text for a graph Arc."""
        with pytest.raises(ValueError, match="takes no graph"):
            client.arcs.create(name=f"test-arc-{uuid4()}", graph=sy.Graph())
        with pytest.raises(ValueError, match="takes no text"):
            client.arcs.create(
                name=f"test-arc-{uuid4()}", mode="graph", text=sy.Text(raw="x := 1")
            )

    def test_create_module(self, client: sy.Synnax) -> None:
        """Should create a module and read its type back."""
        name = f"test_module_{uuid4().hex}"
        created = client.arcs.create(name=name, mode="text", type="module")
        assert isinstance(created, sy.arc.Module)
        assert isinstance(client.arcs.retrieve(key=created.key), sy.arc.Module)

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple Arcs at once."""
        arcs = [
            sy.arc.Automation(name=f"test-arc-multi-{uuid4()}"),
            sy.arc.Automation(name=f"test-arc-multi-{uuid4()}"),
        ]
        created = client.arcs.create(arcs)
        assert len(created) == 2
        for arc in created:
            assert arc.name.startswith("test-arc-multi-")
            assert arc.key != NIL

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve an Arc by key."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        retrieved = client.arcs.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == name

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve an Arc by name."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        retrieved = client.arcs.retrieve(name=name)
        assert retrieved.key == created.key
        assert retrieved.name == name

    def test_retrieve_multiple_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple Arcs by keys."""
        arcs = [
            sy.arc.Automation(name=f"test-arc-keys-{uuid4()}"),
            sy.arc.Automation(name=f"test-arc-keys-{uuid4()}"),
        ]
        created = client.arcs.create(arcs)
        keys = [arc.key for arc in created]
        retrieved = client.arcs.retrieve(keys=keys)
        assert len(retrieved) == 2
        retrieved_keys = {arc.key for arc in retrieved}
        assert retrieved_keys == set(keys)

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep Arcs that match a filter tree and the given keys."""
        prefix = f"test-arc-where-{uuid4()}"
        created = client.arcs.create(
            [
                sy.arc.Automation(name=f"{prefix}-a"),
                sy.arc.Automation(name=f"{prefix}-b"),
            ]
        )
        keys = [arc.key for arc in created]
        where = sy.arc.Where(name=filter.String(regex=f"^{prefix}-b$"))
        retrieved = client.arcs.retrieve(where=where, keys=keys)
        assert [arc.key for arc in retrieved] == [created[1].key]

    def test_retrieve_by_directory(self, client: sy.Synnax) -> None:
        """Should keep no Arcs for a directory that holds none."""
        created = client.arcs.create(name=f"test-arc-{uuid4()}", mode="text")
        assert client.arcs.retrieve(key=[created.key], directories=uuid4()) == []

    def test_retrieve_name_operator(self, client: sy.Synnax) -> None:
        """Should retrieve the Arcs whose name matches an operator."""
        prefix = f"test-arc-op-{uuid4()}"
        created = client.arcs.create(
            [
                sy.arc.Automation(name=f"{prefix}-a"),
                sy.arc.Automation(name=f"{prefix}-b"),
            ]
        )
        retrieved = client.arcs.retrieve(name={"starts_with": prefix})
        assert {arc.key for arc in retrieved} == {arc.key for arc in created}

    def test_retrieve_plain_key_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one Arc for a plain key beside other fields."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        found = client.arcs.retrieve(key=created.key, name={"in": [name]})
        assert found.key == created.key
        with pytest.raises(sy.NotFoundError):
            client.arcs.retrieve(key=created.key, name="other")

    def test_retrieve_one(self, client: sy.Synnax) -> None:
        """Should return one Arc for any filter when one is set."""
        name = f"test-arc-{uuid4()}"
        created = client.arcs.create(name=name, mode="text")
        found = client.arcs.retrieve(name={"starts_with": name}, one=True)
        assert found.key == created.key

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every Arc."""
        with pytest.raises(sy.ValidationError, match="key is None"):
            client.arcs.retrieve(key=None)  # type: ignore[call-overload]

    def test_retrieve_compiled(self, client: sy.Synnax) -> None:
        """Should return the compiled program when include_program is True."""
        created = client.arcs.create(
            name=f"test-arc-{uuid4()}",
            mode="text",
            text=sy.Text(raw="func f() {}"),
        )
        retrieved = client.arcs.retrieve(key=created.key, include_program=True)
        assert isinstance(retrieved, sy.arc.Automation)
        assert retrieved.program is not None
        assert retrieved.program.wasm.startswith(b"\x00asm")

    def test_retrieve_module_status(self, client: sy.Synnax) -> None:
        """Should return a module's compile status when include_status is True."""
        created = client.arcs.create(
            name=f"test_module_{uuid4().hex}",
            type="module",
            mode="text",
            text=sy.Text(raw="export func f() i64 { return 1 }"),
        )
        assert isinstance(client.arcs.retrieve(key=created.key), sy.arc.Module)

        def compiled() -> None:
            retrieved = client.arcs.retrieve(key=created.key, include_status=True)
            assert isinstance(retrieved, sy.arc.Module)
            assert retrieved.status is not None
            assert retrieved.status.message == "Compiled"

        assert_eventually(compiled)

    def test_retrieve_changes(self, client: sy.Synnax) -> None:
        """Should name an imported module that changed after the deploy."""
        path = f"test_changes_{uuid4().hex}"
        module = client.arcs.create(
            name=path,
            type="module",
            mode="text",
            text=sy.Text(raw="export func g() i64 { return 1 }"),
        )
        created = client.arcs.create(
            name=f"test_changes_{uuid4().hex}",
            mode="text",
            text=sy.Text(raw=f"import {path}\n\nfunc f() i64 {{ return {path}.g() }}"),
        )
        assert isinstance(created, sy.arc.Automation)
        rack = client.racks.create(name=f"test_changes_{uuid4().hex}")
        tsk = client.arcs.update_task(created.key, rack=rack.key)
        assert tsk is not None
        retrieved = client.arcs.retrieve(key=created.key, include_changes=True)
        assert isinstance(retrieved, sy.arc.Automation)
        assert not retrieved.changes
        client.statuses.set(
            sy.task.Status(
                variant=sy.status.VARIANT_SUCCESS,
                message="Task running",
                details=sy.task.StatusDetails(
                    running=True, config_hash=tsk.config_hash
                ),
                owner=tsk.resource_id,
            )
        )
        client.arcs.set_text(module.key, "export func g() i64 { return 2 }")

        def changed() -> None:
            retrieved = client.arcs.retrieve(key=created.key, include_changes=True)
            assert isinstance(retrieved, sy.arc.Automation)
            assert retrieved.changes == [
                sy.arc.Change(type="module", path=path, kind="changed")
            ]

        assert_eventually(changed)

    def test_retrieve_not_compiled(self, client: sy.Synnax) -> None:
        """Should return no program when include_program is not set."""
        created = client.arcs.create(
            name=f"test-arc-{uuid4()}",
            mode="text",
            text=sy.Text(raw="func f() {}"),
        )
        retrieved = client.arcs.retrieve(key=created.key)
        assert isinstance(retrieved, sy.arc.Automation)
        assert retrieved.program is None

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when Arc is not found."""
        with pytest.raises(sy.NotFoundError):
            client.arcs.retrieve(name=f"nonexistent-arc-{uuid4()}")

    def test_set_text(self, client: sy.Synnax) -> None:
        """Should replace the text of an Arc on each call."""
        arc = client.arcs.create(
            name=f"test-arc-{uuid4()}", mode="text", text=sy.Text(raw="x := 1")
        )
        client.arcs.set_text(arc.key, "y := 2")
        assert text_of(client.arcs.retrieve(key=arc.key)).raw == "y := 2"
        client.arcs.set_text(arc.key, "z := 3 // é")
        assert text_of(client.arcs.retrieve(key=arc.key)).raw == "z := 3 // é"

    def test_set_text_of_a_graph(self, client: sy.Synnax) -> None:
        """Should refuse to set the text of a graph-mode Arc."""
        arc = client.arcs.create(name=f"test-arc-{uuid4()}", mode="graph")
        with pytest.raises(ValueError, match="graph-mode Arc and has no text"):
            client.arcs.set_text(arc.key, "x := 1")

    def test_delete_single(self, client: sy.Synnax) -> None:
        """Should delete a single Arc."""
        name = f"test-arc-delete-{uuid4()}"
        arc = client.arcs.create(name=name, mode="text")
        client.arcs.delete(arc.key)
        with pytest.raises(sy.NotFoundError):
            client.arcs.retrieve(key=arc.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple Arcs."""
        arcs = [
            sy.arc.Automation(name=f"test-arc-delete-multi-{uuid4()}"),
            sy.arc.Automation(name=f"test-arc-delete-multi-{uuid4()}"),
        ]
        created = client.arcs.create(arcs)
        keys = [arc.key for arc in created]
        client.arcs.delete(keys)
        for key in keys:
            with pytest.raises(sy.NotFoundError):
                client.arcs.retrieve(key=key)

    def test_search(self, client: sy.Synnax) -> None:
        """Should search for Arcs by name."""
        unique_prefix = f"searchable-arc-{uuid4()}"
        client.arcs.create(
            [
                sy.arc.Automation(name=f"{unique_prefix}-1"),
                sy.arc.Automation(name=f"{unique_prefix}-2"),
            ]
        )

        def check() -> None:
            results = client.arcs.retrieve(search_term=unique_prefix)
            assert len(results) >= 2
            for arc in results:
                assert unique_prefix in arc.name

        assert_eventually(check)


@pytest.mark.arc
class TestArcGraph:
    """Tests for Arc graph mode functionality."""

    def test_graph_with_edges(self, client: sy.Synnax) -> None:
        """Should create an Arc with graph nodes and edges."""
        name = f"test-arc-graph-{uuid4()}"
        graph = sy.Graph(
            nodes=[
                sy.GraphNode(key="node1", position=sy.Position(x=0, y=0)),
                sy.GraphNode(key="node2", position=sy.Position(x=100, y=0)),
            ],
            edges=[
                sy.GraphEdge(
                    key="edge1",
                    source=sy.Handle(param="output", node="node1"),
                    target=sy.Handle(param="input1", node="node2"),
                    kind=sy.arc.ir.EdgeKind.continuous,
                ),
            ],
            inputs={
                "node1": {"type": "constant", "value": 1},
                "node2": {"type": "add"},
            },
        )
        arc = client.arcs.create(name=name, mode="graph", graph=graph)
        assert arc.source.mode == "graph"
        assert len(graph_of(arc).nodes) == 2
        assert len(graph_of(arc).edges) == 1
        assert graph_of(arc).nodes[0].key == "node1"
        assert graph_of(arc).edges[0].source.node == "node1"
        assert graph_of(arc).edges[0].target.node == "node2"


@pytest.mark.arc
class TestArcTask:
    """Tests for Arc task functionality."""

    def test_create_arc_task(self, client: sy.Synnax) -> None:
        """Should create an Arc task."""
        arc = client.arcs.create(
            name=f"test-arc-task-{uuid4()}",
            mode="text",
            text=sy.Text(raw="x := 1"),
        )
        task = sy.arc.Task(name="Test Arc Task", arc_key=arc.key, auto_start=False)
        assert task.config.arc_key == arc.key
        assert task.config.auto_start is False
        assert task.TYPE == "arc"

    def test_arc_task_to_payload(self, client: sy.Synnax) -> None:
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

    def test_arc_task_keeps_stored_fields(self) -> None:
        """Should keep the hash and data saving flag of a stored config."""
        stored = {
            "arc_key": str(uuid4()),
            "hash": "abc",
            "data_saving_disabled": True,
        }
        task = sy.arc.Task(internal=sy.task.Task(type="arc", config=stored))
        config = task.to_payload().config
        assert config["hash"] == "abc"
        assert config["data_saving_disabled"] is True

    def test_arc_task_requires_arc_key(self) -> None:
        """Should raise error when arc_key is not provided."""
        with pytest.raises(ValueError, match="arc_key is required"):
            sy.arc.Task(name="Test")

    def test_create_arc_task_on_cluster(self, client: sy.Synnax) -> None:
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
        assert created.key is not None
        assert created.name == task_name
        assert created.type == "arc"
        assert created.config["arc_key"] == str(arc.key)

    def test_retrieve_arc_task(self, client: sy.Synnax) -> None:
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

    def test_retrieve_tasks_by_arc(self, client: sy.Synnax) -> None:
        """Should retrieve only the tasks that run the given Arcs."""
        arcs = [
            client.arcs.create(name=f"test-arc-by-arc-{uuid4()}", mode="text")
            for _ in range(2)
        ]
        created = [
            client.tasks.create(
                name=f"Test Arc Task {uuid4()}",
                type="arc",
                config=sy.arc.Task(name="", arc_key=a.key).to_payload().config,
            )
            for a in arcs
        ]
        retrieved = client.tasks.retrieve(arcs=[arcs[0].key])
        assert [t.key for t in retrieved] == [created[0].key]

    def test_wrap_retrieved_task_as_arc_task(self, client: sy.Synnax) -> None:
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
        assert arc_task.config.arc_key == arc.key
        assert arc_task.config.auto_start is False

    def test_update_task_binds_to_rack(self, client: sy.Synnax) -> None:
        """Should create the automation's task on the rack with its compiled hash."""
        arc = client.arcs.create(
            name=f"test-arc-bind-{uuid4()}", mode="text", text=sy.Text(raw="x := 1")
        )
        rack = client.racks.create(name=f"test-arc-bind-{uuid4()}")
        task = client.arcs.update_task(arc.key, rack=rack.key)
        assert task is not None
        assert task.rack == rack.key
        config = sy.arc.Task(internal=task).config
        assert config.arc_key == arc.key
        assert config.hash != ""

    def test_update_task_unbinds(self, client: sy.Synnax) -> None:
        """Should delete the task and return None for a rack of 0."""
        arc = client.arcs.create(name=f"test-arc-unbind-{uuid4()}", mode="text")
        rack = client.racks.create(name=f"test-arc-unbind-{uuid4()}")
        task = client.arcs.update_task(arc.key, rack=rack.key)
        assert task is not None
        assert client.arcs.update_task(arc.key, rack=0) is None
        with pytest.raises(sy.NotFoundError):
            client.tasks.retrieve(key=task.key)

    def test_update_task_rejects_module(self, client: sy.Synnax) -> None:
        """Should raise ValidationError when binding a module to a rack."""
        module = client.arcs.create(
            name=f"test_arc_bind_module_{uuid4().hex}", type="module", mode="text"
        )
        rack = client.racks.create(name=f"test-arc-bind-{uuid4()}")
        with pytest.raises(sy.ValidationError):
            client.arcs.update_task(module.key, rack=rack.key)

    def test_performance_defaults_to_auto(self) -> None:
        """Should default the performance level to auto."""
        task = sy.arc.Task(name="Test", arc_key=uuid4())
        assert task.to_payload().config["performance"] == "auto"

    def test_keep_performance_through_retrieve(self, client: sy.Synnax) -> None:
        """Should keep the performance level when a retrieved task is saved."""
        arc = client.arcs.create(name=f"test-arc-task-perf-{uuid4()}", mode="text")
        task = sy.arc.Task(
            name=f"Test Arc Task {uuid4()}", arc_key=arc.key, performance="high"
        )
        created = client.tasks.create(
            name=task.name, type="arc", config=task.to_payload().config
        )
        retrieved = sy.arc.Task(internal=client.tasks.retrieve(key=created.key))
        assert retrieved.config.performance == "high"
        assert retrieved.to_payload().config["performance"] == "high"
