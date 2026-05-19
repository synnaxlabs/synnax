#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from importlib.metadata import version as _version

from synnax import ethercat, http, labjack, modbus, ni, opcua, pagerduty, status
from synnax.access.policy import Policy
from synnax.access.role import Role
from synnax.arc import (
    Arc,
    Edge,
    Graph,
    GraphNode,
    Handle,
    Position,
    Text,
)
from synnax.arc import Task as ArcTask
from synnax.channel import Channel
from synnax.control import Controller
from synnax.device import Device
from synnax.exceptions import (
    AuthError,
    ConfigurationError,
    ControlError,
    ExpiredToken,
    InvalidToken,
    MultipleFoundError,
    NotFoundError,
    PathError,
    QueryError,
    RouteError,
    UnauthorizedError,
    UnexpectedError,
    ValidationError,
)
from synnax.framer import (
    AUTO_SPAN,
    Frame,
    Iterator,
    Streamer,
    Writer,
    WriterMode,
)
from synnax.options import Options
from synnax.rack import Rack
from synnax.ranger import Range
from synnax.status import Status
from synnax.synnax import Synnax
from synnax.task import Status as TaskStatus
from synnax.task import StatusDetails as TaskStatusDetails
from synnax.task import Task
from synnax.telem import (
    Alignment,
    Authority,
    Bounds,
    CrudeAlignment,
    CrudeAuthority,
    CrudeDataType,
    CrudeDensity,
    CrudeRate,
    CrudeTimeSpan,
    CrudeTimeStamp,
    DataType,
    Density,
    MultiSeries,
    Rate,
    Series,
    Size,
    TimeRange,
    TimeSpan,
    TimeSpanUnits,
    TimeStamp,
    convert_time_units,
    elapsed_seconds,
)
from synnax.timing import Loop, Timer, poll, sleep
from synnax.user.payload import User
from synnax.view import View
from x import color
from x.color import Color
from x.deprecation import deprecated_getattr
from x.exceptions import ContiguityError

__version__ = _version("synnax")

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "ArcTask": ("synnax.arc.Task", "ArcTask"),
    "TaskStatus": ("synnax.task.Status", "TaskStatus"),
    "TaskStatusDetails": ("synnax.task.StatusDetails", "TaskStatusDetails"),
    "SynnaxOptions": "Options",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "ArcTask",
    "TaskStatus",
    "TaskStatusDetails",
    "Alignment",
    "Arc",
    "AUTO_SPAN",
    "AuthError",
    "Authority",
    "Controller",
    "Bounds",
    "Channel",
    "Color",
    "Edge",
    "ConfigurationError",
    "ContiguityError",
    "ControlError",
    "convert_time_units",
    "CrudeAlignment",
    "CrudeAuthority",
    "CrudeDataType",
    "CrudeDensity",
    "CrudeRate",
    "CrudeTimeSpan",
    "CrudeTimeStamp",
    "DataType",
    "Density",
    "Device",
    "View",
    "elapsed_seconds",
    "ExpiredToken",
    "PathError",
    "Frame",
    "Graph",
    "GraphNode",
    "Handle",
    "Iterator",
    "InvalidToken",
    "Loop",
    "MultipleFoundError",
    "NotFoundError",
    "Options",
    "poll",
    "Policy",
    "Position",
    "QueryError",
    "Rack",
    "Range",
    "Rate",
    "Role",
    "RouteError",
    "Series",
    "MultiSeries",
    "Size",
    "sleep",
    "Status",
    "Streamer",
    "Synnax",
    "Task",
    "Text",
    "Timer",
    "TimeRange",
    "TimeSpan",
    "TimeSpanUnits",
    "TimeStamp",
    "UnauthorizedError",
    "UnexpectedError",
    "User",
    "ValidationError",
    "Writer",
    "WriterMode",
    "color",
    "ethercat",
    "http",
    "labjack",
    "modbus",
    "ni",
    "opcua",
    "pagerduty",
    "ontology",
    "status",
    "Status",
    "group",
]
