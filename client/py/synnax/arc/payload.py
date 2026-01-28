#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal
from uuid import UUID

from pydantic import BaseModel

from freighter import Payload
from synnax.ontology.payload import ID

ARC_ONTOLOGY_TYPE = ID(type="arc")
"""Ontology type identifier for Arc entities."""


def ontology_id(key: UUID) -> ID:
    """Returns the ontology ID for the Arc entity."""
    return ID(type=ARC_ONTOLOGY_TYPE.type, key=str(key))


ArcKey = UUID
ArcMode = Literal["graph", "text"]


class Handle(BaseModel):
    """A connection point on a graph node."""

    param: str
    """The parameter name on the node."""
    node: str
    """The key of the node this handle belongs to."""


class Edge(BaseModel):
    """A connection between two handles in the graph."""

    source: Handle
    """The source handle of the edge."""
    target: Handle
    """The target handle of the edge."""


class Position(BaseModel):
    """A 2D position in the graph editor."""

    x: float = 0
    y: float = 0


class GraphNode(BaseModel):
    """A node in the visual graph representation of an Arc program."""

    key: str
    """Unique identifier for the node."""
    type: str
    """The type of node (e.g., 'add', 'multiply', 'channel')."""
    config: dict[str, Any] = {}
    """Configuration parameters for the node."""
    position: Position = Position()
    """The position of the node in the graph editor."""


class Graph(BaseModel):
    """The visual graph representation of an Arc program."""

    nodes: list[GraphNode] = []
    """The nodes in the graph."""
    edges: list[Edge] = []
    """The edges connecting nodes in the graph."""


class Text(BaseModel):
    """The text-based representation of an Arc program."""

    raw: str = ""
    """The raw source code of the Arc program."""


class ArcPayload(Payload):
    """Network transportable payload representing an Arc program."""

    key: UUID = UUID(int=0)
    """Unique identifier for the Arc."""
    name: str = ""
    """Human-readable name for the Arc."""
    text: Text = Text()
    """Text-based representation of the program."""
    version: str = ""
    """Version string for the Arc."""
    mode: ArcMode = "text"
    """The active mode of the Arc (graph or text)."""
    graph: Graph = Graph()
    """Visual graph representation of the program."""
