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

from synnax import schematic
from synnax.schematic import symbol


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
