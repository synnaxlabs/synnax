// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <cstddef>
#include <string>
#include <vector>

/// external
#include "nlohmann/json.hpp"

/// internal
#include "x/cpp/telem/telem.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

using json = nlohmann::json;

constexpr char NEWLINE_CHAR = '\n';
constexpr auto NEWLINE_TERMINATOR = static_cast<std::byte>(NEWLINE_CHAR);

namespace telem {
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


/// @brief Series is a strongly typed array of telemetry samples backed by an underlying binary buffer.
class Series {
public:
    /// @brief Holds what type of data is being used.
    const DataType data_type;
    /// @brief the capacity of the series in number of samples.
    const size_t cap;

private:
    /// @brief cached_byte_size is an optimization for variable rate channels that
    /// caches the byte size of the series so it doesn't need to be re-calculated.
    size_t cached_byte_size = 0;
    /// @brief the size of the series in number of samples.
    size_t size_;
    /// @brief Holds the underlying data.
    std::unique_ptr<std::byte[]> data;
    /// @brief an optional property that defines the time range occupied by the Series' data. This property is
    /// guaranteed to be defined when reading data from a Synnax cluster, and is particularly useful for understanding
    /// the alignment of samples in relation to another series. When read from a cluster, the start of the time range
    /// represents the timestamp of the first sample in the array (inclusive), while the end of the time
    /// range is set to the nanosecond AFTER the last sample in the array (exclusive).
    telem::TimeRange time_range = telem::TimeRange();
    /// @brief validates the input index is within the bounds of the series. If the
    /// write size is provided, it will also validate that the write does not exceed
    /// the capacity of the series.
    [[nodiscard]] int validate_bounds(
        const int &index,
        const size_t write_size = 0
    ) const {
        auto adjusted = index;
        if (index < 0) adjusted = static_cast<int>(this->size()) + index;
        if (adjusted + write_size > this->size() || adjusted < 0)
            throw std::runtime_error(
                "index " + std::to_string(index) +
                " out of bounds for series of size " +
                std::to_string(this->size())
            );
        return adjusted;
    }

    /// @brief Private copy constructor that performs a deep copy.
    /// This is private to prevent accidental copying - use deep_copy() instead.
    Series(const Series &other):
        data_type(other.data_type),
        cap(other.cap),
        cached_byte_size(other.cached_byte_size),
        size_(other.size_),
        data(std::make_unique<std::byte[]>(other.byte_size())),
        time_range(other.time_range) {
        memcpy(data.get(), other.data.get(), other.byte_size());
    }
public:
    [[nodiscard]] size_t size() const { return size_; }

    [[nodiscard]] bool empty() const { return size_ == 0; }

    /// @brief move constructor.
    Series(Series &&other) noexcept:
        data_type(other.data_type),
        cap(other.cap),
        cached_byte_size(other.cached_byte_size),
        size_(other.size_),
        data(std::move(other.data)),
        time_range(other.time_range) {
    }

    /// @brief allocates a series with the given data type and capacity (in samples).
    /// Allocated series are treated as buffers and are not initialized with any data.
    /// Calls to write can be used to populate the series.
    /// @param data_type the type of data being stored.
    /// @param cap the number of samples that can be stored in the series.
    Series(const DataType &data_type, const size_t cap) :
        data_type(data_type),
        cap(cap),
        size_(0),
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
        const DataType dt = DATA_TYPE_UNKNOWN
    ):
        data_type(telem::DataType::infer<NumericType>(dt)),
        cap(d.size()),
        size_(d.size()),
        data(std::make_unique<std::byte[]>(d.size() * this->data_type.density())) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        memcpy(this->data.get(), d.data(), d.size() * this->data_type.density());
    }

    /// @brief constructs a series of size 1 with a data type of TIMESTAMP from the
    /// given timestamp.
    /// @param v the timestamp to be used.
    explicit Series(const TimeStamp v) :
        data_type(telem::TIMESTAMP_T),
        cap(1),
        size_(1),
        data(std::make_unique<std::byte[]>(this->byte_size())) {
        memcpy(data.get(), &v.value, this->byte_size());
    }

    /// @brief constructs a series of size 1 from the given number.
    /// @param v the number to be used.
    /// @param override_dt an optional data type to use. If not specified, the data type
    /// will be inferred from the numeric type. If you do choose to override the
    /// data type, it's up to you to ensure that the contents of the series are
    /// compatible with the data type.
    template<typename NumericType>
    explicit Series(
        NumericType v,
        const DataType override_dt = DATA_TYPE_UNKNOWN
    ) :
        data_type(telem::DataType::infer<NumericType>(override_dt)),
        cap(1),
        size_(1),
        data(std::make_unique<std::byte[]>(this->byte_size())) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        memcpy(this->data.get(), &v, this->byte_size());
    }

    /// @brief constructs the series from the given vector of strings. These can also
    /// be JSON encoded strings, in which case the data type should be set to JSON.
    /// @param d the vector of strings to be used as the data.
    /// @param data_type_ the type of data being used.
    explicit Series(const std::vector<std::string> &d, DataType data_type_ = STRING_T):
        data_type(std::move(data_type_)),
        cap(d.size()),
        size_(d.size()) {
        if (!this->data_type.is_variable())
            throw std::runtime_error("expected data type to be STRING or JSON");
        this->cached_byte_size = 0;
        for (const auto &s: d) this->cached_byte_size += s.size() + 1;
        this->data = std::make_unique<std::byte[]>(byte_size());
        size_t offset = 0;
        for (const auto &s: d) {
            memcpy(this->data.get() + offset, s.data(), s.size());
            offset += s.size();
            this->data[offset] = NEWLINE_TERMINATOR;
            offset++;
        }
    }

    /// @brief constructs the series from the given string. This can also be a JSON
    /// encoded string, in which case the data type should be set to JSON.
    /// @param data the string to be used as the data.
    /// @param data_type_ the type of data being used. Defaults to STRING, but can
    /// also be set to JSON.
    explicit Series(const std::string &data, DataType data_type_ = STRING_T):
        data_type(std::move(data_type_)),
        cap(1),
        cached_byte_size(data.size() + 1),
        size_(1),
        data(std::make_unique<std::byte[]>(this->byte_size())) {
        if (!this->data_type.matches({STRING_T, JSON_T}))
            throw std::runtime_error(
                "cannot set a string value on a non-string or JSON series");
        memcpy(this->data.get(), data.data(), data.size());
        this->data[byte_size() - 1] = NEWLINE_TERMINATOR;
    }


    /// @brief constructs the series from its protobuf representation.
    explicit Series(const telem::PBSeries &s)
        : data_type(s.data_type()),
          cap(this->size()),
          cached_byte_size(s.data().size()),
          size_(0) {
        if (!this->data_type.is_variable())
            this->size_ = s.data().size() / this->data_type.density();
        for (const char &v: s.data())if (v == NEWLINE_CHAR) ++this->size_;
        this->data = std::make_unique<std::byte[]>(byte_size());
        memcpy(this->data.get(), s.data().data(), byte_size());
    }

    /// @brief constructs the series from the given JSON value.
    explicit Series(const json &value): Series(value.dump(), JSON_T) {
    }

    /// @brief sets a number at an index.
    /// @param index the index to set the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the value to set. The provided value should be compatible with
    /// the series' data type. It is up to you to ensure that this is the case.
    template<typename NumericType>
    void set(const int &index, const NumericType value) {
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
    void set_array(const NumericType *d, const int &index, const size_t size_) {
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
    void set(const std::vector<NumericType> &d, const int &index) {
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
    template<typename T>
    size_t write(const std::vector<T> &d) {
        if constexpr (std::is_same_v<T, std::string>) {
            if (!this->data_type.matches({STRING_T, JSON_T}))
                throw std::runtime_error(
                    "cannot write strings to non-string/JSON series");
            const size_t count = std::min(d.size(), this->cap - this->size());
            if (count == 0) return 0;
            size_t offset = 0;
            for (size_t i = 0; i < count; i++) {
                const auto &s = d[i];
                memcpy(this->data.get() + offset, s.data(), s.size());
                offset += s.size();
                this->data.get()[offset] = NEWLINE_TERMINATOR;
                offset++;
            }
            this->cached_byte_size = offset;
            this->size_ += count;
            return count;
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "T must be a numeric type or string"
            );
            const size_t count = std::min(d.size(), this->cap - this->size());
            if (count == 0) return 0;
            memcpy(this->data.get(), d.data(), count * this->data_type.density());
            this->size_ += count;
            return count;
        }
    }

    /// @brief writes a single number to the series.
    /// @param d the number to be written.
    /// @returns 1 if the number was written, 0 if the series is at capacity and the
    /// sample was not written.
    template<typename T>
    size_t write(const T d) {
        if constexpr (std::is_same_v<T, std::string> ||
                      std::is_same_v<T, const char *> ||
                      std::is_same_v<T, char *>) {
            if (!this->data_type.matches({STRING_T, JSON_T}))
                throw std::runtime_error(
                    "cannot write string to non-string/JSON series");
            if (this->size() >= this->cap) return 0;

            const char *str_data;
            size_t str_len;
            if constexpr (std::is_same_v<T, std::string>) {
                str_data = d.c_str();
                str_len = d.length();
            } else {
                str_data = d;
                str_len = strlen(d);
            }

            memcpy(this->data.get() + cached_byte_size, str_data, str_len);
            this->data.get()[cached_byte_size + str_len] = NEWLINE_TERMINATOR;
            this->cached_byte_size += str_len + 1;
            this->size_++;
            return 1;
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "generic argument to write must be a numeric type or string"
            );
            if (this->size() >= this->cap) return 0;
            memcpy(
                data.get() + this->size() * this->data_type.density(),
                &d,
                this->data_type.density()
            );
            this->size_++;
            return 1;
        }
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
        const size_t count = std::min(size_, this->cap - this->size());
        memcpy(this->data.get(), d, count * this->data_type.density());
        this->size_ += count;
        return count;
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
        if (!data_type.matches({STRING_T, JSON_T}))
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
        std::vector<NumericType> v(this->size());
        memcpy(v.data(), this->data.get(), this->byte_size());
        return v;
    }

    /// @brief accesses the number at the given index.
    /// @param index the index to get the number at. If negative, the index is treated
    /// as an offset from the end of the series.
    template<typename T>
    [[nodiscard]] T at(const int &index) const {
        if constexpr (std::is_same_v<T, std::string>) {
            std::string value;
            this->at(index, value);
            return value;
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "template argument to at must be a numeric type or string"
            );
            const auto adjusted = this->validate_bounds(index);
            T value;
            memcpy(
                &value,
                this->data.get() + adjusted * this->data_type.density(),
                this->data_type.density()
            );
            return value;
        }
    }

    template<typename T>
    [[nodiscard]] T operator[](int index) { return this->at(index); }

    /// @returns the value at the given index.
    [[nodiscard]] SampleValue at(const int &index) const {
        const auto adjusted = validate_bounds(index);
        const auto dt = this->data_type;
        if (dt == FLOAT64_T) return this->at<double>(adjusted);
        if (dt == FLOAT32_T) return this->at<float>(adjusted);
        if (dt == INT64_T) return this->at<int64_t>(adjusted);
        if (dt == INT32_T) return this->at<int32_t>(adjusted);
        if (dt == INT16_T) return this->at<int16_t>(adjusted);
        if (dt == INT8_T) return this->at<int8_t>(adjusted);
        if (dt == UINT64_T) return this->at<uint64_t>(adjusted);
        if (dt == UINT32_T) return this->at<uint32_t>(adjusted);
        if (dt == UINT16_T) return this->at<uint16_t>(adjusted);
        if (dt == UINT8_T) return this->at<uint8_t>(adjusted);
        if (dt == STRING_T || dt == JSON_T) {
            std::string value;
            this->at(adjusted, value);
            return value;
        }
        throw std::runtime_error(
            "unsupported data type for value_at: " + data_type.name()
        );
    }


    /// @brief binds the string value at the given index to the provided string. The
    /// series' data type must be STRING or JSON.
    /// @param index the index to get the string at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the string to bind the value to.
    void at(const int &index, std::string &value) const {
        if (!data_type.matches({STRING_T, JSON_T}))
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

    /// @brief binds the JSON value at the given index to the provided json object. The
    /// series' data type must be JSON.
    /// @param index the index to get the JSON at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the json object to bind the value to.
    void at(const int &index, json &value) const {
        if (!data_type.matches({JSON_T}))
            throw std::runtime_error("cannot bind a JSON value on a non-JSON series");
        std::string str_value;
        this->at(index, str_value);
        value = json::parse(str_value);
    }

    friend std::ostream &operator<<(std::ostream &os, const telem::Series &s) {
        os << "Series(type: " << s.data_type.name() << ", size: " << s.size()
                << ", cap: "
                << s.cap << ", data: [";
        if (s.data_type == telem::STRING_T || s.data_type == telem::JSON_T)
            output_partial_vector(os, s.strings());
        else if (s.data_type == telem::FLOAT32_T)
            output_partial_vector(os, s.values<float>());
        else if (s.data_type == telem::INT64_T)
            output_partial_vector(os, s.values<int64_t>());
        else if (s.data_type == telem::UINT64_T || s.data_type == telem::TIMESTAMP_T)
            output_partial_vector(os, s.values<uint64_t>());
        else if (s.data_type == telem::UINT8_T)
            output_partial_vector_byte(os, s.values<uint8_t>());
        else if (s.data_type == telem::INT32_T)
            output_partial_vector(os, s.values<int32_t>());
        else if (s.data_type == telem::INT16_T)
            output_partial_vector(os, s.values<int16_t>());
        else if (s.data_type == telem::UINT16_T)
            output_partial_vector(os, s.values<uint16_t>());
        else if (s.data_type == telem::UINT32_T)
            output_partial_vector(os, s.values<uint32_t>());
        else if (s.data_type == telem::FLOAT64_T)
            output_partial_vector(os, s.values<double>());
        else os << "unknown data type";
        os << "])";
        return os;
    }


    /// @brief returns the size of the series in bytes.
    [[nodiscard]] size_t byte_size() const {
        if (this->data_type.is_variable()) return this->cached_byte_size;
        return this->size() * this->data_type.density();
    }

    /// @brief returns the capacity of the series in bytes.
    [[nodiscard]] size_t byte_cap() const {
        if (this->cap == 0 || this->data_type.is_variable())
            return this->
                    cached_byte_size;
        return this->cap * this->data_type.density();
    }

    template<typename NumericType>
    void map_inplace(const std::function<NumericType(const NumericType &)> &func) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "template argument to transform_inplace must be a numeric type"
        );
        if (size() == 0) return;
        auto vals = this->values<NumericType>();
        std::transform(vals.begin(), vals.end(), vals.begin(), func);
        set_array(vals.data(), 0, vals.size());
    }

    /// @brief deep copies the series, including all of its data. This function
    /// should be called explicitly (as opposed to an implicit copy constructor) to
    /// avoid accidental deep copies.
    [[nodiscard]] Series deep_copy() const { return Series(*this); }
}; // class Series
} // namespace telem
