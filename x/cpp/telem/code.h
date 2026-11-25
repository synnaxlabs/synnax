// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include "x/cpp/telem/telem.h"
#include "x/cpp/telem/types.gen.h"

namespace x::telem {
namespace details {
/// @brief returns the data type of each type code, indexed by code.
inline const std::array<DataType, 17> &data_types_by_code() {
    static const std::array<DataType, 17> types = {
        ::x::telem::UNKNOWN_T,
        ::x::telem::UINT8_T,
        ::x::telem::UINT16_T,
        ::x::telem::UINT32_T,
        ::x::telem::UINT64_T,
        ::x::telem::INT8_T,
        ::x::telem::INT16_T,
        ::x::telem::INT32_T,
        ::x::telem::INT64_T,
        ::x::telem::FLOAT32_T,
        ::x::telem::FLOAT64_T,
        ::x::telem::TIMESTAMP_T,
        ::x::telem::UUID_T,
        ::x::telem::STRING_T,
        ::x::telem::BYTES_T,
        ::x::telem::JSON_T,
        ::x::telem::BOOLEAN_T,
    };
    return types;
}
}

/// @brief returns the type code of dt, or DataTypeCode::Unknown when it has none.
inline DataTypeCode code_of(const DataType &dt) {
    const auto &types = details::data_types_by_code();
    for (std::size_t i = 1; i < types.size(); i++)
        if (types[i] == dt) return static_cast<DataTypeCode>(i);
    return DataTypeCode::Unknown;
}

/// @brief returns the data type of code, or UNKNOWN_T for a code with no data type.
inline DataType data_type_of(const std::uint8_t code) {
    const auto &types = details::data_types_by_code();
    if (code >= types.size()) return UNKNOWN_T;
    return types[code];
}
}
