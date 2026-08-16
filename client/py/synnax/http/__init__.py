#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.http.types import (
    MAKE,
    MODEL,
    Device,
    ExpectedResponse,
    HealthCheck,
    ReadTask,
    WriteTask,
)
from synnax.http.types_gen import (
    BaseWriteField,
    ChannelField,
    EnumEntry,
    GeneratedWriteField,
    GeneratorType,
    Header,
    JSONType,
    Method,
    QueryParam,
    ReadConfig,
    ReadEndpoint,
    ReadField,
    ScanConfig,
    StaticWriteField,
    TimeFormat,
    WriteConfig,
    WriteEndpoint,
    WriteField,
)
from x.deprecation import deprecated_getattr

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "GeneratedField": "GeneratedWriteField",
    "StaticField": "StaticWriteField",
    "HeaderEntry": "Header",
    "QueryParamEntry": "QueryParam",
    "ReadEnumEntry": "EnumEntry",
    "WriteEnumEntry": "EnumEntry",
    "ReadTaskConfig": "ReadConfig",
    "WriteTaskConfig": "WriteConfig",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "BaseWriteField",
    "ChannelField",
    "Device",
    "EnumEntry",
    "ExpectedResponse",
    "GeneratedWriteField",
    "GeneratorType",
    "Header",
    "HealthCheck",
    "JSONType",
    "MAKE",
    "MODEL",
    "Method",
    "QueryParam",
    "ReadConfig",
    "ReadEndpoint",
    "ReadField",
    "ReadTask",
    "ScanConfig",
    "StaticWriteField",
    "TimeFormat",
    "WriteConfig",
    "WriteEndpoint",
    "WriteField",
    "WriteTask",
]
