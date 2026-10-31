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
/// @brief converts an OPC UA data type to a Synnax telemetry data type.
x::telem::DataType ua_to_data_type(const UA_DataType *dt);

/// @brief converts a Synnax telemetry data type to an OPC UA data type.
UA_DataType *data_type_to_ua(const x::telem::DataType &data_type);

/// @brief converts a scalar variant to a sample of the data type.
/// @returns x::errors::VALIDATION if the variant is empty or an array, or holds a type
/// or value that the data type cannot hold.
std::pair<x::telem::SampleValue, x::errors::Error>
to_sample(const UA_Variant &v, const x::telem::DataType &dt);

/// @brief builds a series of the data type from samples of that data type.
x::telem::Series to_series(
    const x::telem::DataType &dt,
    const std::vector<x::telem::SampleValue> &samples
);

/// @brief converts a telemetry series to a UA_Variant.
std::pair<UA_Variant, x::errors::Error> series_to_variant(const x::telem::Series &s);

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
}
