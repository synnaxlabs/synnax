#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

__version__ = "0.3.0"

from .channel import Channel
from .exceptions import (
    AuthError,
    ContiguityError,
    Field,
    GeneralError,
    ParseError,
    QueryError,
    RouteError,
    UnexpectedError,
    ValidationError,
)
from freighter.exceptions import Unreachable
from .options import SynnaxOptions
from .synnax import Synnax
from .telem import (
    DataType,
    Density,
    Rate,
    Size,
    TimeRange,
    TimeSpan,
    TimeStamp,
    UnparsedDataType,
    UnparsedDensity,
    UnparsedRate,
    UnparsedTimeSpan,
    UnparsedTimeStamp,
    NumpyArray,
    BinaryArray,
)
from .framer import (
    BinaryFrame,
    NumpyFrame,
    BufferedDataFrameWriter,
    DataFrameWriter,
    FrameWriter,
)
