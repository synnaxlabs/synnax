// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/telem/telem.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

constexpr auto NEWLINE_TERMINATOR = static_cast<std::byte>('\n');
constexpr char NEWLINE_TERMINATOR_CHAR = '\n';

namespace synnax {
template<typename T>
static inline void
output_partial_vector(std::ostream &os, const std::vector<T> &v) {
    if (v.size() <= 6) {
        for (const auto &i: v) os << i << " ";
        return;
    }
    for (size_t i = 0; i < 3; i++) os << v[i] << " ";
    os << "... ";
    for (size_t i = v.size() - 3; i < v.size(); ++i) os << v[i] << " ";
}

static inline void
output_partial_vector_byte(std::ostream &os, const std::vector<uint8_t> &v) {
    if (v.size() <= 6) {
        for (size_t i = 0; i < v.size(); ++i)
            os << static_cast<uint32_t>(v[i]) << " ";
        return;
    }
    for (size_t i = 0; i < 3; ++i) os << static_cast<uint64_t>(v[i]) << " ";
    os << "... ";
    for (size_t i = v.size() - 3; i < v.size(); ++i)
        os << static_cast<uint64_t>(v[i]) << " ";
}


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
    ) : size(0),
        cap(cap),
        data_type(data_type),
        data(std::make_unique<std::byte[]>(cap * data_type.density())) {
        if (data_type.is_variable())
            throw std::runtime_error(
                "cannot pre-allocate a series with a variable data type");
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
        if (data_type == DATA_TYPE_UNKNOWN)
            data_type = DataType::infer<NumericType>();
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
        if (data_type == DATA_TYPE_UNKNOWN)
            data_type = DataType::infer<NumericType>();
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
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
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
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
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
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
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
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        const size_t count = std::min(d.size(), cap - size);
        if (count == 0) return 0;
        memcpy(data.get(), d.data(), count * data_type.density());
        size += count;
        return count;
    }

    /// @brief writes a single number to the series.
    /// @param d the number to be written.
    /// @returns 1 if the number was written, 0 if the series is at capacity and the
    /// sample was not written.
    template<typename NumericType>
    size_t write(const NumericType d) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        if (size >= cap) return 0;
        memcpy(data.get() + size * data_type.density(), &d, data_type.density());
        size++;
        return 1;
    }

    /// @brief writes the given array of numeric data to the series.
    /// @param d the array of numeric data to be written.
    /// @param size_ the number of samples to write.
    /// @returns the number of samples written. If the capacity of the series is exceeded,
    /// it will only write as many samples as it can hold.
    template<typename NumericType>
    size_t write(const NumericType *d, const size_t size_) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
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
        data = std::make_unique<std::byte[]>(byteSize());
        size_t offset = 0;
        for (const auto &s: d) {
            memcpy(data.get() + offset, s.data(), s.size());
            offset += s.size();
            data[offset] = static_cast<std::byte>('\n');
            offset++;
            cached_byte_size += s.size() + 1;
        }
        size = d.size();
        cap = size;
    }

    /// @brief constructs the series from the given string. This can also be a JSON
    /// encoded string, in which case the data type should be set to JSON.
    ///  @param data the string to be used as the data.
    ///  @param data_type_ the type of data being used. Defaults to STRING, but can
    ///  also be set to JSON.
    explicit Series(
        const std::string &data,
        DataType data_type_ = STRING
    ) : size(1), cap(1), data_type(std::move(data_type_)) {
        if (data_type != STRING && data_type != JSON)
            throw std::runtime_error("invalid data type c");
        cached_byte_size = data.size() + 1;
        this->data = std::make_unique<std::byte[]>(byteSize());
        memcpy(this->data.get(), data.data(), data.size());
        this->data[byteSize() - 1] = static_cast<std::byte>('\n');
    }

    /// @brief constructs the series from its protobuf representation.
    explicit Series(const telem::PBSeries &s) : data_type(s.data_type()) {
        if (data_type.is_variable()) {
            size = 0;
            for (const char &v: s.data()) if (v == NEWLINE_TERMINATOR_CHAR) size++;
            cached_byte_size = s.data().size();
        } else size = s.data().size() / data_type.density();
        cap = size;
        data = std::make_unique<std::byte[]>(byteSize());
        memcpy(data.get(), s.data().data(), byteSize());
    }

    /// @brief encodes the series' fields into the given protobuf message.
    /// @param pb the protobuf message to encode the fields into.
    void to_proto(telem::PBSeries *pb) const {
        pb->set_data_type(data_type.name());
        pb->set_data(data.get(), byteSize());
    }

    /// @brief returns the data as a vector of strings. This method can only be used
    /// if the data type is STRING or JSON.
    [[nodiscard]] std::vector<std::string> strings() const {
        if (data_type != synnax::STRING && data_type != synnax::JSON)
            throw std::runtime_error("invalid data type");
        std::vector<std::string> v;
        std::string s;
        for (size_t i = 0; i < byteSize(); i++) {
            if (data[i] == NEWLINE_TERMINATOR) {
                v.push_back(s);
                s.clear();
                // WARNING: This might be very slow due to copying.
            } else s += static_cast<char>(data[i]);
        }
        return v;
    }

    /// @brief returns the data as a vector of numeric values. It is up to the caller
    /// to ensure that the numeric type is compatible with the series' data type.
    template<typename NumericType>
    [[nodiscard]] std::vector<NumericType> values() const {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        std::vector<NumericType> v(size);
        memcpy(v.data(), data.get(), byteSize());
        return v;
    }


    /// @brief accesses the number at the given index.
    /// @param index the index to get the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    template<typename NumericType>
    NumericType operator[](const int index) const {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        return at<NumericType>(index);
    }

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

    /// @brief binds the string value at the given index to the provided string. The
    /// series' data type must be STRING or JSON.
    /// @param index the index to get the string at. If negative, the index is treated
    /// as an offset from the end of the series.
    void at(const int index, std::string &value) const {
        if (data_type != synnax::STRING && data_type != synnax::JSON)
            throw std::runtime_error("invalid data type");
        const auto adjusted = validateBounds(index);
        // iterate through the data byte by byte, incrementing the index every time we
        // hit a newline character until we reach the desired index.
        for (size_t i = 0, j = 0; i < byteSize(); i++) {
            if (data[i] == NEWLINE_TERMINATOR) {
                if (j == adjusted) return;
                value.clear();
                j++;
            } else value += static_cast<char>(data[i]);
        }
    }


    /// @brief returns the number at the given index.
    /// @param index the index to get the number at.
    template<typename NumericType>
    [[nodiscard]] NumericType at(const size_t index) const {
        NumericType value;
        memcpy(&value, data.get() + index * data_type.density(),
               data_type.density());
        return value;
    }

    friend std::ostream &operator<<(std::ostream &os, const Series &s) {
        os << "Series(type: " << s.data_type.name() << ", size: " << s.size
                << ", cap: "
                << s.cap << ", data: [";
        if (s.data_type == synnax::STRING || s.data_type == synnax::JSON)
            output_partial_vector(os, s.strings());
        else if (s.data_type == synnax::FLOAT32)
            output_partial_vector(os, s.values<float>());
        else if (s.data_type == synnax::INT64)
            output_partial_vector(os, s.values<int64_t>());
        else if (s.data_type == synnax::UINT64 || s.data_type == synnax::TIMESTAMP)
            output_partial_vector(os, s.values<uint64_t>());
        else if (s.data_type == synnax::SY_UINT8)
            output_partial_vector_byte(os, s.values<uint8_t>());
        else if (s.data_type == synnax::INT32)
            output_partial_vector(os, s.values<int32_t>());
        else if (s.data_type == synnax::INT16)
            output_partial_vector(os, s.values<int16_t>());
        else if (s.data_type == synnax::SY_UINT16)
            output_partial_vector(os, s.values<uint16_t>());
        else if (s.data_type == synnax::UINT32)
            output_partial_vector(os, s.values<uint32_t>());
        else if (s.data_type == synnax::FLOAT64)
            output_partial_vector(os, s.values<double>());
        else os << "unknown data type";
        os << "])";
        return os;
    }

    /// @brief the size of the series in number of samples.
    size_t size;
    /// @brief the capacity of the series in number of samples.
    size_t cap;

    /// @brief returns the size of the series in bytes.
    [[nodiscard]] size_t byteSize() const {
        if (data_type.is_variable()) return cached_byte_size;
        return size * data_type.density();
    }

    /// @brief returns the capacity of the series in bytes.
    [[nodiscard]] size_t byteCap() const {
        if (cap == 0 || data_type.is_variable()) return cached_byte_size;
        return cap * data_type.density();
    }

    template<typename NumericType>
    void transform_inplace(const std::function<NumericType(NumericType)>& func){
        static_assert(std::is_arithmetic_v<NumericType>, "NumericType must be a numeric type");
        if (size == 0) return;
        auto vals = values<NumericType>();
        std::transform(vals.begin(), vals.end(), vals.begin(), func);
        set_array(vals.data(), 0, vals.size());
    }


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
        if (adjusted + write_size > size || adjusted < 0)
            throw std::runtime_error(
                "index " + std::to_string(index) +
                " out of bounds for series of size " +
                std::to_string(size)
            );
        return adjusted;
    }
}; // class Series
} // namespace synnax
