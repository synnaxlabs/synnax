#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, get_args
from uuid import uuid4

import pytest
from pydantic import BaseModel, TypeAdapter, ValidationError

import synnax as sy
from synnax import group, schematic
from synnax.exceptions import NotFoundError
from synnax.ontology import ID
from synnax.schematic import symbol
from x import spatial
from x.testutil import assert_eventually


def variants(union: Any) -> list[type[BaseModel]]:
    return list(get_args(get_args(union)[0]))


def defaults_only(cls: type[BaseModel]) -> bool:
    return all(not f.is_required() for f in cls.model_fields.values())


ELEMENT = TypeAdapter(schematic.ElementConfig)
NODE = TypeAdapter(schematic.NodeConfig)
EDGE = TypeAdapter(schematic.EdgeConfig)


class TestSchematic:
    def test_defaults(self) -> None:
        """Should create an empty, editable schematic from only a name."""
        s = schematic.Schematic(name="Tank")
        assert s.snapshot is False
        assert s.nodes == []
        assert s.edges == []
        assert s.configs == {}

    def test_round_trip(self) -> None:
        """Should parse nodes, edges, and configs and dump them back unchanged."""
        s = schematic.Schematic.model_validate(
            {
                "name": "Tank",
                "nodes": [
                    {"key": "valve", "position": {"x": 10, "y": 20}},
                    {"key": "tank", "position": {"x": 50, "y": 20}, "z_index": 2},
                ],
                "edges": [
                    {
                        "key": "pipe",
                        "source": {"node": "valve", "param": "right"},
                        "target": {"node": "tank", "param": "left"},
                    }
                ],
                "configs": {
                    "valve": {"variant": "solenoid_valve"},
                    "tank": {"variant": "tank"},
                    "pipe": {"variant": "pipe"},
                },
            }
        )
        assert isinstance(s.configs["valve"], schematic.SolenoidValveElementConfig)
        assert isinstance(s.configs["tank"], schematic.TankElementConfig)
        assert isinstance(s.configs["pipe"], schematic.PipeElementConfig)
        assert schematic.Schematic.model_validate_json(s.model_dump_json()) == s

    def test_unknown_variant(self) -> None:
        """Should reject a config with an unknown variant."""
        with pytest.raises(ValidationError):
            schematic.Schematic.model_validate(
                {"name": "Tank", "configs": {"a": {"variant": "teleporter"}}}
            )

    def test_hash(self) -> None:
        """Should hash a schematic by its key."""
        s = schematic.Schematic(name="Tank")
        assert hash(s) == hash(s.key)

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a schematic."""
        key = uuid4()
        id = schematic.ontology_id(key)
        assert id.type == schematic.ONTOLOGY_TYPE.type == "schematic"
        assert id.key == str(key)


class TestElementConfig:
    @pytest.mark.parametrize(
        "cls",
        [c for c in variants(schematic.ElementConfig) if defaults_only(c)],
        ids=lambda c: c.__name__,
    )
    def test_dispatch(self, cls: type[BaseModel]) -> None:
        """Should select each element variant by its discriminator."""
        variant = cls.model_fields["variant"].default
        config = ELEMENT.validate_python({"variant": variant})
        assert type(config) is cls
        assert ELEMENT.validate_json(ELEMENT.dump_json(config)) == config

    def test_custom_actuator(self) -> None:
        """Should require the spec key of a custom actuator."""
        config = ELEMENT.validate_python(
            {"variant": "custom_actuator", "spec_key": "pump"}
        )
        assert isinstance(config, schematic.CustomActuatorElementConfig)
        assert config.state_overrides == []
        with pytest.raises(ValidationError):
            ELEMENT.validate_python({"variant": "custom_actuator"})

    def test_group_box(self) -> None:
        """Should keep the members of a group box."""
        config = ELEMENT.validate_python(
            {"variant": "group_box", "members": ["a", "b"]}
        )
        assert isinstance(config, schematic.GroupBoxElementConfig)
        assert config.members == ["a", "b"]
        assert config.locked is False


class TestNodeConfig:
    @pytest.mark.parametrize(
        "cls",
        [c for c in variants(schematic.NodeConfig) if defaults_only(c)],
        ids=lambda c: c.__name__,
    )
    def test_dispatch(self, cls: type[BaseModel]) -> None:
        """Should select each node variant by its discriminator."""
        variant = cls.model_fields["variant"].default
        assert type(NODE.validate_python({"variant": variant})) is cls


class TestEdgeConfig:
    @pytest.mark.parametrize(
        "cls", variants(schematic.EdgeConfig), ids=lambda c: c.__name__
    )
    def test_dispatch(self, cls: type[BaseModel]) -> None:
        """Should select each edge variant by its discriminator."""
        variant = cls.model_fields["variant"].default
        assert type(EDGE.validate_python({"variant": variant})) is cls

    def test_segments(self) -> None:
        """Should keep the segments of a segmented edge."""
        config = EDGE.validate_python(
            {"variant": "pipe", "segments": [{"direction": "x", "length": 10}]}
        )
        assert config.segments == [schematic.Segment(direction="x", length=10)]


class TestSymbol:
    def test_round_trip(self) -> None:
        """Should parse a symbol spec and dump it back unchanged."""
        s = symbol.Symbol.model_validate(
            {
                "name": "Pump",
                "data": {
                    "svg": "<svg/>",
                    "variant": "actuator",
                    "states": [
                        {
                            "key": "on",
                            "name": "On",
                            "regions": [
                                {"key": "body", "name": "Body", "selectors": ["#b"]}
                            ],
                        }
                    ],
                    "handles": [
                        {
                            "key": "in",
                            "position": {"x": 0, "y": 0.5},
                            "orientation": "left",
                        }
                    ],
                },
            }
        )
        assert s.data.states[0].regions[0].selectors == ["#b"]
        assert s.data.handles[0].orientation == "left"
        assert s.data.scale == 1
        assert symbol.Symbol.model_validate_json(s.model_dump_json()) == s
        assert hash(s) == hash(s.key)

    def test_empty_svg(self) -> None:
        """Should reject a symbol spec with an empty SVG."""
        with pytest.raises(ValidationError):
            symbol.Spec(svg="", variant="actuator")

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a symbol."""
        key = uuid4()
        assert symbol.ontology_id(key).type == symbol.ONTOLOGY_TYPE.type
        assert symbol.ontology_id(key).key == str(key)


class TestBorderRadius:
    def test_shorthand(self) -> None:
        """Should expand a shorthand border radius on an element config."""
        config = ELEMENT.validate_python({"variant": "cylinder", "border_radius": 4})
        assert isinstance(config, schematic.CylinderElementConfig)
        assert config.border_radius is not None
        assert config.border_radius.top_left.x == 4


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name=f"schematic-project-{uuid4()}")


@pytest.mark.schematic
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a schematic from a name."""
        s = client.schematics.create(project.key, name="Test Schematic")
        assert s.name == "Test Schematic"

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a schematic from a Schematic object."""
        s = client.schematics.create(project.key, sy.Schematic(name="From Object"))
        assert s.name == "From Object"

    def test_diagram_round_trip(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should keep nodes, edges, and configs through create, retrieve, and copy."""
        diagram = sy.Schematic(
            name="Diagram",
            nodes=[
                schematic.Node(key="a", position=spatial.XY(x=10, y=20)),
                schematic.Node(key="b", position=spatial.XY(x=30, y=40), z_index=2),
            ],
            edges=[
                schematic.Edge(
                    key="e",
                    source=schematic.Handle(node="a", param="outlet"),
                    target=schematic.Handle(node="b", param="inlet"),
                )
            ],
            configs={
                "a": schematic.TextBoxElementConfig(value="Inlet"),
                "b": schematic.TankElementConfig(),
                "e": schematic.PipeElementConfig(),
            },
        )
        created = client.schematics.create(project.key, diagram)
        retrieved = client.schematics.retrieve(key=created.key)
        copied = client.schematics.copy(created.key, name="Diagram Copy")
        for s in (created, retrieved, copied):
            assert s.nodes == diagram.nodes
            assert s.edges == diagram.edges
            assert s.configs == diagram.configs

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple schematics."""
        schematics = client.schematics.create(
            project.key,
            [sy.Schematic(name="Multi 1"), sy.Schematic(name="Multi 2")],
        )
        assert [s.name for s in schematics] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_key(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve a schematic by key."""
        created = client.schematics.create(project.key, name="Retrieve Me")
        retrieved = client.schematics.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple schematics by keys."""
        s1 = client.schematics.create(project.key, name="Schematic A")
        s2 = client.schematics.create(project.key, name="Schematic B")
        retrieved = client.schematics.retrieve(keys=[s1.key, s2.key])
        assert {s.key for s in retrieved} == {s1.key, s2.key}

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the key does not exist."""
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a schematic."""
        s = client.schematics.create(project.key, name="Before Rename")
        client.schematics.rename(s.key, "After Rename")
        assert client.schematics.retrieve(key=s.key).name == "After Rename"

    def test_copy(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should copy a schematic under a new key and name."""
        s = client.schematics.create(project.key, name="Original")
        c = client.schematics.copy(s.key, name="Copy")
        assert c.key != s.key
        assert c.name == "Copy"
        assert not c.snapshot
        assert client.schematics.retrieve(key=c.key).name == "Copy"

    def test_copy_snapshot(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should copy a schematic as an immutable snapshot."""
        s = client.schematics.create(project.key, name="Original")
        c = client.schematics.copy(s.key, name="Snapshot", snapshot=True)
        assert c.snapshot

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a schematic."""
        s = client.schematics.create(project.key, name="To Delete")
        client.schematics.delete(s.key)
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(key=s.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple schematics."""
        s1 = client.schematics.create(project.key, name="Delete 1")
        s2 = client.schematics.create(project.key, name="Delete 2")
        client.schematics.delete([s1.key, s2.key])
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(keys=[s1.key, s2.key])


def _symbol(name: str) -> symbol.Symbol:
    return symbol.Symbol(
        name=name, data=symbol.Spec(svg="<svg></svg>", variant="static")
    )


@pytest.fixture
def parent_group(client: sy.Synnax) -> group.Group:
    root = client.schematics.symbols.retrieve_group()
    return client.groups.create(group.ontology_id(root.key), f"symbols-{uuid4()}")


@pytest.fixture
def parent(parent_group: group.Group) -> ID:
    return group.ontology_id(parent_group.key)


@pytest.mark.schematic
class TestSymbolClient:
    def test_create(self, client: sy.Synnax, parent: ID) -> None:
        """Should create a single symbol."""
        s = client.schematics.symbols.create(_symbol("Test Symbol"), parent=parent)
        assert s.name == "Test Symbol"
        assert s.data.svg == "<svg></svg>"

    def test_create_multiple(self, client: sy.Synnax, parent: ID) -> None:
        """Should create multiple symbols."""
        symbols = client.schematics.symbols.create(
            [_symbol("Multi 1"), _symbol("Multi 2")], parent=parent
        )
        assert [s.name for s in symbols] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_key(self, client: sy.Synnax, parent: ID) -> None:
        """Should retrieve a symbol by key."""
        created = client.schematics.symbols.create(
            _symbol("Retrieve Me"), parent=parent
        )
        retrieved = client.schematics.symbols.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, parent: ID) -> None:
        """Should retrieve multiple symbols by keys."""
        s1, s2 = client.schematics.symbols.create(
            [_symbol("Symbol A"), _symbol("Symbol B")], parent=parent
        )
        retrieved = client.schematics.symbols.retrieve(keys=[s1.key, s2.key])
        assert {s.key for s in retrieved} == {s1.key, s2.key}

    def test_retrieve_with_search_term(self, client: sy.Synnax, parent: ID) -> None:
        """Should search for symbols by name."""
        prefix = f"searchable-symbol-{uuid4()}"
        client.schematics.symbols.create(
            [_symbol(f"{prefix}-1"), _symbol(f"{prefix}-2")], parent=parent
        )

        def check() -> None:
            results = client.schematics.symbols.retrieve(search_term=prefix)
            assert len(results) >= 2
            assert all(prefix in s.name for s in results)

        assert_eventually(check)

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the key does not exist."""
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, parent: ID) -> None:
        """Should rename a symbol."""
        s = client.schematics.symbols.create(_symbol("Before Rename"), parent=parent)
        client.schematics.symbols.rename(s.key, "After Rename")
        assert client.schematics.symbols.retrieve(key=s.key).name == "After Rename"

    def test_delete(self, client: sy.Synnax, parent: ID) -> None:
        """Should delete symbols."""
        s1, s2 = client.schematics.symbols.create(
            [_symbol("Delete 1"), _symbol("Delete 2")], parent=parent
        )
        client.schematics.symbols.delete([s1.key, s2.key])
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=s1.key)

    def test_retrieve_group(self, client: sy.Synnax) -> None:
        """Should retrieve the same symbol group on each call."""
        g = client.schematics.symbols.retrieve_group()
        assert client.schematics.symbols.retrieve_group().key == g.key

    def test_delete_group(
        self, client: sy.Synnax, parent_group: group.Group, parent: ID
    ) -> None:
        """Should delete a group and every symbol in it."""
        s = client.schematics.symbols.create(_symbol("In Group"), parent=parent)
        client.schematics.symbols.delete_group(parent_group.key)
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=s.key)
        with pytest.raises(NotFoundError):
            client.groups.retrieve(parent_group.key)
