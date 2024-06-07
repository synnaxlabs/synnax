// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/telem/telem.h"
#include "x/go/telem/x/go/telem/telem.pb.h"
#include <string>
#include <utility>
#include <vector>
#include <cstddef>

constexpr auto NEWLINE_TERMINATOR = static_cast<std::byte>('\n');

namespace synnax {
/// @brief Series is a strongly typed array of telemetry samples backed by an underlying binary buffer.
class Series {
public:
    /// @brief allocates a series with the given data type and capacity (in samples).
    /// Allocated series are treated as buffers and are not initialized with any data.
    /// Calls to write can be used to populate the series.
    /// @param data_type the type of data being stored.
    /// @param cap the number of samples that can be stored in the series.
    Series(
        const DataType &data_type,
        const size_t cap
    ): size(0),
       cap(cap),
       data_type(data_type),
       data(std::make_unique<std::byte[]>(cap * data_type.density())) {
    }

    /// @brief constructs a series from the given vector of numeric data and an optional
    /// data type.
    /// @param d the vector of numeric data to be used.
    /// @param dt the type of data being used. In most cases, this should not
    /// be specified and the data type will be inferred from the numeric type. If
    /// you do choose to override the data type, it's up to you to ensure that the
    /// contents of the series are compatible with the data type.
    template<typename NumericType>
    explicit Series(
        const std::vector<NumericType> &d,
        DataType dt = DATA_TYPE_UNKNOWN
    ) : size(d.size()),
        cap(d.size()),
        data_type(std::move(dt)) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        if (data_type == DATA_TYPE_UNKNOWN) data_type = DataType::infer<NumericType>();
        data = std::make_unique<std::byte[]>(byteSize());
        memcpy(data.get(), d.data(), byteSize());
    }

    /// @brief constructs a series of size 1 with a data type of TIMESTAMP from the
    /// given timestamp.
    /// @param v the timestamp to be used.
    explicit Series(
        const TimeStamp v
    ) : size(1),
        cap(1),
        data_type(synnax::TIMESTAMP) {
        memcpy(data.get(), &v.value, this->byteSize());
    }

    /// @brief constructs a series of size 1 from the given number.
    /// @param v the number to be used.
    /// @param dt an optional data type to use. If not specified, the data type
    /// will be inferred from the numeric type. If you do choose to override the
    /// data type, it's up to you to ensure that the contents of the series are
    /// compatible with the data type.
    template<typename NumericType>
    explicit Series(
        NumericType v,
        DataType dt = DATA_TYPE_UNKNOWN
    ): size(1),
       cap(1),
       data_type(std::move(dt)) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        if (data_type == DATA_TYPE_UNKNOWN) data_type = DataType::infer<NumericType>();
        data = std::make_unique<std::byte[]>(this->byteSize());
        memcpy(data.get(), &v, this->byteSize());
    }


    /// @brief sets a number at an index.
    /// @param index the index to set the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the value to set. The provided value should be compatible with
    /// the series' data type. It is up to you to ensure that this is the case.
    template<typename NumericType>
    void set(const int index, const NumericType value) {
        const auto adjusted = this->validateBounds(index);
        memcpy(
            data.get() + adjusted * data_type.density(),
            &value,
            data_type.density()
        );
    }

    /// @brief sets the given array of numeric data at the given index.
    /// @param d the array of numeric data to be written.
    /// @param index the index to write the data at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param size_ the number of samples to write.
    /// @throws std::runtime_error if the index is out of bounds or the write would
    /// exceed the capacity of the series.
    template<typename NumericType>
    void set_array(const NumericType *d, const int index, const size_t size_) {
        const auto adjusted = this->validateBounds(index, size_);
        memcpy(
            data.get() + adjusted * data_type.density(),
            d,
            size_ * data_type.density()
        );
    }

    /// @brief sets the given vector of numeric data at the given index.
    /// @param d the vector of numeric data to be written.
    /// @param index the index to write the data at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @throws std::runtime_error if the index is out of bounds or the write would
    template<typename NumericType>
    void set(const std::vector<NumericType> &d, const int index) {
        const auto adjusted = this->validateBounds(index, d.size());
        memcpy(
            data.get() + adjusted * data_type.density(),
            d.data(),
            d.size() * data_type.density()
        );
    }

    /// @brief writes the given vector of numeric data to the series.
    /// @param d the vector of numeric data to be written.
    /// @returns the number of samples written. If the capacity of the series is exceeded,
    /// it will only write as many samples as it can hold.
    template<typename NumericType>
    size_t write(const std::vector<NumericType> &d) {
        const size_t count = std::min(d.size(), cap - size);
        memcpy(data.get(), d.data(), count * data_type.density());
        size += count;
        return count;
    }

    /// @brief writes the given array of numeric data to the series.
    /// @param d the array of numeric data to be written.
    /// @param size_ the number of samples to write.
    /// @returns the number of samples written. If the capacity of the series is exceeded,
    /// it will only write as many samples as it can hold.
    template<typename NumericType>
    size_t write(const NumericType *d, const size_t size_) {
        const size_t count = std::min(size_, cap - size);
        memcpy(data.get(), d, count * data_type.density());
        size += count;
        return count;
    }

    /// @brief constructs the series from the given vector of strings. These can also
    /// be JSON encoded strings, in which case the data type should be set to JSON.
    /// @param d the vector of strings to be used as the data.
    /// @param data_type_ the type of data being used.
    explicit Series(
        const std::vector<std::string> &d,
        DataType data_type_ = STRING
    ) : data_type(std::move(data_type_)) {
        if (!data_type.is_variable())
            throw std::runtime_error("expected data type to be STRING or JSON");
        cached_byte_size = 0;
        for (const auto &s: d) cached_byte_size += s.size() + 1;
        data = std::make_unique<std::byte[]>(byteSize());
        size_t offset = 0;
        for (const auto &s: d) {
            memcpy(data.get() + offset, s.data(), s.size());
            offset += s.size();
            data[offset] = static_cast<std::byte>('\n');
            offset++;
        }
        size = total_size;
        cap = size;
    }

    explicit Series(
        const std::string &data,
        DataType data_type_ = STRING
    ): data_type(std::move(data_type_)) {
        if (data_type != STRING && data_type != JSON)
            throw std::runtime_error("invalid data type c");
        size = data.size() + 1;
        cap = size;
        this->data = std::make_unique<std::byte[]>(size);
        memcpy(this->data.get(), data.data(), data.size());
        this->data[size - 1] = static_cast<std::byte>('\n');
    }

    Series(const Series &s) : data_type(s.data_type) {
        data = std::make_unique<std::byte[]>(s.size);
        memcpy(data.get(), s.data.get(), s.size);
        size = s.size;
        cap = s.cap;
    }

    /// @brief constructs the series from its protobuf representation.
    explicit Series(const telem::PBSeries &s) : data_type(s.data_type()) {
        size = s.data().size();
        cap = size;
        data = std::make_unique<std::byte[]>(size);
        memcpy(data.get(), s.data().data(), size);
    }

    /// @brief encodes the series' fields into the given protobuf message.
    /// @param pb the protobuf message to encode the fields into.
    void to_proto(telem::PBSeries *pb) const {
        pb->set_data_type(data_type.name());
        pb->set_data(data.get(), size);
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
        if (data_type != synnax::FLOAT32)
            throw std::runtime_error("invalid data type");
        std::vector<float> v(size / sizeof(float));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<int64_t> int64() const {
        if (data_type != synnax::INT64)
            throw std::runtime_error("invalid data type");
        std::vector<int64_t> v(size / sizeof(int64_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<uint64_t> uint64() const {
        if (data_type != synnax::UINT64 && data_type != synnax::TIMESTAMP)
            throw std::runtime_error("invalid data type");
        std::vector<uint64_t> v(size / sizeof(uint64_t));
        memcpy(v.data(), data.get(), size);
        return v;
    }

    [[nodiscard]] std::vector<std::string> string() const {
        if (data_type != synnax::STRING && data_type != synnax::JSON)
            throw std::runtime_error("invalid data type");
        std::vector<std::string> v;
        std::string s;
        for (size_t i = 0; i < size; i++) {
            if (data[i] == static_cast<std::byte>('\n')) {
                v.push_back(s);
                s.clear();
                // WARNING: This might be very slow due to copying.
            } else s += char(data[i]);
        }
        return v;
    }

    template<typename NumericType>
    NumericType operator[](const int index) const { return at<NumericType>(index); }

    /// @brief returns the number at the given index. It is up to the caller to ensure
    /// that the numeric type is compatible with the series' data type.
    /// @param index the index to get the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    template<typename NumericType>
    [[nodiscard]] NumericType at(const int index) const {
        const auto adjusted = validateBounds(index);
        NumericType value;
        memcpy(&value, data.get() + adjusted * data_type.density(),
               data_type.density());
        return value;
    }

    /// @brief returns the number at the given index.
    /// @param index the index to get the number at.
    template<typename NumericType>
    [[nodiscard]] NumericType at(const size_t index) const {
        NumericType value;
        memcpy(&value, data.get() + index * data_type.density(), data_type.density());
        return value;
    }

    // implement the ostream operator
    friend std::ostream &operator<<(std::ostream &os, const Series &s) {
        os << "Series(" << s.data_type.name() << ", [";
        if (s.data_type == synnax::STRING || s.data_type == synnax::JSON) {
            const auto strings = s.string();
            for (const auto &string: strings) os << "\"" << string << "\" ";
        } else if (s.data_type == synnax::FLOAT32)
            for (size_t i = 0; i < s.size; i++) os << s.at<float>(i) << " ";
        else if (s.data_type == synnax::INT64)
            for (size_t i = 0; i < s.size; i++) os << s.at<int64_t>(i) << " ";
        else if (s.data_type == synnax::UINT64 || s.data_type == synnax::TIMESTAMP)
            for (size_t i = 0; i < s.size; i++) os << s.at<uint64_t>(i) << " ";
        else if (s.data_type == synnax::UINT8)
            for (size_t i = 0; i < s.size; i++) os << s.at<uint8_t>(i) << " ";
        else os << "unknown data type";
        os << "])";
        return os;
    }

    /// @brief the size of the series in number of samples.
    size_t size;
    /// @brief the capacity of the series in number of samples.
    size_t cap;
    [[nodiscard]] size_t byteSize() const {
        if (data_type.is_variable()) {
            if (cached_byte_size != 0) return cached_byte_size;
            size_t count = 0;
            for (const auto &b: data) if (b == NEWLINE_TERMINATOR) count++;
        }
        return size * data_type.density();
    }
    [[nodiscard]] size_t byteCap() const { return cap * data_type.density(); }
    /// @brief Holds what type of data is being used.
    DataType data_type;
    /// @brief Holds the underlying data.
    std::unique_ptr<std::byte[]> data;
    /// @brief an optional property that defines the time range occupied by the Series' data. This property is
    /// guaranteed to be defined when reading data from a Synnax cluster, and is particularly useful for understanding
    /// the alignment of samples in relation to another series. When read from a cluster, the start of the time range
    /// represents the timestamp of the first sample in the array (inclusive), while the end of the time
    /// range is set to the nanosecond AFTER the last sample in the array (exclusive).
    synnax::TimeRange time_range = synnax::TimeRange();

private:
    size_t cached_byte_size = 0;

    [[nodiscard]] int validateBounds(
        const int index,
        const size_t write_size = 0
    ) const {
        auto adjusted = index;
        if (index < 0) adjusted = static_cast<int>(size) + index;
        if (adjusted + write_size >= size || adjusted < 0)
            throw std::runtime_error(
                "index" + std::to_string(index) + " out of bounds for series of size" +
                std::to_string(size));
        return adjusted;
    }
};
}
