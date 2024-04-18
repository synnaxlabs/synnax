// Copyright 2024 Synnax Labs, Inc.
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

/// @brief Series is a strongly typed array of telemetry samples backed by an underlying binary buffer.
class Series {
public:
    static const DataType validateDataType(DataType expected, DataType value, bool validate = true) {
        if (validate && expected != value)
            throw std::runtime_error("invalid data type. Expected " + expected.name() + ", got " + value.name());
        return value;
    }


    /// @brief Constructs the series from a vector of uint8's,
    explicit Series(const std::vector<uint8_t> &d, bool validate_data_type = true) : data_type(synnax::UINT8) {
        data = std::make_unique<std::byte[]>(d.size());
        memcpy(data.get(), d.data(), d.size());
        size = d.size();
    }

    explicit Series(const std::vector<float> &d) : data_type(synnax::FLOAT32) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(float));
        memcpy(data.get(), d.data(), d.size() * sizeof(float));
        size = d.size() * sizeof(float);
    }

    explicit Series(const std::vector<int64_t> &d) : data_type(synnax::INT64) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(int64_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(int64_t));
        size = d.size() * sizeof(int64_t);
    }

    explicit Series(const std::vector<uint64_t> &d) : data_type(synnax::UINT64) {
        data = std::make_unique<std::byte[]>(d.size() * sizeof(uint64_t));
        memcpy(data.get(), d.data(), d.size() * sizeof(uint64_t));
        size = d.size() * sizeof(uint64_t);
    }

    explicit Series(
            const std::vector<std::string> &d,
            synnax::DataType data_type = synnax::STRING
    ) : data_type(data_type) {
        if (data_type != synnax::STRING && data_type != synnax::JSON)
            throw std::runtime_error("invalid data type");
        size_t total_size = 0;
        for (const auto &s: d) total_size += s.size() + 1;
        data = std::make_unique<std::byte[]>(total_size);
        size_t offset = 0;
        for (const auto &s: d) {
            memcpy(data.get() + offset, s.data(), s.size());
            offset += s.size();
            data[offset] = std::byte('\n');
            offset++;
        }
        size = total_size;
    }

    explicit Series(const telempb::Series &s) : data_type(s.data_type()) {
        size = s.data().size();
        data = std::make_unique<std::byte[]>(size);
        memcpy(data.get(), s.data().data(), size);
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

    /// @brief Holds the underlying data.
    std::unique_ptr<std::byte[]> data;

    /// @brief an optional property that defines the time range occupied by the Series' data. This property is
    /// guaranteed to be defined when reading data from a Synnax cluster, and is particularly useful for understanding
    /// the alignment of samples in relation to another series. When read from a cluster, the start of the time range
    /// represents the timestamp of the first sample in the array (inclusive), while the end of the time
    /// range is set to the nanosecond AFTER the last sample in the array (exclusive).
    synnax::TimeRange time_range = synnax::TimeRange();

    /// @brief Holds what type of data is being used.
    const DataType data_type;
private:
    size_t size;
};
}