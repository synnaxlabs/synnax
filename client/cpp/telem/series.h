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
#include <vector>
#include <variant>

#include "client/cpp/telem/telem.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

constexpr auto NEWLINE_TERMINATOR = static_cast<std::byte>('\n');
constexpr char NEWLINE_TERMINATOR_CHAR = '\n';

namespace synnax {
template<typename T>
static void
output_partial_vector(
    std::ostream &os,
    const std::vector<T> &v
) {
    if (v.size() <= 6) {
        for (const auto &i: v) os << i << " ";
        return;
    }
    for (size_t i = 0; i < 3; i++) os << v[i] << " ";
    os << "... ";
    for (size_t i = v.size() - 3; i < v.size(); ++i) os << v[i] << " ";
}

static void output_partial_vector_byte(
    std::ostream &os,
    const std::vector<uint8_t> &vec
) {
    if (vec.size() <= 6) {
        for (const unsigned char v: vec)
            os << static_cast<uint32_t>(v) << " ";
        return;
    }
    for (size_t i = 0; i < 3; ++i) os << static_cast<uint64_t>(vec[i]) << " ";
    os << "... ";
    for (size_t i = vec.size() - 3; i < vec.size(); ++i)
        os << static_cast<uint64_t>(vec[i]) << " ";
}


/// @brief all the possible types for a sample within the series.
/// THE ORDER OF THESE TYPES IS VERY IMPORTANT. DO NOT CHANGE IT.
using SampleValue = std::variant<
    double, // FLOAT64
    float, // FLOAT32
    int64_t, // INT64
    int32_t, // INT32
    int16_t, // INT16
    int8_t, // INT8
    uint64_t, // UINT64
    uint32_t, // UINT32
    uint16_t, // UINT16
    uint8_t, // UINT8
    std::string // STRING
>;

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
        data = std::make_unique<std::byte[]>(byte_size());
        memcpy(data.get(), d.data(), byte_size());
    }

    /// @brief constructs a series of size 1 with a data type of TIMESTAMP from the
    /// given timestamp.
    /// @param v the timestamp to be used.
    explicit Series(
        const TimeStamp v
    ) : size(1),
        cap(1),
        data_type(synnax::TIMESTAMP) {
        data = std::make_unique<std::byte[]>(this->byte_size());
        memcpy(data.get(), &v.value, this->byte_size());
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
        if (this->data_type == DATA_TYPE_UNKNOWN)
            this->data_type = DataType::infer<NumericType>();
        this->data = std::make_unique<std::byte[]>(this->byte_size());
        memcpy(this->data.get(), &v, this->byte_size());
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
        const auto adjusted = this->validate_bounds(index);
        memcpy(
            this->data.get() + adjusted * this->data_type.density(),
            &value,
            this->data_type.density()
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
        const auto adjusted = this->validate_bounds(index, size_);
        memcpy(
            this->data.get() + adjusted * this->data_type.density(),
            d,
            size_ * this->data_type.density()
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
        const auto adjusted = this->validate_bounds(index, d.size());
        memcpy(
            this->data.get() + adjusted * this->data_type.density(),
            d.data(),
            d.size() * this->data_type.density()
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
        const size_t count = std::min(d.size(), this->cap - this->size);
        if (count == 0) return 0;
        memcpy(this->data.get(), d.data(), count * this->data_type.density());
        this->size += count;
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
            "generic argument to write must be a numeric type"
        );
        if (this->size >= this->cap) return 0;
        memcpy(
            data.get() + this->size * this->data_type.density(),
            &d,
            this->data_type.density()
        );
        this->size++;
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
            "generic argument to write must be a numeric type"
        );
        const size_t count = std::min(size_, this->cap - this->size);
        memcpy(this->data.get(), d, count * this->data_type.density());
        this->size += count;
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
        if (!this->data_type.is_variable())
            throw std::runtime_error("expected data type to be STRING or JSON");
        this->cached_byte_size = 0;
        this->data = std::make_unique<std::byte[]>(byte_size());
        size_t offset = 0;
        for (const auto &s: d) {
            memcpy(this->data.get() + offset, s.data(), s.size());
            offset += s.size();
            this->data[offset] = static_cast<std::byte>('\n');
            offset++;
            this->cached_byte_size += s.size() + 1;
        }
        this->size = d.size();
        this->cap = size;
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
        if (!this->data_type.matches({STRING, JSON}))
            throw std::runtime_error(
                "cannot set a string value on a non-string or JSON series");
        this->cached_byte_size = data.size() + 1;
        this->data = std::make_unique<std::byte[]>(byte_size());
        memcpy(this->data.get(), data.data(), data.size());
        this->data[byte_size() - 1] = static_cast<std::byte>('\n');
    }

    /// @brief constructs the series from its protobuf representation.
    explicit Series(const telem::PBSeries &s) : data_type(s.data_type()) {
        if (this->data_type.is_variable()) {
            this->size = 0;
            for (const char &v: s.data())
                if (v == NEWLINE_TERMINATOR_CHAR)
                    this->size
                            ++;
            this->cached_byte_size = s.data().size();
        } else this->size = s.data().size() / this->data_type.density();
        this->cap = this->size;
        this->data = std::make_unique<std::byte[]>(byte_size());
        memcpy(this->data.get(), s.data().data(), byte_size());
    }

    /// @brief encodes the series' fields into the given protobuf message.
    /// @param pb the protobuf message to encode the fields into.
    void to_proto(telem::PBSeries *pb) const {
        pb->set_data_type(this->data_type.name());
        pb->set_data(this->data.get(), byte_size());
    }

    /// @brief returns the data as a vector of strings. This method can only be used
    /// if the data type is STRING or JSON.
    [[nodiscard]] std::vector<std::string> strings() const {
        if (!data_type.matches({STRING, JSON}))
            throw std::runtime_error(
                "cannot convert a non-JSON or non-string series to strings");
        std::vector<std::string> v;
        std::string buf;
        for (size_t i = 0; i < this->byte_size(); i++) {
            if (this->data[i] == NEWLINE_TERMINATOR) {
                v.push_back(buf);
                buf.clear();
                // WARNING: This might be very slow due to copying.
            } else buf += static_cast<char>(this->data[i]);
        }
        return v;
    }

    /// @brief returns the data as a vector of numeric values. It is up to the caller
    /// to ensure that the numeric type is compatible with the series' data type.
    template<typename NumericType>
    [[nodiscard]] std::vector<NumericType> values() const {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "template argument to values() must be a numeric type"
        );
        std::vector<NumericType> v(this->size);
        memcpy(v.data(), this->data.get(), this->byte_size());
        return v;
    }


    /// @brief accesses the number at the given index.
    /// @param index the index to get the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    template<typename NumericType>
    NumericType operator[](const int index) const {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "template argument to operator[] must be a numeric type"
        );
        return this->at<NumericType>(index);
    }

    /// @brief returns the number at the given index. It is up to the caller to ensure
    /// that the numeric type is compatible with the series' data type.
    /// @param index the index to get the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    template<typename NumericType>
    [[nodiscard]] NumericType at(const int index) const {
        const auto adjusted = this->validate_bounds(index);
        NumericType value;
        memcpy(
            &value,
            this->data.get() + adjusted * this->data_type.density(),
            this->data_type.density()
        );
        return value;
    }

    /// @brief binds the string value at the given index to the provided string. The
    /// series' data type must be STRING or JSON.
    /// @param index the index to get the string at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the string to bind the value to.
    void at(const int index, std::string &value) const {
        if (!data_type.matches({STRING, JSON}))
            throw std::runtime_error(
                "cannot bind a string value on a non-string or JSON series"
            );
        const auto adjusted = this->validate_bounds(index);
        // iterate through the data byte by byte, incrementing the index every time we
        // hit a newline character until we reach the desired index.
        for (size_t i = 0, j = 0; i < this->byte_size(); i++)
            if (data[i] == NEWLINE_TERMINATOR) {
                if (j == adjusted) return;
                value.clear();
                j++;
            } else value += static_cast<char>(this->data[i]);
    }


    /// @brief returns the number at the given index.
    /// @param index the index to get the number at.
    template<typename NumericType>
    [[nodiscard]] NumericType at(const size_t index) const {
        NumericType value;
        memcpy(
            &value,
            this->data.get() + index * this->data_type.density(),
            this->data_type.density()
        );
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
    [[nodiscard]] size_t byte_size() const {
        if (this->data_type.is_variable()) return this->cached_byte_size;
        return this->size * this->data_type.density();
    }

    /// @brief returns the capacity of the series in bytes.
    [[nodiscard]] size_t byte_cap() const {
        if (this->cap == 0 || this->data_type.is_variable())
            return this->
                    cached_byte_size;
        return this->cap * this->data_type.density();
    }

    template<typename NumericType>
    void transform_inplace(const std::function<NumericType(NumericType)> &func) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "template argument to transform_inplace must be a numeric type"
        );
        if (size == 0) return;
        auto vals = this->values<NumericType>();
        std::transform(vals.begin(), vals.end(), vals.begin(), func);
        set_array(vals.data(), 0, vals.size());
    }

    /// @brief deep copies the series, including all of its data. This function
    /// should be called explicitly (as opposed to an implicit copy constructor) to
    /// avoid accidental deep copies.
    [[nodiscard]] Series deep_copy() const {
        Series s(data_type, cap);
        s.size = size;
        s.cached_byte_size = this->cached_byte_size;
        memcpy(s.data.get(), this->data.get(), this->byte_size());
        return s;
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

    /// @returns the
    [[nodiscard]] SampleValue at(const int index) const {
        const auto adjusted = validate_bounds(index);
        const auto dt = this->data_type;
        if (dt == FLOAT64) return this->at<double>(adjusted);
        if (dt == FLOAT32) return this->at<float>(adjusted);
        if (dt == INT64) return this->at<int64_t>(adjusted);
        if (dt == INT32) return this->at<int32_t>(adjusted);
        if (dt == INT16) return this->at<int16_t>(adjusted);
        if (dt == INT8) return this->at<int8_t>(adjusted);
        if (dt == UINT64) return this->at<uint64_t>(adjusted);
        if (dt == UINT32) return this->at<uint32_t>(adjusted);
        if (dt == SY_UINT16) return this->at<uint16_t>(adjusted);
        if (dt == SY_UINT8) return this->at<uint8_t>(adjusted);
        if (dt == STRING || dt == JSON) {
            std::string value;
            this->at(adjusted, value);
            return value;
        }
        throw std::runtime_error(
            "unsupported data type for value_at: " + data_type.name()
        );
    }

private:
    /// @brief cached_byte_size is an optimization for variable rate channels that
    /// caches the byte size of the series so it doesn't need to be re-calculated.
    size_t cached_byte_size = 0;

    /// @brief validates the input index is within the bounds of the series. If the
    /// write size is provided, it will also validate that the write does not exceed
    /// the capacity of the series.
    [[nodiscard]] int validate_bounds(
        const int index,
        const size_t write_size = 0
    ) const {
        auto adjusted = index;
        if (index < 0) adjusted = static_cast<int>(this->size) + index;
        if (adjusted + write_size > this->size || adjusted < 0)
            throw std::runtime_error(
                "index " + std::to_string(index) +
                " out of bounds for series of size " +
                std::to_string(this->size)
            );
        return adjusted;
    }
}; // class Series
} // namespace synnax
