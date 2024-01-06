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

    explicit Series(const std::vector<uint8_t> &d) {
        data = std::make_unique<std::byte[]>(d.size());
        memcpy(data.get(), d.data(), d.size());
        size = d.size();
        data_type = synnax::UINT8;
    }

    explicit Series(const std::vector<float> &d) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(float));
        memcpy(data.get(), d.data(), d.size() * sizeof(float));
        size = d.size() * sizeof(float);
        data_type = synnax::FLOAT32;
    }

    explicit Series(const std::vector<int64_t> &d) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(int64_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(int64_t));
        size = d.size() * sizeof(int64_t);
        data_type = synnax::INT64;
    }

    explicit Series(const std::vector<uint64_t> &d) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(uint64_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(uint64_t));
        size = d.size() * sizeof(uint64_t);
        data_type = synnax::UINT64;
    }

    explicit Series(const std::vector<std::string> &d, synnax::DataType data_type = synnax::STRING): data_type(data_type) {
        size_t total_size = 0;
        for (const auto &s : d) total_size += s.size() + 1;
        data = std::make_unique<std::byte[]>(total_size);
        size_t offset = 0;
        for (const auto &s : d) {
            memcpy(data.get() + offset, s.data(), s.size());
            offset += s.size();
            data[offset] = std::byte('\n');
            offset++;
        }
        size = total_size;
    }

    explicit Series(const telempb::Series &s) {
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

    [[nodiscard]] std::vector<uint8_t> uint8() const {
        if (data_type != synnax::UINT8) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<uint8_t> v(size);
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<float> float32() const {
        if (data_type != synnax::FLOAT32) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<float> v(size / sizeof(float));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<int64_t> int64() const {
        if (data_type != synnax::INT64) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<int64_t> v(size / sizeof(int64_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<uint64_t> uint64() const {
        if (data_type != synnax::UINT64) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<uint64_t> v(size / sizeof(uint64_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<std::string> string() const {
        if (data_type != synnax::STRING) {
            throw std::runtime_error("invalid data type");
        }
        std::vector<std::string> v;
        std::string s;
        for (size_t i = 0; i < size; i++) {
            if (data[i] == std::byte('\n')) {
                v.push_back(s);
                s.clear();
                // WARNING: This might be very slow due to copying.
            } else s += char(data[i]);
        }
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