// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "open62541/types.h"

/// module
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/errors/errors.h"

namespace driver::opc::telem {
/// @brief converts an OPC UA data type to a Synnax telemetry data type.
::x::telem::DataType ua_to_data_type(const UA_DataType *dt);

/// @brief converts a Synnax telemetry data type to an OPC UA data type.
UA_DataType *data_type_to_ua(const ::x::telem::DataType &data_type);

/// @brief writes data from a UA_Variant to a telemetry series.
/// @return a pair containing the number of samples written and any error.
std::pair<size_t, x::errors::Error>
write_to_series(::x::telem::Series &s, const UA_Variant &v);

/// @brief converts a telemetry series to a UA_Variant.
std::pair<UA_Variant, x::errors::Error> series_to_variant(const ::x::telem::Series &s);

/// @brief writes data from a UA_Variant array to a telemetry series.
/// @param series the series to write to.
/// @param val the variant containing the array data.
/// @param target_size the expected size of the array.
/// @param name optional name for error messages.
/// @return a pair containing the number of samples written and any error.
std::pair<size_t, x::errors::Error> ua_array_write_to_series(
    ::x::telem::Series &series,
    const UA_Variant *val,
    size_t target_size,
    const std::string &name = ""
);
}
