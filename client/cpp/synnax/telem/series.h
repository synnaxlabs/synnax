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

/// @brief Series type, able to hold generic types under the hood.
class Series {
public:
    Series(std::vector<std::any> vals) {
        data_type.setDataType(vals[0].type().name());
        data_type = DataType(vals[0].type().name());
        data = vals;
    }

    Series(const telempb::Series &s) {
        data_type = DataType(s.data_type());
        for (auto &val: s.data()) {
            data.push_back(val);
        }
    }

    std::vector<std::any> &getRaw() {
        return data;
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
    std::vector<std::any> data;
};
}