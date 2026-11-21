#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.telem.telem import DataType
from x.telem.types_gen import DataTypeCode

_DATA_TYPES: dict[DataTypeCode, DataType] = {
    DataTypeCode.uint8: DataType.UINT8,
    DataTypeCode.uint16: DataType.UINT16,
    DataTypeCode.uint32: DataType.UINT32,
    DataTypeCode.uint64: DataType.UINT64,
    DataTypeCode.int8: DataType.INT8,
    DataTypeCode.int16: DataType.INT16,
    DataTypeCode.int32: DataType.INT32,
    DataTypeCode.int64: DataType.INT64,
    DataTypeCode.float32: DataType.FLOAT32,
    DataTypeCode.float64: DataType.FLOAT64,
    DataTypeCode.timestamp: DataType.TIMESTAMP,
    DataTypeCode.uuid: DataType.UUID,
    DataTypeCode.string: DataType.STRING,
    DataTypeCode.bytes: DataType.BYTES,
    DataTypeCode.json: DataType.JSON,
    DataTypeCode.boolean: DataType.BOOLEAN,
}

_CODES: dict[DataType, DataTypeCode] = {dt: c for c, dt in _DATA_TYPES.items()}


def code_of(data_type: DataType) -> DataTypeCode:
    """:returns: the type code of data_type, or the unknown code when it has none."""
    return _CODES.get(data_type, DataTypeCode.unknown)


def data_type_of(code: int) -> DataType:
    """:returns: the data type of code, or DataType.UNKNOWN for a code with none."""
    for c, dt in _DATA_TYPES.items():
        if c == code:
            return dt
    return DataType.UNKNOWN
