#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.arc import compiler, graph, ir, program, text, types
from synnax.arc.client import Arc, Client
from synnax.arc.graph import Edge as GraphEdge
from synnax.arc.graph import Graph
from synnax.arc.graph import Node as GraphNode
from synnax.arc.ir import Edge, Handle
from synnax.arc.task import Task, TaskConfig
from synnax.arc.text import Text
from synnax.arc.types_gen import (
    ONTOLOGY_TYPE,
    Key,
    Mode,
    Payload,
    Status,
    StatusDetails,
    ontology_id,
)
from x.deprecation import deprecated_getattr
from x.spatial import XY as Position

_DEPRECATED = {
    "ArcTask": "Task",
    "ArcTaskConfig": "TaskConfig",
    "ArcClient": "Client",
    "ArcKey": "Key",
    "ArcMode": "Mode",
    "ArcPayload": "Payload",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "ONTOLOGY_TYPE",
    "Arc",
    "Client",
    "Edge",
    "Graph",
    "GraphEdge",
    "GraphNode",
    "Handle",
    "Key",
    "Mode",
    "Payload",
    "Position",
    "Status",
    "StatusDetails",
    "Task",
    "TaskConfig",
    "Text",
    "compiler",
    "graph",
    "ir",
    "ontology_id",
    "program",
    "text",
    "types",
]
