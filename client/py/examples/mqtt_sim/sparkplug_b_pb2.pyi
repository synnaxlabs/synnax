#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# Copyright (c) 2015, 2018 Cirrus Link Solutions and others
#
# This program and the accompanying materials are made available under the terms of
# the Eclipse Public License 2.0 which is available at
# http://www.eclipse.org/legal/epl-2.0.
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Cirrus Link Solutions - initial implementation

from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf.internal import python_message as _python_message
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DataType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    Unknown: _ClassVar[DataType]
    Int8: _ClassVar[DataType]
    Int16: _ClassVar[DataType]
    Int32: _ClassVar[DataType]
    Int64: _ClassVar[DataType]
    UInt8: _ClassVar[DataType]
    UInt16: _ClassVar[DataType]
    UInt32: _ClassVar[DataType]
    UInt64: _ClassVar[DataType]
    Float: _ClassVar[DataType]
    Double: _ClassVar[DataType]
    Boolean: _ClassVar[DataType]
    String: _ClassVar[DataType]
    DateTime: _ClassVar[DataType]
    Text: _ClassVar[DataType]
    UUID: _ClassVar[DataType]
    DataSet: _ClassVar[DataType]
    Bytes: _ClassVar[DataType]
    File: _ClassVar[DataType]
    Template: _ClassVar[DataType]
    PropertySet: _ClassVar[DataType]
    PropertySetList: _ClassVar[DataType]
    Int8Array: _ClassVar[DataType]
    Int16Array: _ClassVar[DataType]
    Int32Array: _ClassVar[DataType]
    Int64Array: _ClassVar[DataType]
    UInt8Array: _ClassVar[DataType]
    UInt16Array: _ClassVar[DataType]
    UInt32Array: _ClassVar[DataType]
    UInt64Array: _ClassVar[DataType]
    FloatArray: _ClassVar[DataType]
    DoubleArray: _ClassVar[DataType]
    BooleanArray: _ClassVar[DataType]
    StringArray: _ClassVar[DataType]
    DateTimeArray: _ClassVar[DataType]
Unknown: DataType
Int8: DataType
Int16: DataType
Int32: DataType
Int64: DataType
UInt8: DataType
UInt16: DataType
UInt32: DataType
UInt64: DataType
Float: DataType
Double: DataType
Boolean: DataType
String: DataType
DateTime: DataType
Text: DataType
UUID: DataType
DataSet: DataType
Bytes: DataType
File: DataType
Template: DataType
PropertySet: DataType
PropertySetList: DataType
Int8Array: DataType
Int16Array: DataType
Int32Array: DataType
Int64Array: DataType
UInt8Array: DataType
UInt16Array: DataType
UInt32Array: DataType
UInt64Array: DataType
FloatArray: DataType
DoubleArray: DataType
BooleanArray: DataType
StringArray: DataType
DateTimeArray: DataType

class Payload(_message.Message):
    __slots__ = ("timestamp", "metrics", "seq", "uuid", "body")
    Extensions: _python_message._ExtensionDict
    class Template(_message.Message):
        __slots__ = ("version", "metrics", "parameters", "template_ref", "is_definition")
        Extensions: _python_message._ExtensionDict
        class Parameter(_message.Message):
            __slots__ = ("name", "type", "int_value", "long_value", "float_value", "double_value", "boolean_value", "string_value", "extension_value")
            class ParameterValueExtension(_message.Message):
                __slots__ = ()
                Extensions: _python_message._ExtensionDict
                def __init__(self) -> None: ...
            NAME_FIELD_NUMBER: _ClassVar[int]
            TYPE_FIELD_NUMBER: _ClassVar[int]
            INT_VALUE_FIELD_NUMBER: _ClassVar[int]
            LONG_VALUE_FIELD_NUMBER: _ClassVar[int]
            FLOAT_VALUE_FIELD_NUMBER: _ClassVar[int]
            DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
            BOOLEAN_VALUE_FIELD_NUMBER: _ClassVar[int]
            STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
            EXTENSION_VALUE_FIELD_NUMBER: _ClassVar[int]
            name: str
            type: int
            int_value: int
            long_value: int
            float_value: float
            double_value: float
            boolean_value: bool
            string_value: str
            extension_value: Payload.Template.Parameter.ParameterValueExtension
            def __init__(self, name: _Optional[str] = ..., type: _Optional[int] = ..., int_value: _Optional[int] = ..., long_value: _Optional[int] = ..., float_value: _Optional[float] = ..., double_value: _Optional[float] = ..., boolean_value: _Optional[bool] = ..., string_value: _Optional[str] = ..., extension_value: _Optional[_Union[Payload.Template.Parameter.ParameterValueExtension, _Mapping]] = ...) -> None: ...
        VERSION_FIELD_NUMBER: _ClassVar[int]
        METRICS_FIELD_NUMBER: _ClassVar[int]
        PARAMETERS_FIELD_NUMBER: _ClassVar[int]
        TEMPLATE_REF_FIELD_NUMBER: _ClassVar[int]
        IS_DEFINITION_FIELD_NUMBER: _ClassVar[int]
        version: str
        metrics: _containers.RepeatedCompositeFieldContainer[Payload.Metric]
        parameters: _containers.RepeatedCompositeFieldContainer[Payload.Template.Parameter]
        template_ref: str
        is_definition: bool
        def __init__(self, version: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[Payload.Metric, _Mapping]]] = ..., parameters: _Optional[_Iterable[_Union[Payload.Template.Parameter, _Mapping]]] = ..., template_ref: _Optional[str] = ..., is_definition: _Optional[bool] = ...) -> None: ...
    class DataSet(_message.Message):
        __slots__ = ("num_of_columns", "columns", "types", "rows")
        Extensions: _python_message._ExtensionDict
        class DataSetValue(_message.Message):
            __slots__ = ("int_value", "long_value", "float_value", "double_value", "boolean_value", "string_value", "extension_value")
            class DataSetValueExtension(_message.Message):
                __slots__ = ()
                Extensions: _python_message._ExtensionDict
                def __init__(self) -> None: ...
            INT_VALUE_FIELD_NUMBER: _ClassVar[int]
            LONG_VALUE_FIELD_NUMBER: _ClassVar[int]
            FLOAT_VALUE_FIELD_NUMBER: _ClassVar[int]
            DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
            BOOLEAN_VALUE_FIELD_NUMBER: _ClassVar[int]
            STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
            EXTENSION_VALUE_FIELD_NUMBER: _ClassVar[int]
            int_value: int
            long_value: int
            float_value: float
            double_value: float
            boolean_value: bool
            string_value: str
            extension_value: Payload.DataSet.DataSetValue.DataSetValueExtension
            def __init__(self, int_value: _Optional[int] = ..., long_value: _Optional[int] = ..., float_value: _Optional[float] = ..., double_value: _Optional[float] = ..., boolean_value: _Optional[bool] = ..., string_value: _Optional[str] = ..., extension_value: _Optional[_Union[Payload.DataSet.DataSetValue.DataSetValueExtension, _Mapping]] = ...) -> None: ...
        class Row(_message.Message):
            __slots__ = ("elements",)
            Extensions: _python_message._ExtensionDict
            ELEMENTS_FIELD_NUMBER: _ClassVar[int]
            elements: _containers.RepeatedCompositeFieldContainer[Payload.DataSet.DataSetValue]
            def __init__(self, elements: _Optional[_Iterable[_Union[Payload.DataSet.DataSetValue, _Mapping]]] = ...) -> None: ...
        NUM_OF_COLUMNS_FIELD_NUMBER: _ClassVar[int]
        COLUMNS_FIELD_NUMBER: _ClassVar[int]
        TYPES_FIELD_NUMBER: _ClassVar[int]
        ROWS_FIELD_NUMBER: _ClassVar[int]
        num_of_columns: int
        columns: _containers.RepeatedScalarFieldContainer[str]
        types: _containers.RepeatedScalarFieldContainer[int]
        rows: _containers.RepeatedCompositeFieldContainer[Payload.DataSet.Row]
        def __init__(self, num_of_columns: _Optional[int] = ..., columns: _Optional[_Iterable[str]] = ..., types: _Optional[_Iterable[int]] = ..., rows: _Optional[_Iterable[_Union[Payload.DataSet.Row, _Mapping]]] = ...) -> None: ...
    class PropertyValue(_message.Message):
        __slots__ = ("type", "is_null", "int_value", "long_value", "float_value", "double_value", "boolean_value", "string_value", "propertyset_value", "propertysets_value", "extension_value")
        class PropertyValueExtension(_message.Message):
            __slots__ = ()
            Extensions: _python_message._ExtensionDict
            def __init__(self) -> None: ...
        TYPE_FIELD_NUMBER: _ClassVar[int]
        IS_NULL_FIELD_NUMBER: _ClassVar[int]
        INT_VALUE_FIELD_NUMBER: _ClassVar[int]
        LONG_VALUE_FIELD_NUMBER: _ClassVar[int]
        FLOAT_VALUE_FIELD_NUMBER: _ClassVar[int]
        DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
        BOOLEAN_VALUE_FIELD_NUMBER: _ClassVar[int]
        STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
        PROPERTYSET_VALUE_FIELD_NUMBER: _ClassVar[int]
        PROPERTYSETS_VALUE_FIELD_NUMBER: _ClassVar[int]
        EXTENSION_VALUE_FIELD_NUMBER: _ClassVar[int]
        type: int
        is_null: bool
        int_value: int
        long_value: int
        float_value: float
        double_value: float
        boolean_value: bool
        string_value: str
        propertyset_value: Payload.PropertySet
        propertysets_value: Payload.PropertySetList
        extension_value: Payload.PropertyValue.PropertyValueExtension
        def __init__(self, type: _Optional[int] = ..., is_null: _Optional[bool] = ..., int_value: _Optional[int] = ..., long_value: _Optional[int] = ..., float_value: _Optional[float] = ..., double_value: _Optional[float] = ..., boolean_value: _Optional[bool] = ..., string_value: _Optional[str] = ..., propertyset_value: _Optional[_Union[Payload.PropertySet, _Mapping]] = ..., propertysets_value: _Optional[_Union[Payload.PropertySetList, _Mapping]] = ..., extension_value: _Optional[_Union[Payload.PropertyValue.PropertyValueExtension, _Mapping]] = ...) -> None: ...
    class PropertySet(_message.Message):
        __slots__ = ("keys", "values")
        Extensions: _python_message._ExtensionDict
        KEYS_FIELD_NUMBER: _ClassVar[int]
        VALUES_FIELD_NUMBER: _ClassVar[int]
        keys: _containers.RepeatedScalarFieldContainer[str]
        values: _containers.RepeatedCompositeFieldContainer[Payload.PropertyValue]
        def __init__(self, keys: _Optional[_Iterable[str]] = ..., values: _Optional[_Iterable[_Union[Payload.PropertyValue, _Mapping]]] = ...) -> None: ...
    class PropertySetList(_message.Message):
        __slots__ = ("propertyset",)
        Extensions: _python_message._ExtensionDict
        PROPERTYSET_FIELD_NUMBER: _ClassVar[int]
        propertyset: _containers.RepeatedCompositeFieldContainer[Payload.PropertySet]
        def __init__(self, propertyset: _Optional[_Iterable[_Union[Payload.PropertySet, _Mapping]]] = ...) -> None: ...
    class MetaData(_message.Message):
        __slots__ = ("is_multi_part", "content_type", "size", "seq", "file_name", "file_type", "md5", "description")
        Extensions: _python_message._ExtensionDict
        IS_MULTI_PART_FIELD_NUMBER: _ClassVar[int]
        CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
        SIZE_FIELD_NUMBER: _ClassVar[int]
        SEQ_FIELD_NUMBER: _ClassVar[int]
        FILE_NAME_FIELD_NUMBER: _ClassVar[int]
        FILE_TYPE_FIELD_NUMBER: _ClassVar[int]
        MD5_FIELD_NUMBER: _ClassVar[int]
        DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
        is_multi_part: bool
        content_type: str
        size: int
        seq: int
        file_name: str
        file_type: str
        md5: str
        description: str
        def __init__(self, is_multi_part: _Optional[bool] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ..., seq: _Optional[int] = ..., file_name: _Optional[str] = ..., file_type: _Optional[str] = ..., md5: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...
    class Metric(_message.Message):
        __slots__ = ("name", "alias", "timestamp", "datatype", "is_historical", "is_transient", "is_null", "metadata", "properties", "int_value", "long_value", "float_value", "double_value", "boolean_value", "string_value", "bytes_value", "dataset_value", "template_value", "extension_value")
        class MetricValueExtension(_message.Message):
            __slots__ = ()
            Extensions: _python_message._ExtensionDict
            def __init__(self) -> None: ...
        NAME_FIELD_NUMBER: _ClassVar[int]
        ALIAS_FIELD_NUMBER: _ClassVar[int]
        TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
        DATATYPE_FIELD_NUMBER: _ClassVar[int]
        IS_HISTORICAL_FIELD_NUMBER: _ClassVar[int]
        IS_TRANSIENT_FIELD_NUMBER: _ClassVar[int]
        IS_NULL_FIELD_NUMBER: _ClassVar[int]
        METADATA_FIELD_NUMBER: _ClassVar[int]
        PROPERTIES_FIELD_NUMBER: _ClassVar[int]
        INT_VALUE_FIELD_NUMBER: _ClassVar[int]
        LONG_VALUE_FIELD_NUMBER: _ClassVar[int]
        FLOAT_VALUE_FIELD_NUMBER: _ClassVar[int]
        DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
        BOOLEAN_VALUE_FIELD_NUMBER: _ClassVar[int]
        STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
        BYTES_VALUE_FIELD_NUMBER: _ClassVar[int]
        DATASET_VALUE_FIELD_NUMBER: _ClassVar[int]
        TEMPLATE_VALUE_FIELD_NUMBER: _ClassVar[int]
        EXTENSION_VALUE_FIELD_NUMBER: _ClassVar[int]
        name: str
        alias: int
        timestamp: int
        datatype: int
        is_historical: bool
        is_transient: bool
        is_null: bool
        metadata: Payload.MetaData
        properties: Payload.PropertySet
        int_value: int
        long_value: int
        float_value: float
        double_value: float
        boolean_value: bool
        string_value: str
        bytes_value: bytes
        dataset_value: Payload.DataSet
        template_value: Payload.Template
        extension_value: Payload.Metric.MetricValueExtension
        def __init__(self, name: _Optional[str] = ..., alias: _Optional[int] = ..., timestamp: _Optional[int] = ..., datatype: _Optional[int] = ..., is_historical: _Optional[bool] = ..., is_transient: _Optional[bool] = ..., is_null: _Optional[bool] = ..., metadata: _Optional[_Union[Payload.MetaData, _Mapping]] = ..., properties: _Optional[_Union[Payload.PropertySet, _Mapping]] = ..., int_value: _Optional[int] = ..., long_value: _Optional[int] = ..., float_value: _Optional[float] = ..., double_value: _Optional[float] = ..., boolean_value: _Optional[bool] = ..., string_value: _Optional[str] = ..., bytes_value: _Optional[bytes] = ..., dataset_value: _Optional[_Union[Payload.DataSet, _Mapping]] = ..., template_value: _Optional[_Union[Payload.Template, _Mapping]] = ..., extension_value: _Optional[_Union[Payload.Metric.MetricValueExtension, _Mapping]] = ...) -> None: ...
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    SEQ_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    timestamp: int
    metrics: _containers.RepeatedCompositeFieldContainer[Payload.Metric]
    seq: int
    uuid: str
    body: bytes
    def __init__(self, timestamp: _Optional[int] = ..., metrics: _Optional[_Iterable[_Union[Payload.Metric, _Mapping]]] = ..., seq: _Optional[int] = ..., uuid: _Optional[str] = ..., body: _Optional[bytes] = ...) -> None: ...
