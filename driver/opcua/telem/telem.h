// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <vector>

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace driver::opcua::telem {
/// @brief converts an OPC UA data type to the Synnax data type that holds its values. A
/// subtype, such as Duration, converts as its builtin type, and an enumeration as
/// int32.
/// @returns x::telem::UNKNOWN_T for a type that no Synnax data type holds.
x::telem::DataType to_data_type(const UA_DataType *dt);

/// @brief converts the DataType attribute of a node to a Synnax data type.
/// @returns x::telem::UNKNOWN_T for an abstract type, a structure, or a type that the
/// client does not know.
x::telem::DataType to_data_type(const UA_NodeId &data_type);

/// @brief converts a Synnax telemetry data type to an OPC UA data type.
UA_DataType *data_type_to_ua(const x::telem::DataType &data_type);

/// @brief converts a scalar variant to a sample of the data type. A uuid sample holds
/// the 16 bytes of the UUID as a string, since a SampleValue has no UUID.
/// @returns x::errors::VALIDATION if the variant is empty or an array, or holds a type
/// or value that the data type cannot hold.
std::pair<x::telem::SampleValue, x::errors::Error>
to_sample(const UA_Variant &v, const x::telem::DataType &dt);

/// @brief builds a series of the data type from samples of that data type.
x::telem::Series to_series(
    const x::telem::DataType &dt,
    const std::vector<x::telem::SampleValue> &samples
);

/// @brief converts the last sample of a series to a variant of the builtin type of
/// target, the data type of the node that the variant goes to. A null target, for a
/// node of an unknown type, and a Variant target, for a node of BaseDataType, convert
/// to the OPC UA type that matches the series.
/// @returns x::errors::VALIDATION if the series is empty, or the sample is out of the
/// range of target, is not a whole number for an integer target, or is text that does
/// not parse as target.
std::pair<UA_Variant, x::errors::Error>
to_variant(const x::telem::Series &s, const UA_DataType *target);

/// @brief writes data from a UA_Variant array to a telemetry series.
/// @param series the series to write to.
/// @param val the variant containing the array data.
/// @param target_size the expected size of the array.
/// @param name optional name for error messages.
/// @return a pair containing the number of samples written and any error.
std::pair<size_t, x::errors::Error> ua_array_write_to_series(
    x::telem::Series &series,
    const UA_Variant *val,
    size_t target_size,
    const std::string &name = ""
);

/// @returns the time that t holds.
/// @returns x::errors::VALIDATION if t is outside the years 1677 to 2262 that a
/// timestamp holds.
std::pair<x::telem::TimeStamp, x::errors::Error> to_timestamp(UA_DateTime t);
}
