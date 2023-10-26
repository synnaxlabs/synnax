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

    explicit Series(std::vector<std::uint8_t> d) {
        data = std::make_unique<std::byte[]>(d.size());
        memcpy(data.get(), d.data(), d.size());
        size = d.size();
        data_type = synnax::UINT8;
    }

    explicit Series(std::vector<std::float_t> d) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(std::float_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(std::float_t));
        size = d.size() * sizeof(std::float_t);
        data_type = synnax::FLOAT32;
    }

    explicit Series(std::vector<std::int64_t> d) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(std::int64_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(std::int64_t));
        size = d.size() * sizeof(std::int64_t);
        data_type = synnax::INT64;
    }

    explicit Series(telempb::Series s) {
        data_type = synnax::DataType(s.data_type());
        size = s.data().size();
        data = std::make_unique<std::byte[]>(size);
        memcpy(data.get(), s.data().data(), size);
    }

    DataType &getDataType() {
        return data_type;
    }

    void to_proto(telempb::Series *s) const {
        s->set_data_type(data_type.name());
        s->set_data(data.get(), size);
    }

    std::vector<std::uint8_t> uint8()  {
        if (data_type != synnax::UINT8) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<std::uint8_t> v(size);
        memcpy(v.data(), data.get(), size);
        return v;
    }

    std::vector<std::float_t> float32() {
        if (data_type != synnax::FLOAT32) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<std::float_t> v(size / sizeof(std::float_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    std::vector<std::int64_t> int64() {
        if (data_type != synnax::INT64) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<std::int64_t> v(size / sizeof(std::int64_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    /// @brief Holds the data.
    /// use a c character array to hold the data.
    std::unique_ptr<std::byte[]> data;

private:
    /// @brief Holds what type of data is being used.
    DataType data_type;

    size_t size;
};
}