// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Local hdrs.
#include "synnax/telem/telem.h"
#include "telempb/telem.pb.h"

// std.
#include <string>
#include <vector>
#include <cstddef>
#include <typeinfo>

namespace synnax {

template<typename T = std::byte>
/// @brief Series type, able to hold generic types under the hood.
class Series {
public:
    // allow construction of series from int iterator
    explicit Series(std::vector<T> d) {
        // interpret the data as a byte array.
        data = reinterpret_cast<std::byte *>(d.data());
        // set the size of the data. in bytes.
        size = d.size() * sizeof(T);
    }

    DataType &getDataType() {
        return data_type;
    }

    void to_proto(telempb::Series *s) const {
        s->set_data_type(data_type.name());
    }

private:
    /// @brief Holds what type of data is being used.
    DataType data_type;

    /// @brief Holds the data.
    /// use a c character array to hold the data.
    std::byte *data;
    size_t size;
}