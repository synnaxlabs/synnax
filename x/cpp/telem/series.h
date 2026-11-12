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
#include <cmath>
#include <cstddef>
#include <cstring>
#include <iostream>
#include <memory>
#include <string>
#include <variant>
#include <vector>

#include "x/cpp/binary/binary.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "x/go/telem/pb/frame.pb.h"
#include "x/go/telem/pb/telem.pb.h"

constexpr size_t VARIABLE_LENGTH_PREFIX_SIZE = sizeof(uint32_t);

static inline uint32_t read_u32_le(const std::byte *ptr) {
    uint32_t v = 0;
    memcpy(&v, ptr, VARIABLE_LENGTH_PREFIX_SIZE);
    return v;
}

static inline void write_u32_le(std::byte *ptr, uint32_t v) {
    memcpy(ptr, &v, VARIABLE_LENGTH_PREFIX_SIZE);
}

namespace x::telem {
// NOLINTBEGIN(bugprone-signed-char-misuse): an int8_t sign-extends on purpose.
template<typename DestType, typename SrcType>
static void cast_to_type(std::byte *dest, SrcType *src, const size_t count) {
    auto *typed_dest = reinterpret_cast<DestType *>(dest);
    for (size_t i = 0; i < count; i++)
        typed_dest[i] = static_cast<DestType>(src[i]);
}
// NOLINTEND(bugprone-signed-char-misuse)

template<typename T>
void output_partial_vector(std::ostream &os, const std::vector<T> &v) {
    if (v.size() <= 6) {
        for (const auto &i: v)
            os << i << " ";
        return;
    }
    for (size_t i = 0; i < 3; i++)
        os << v[i] << " ";
    os << "... ";
    for (size_t i = v.size() - 3; i < v.size(); ++i)
        os << v[i] << " ";
}

inline void
output_partial_vector_byte(std::ostream &os, const std::vector<uint8_t> &vec) {
    if (vec.size() <= 6) {
        for (const unsigned char v: vec)
            os << static_cast<uint32_t>(v) << " ";
        return;
    }
    for (size_t i = 0; i < 3; ++i)
        os << static_cast<uint64_t>(vec[i]) << " ";
    os << "... ";
    for (size_t i = vec.size() - 3; i < vec.size(); ++i)
        os << static_cast<uint64_t>(vec[i]) << " ";
}

/// @brief Series is a strongly typed array of telemetry samples backed by an
/// underlying binary buffer.
class Series {
    /// @brief the data type of the series.
    DataType type;
    /// @brief the capacity of the series in number of samples.
    size_t capacity;
    /// @brief cached_byte_size is an optimization for variable rate channels that
    /// caches the byte size of the series so it doesn't need to be re-calculated.
    size_t cached_byte_size = 0;
    /// @brief cached_bye_cap is an optimization for variable rate channels that caches
    /// the byte capacity of the series so it doesn't need to be re-calculated.
    size_t cached_byte_cap = 0;
    /// @brief the size of the series in number of samples.
    size_t sample_count;
    /// @brief Holds the underlying data. Mutable to support copy-on-write:
    /// shallow_copy() shares the buffer via shared_ptr, and ensure_exclusive()
    /// materializes a private copy before any mutation.
    mutable std::shared_ptr<std::byte[]> buffer;

    /// @brief allocates an uninitialized shared byte buffer. Callers always overwrite
    /// the buffer immediately (memcpy, write, etc.), so zero-initialization is
    /// unnecessary overhead.
    static std::shared_ptr<std::byte[]> alloc(size_t byte_size) {
        return std::shared_ptr<std::byte[]>(new std::byte[byte_size]);
    }

    /// @brief ensures this Series has exclusive ownership of its data buffer. If the
    /// buffer is shared (use_count > 1), materializes a private copy.
    void ensure_exclusive() const {
        if (this->buffer && this->buffer.use_count() > 1) {
            const auto bc = this->byte_cap();
            auto exclusive = alloc(bc);
            memcpy(exclusive.get(), this->buffer.get(), this->byte_size());
            this->buffer = std::move(exclusive);
        }
    }

    /// @brief moves the data into a buffer of byte_cap bytes.
    void reserve_bytes(const size_t byte_cap) {
        auto data = alloc(byte_cap);
        if (this->byte_size() > 0)
            memcpy(data.get(), this->buffer.get(), this->byte_size());
        this->buffer = std::move(data);
        this->cached_byte_cap = byte_cap;
    }

public:
    /// @brief an optional property that defines the time range occupied by the
    /// Series' buffer. This property is guaranteed to be defined when reading data
    /// from a Synnax Cluster, and is particularly useful for understanding the
    /// alignment of samples in relation to another series.
    ///
    /// When reading from a cluster:
    ///   - The start of the time range represents the timestamp of the first
    ///     sample in the array (inclusive),
    ///   - The end of the time range is set to the nanosecond AFTER the last sample
    ///     in the array (exclusive).
    ///
    TimeRange time_range;
    /// @brief alignment defines the location of the series relative to other series in
    /// a logical group. This is typically used to define the location of the series
    /// within a channel's data.
    Alignment alignment;

private:
    /// @brief validates that the count samples from index are within the series. A
    /// negative index counts from the end.
    /// @returns the index as an offset from the start.
    [[nodiscard]] size_t
    validate_bounds(const int &index, const size_t count = 1) const {
        auto adjusted = index;
        if (index < 0) adjusted = static_cast<int>(this->size()) + index;
        if (adjusted < 0 || adjusted + count > this->size())
            throw std::runtime_error(
                "index " + std::to_string(index) +
                " out of bounds for series of size " + std::to_string(this->size())
            );
        return adjusted;
    }

    /// @brief Private copy constructor that performs a deep copy.
    /// This is private to prevent accidental copying - use deep_copy() instead.
    Series(const Series &other):
        type(other.type),
        capacity(other.capacity),
        cached_byte_size(other.cached_byte_size),
        cached_byte_cap(other.cached_byte_cap),
        sample_count(other.sample_count),
        buffer(alloc(other.byte_cap())),
        time_range(other.time_range),
        alignment(other.alignment) {
        memcpy(this->buffer.get(), other.buffer.get(), other.byte_size());
    }

    /// @brief Private shallow copy constructor that shares the data buffer.
    struct ShallowTag {};

    Series(const Series &other, ShallowTag):
        type(other.type),
        capacity(other.capacity),
        cached_byte_size(other.cached_byte_size),
        cached_byte_cap(other.cached_byte_cap),
        sample_count(other.sample_count),
        buffer(other.buffer),
        time_range(other.time_range),
        alignment(other.alignment) {}

    template<typename SourceType, typename TargetType, typename Op>
#ifdef _MSC_VER
    __declspec(noinline)
#endif
    void apply_numeric_op(const TargetType &rhs, Op op) const {
        this->ensure_exclusive();
        auto *data_ptr = reinterpret_cast<SourceType *>(this->buffer.get());
        const auto size = this->size();
        const auto cast_rhs = static_cast<SourceType>(rhs);
        for (size_t i = 0; i < size; i++)
            data_ptr[i] = op(data_ptr[i], cast_rhs);
    }

    template<typename T, typename F>
#ifdef _MSC_VER
    __declspec(noinline)
#endif
    void map_numeric(F f) const {
        this->ensure_exclusive();
        auto *data_ptr = reinterpret_cast<T *>(this->buffer.get());
        const auto size = this->size();
        for (size_t i = 0; i < size; i++)
            data_ptr[i] = static_cast<T>(f(static_cast<double>(data_ptr[i])));
    }

    template<typename T, typename Op>
#ifdef _MSC_VER
    __declspec(noinline)
#endif
    void cast_and_apply_numeric_op(const T &rhs, Op op) const {
        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_numeric_op<double, T>(rhs, op);
        else if (dt == FLOAT32_T)
            apply_numeric_op<float, T>(rhs, op);
        else if (dt == INT64_T)
            apply_numeric_op<int64_t, T>(rhs, op);
        else if (dt == INT32_T)
            apply_numeric_op<int32_t, T>(rhs, op);
        else if (dt == INT16_T)
            apply_numeric_op<int16_t, T>(rhs, op);
        else if (dt == INT8_T)
            apply_numeric_op<int8_t, T>(rhs, op);
        else if (dt == UINT64_T)
            apply_numeric_op<uint64_t, T>(rhs, op);
        else if (dt == UINT32_T)
            apply_numeric_op<uint32_t, T>(rhs, op);
        else if (dt == UINT16_T)
            apply_numeric_op<uint16_t, T>(rhs, op);
        else if (dt == UINT8_T)
            apply_numeric_op<uint8_t, T>(rhs, op);
    }

    template<typename T, typename Op>
    void apply_binary_op_typed(const Series &other, Series &result, Op op) const {
        auto *lhs = reinterpret_cast<const T *>(this->buffer.get());
        auto *rhs = reinterpret_cast<const T *>(other.buffer.get());
        auto *out = reinterpret_cast<T *>(result.buffer.get());
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(lhs[i], rhs[i]);
    }

    template<typename Op>
    Series apply_binary_op(const Series &other, Op op) const {
        if (this->size() != other.size())
            throw std::runtime_error("series length mismatch for binary operation");
        if (this->data_type() != other.data_type())
            throw std::runtime_error("series type mismatch for binary operation");

        auto result = Series(this->data_type(), this->size());
        result.resize(this->size());

        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_binary_op_typed<double>(other, result, op);
        else if (dt == FLOAT32_T)
            apply_binary_op_typed<float>(other, result, op);
        else if (dt == INT64_T)
            apply_binary_op_typed<int64_t>(other, result, op);
        else if (dt == INT32_T)
            apply_binary_op_typed<int32_t>(other, result, op);
        else if (dt == INT16_T)
            apply_binary_op_typed<int16_t>(other, result, op);
        else if (dt == INT8_T)
            apply_binary_op_typed<int8_t>(other, result, op);
        else if (dt == UINT64_T)
            apply_binary_op_typed<uint64_t>(other, result, op);
        else if (dt == UINT32_T)
            apply_binary_op_typed<uint32_t>(other, result, op);
        else if (dt == UINT16_T)
            apply_binary_op_typed<uint16_t>(other, result, op);
        else if (dt == UINT8_T)
            apply_binary_op_typed<uint8_t>(other, result, op);

        return result;
    }

    template<typename T, typename Op>
    void apply_comparison_op_typed(const Series &other, Series &result, Op op) const {
        auto *lhs = reinterpret_cast<const T *>(this->buffer.get());
        auto *rhs = reinterpret_cast<const T *>(other.buffer.get());
        auto *out = reinterpret_cast<uint8_t *>(result.buffer.get());
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(lhs[i], rhs[i]) ? 1 : 0;
    }

    template<typename Op>
    Series apply_comparison_op(const Series &other, Op op) const {
        if (this->size() != other.size())
            throw std::runtime_error("series length mismatch for comparison");
        if (this->data_type() != other.data_type())
            throw std::runtime_error("series type mismatch for comparison");

        auto result = Series(BOOLEAN_T, this->size());
        result.resize(this->size());

        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_comparison_op_typed<double>(other, result, op);
        else if (dt == FLOAT32_T)
            apply_comparison_op_typed<float>(other, result, op);
        else if (dt == INT64_T)
            apply_comparison_op_typed<int64_t>(other, result, op);
        else if (dt == INT32_T)
            apply_comparison_op_typed<int32_t>(other, result, op);
        else if (dt == INT16_T)
            apply_comparison_op_typed<int16_t>(other, result, op);
        else if (dt == INT8_T)
            apply_comparison_op_typed<int8_t>(other, result, op);
        else if (dt == UINT64_T)
            apply_comparison_op_typed<uint64_t>(other, result, op);
        else if (dt == UINT32_T)
            apply_comparison_op_typed<uint32_t>(other, result, op);
        else if (dt == UINT16_T)
            apply_comparison_op_typed<uint16_t>(other, result, op);
        else if (dt == UINT8_T || dt == BOOLEAN_T)
            apply_comparison_op_typed<uint8_t>(other, result, op);

        return result;
    }

    template<typename T, typename Op>
    void apply_unary_op_typed(Series &result, Op op) const {
        auto *src = reinterpret_cast<const T *>(this->buffer.get());
        auto *out = reinterpret_cast<T *>(result.buffer.get());
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(src[i]);
    }

    template<typename Op>
    Series apply_unary_op(Op op) const {
        auto result = Series(this->data_type(), this->size());
        result.resize(this->size());

        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_unary_op_typed<double>(result, op);
        else if (dt == FLOAT32_T)
            apply_unary_op_typed<float>(result, op);
        else if (dt == INT64_T)
            apply_unary_op_typed<int64_t>(result, op);
        else if (dt == INT32_T)
            apply_unary_op_typed<int32_t>(result, op);
        else if (dt == INT16_T)
            apply_unary_op_typed<int16_t>(result, op);
        else if (dt == INT8_T)
            apply_unary_op_typed<int8_t>(result, op);
        else if (dt == UINT64_T)
            apply_unary_op_typed<uint64_t>(result, op);
        else if (dt == UINT32_T)
            apply_unary_op_typed<uint32_t>(result, op);
        else if (dt == UINT16_T)
            apply_unary_op_typed<uint16_t>(result, op);
        else if (dt == UINT8_T)
            apply_unary_op_typed<uint8_t>(result, op);

        return result;
    }

    template<typename SourceType, typename T, typename Op>
    void apply_scalar_comparison_op_typed(T scalar, Series &result, Op op) const {
        auto *src = reinterpret_cast<const SourceType *>(this->buffer.get());
        auto *out = reinterpret_cast<uint8_t *>(result.buffer.get());
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(src[i], static_cast<SourceType>(scalar)) ? 1 : 0;
    }

    template<typename T, typename Op>
    Series apply_scalar_comparison_op(T scalar, Op op) const {
        auto result = Series(BOOLEAN_T, this->size());
        result.resize(this->size());

        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_scalar_comparison_op_typed<double>(scalar, result, op);
        else if (dt == FLOAT32_T)
            apply_scalar_comparison_op_typed<float>(scalar, result, op);
        else if (dt == INT64_T)
            apply_scalar_comparison_op_typed<int64_t>(scalar, result, op);
        else if (dt == INT32_T)
            apply_scalar_comparison_op_typed<int32_t>(scalar, result, op);
        else if (dt == INT16_T)
            apply_scalar_comparison_op_typed<int16_t>(scalar, result, op);
        else if (dt == INT8_T)
            apply_scalar_comparison_op_typed<int8_t>(scalar, result, op);
        else if (dt == UINT64_T)
            apply_scalar_comparison_op_typed<uint64_t>(scalar, result, op);
        else if (dt == UINT32_T)
            apply_scalar_comparison_op_typed<uint32_t>(scalar, result, op);
        else if (dt == UINT16_T)
            apply_scalar_comparison_op_typed<uint16_t>(scalar, result, op);
        else if (dt == UINT8_T || dt == BOOLEAN_T)
            apply_scalar_comparison_op_typed<uint8_t>(scalar, result, op);

        return result;
    }

    template<typename Op>
    Series apply_logical_op(const Series &other, Op op) const {
        const auto lhs_len = this->size();
        const auto rhs_len = other.size();
        const auto max_len = lhs_len > rhs_len ? lhs_len : rhs_len;
        auto result = Series(BOOLEAN_T, max_len);
        result.resize(max_len);

        const auto *lhs = reinterpret_cast<const uint8_t *>(this->buffer.get());
        const auto *rhs = reinterpret_cast<const uint8_t *>(other.buffer.get());
        auto *out = reinterpret_cast<uint8_t *>(result.buffer.get());

        // Equal-length fast path: avoids the per-iteration branches in the broadcast
        // loop below, which defeat the compiler's ability to keep the inner loop tight.
        if (lhs_len == rhs_len) {
            for (size_t i = 0; i < lhs_len; i++)
                out[i] = op(lhs[i] != 0, rhs[i] != 0) ? 1 : 0;
            return result;
        }

        const bool lhs_last = lhs_len > 0 && lhs[lhs_len - 1] != 0;
        const bool rhs_last = rhs_len > 0 && rhs[rhs_len - 1] != 0;
        for (size_t i = 0; i < max_len; i++) {
            const bool lhs_val = i < lhs_len ? lhs[i] != 0 : lhs_last;
            const bool rhs_val = i < rhs_len ? rhs[i] != 0 : rhs_last;
            out[i] = op(lhs_val, rhs_val) ? 1 : 0;
        }
        return result;
    }

    template<typename Op>
    Series apply_logical_scalar_op(const bool scalar, Op op) const {
        auto result = Series(BOOLEAN_T, this->size());
        result.resize(this->size());
        const auto *src = reinterpret_cast<const uint8_t *>(this->buffer.get());
        auto *out = reinterpret_cast<uint8_t *>(result.buffer.get());
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(src[i] != 0, scalar) ? 1 : 0;
        return result;
    }

    template<typename T, typename Op>
    Series apply_scalar_op(T scalar, Op op) const {
        auto result = this->deep_copy();
        result.cast_and_apply_numeric_op(scalar, op);
        return result;
    }

    template<typename SourceType, typename T, typename Op>
    void apply_reverse_scalar_op_typed(T scalar, Series &result, Op op) const {
        auto *src = reinterpret_cast<const SourceType *>(this->buffer.get());
        auto *out = reinterpret_cast<SourceType *>(result.buffer.get());
        const auto cast_scalar = static_cast<SourceType>(scalar);
        for (size_t i = 0; i < this->size(); i++)
            out[i] = op(cast_scalar, src[i]);
    }

    template<typename T, typename Op>
    Series apply_reverse_scalar_op(T scalar, Op op) const {
        auto result = Series(this->data_type(), this->size());
        result.resize(this->size());

        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            apply_reverse_scalar_op_typed<double>(scalar, result, op);
        else if (dt == FLOAT32_T)
            apply_reverse_scalar_op_typed<float>(scalar, result, op);
        else if (dt == INT64_T)
            apply_reverse_scalar_op_typed<int64_t>(scalar, result, op);
        else if (dt == INT32_T)
            apply_reverse_scalar_op_typed<int32_t>(scalar, result, op);
        else if (dt == INT16_T)
            apply_reverse_scalar_op_typed<int16_t>(scalar, result, op);
        else if (dt == INT8_T)
            apply_reverse_scalar_op_typed<int8_t>(scalar, result, op);
        else if (dt == UINT64_T)
            apply_reverse_scalar_op_typed<uint64_t>(scalar, result, op);
        else if (dt == UINT32_T)
            apply_reverse_scalar_op_typed<uint32_t>(scalar, result, op);
        else if (dt == UINT16_T)
            apply_reverse_scalar_op_typed<uint16_t>(scalar, result, op);
        else if (dt == UINT8_T)
            apply_reverse_scalar_op_typed<uint8_t>(scalar, result, op);

        return result;
    }

public:
    /// @brief returns the number of samples in the series.
    [[nodiscard]] size_t size() const { return this->sample_count; }

    /// @brief returns the size of the series in bytes.
    [[nodiscard]] size_t byte_size() const {
        if (this->data_type().is_variable()) return this->cached_byte_size;
        return this->size() * this->data_type().density();
    }

    /// @brief returns true if the series is empty.
    [[nodiscard]] bool empty() const { return this->sample_count == 0; }

    /// @brief returns the data type of the series.
    [[nodiscard]] DataType data_type() const { return this->type; }

    /// @brief returns the capacity of the series in number of samples. If the
    /// series was not pre-allocated, this is the same as size().
    [[nodiscard]] size_t cap() const { return this->capacity; }

    /// @brief returns the capacity of the series in bytes.
    [[nodiscard]] size_t byte_cap() const {
        if (this->cached_byte_cap != 0) return this->cached_byte_cap;
        if (this->cap() == 0 || this->data_type().is_variable())
            return this->cached_byte_size;
        return this->cap() * this->data_type().density();
    }

    /// @brief move constructor.
    Series(Series &&other) noexcept:
        type(std::move(other.type)),
        capacity(other.capacity),
        cached_byte_size(other.cached_byte_size),
        cached_byte_cap(other.cached_byte_cap),
        sample_count(other.sample_count),
        buffer(std::move(other.buffer)),
        time_range(other.time_range),
        alignment(other.alignment) {
        other.buffer = nullptr;
    }

    /// @brief move assignment operator.
    Series &operator=(Series &&other) noexcept {
        if (this != &other) {
            this->type = std::move(other.type);
            this->capacity = other.capacity;
            cached_byte_size = other.cached_byte_size;
            cached_byte_cap = other.cached_byte_cap;
            sample_count = other.sample_count;
            this->buffer = std::move(other.buffer);
            time_range = other.time_range;
            alignment = other.alignment;
            other.buffer = nullptr;
        }
        return *this;
    }

    /// @brief returns a raw pointer to the underlying buffer. This pointer is only
    /// valid for the lifetime of the Series. If this Series was created via
    /// shallow_copy(), writing through this pointer will corrupt other copies. Use
    /// write(), set(), or other mutation methods instead, which handle copy-on-write
    /// automatically.
    [[nodiscard]] std::byte *data() const { return this->buffer.get(); }

    /// @brief allocates a series with the given data type and capacity. If the data
    /// type of the series is variable, then the capacity is treated as the number
    /// of bytes to allocate. If fixed, it is treated as the number of samples.
    /// Allocated series are treated as buffers and are not initialized
    /// with any data. Calls to write can be used to populate the series.
    /// @param data_type the type of data being stored.
    /// @param cap the number of samples that can be stored in the series.
    Series(const DataType &data_type, const size_t cap):
        type(data_type), sample_count(0) {
        if (data_type.is_variable()) {
            this->buffer = alloc(cap);
            this->cached_byte_cap = cap;
            this->capacity = 0;
        } else {
            this->buffer = alloc(cap * data_type.density());
            this->capacity = cap;
            this->cached_byte_cap = cap * data_type.density();
        }
    }

    /// @brief constructs a series from the given array of numeric data and a
    /// length.
    /// @param d the array of numeric data to be used.
    /// @param size the number of samples to be used.
    /// @param dt the data type of the series.
    template<typename NumericType>
    Series(const NumericType *d, const size_t size, const DataType &dt = UNKNOWN_T):
        type(DataType::infer<NumericType>(dt)),
        capacity(size),
        sample_count(size),
        buffer(alloc(this->size() * this->data_type().density())) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        memcpy(this->buffer.get(), d, this->size() * this->data_type().density());
    }

    /// @brief constructs a series from the given vector of numeric data and an
    /// optional data type.
    /// @param d the vector of numeric data to be used.
    /// @param dt the type of data being used. In most cases, this should not
    /// be specified and the data type will be inferred from the numeric type. If
    /// you do choose to override the data type, it's up to you to ensure that the
    /// contents of the series are compatible with the data type.
    template<typename NumericType>
    explicit Series(const std::vector<NumericType> &d, const DataType &dt = UNKNOWN_T):
        Series(d.data(), d.size(), dt) {}

    /// @brief constructs a series with a data type of TIMESTAMP containing the given
    /// vector of timestamps.
    explicit Series(const std::vector<TimeStamp> &d):
        type(TIMESTAMP_T),
        capacity(d.size()),
        sample_count(d.size()),
        buffer(alloc(d.size() * this->data_type().density())) {
        for (size_t i = 0; i < d.size(); i++) {
            const auto ov = d[i].nanoseconds();
            memcpy(
                this->buffer.get() + i * this->data_type().density(),
                &ov,
                this->data_type().density()
            );
        }
    }

    /// @brief constructs a series of size 1 with a data type of TIMESTAMP from the
    /// given timestamp.
    /// @param v the timestamp to be used.
    explicit Series(const TimeStamp v):
        type(TIMESTAMP_T),
        capacity(1),
        sample_count(1),
        buffer(alloc(this->byte_size())) {
        const auto ov = v.nanoseconds();
        memcpy(this->buffer.get(), &ov, this->byte_size());
    }

    /// @brief constructs a series of size 1 from the given number.
    /// @param v the number to be used.
    /// @param override_dt an optional data type to use. If not specified, the data
    /// type will be inferred from the numeric type. If you do choose to override
    /// the data type, it's up to you to ensure that the contents of the series are
    /// compatible with the data type.
    template<typename NumericType>
    explicit Series(NumericType v, const DataType &override_dt = UNKNOWN_T):
        type(DataType::infer<NumericType>(override_dt)),
        capacity(1),
        sample_count(1),
        buffer(alloc(this->byte_size())) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        memcpy(this->buffer.get(), &v, this->byte_size());
    }

    /// @brief constructs the series from the given vector of strings. These can
    /// also be JSON encoded strings, in which case the data type should be set to
    /// JSON.
    /// @param d the vector of strings to be used as the buffer.
    /// @param data_type the type of data being used.
    explicit Series(const std::vector<std::string> &d, DataType data_type = STRING_T):
        type(std::move(data_type)), capacity(d.size()), sample_count(d.size()) {
        if (!this->data_type().is_variable())
            throw std::runtime_error("expected data type to be STRING or JSON");
        this->cached_byte_size = 0;
        for (const auto &s: d)
            this->cached_byte_size += s.size() + VARIABLE_LENGTH_PREFIX_SIZE;
        this->buffer = alloc(this->byte_size());
        size_t offset = 0;
        for (const auto &s: d) {
            write_u32_le(this->buffer.get() + offset, static_cast<uint32_t>(s.size()));
            offset += 4;
            memcpy(this->buffer.get() + offset, s.data(), s.size());
            offset += s.size();
        }
    }

    /// @brief constructs the series from the given string.
    /// @param data the string to be used as the buffer.
    /// @param type the variable-length data type of the series. Defaults to STRING.
    explicit Series(const std::string &data, DataType type = STRING_T):
        type(std::move(type)),
        capacity(1),
        cached_byte_size(data.size() + VARIABLE_LENGTH_PREFIX_SIZE),
        sample_count(1),
        buffer(alloc(this->byte_size())) {
        if (!this->data_type().is_variable())
            throw std::runtime_error(
                "cannot set a string value on a series of a fixed-length data type"
            );
        write_u32_le(this->buffer.get(), static_cast<uint32_t>(data.size()));
        memcpy(this->buffer.get() + 4, data.data(), data.size());
    }

    /// @brief constructs a series from its protobuf representation.
    /// @param pb the protobuf representation to convert from.
    /// @return a pair containing the series and any error that occurred.
    static std::pair<Series, x::errors::Error> from_proto(const pb::Series &pb);

    /// @brief constructs the series from the given JSON value.
    explicit Series(const json::json &value): Series(value.dump(), JSON_T) {}

    /// @brief constructs a series of size 1 from the given SampleValue.
    /// @param v the SampleValue to be used.
    explicit Series(const SampleValue &v):
        type(DataType::infer(v)), capacity(1), sample_count(1) {
        if (this->data_type().is_variable()) {
            const auto &str = std::get<std::string>(v);
            cached_byte_size = str.size() + VARIABLE_LENGTH_PREFIX_SIZE;
            this->buffer = alloc(this->byte_size());
            write_u32_le(this->buffer.get(), static_cast<uint32_t>(str.size()));
            memcpy(this->buffer.get() + 4, str.data(), str.size());
            return;
        }
        std::visit(
            [this]<typename IT>(const IT &arg) {
                if constexpr (std::is_trivially_copyable_v<IT>) {
                    this->buffer = alloc(this->byte_size());
                    memcpy(this->buffer.get(), &arg, this->byte_size());
                }
            },
            v
        );
    }

    /// @brief constructs a series of size 1 from the given SampleValue that keeps the
    /// given data type. A JSON or bytes sample is a string, and a UUID sample is a
    /// string of the 16 bytes of the UUID.
    /// @param v the sample to hold.
    /// @param dt the data type of the series.
    /// @throws std::runtime_error if the sample cannot be held by the data type.
    explicit Series(const SampleValue &v, const DataType &dt):
        Series(dt, dt.is_variable() ? 0 : 1) {
        const auto inferred = DataType::infer(v);
        if ((dt.is_variable() || dt == UUID_T) && inferred == STRING_T) {
            const auto &str = std::get<std::string>(v);
            if (dt.is_variable()) {
                this->write(str);
                return;
            }
            if (str.size() != x::uuid::UUID::size())
                throw std::runtime_error(
                    "a uuid sample must hold 16 bytes, not " +
                    std::to_string(str.size())
                );
            this->write(reinterpret_cast<const uint8_t *>(str.data()), 1);
            return;
        }
        if (inferred != dt && !(dt == BOOLEAN_T && inferred == UINT8_T))
            throw std::runtime_error(
                "cannot hold a " + inferred.name() + " sample in a " + dt.name() +
                " series"
            );
        this->write(v);
    }

    /// @brief constructs the series from a vector of JSON values.
    /// @param values the vector of JSON values to be used.
    explicit Series(const std::vector<json::json> &values):
        type(JSON_T), capacity(values.size()), sample_count(values.size()) {
        for (const auto &value: values)
            this->cached_byte_size += value.dump().size() + VARIABLE_LENGTH_PREFIX_SIZE;

        this->buffer = alloc(this->byte_size());
        size_t offset = 0;
        for (const auto &value: values) {
            const auto str = value.dump();
            write_u32_le(
                this->buffer.get() + offset,
                static_cast<uint32_t>(str.size())
            );
            offset += 4;
            memcpy(this->buffer.get() + offset, str.data(), str.size());
            offset += str.size();
        }
    }

private:
    /// @brief private constructor for protobuf deserialization
    explicit Series(const pb::Series &pb, const TimeRange &tr):
        type(pb.data_type()),
        capacity(0),
        cached_byte_size(pb.data().size()),
        sample_count(0),
        time_range(tr),
        alignment(pb.alignment()) {
        // Calculate size based on data type
        if (!this->type.is_variable()) {
            this->sample_count = pb.data().size() / this->type.density();
        } else {
            const auto *ptr = reinterpret_cast<const std::byte *>(pb.data().data());
            const auto *end = ptr + pb.data().size();
            while (ptr + VARIABLE_LENGTH_PREFIX_SIZE <= end) {
                const uint32_t len = read_u32_le(ptr);
                ptr += 4 + len;
                ++this->sample_count;
            }
        }
        this->capacity = this->sample_count;

        // Copy data
        this->buffer = alloc(byte_size());
        memcpy(this->buffer.get(), pb.data().data(), byte_size());
    }

public:
    /// @brief sets a number at an index with type casting based on series data type.
    /// @param index the index to set the number at. If negative, the index is
    /// treated as an offset from the end of the series.
    /// @param value the value to set. The value will be cast to match the series'
    /// data type.
    // NOLINTBEGIN(bugprone-signed-char-misuse): an int8_t sign-extends on purpose.
    template<typename NumericType>
    void set(const int &index, const NumericType value) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        this->ensure_exclusive();
        const auto adjusted = this->validate_bounds(index);
        const auto dt = this->data_type();
        auto *base_ptr = this->buffer.get() + adjusted * dt.density();

        if (dt == FLOAT64_T) {
            *reinterpret_cast<double *>(base_ptr) = static_cast<double>(value);
        } else if (dt == FLOAT32_T) {
            *reinterpret_cast<float *>(base_ptr) = static_cast<float>(value);
        } else if (dt == INT64_T || dt == TIMESTAMP_T) {
            *reinterpret_cast<int64_t *>(base_ptr) = static_cast<int64_t>(value);
        } else if (dt == INT32_T) {
            *reinterpret_cast<int32_t *>(base_ptr) = static_cast<int32_t>(value);
        } else if (dt == INT16_T) {
            *reinterpret_cast<int16_t *>(base_ptr) = static_cast<int16_t>(value);
        } else if (dt == INT8_T) {
            *reinterpret_cast<int8_t *>(base_ptr) = static_cast<int8_t>(value);
        } else if (dt == UINT64_T) {
            *reinterpret_cast<uint64_t *>(base_ptr) = static_cast<uint64_t>(value);
        } else if (dt == UINT32_T) {
            *reinterpret_cast<uint32_t *>(base_ptr) = static_cast<uint32_t>(value);
        } else if (dt == UINT16_T) {
            *reinterpret_cast<uint16_t *>(base_ptr) = static_cast<uint16_t>(value);
        } else if (dt == UINT8_T) {
            *reinterpret_cast<uint8_t *>(base_ptr) = static_cast<uint8_t>(value);
        } else if (dt == BOOLEAN_T) {
            *reinterpret_cast<uint8_t *>(base_ptr) = static_cast<uint8_t>(
                value != 0 ? 1 : 0
            );
        }
    }
    // NOLINTEND(bugprone-signed-char-misuse)

    /// @brief sets a TimeStamp at an index.
    /// @param index the index to set the timestamp at. If negative, the index is
    /// treated as an offset from the end of the series.
    /// @param value the timestamp value to set.
    void set(const int &index, const TimeStamp value) {
        this->set(index, value.nanoseconds());
    }

    /// @brief sets a SampleValue at an index.
    /// @param index the index to set the value at. If negative, the index is
    /// treated as an offset from the end of the series.
    /// @param val the SampleValue to set. The value will be written based on
    /// the series' data type.
    void set(const int &index, const SampleValue &val) {
        if (this->data_type().is_variable()) {
            throw std::runtime_error(
                "set() with SampleValue is not supported for variable-size data types"
            );
        }
        if (std::holds_alternative<std::string>(val)) {
            throw std::runtime_error("cannot set string value on non-string series");
        }
        if (std::holds_alternative<TimeStamp>(val)) {
            this->set(index, std::get<TimeStamp>(val));
            return;
        }
        std::visit(
            [this, index]<typename T>(const T &v) {
                if constexpr (
                    !std::is_same_v<T, std::string> && !std::is_same_v<T, TimeStamp>
                ) {
                    this->set(index, v);
                }
            },
            val
        );
    }

    /// @brief sets the given array of numeric data at the given index.
    /// @param d the array of numeric data to be written.
    /// @param index the index to write the data at. If negative, the index is
    /// treated as an offset from the end of the series.
    /// @param count the number of samples to write.
    /// @throws std::runtime_error if the write would start before the first sample or
    /// run past the last one.
    template<typename NumericType>
    void set(const NumericType *d, const int &index, const size_t count) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "NumericType must be a numeric type"
        );
        this->ensure_exclusive();
        const auto adjusted = this->validate_bounds(index, count);
        memcpy(
            this->buffer.get() + adjusted * this->data_type().density(),
            d,
            count * this->data_type().density()
        );
    }

    /// @brief sets the given vector of numeric data at the given index.
    /// @param d the vector of numeric data to be written.
    /// @param index the index to write the data at. If negative, the index is
    /// treated as an offset from the end of the series.
    /// @throws std::runtime_error if the write would start before the first sample or
    /// run past the last one.
    template<typename NumericType>
    void set(const std::vector<NumericType> &d, const int &index) {
        this->set(d.data(), index, d.size());
    }

    /// @brief writes the given vector of numbers or strings to the series. A string or
    /// JSON series grows its buffer to fit the strings.
    /// @param d the vector of data to be written.
    /// @returns the number of samples written. A numeric series writes only as many
    /// samples as its capacity holds.
    template<typename T>
    size_t write(const std::vector<T> &d) {
        if constexpr (std::is_same_v<T, std::string>) {
            for (const auto &s: d)
                this->write(s);
            return d.size();
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "T must be a numeric type or string"
            );
            this->ensure_exclusive();
            const size_t count = std::min(d.size(), this->cap() - this->size());
            if (count == 0) return 0;
            memcpy(
                this->buffer.get() + this->byte_size(),
                d.data(),
                count * this->data_type().density()
            );
            this->sample_count += count;
            return count;
        }
    }

    /// @brief writes the given SampleValue to the series.
    /// @param value the SampleValue to be written.
    /// @returns 1 if the value was written, 0 if the series is at capacity and the
    /// sample was not written.
    size_t write(const SampleValue &value) {
        if (std::holds_alternative<std::string>(value))
            return write(std::get<std::string>(value));
        return std::visit([this](const auto &v) { return this->write(v); }, value);
    }

    /// @brief writes a single number or string to the series. A string or JSON series
    /// grows its buffer to fit the string.
    /// @param d the number or string to be written.
    /// @returns 1 if the sample was written, 0 if the series is at capacity and the
    /// sample was not written.
    template<typename T>
    size_t write(const T &d) {
        this->ensure_exclusive();
        if constexpr (
            std::is_same_v<T, std::string> || std::is_same_v<T, const char *> ||
            std::is_same_v<T, char *>
        ) {
            if (!this->data_type().is_variable())
                throw std::runtime_error(
                    "cannot write a string to a series of a fixed-length data type"
                );

            const char *str_data = nullptr;
            size_t str_len = 0;
            if constexpr (std::is_same_v<T, std::string>) {
                str_data = d.c_str();
                str_len = d.length();
            } else {
                str_data = d;
                str_len = strlen(d);
            }

            const auto needed = this->cached_byte_size + VARIABLE_LENGTH_PREFIX_SIZE +
                                str_len;
            if (needed > this->byte_cap())
                this->reserve_bytes(std::max(needed, 2 * this->byte_cap()));
            write_u32_le(
                this->buffer.get() + this->cached_byte_size,
                static_cast<uint32_t>(str_len)
            );
            memcpy(this->buffer.get() + this->cached_byte_size + 4, str_data, str_len);
            this->cached_byte_size += str_len + VARIABLE_LENGTH_PREFIX_SIZE;
            this->sample_count++;
            this->capacity = this->sample_count;
            return 1;
        } else if constexpr (std::is_same_v<T, TimeStamp>) {
            if (this->size() >= this->cap()) return 0;
            const auto v = d.nanoseconds();
            auto *dest = reinterpret_cast<int64_t *>(
                this->buffer.get() + this->size() * this->data_type().density()
            );
            *dest = v;
            this->sample_count++;
            return 1;
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "generic argument to write must be a numeric type, string, or "
                "TimeStamp"
            );
            if (this->size() >= this->cap()) return 0;
            const auto density = this->data_type().density();
            auto *dest = reinterpret_cast<T *>(
                this->buffer.get() + this->sample_count++ * density
            );
            *dest = d;
            return 1;
        }
    }

    /// @brief Optimized hot path for writing timestamps to the series.
    /// @param ts the timestamp to write
    /// @returns 1 if the timestamp was written, 0 if the series is at capacity
    size_t write(const TimeStamp &ts) { return this->write(ts.nanoseconds()); }

    /// @brief writes the given array of numeric data to the series.
    /// @param d the array of numeric data to be written.
    /// @param count the number of samples to write.
    /// @returns the number of samples written. If the capacity of the series is
    /// exceeded, it will only write as many samples as it can hold.
    template<typename NumericType>
    size_t write(const NumericType *d, const size_t count) {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "generic argument to write must be a numeric type"
        );
        this->ensure_exclusive();
        const size_t capped_count = std::min(count, this->cap() - this->size());
        memcpy(
            this->buffer.get() + this->byte_size(),
            d,
            capped_count * this->data_type().density()
        );
        this->sample_count += capped_count;
        return capped_count;
    }

    /// @brief converts the series to its protobuf representation.
    /// @return the protobuf representation of the series.
    [[nodiscard]] pb::Series to_proto() const;

    /// @brief returns the data as a vector of strings. This method can only be used
    /// if the data type is variable-length: STRING, JSON, or BYTES.
    [[nodiscard]] std::vector<std::string> strings() const {
        if (!this->data_type().is_variable())
            throw std::runtime_error(
                "cannot convert a series of a fixed-length data type to strings"
            );
        std::vector<std::string> v;
        v.reserve(this->size());
        const auto *ptr = this->buffer.get();
        const auto *end = ptr + this->byte_size();
        while (ptr + VARIABLE_LENGTH_PREFIX_SIZE <= end) {
            const uint32_t len = read_u32_le(ptr);
            ptr += 4;
            v.emplace_back(reinterpret_cast<const char *>(ptr), len);
            ptr += len;
        }
        return v;
    }

    /// @brief returns the data as a vector of UUIDs. This method can only be used
    /// if the data type is UUID.
    [[nodiscard]] std::vector<x::uuid::UUID> uuids() const {
        if (!this->data_type().matches({UUID_T}))
            throw std::runtime_error("cannot convert a non-UUID series to UUIDs");
        std::vector<x::uuid::UUID> v;
        v.reserve(this->size());
        const auto *ptr = this->buffer.get();
        for (size_t i = 0; i < this->size(); i++, ptr += x::uuid::UUID::size()) {
            std::array<std::uint8_t, 16> bytes{};
            std::memcpy(bytes.data(), ptr, bytes.size());
            v.emplace_back(bytes);
        }
        return v;
    }

    /// @brief returns the data as a vector of numeric values. It is up to the
    /// caller to ensure that the numeric type is compatible with the series' data
    /// type.
    template<typename NumericType>
    [[nodiscard]] std::vector<NumericType> values() const {
        static_assert(
            std::is_arithmetic_v<NumericType>,
            "template argument to values() must be a numeric type"
        );
        std::vector<NumericType> v(this->size());
        memcpy(v.data(), this->buffer.get(), this->byte_size());
        return v;
    }

    /// @brief returns the data as a vector of JSON values. This method can only be used
    /// if the data type is JSON.
    [[nodiscard]] std::vector<json::json> json_values() const {
        if (!this->data_type().matches({JSON_T}))
            throw std::runtime_error("cannot convert a non-JSON series to JSON values");

        std::vector<json::json> v;
        v.reserve(this->size());
        for (const auto &str: this->strings())
            v.push_back(json::json::parse(str));
        return v;
    }

    /// @brief accesses the number at the given index.
    /// @param index the index to get the number at. If negative, the index is
    /// treated as an offset from the end of the series.
    template<typename T>
    [[nodiscard]] T at(const int &index) const {
        if constexpr (std::is_same_v<T, std::string>) {
            std::string value;
            if (!this->data_type().is_variable())
                throw std::runtime_error(
                    "cannot bind a string value on a series of a fixed-length data type"
                );
            const auto adjusted = this->validate_bounds(index);
            const auto *ptr = this->buffer.get();
            const auto *end = ptr + this->byte_size();
            size_t j = 0;
            while (ptr + VARIABLE_LENGTH_PREFIX_SIZE <= end) {
                const uint32_t len = read_u32_le(ptr);
                ptr += 4;
                if (j == static_cast<size_t>(adjusted))
                    return std::string(reinterpret_cast<const char *>(ptr), len);
                ptr += len;
                ++j;
            }
            return value;
        } else if constexpr (std::is_same_v<T, TimeStamp>)
            return TimeStamp(this->at<int64_t>(index));
        else if constexpr (std::is_same_v<T, x::uuid::UUID>) {
            if (!this->data_type().matches({UUID_T}))
                throw std::runtime_error("cannot bind a UUID on a non-UUID series");
            std::array<std::uint8_t, 16> bytes{};
            std::memcpy(
                bytes.data(),
                this->buffer.get() + this->validate_bounds(index) * bytes.size(),
                bytes.size()
            );
            return x::uuid::UUID(bytes);
        } else {
            static_assert(
                std::is_arithmetic_v<T>,
                "template argument to at must be a numeric type or string"
            );
            const auto adjusted = this->validate_bounds(index);
            T value;
            memcpy(
                &value,
                this->buffer.get() + adjusted * this->data_type().density(),
                this->data_type().density()
            );
            return value;
        }
    }

    template<typename T>
    [[nodiscard]] T operator[](const int index) {
        return this->at(index);
    }

    /// @returns the value at the given index.
    [[nodiscard]] SampleValue at(const int &index) const {
        const auto dt = this->data_type();
        if (dt == FLOAT64_T) return this->at<double>(index);
        if (dt == FLOAT32_T) return this->at<float>(index);
        if (dt == INT64_T) return this->at<int64_t>(index);
        if (dt == INT32_T) return this->at<int32_t>(index);
        if (dt == INT16_T) return this->at<int16_t>(index);
        if (dt == INT8_T) return this->at<int8_t>(index);
        if (dt == UINT64_T) return this->at<uint64_t>(index);
        if (dt == UINT32_T) return this->at<uint32_t>(index);
        if (dt == UINT16_T) return this->at<uint16_t>(index);
        if (dt == UINT8_T) return this->at<uint8_t>(index);
        if (dt == BOOLEAN_T) return this->at<uint8_t>(index);
        if (dt == TIMESTAMP_T) return this->at<TimeStamp>(index);
        if (dt.is_variable()) return this->at<std::string>(index);
        throw std::runtime_error("unsupported data type for at: " + dt.name());
    }

    /// @brief binds the JSON value at the given index to the provided json object.
    /// The series' data type must be JSON.
    /// @param index the index to get the JSON at. If negative, the index is treated
    /// as an offset from the end of the series.
    /// @param value the json object to bind the value to.
    void at(const int &index, json::json &value) const {
        if (!this->data_type().matches({JSON_T}))
            throw std::runtime_error("cannot bind a JSON value on a non-JSON series");
        value = json::json::parse(this->at<std::string>(index));
    }

    friend std::ostream &operator<<(std::ostream &os, const Series &s) {
        const auto dt = s.data_type();
        os << "Series(type: " << dt.name() << ", size: " << s.size()
           << ", cap: " << s.cap() << ", data: [";
        if (dt == STRING_T || dt == JSON_T)
            output_partial_vector(os, s.strings());
        else if (dt == FLOAT32_T)
            output_partial_vector(os, s.values<float>());
        else if (dt == INT64_T || dt == TIMESTAMP_T)
            output_partial_vector(os, s.values<int64_t>());
        else if (dt == UINT64_T)
            output_partial_vector(os, s.values<uint64_t>());
        else if (dt == UINT8_T || dt == BOOLEAN_T)
            output_partial_vector_byte(os, s.values<uint8_t>());
        else if (dt == INT32_T)
            output_partial_vector(os, s.values<int32_t>());
        else if (dt == INT16_T)
            output_partial_vector(os, s.values<int16_t>());
        else if (dt == UINT16_T)
            output_partial_vector(os, s.values<uint16_t>());
        else if (dt == UINT32_T)
            output_partial_vector(os, s.values<uint32_t>());
        else if (dt == FLOAT64_T)
            output_partial_vector(os, s.values<double>());
        else
            os << "unknown data type";
        os << "])";
        return os;
    }

    /// @brief Writes evenly spaced timestamps between start and end to the series.
    /// @param start The starting timestamp
    /// @param end The ending timestamp
    /// @param count The number of points to write
    /// @param inclusive Whether to include the end timestamp as the last value
    /// @returns The number of timestamps written
    size_t write_linspace(
        const TimeStamp &start,
        const TimeStamp &end,
        const size_t count,
        const bool inclusive = false
    ) {
        this->ensure_exclusive();
        if (count == 0) return 0;
        if (count == 1) return write(start);

        const auto write_count = std::min(count, this->cap() - this->size());
        if (write_count == 0) return 0;

        const auto adjusted_count = inclusive ? write_count - 1 : write_count;
        const int64_t start_ns = start.nanoseconds();
        const int64_t step_ns = (end - start).nanoseconds() /
                                static_cast<int64_t>(adjusted_count);
        auto *data_ptr = reinterpret_cast<int64_t *>(
            this->buffer.get() + this->size() * this->data_type().density()
        );
        for (size_t i = 0; i < write_count; i++)
            data_ptr[i] = start_ns + step_ns * static_cast<int64_t>(i);
        this->sample_count += write_count;
        return write_count;
    }

    /// @brief Creates a timestamp series with evenly spaced values between start
    /// and end (inclusive).
    /// @param start The starting timestamp
    /// @param end The ending timestamp
    /// @param count The number of points to generate
    /// @param inclusive Whether to include the end timestamp as the last value
    /// in the series.
    /// @return A Series containing evenly spaced timestamps
    static Series linspace(
        const TimeStamp &start,
        const TimeStamp &end,
        const size_t count,
        const bool inclusive = false
    ) {
        Series s(TIMESTAMP_T, count);
        s.write_linspace(start, end, count, inclusive);
        return s;
    }

    /// @brief writes data to the series while performing any necessary type casting
    /// @param data the data to write
    /// @param size the number of samples to write
    /// @param source_type the data type of the source data
    /// @returns the number of samples written
    size_t
    write_casted(const void *data, const size_t size, const DataType &source_type) {
        if (source_type == FLOAT64_T)
            return write_casted(static_cast<const double *>(data), size);
        if (source_type == FLOAT32_T)
            return write_casted(static_cast<const float *>(data), size);
        if (source_type == INT64_T)
            return write_casted(static_cast<const int64_t *>(data), size);
        if (source_type == INT32_T)
            return write_casted(static_cast<const int32_t *>(data), size);
        if (source_type == INT16_T)
            return write_casted(static_cast<const int16_t *>(data), size);
        if (source_type == INT8_T)
            return write_casted(static_cast<const int8_t *>(data), size);
        if (source_type == UINT64_T)
            return write_casted(static_cast<const uint64_t *>(data), size);
        if (source_type == UINT32_T)
            return write_casted(static_cast<const uint32_t *>(data), size);
        if (source_type == UINT16_T)
            return write_casted(static_cast<const uint16_t *>(data), size);
        if (source_type == UINT8_T)
            return write_casted(static_cast<const uint8_t *>(data), size);
        throw std::runtime_error(
            "Unsupported data type for casting: " + source_type.name()
        );
    }

    /// @brief constructor that conditionally casts that provided data array to the
    /// given data type.
    /// @param data_type - the data type of the series.
    /// @param data - the data to write to the series. If data_type is the same as
    /// the inferred type of this data, then it will be directly written to the
    /// series. Otherwise, each sample in data will be cast to the correct data
    /// type.
    /// @param size - the number of samples in the data array.
    template<typename T>
    static Series cast(const DataType &data_type, T *data, const size_t size) {
        static_assert(std::is_arithmetic_v<T>, "T must be a numeric type");
        auto s = Series(data_type, size);
        s.write_casted(data, size);
        return s;
    }

    static Series cast(
        const DataType &target_type,
        const void *data,
        const size_t size,
        const DataType &source_type
    ) {
        auto s = Series(target_type, size);
        s.write_casted(data, size, source_type);
        return s;
    }

    template<typename T>
    void add_inplace(const T &rhs) const noexcept {
        cast_and_apply_numeric_op(rhs, std::plus<T>());
    }

    template<typename T>
    void sub_inplace(const T &rhs) const noexcept {
        cast_and_apply_numeric_op(rhs, std::minus<T>());
    }

    template<typename T>
    void multiply_inplace(const T &rhs) const noexcept {
        cast_and_apply_numeric_op(rhs, std::multiplies<T>());
    }

    template<typename T>
    void divide_inplace(const T &rhs) const {
        if (rhs == 0) throw std::runtime_error("division by zero");
        cast_and_apply_numeric_op(rhs, std::divides<T>());
    }

    /// @brief replaces each value v with f(v). f takes and returns a double, and the
    /// result is cast back to the data type of the series.
    /// @throws std::runtime_error if the data type of the series is not numeric.
    template<typename F>
    void map_inplace(F f) const {
        const auto dt = this->data_type();
        if (dt == FLOAT64_T)
            map_numeric<double>(f);
        else if (dt == FLOAT32_T)
            map_numeric<float>(f);
        else if (dt == INT64_T)
            map_numeric<int64_t>(f);
        else if (dt == INT32_T)
            map_numeric<int32_t>(f);
        else if (dt == INT16_T)
            map_numeric<int16_t>(f);
        else if (dt == INT8_T)
            map_numeric<int8_t>(f);
        else if (dt == UINT64_T)
            map_numeric<uint64_t>(f);
        else if (dt == UINT32_T)
            map_numeric<uint32_t>(f);
        else if (dt == UINT16_T)
            map_numeric<uint16_t>(f);
        else if (dt == UINT8_T)
            map_numeric<uint8_t>(f);
        else
            throw std::runtime_error("cannot map a series of type " + dt.name());
    }

    /// @brief Series-Series addition operator. Returns a new Series.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator+(const Series &other) const {
        return apply_binary_op(other, [](auto a, auto b) { return a + b; });
    }

    /// @brief Series-Series subtraction operator. Returns a new Series.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator-(const Series &other) const {
        return apply_binary_op(other, [](auto a, auto b) { return a - b; });
    }

    /// @brief Series-Series multiplication operator. Returns a new Series.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator*(const Series &other) const {
        return apply_binary_op(other, [](auto a, auto b) { return a * b; });
    }

    /// @brief Series-Series division operator. Returns a new Series.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator/(const Series &other) const {
        return apply_binary_op(other, [](auto a, auto b) { return a / b; });
    }

    /// @brief Series-Series modulo operator. Returns a new Series.
    /// Uses % for integer types, std::fmod for floating-point types.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator%(const Series &other) const {
        return apply_binary_op(other, [](auto a, auto b) {
            if constexpr (std::is_integral_v<decltype(a)>) {
                return a % b;
            } else {
                return std::fmod(a, b);
            }
        });
    }

    /// @brief Series + scalar operator. Returns a new Series.
    template<typename T>
    Series operator+(T scalar) const {
        return apply_scalar_op(scalar, std::plus<T>());
    }

    /// @brief Series - scalar operator. Returns a new Series.
    template<typename T>
    Series operator-(T scalar) const {
        return apply_scalar_op(scalar, std::minus<T>());
    }

    /// @brief Series * scalar operator. Returns a new Series.
    template<typename T>
    Series operator*(T scalar) const {
        return apply_scalar_op(scalar, std::multiplies<T>());
    }

    /// @brief Series / scalar operator. Returns a new Series.
    /// @throws std::runtime_error if scalar is zero.
    template<typename T>
    Series operator/(T scalar) const {
        if (scalar == 0) throw std::runtime_error("division by zero");
        return apply_scalar_op(scalar, std::divides<T>());
    }

    /// @brief Series % scalar operator. Returns a new Series.
    /// Uses % for integer types, std::fmod for floating-point types.
    /// @throws std::runtime_error if scalar is zero.
    template<typename T>
    Series operator%(T scalar) const {
        if (scalar == 0) throw std::runtime_error("modulo by zero");
        if constexpr (std::is_integral_v<T>) {
            return apply_scalar_op(scalar, std::modulus<T>());
        } else {
            return apply_scalar_op(scalar, [](auto a, auto b) {
                return std::fmod(a, b);
            });
        }
    }

    /// @brief scalar + Series operator (commutative). Returns a new Series.
    template<typename T>
    friend Series operator+(T scalar, const Series &s) {
        return s + scalar;
    }

    /// @brief scalar * Series operator (commutative). Returns a new Series.
    template<typename T>
    friend Series operator*(T scalar, const Series &s) {
        return s * scalar;
    }

    /// @brief scalar - Series operator. Computes (scalar - element) for each element.
    template<typename T>
    friend Series operator-(T scalar, const Series &s) {
        return s.apply_reverse_scalar_op(scalar, std::minus<T>());
    }

    /// @brief scalar / Series operator. Computes (scalar / element) for each element.
    template<typename T>
    friend Series operator/(T scalar, const Series &s) {
        return s.apply_reverse_scalar_op(scalar, std::divides<T>());
    }

    /// @brief scalar % Series operator. Computes (scalar % element) for each element.
    /// Uses % for integer types, std::fmod for floating-point types.
    template<typename T>
    friend Series operator%(T scalar, const Series &s) {
        if constexpr (std::is_integral_v<T>) {
            return s.apply_reverse_scalar_op(scalar, std::modulus<T>());
        } else {
            return s.apply_reverse_scalar_op(scalar, [](auto a, auto b) {
                return std::fmod(a, b);
            });
        }
    }

    /// @brief Series > Series comparison. Returns BOOLEAN_T Series with 0/1 values.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator>(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a > b; });
    }

    /// @brief Series < Series comparison. Returns BOOLEAN_T Series with 0/1 values.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator<(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a < b; });
    }

    /// @brief Series >= Series comparison. Returns BOOLEAN_T Series with 0/1 values.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator>=(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a >= b; });
    }

    /// @brief Series <= Series comparison. Returns BOOLEAN_T Series with 0/1 values.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator<=(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a <= b; });
    }

    /// @brief Series == Series element-wise comparison. Returns BOOLEAN_T Series.
    /// Note: This performs element-wise comparison, not structural equality.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator==(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a == b; });
    }

    /// @brief Series != Series element-wise comparison. Returns BOOLEAN_T Series.
    /// @throws std::runtime_error if series lengths or types don't match.
    Series operator!=(const Series &other) const {
        return apply_comparison_op(other, [](auto a, auto b) { return a != b; });
    }

    /// @brief Series > scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator>(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) { return a > b; });
    }

    /// @brief Series < scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator<(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) { return a < b; });
    }

    /// @brief Series >= scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator>=(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) {
            return a >= b;
        });
    }

    /// @brief Series <= scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator<=(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) {
            return a <= b;
        });
    }

    /// @brief Series == scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator==(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) {
            return a == b;
        });
    }

    /// @brief Series != scalar comparison. Returns BOOLEAN_T Series with 0/1 values.
    template<typename T>
    Series operator!=(T scalar) const {
        return apply_scalar_comparison_op(scalar, [](auto a, auto b) {
            return a != b;
        });
    }

    /// @brief Unary negation operator. Returns a new Series with negated values.
    /// Works for all numeric types (signed and unsigned).
    Series operator-() const {
        return apply_unary_op([](auto a) { return -a; });
    }

    /// @brief Logical NOT. Returns a BOOLEAN_T Series where each element is
    /// 1 if the original was 0, and 0 if the original was non-zero.
    [[nodiscard]] Series logical_not() const {
        return apply_scalar_comparison_op(0, [](auto a, auto b) { return a == b; });
    }

    /// @brief Logical AND with another boolean series. Returns a BOOLEAN_T Series whose
    /// length is the longer operand's; the shorter operand's last value repeats.
    [[nodiscard]] Series logical_and(const Series &other) const {
        return apply_logical_op(other, [](const bool a, const bool b) {
            return a && b;
        });
    }

    /// @brief Logical OR with another boolean series. Returns a BOOLEAN_T Series whose
    /// length is the longer operand's; the shorter operand's last value repeats.
    [[nodiscard]] Series logical_or(const Series &other) const {
        return apply_logical_op(other, [](const bool a, const bool b) {
            return a || b;
        });
    }

    /// @brief Logical AND with a scalar. Returns a BOOLEAN_T Series of the input's
    /// length.
    [[nodiscard]] Series logical_and(const bool scalar) const {
        return apply_logical_scalar_op(scalar, [](const bool a, const bool b) {
            return a && b;
        });
    }

    /// @brief Logical OR with a scalar. Returns a BOOLEAN_T Series of the input's
    /// length.
    [[nodiscard]] Series logical_or(const bool scalar) const {
        return apply_logical_scalar_op(scalar, [](const bool a, const bool b) {
            return a || b;
        });
    }

    /// @brief returns a one-sample series holding the last sample. Throws when empty.
    [[nodiscard]] Series last() const {
        const auto v = this->at(-1);
        Series out = this->data_type().is_variable()
                       ? Series(std::get<std::string>(v), this->data_type())
                       : Series(v);
        out.type = this->data_type();
        const auto offset = static_cast<std::uint32_t>(this->size() - 1);
        out.alignment = Alignment(
            this->alignment.domain_index(),
            this->alignment.sample_index() + offset
        );
        out.time_range = this->time_range;
        return out;
    }

    /// @brief deep copies the series, including its buffer. This function
    /// should be called explicitly (as opposed to an implicit copy constructor) to
    /// avoid accidental deep copies.
    [[nodiscard]] Series deep_copy() const { return {*this}; }

    /// @brief returns a shallow copy that shares the underlying data buffer. The copy
    /// is safe to read concurrently. If either copy is mutated, ensure_exclusive()
    /// materializes a private copy first (copy-on-write). Not thread-safe for
    /// concurrent mutation of the same Series instance.
    [[nodiscard]] Series shallow_copy() const { return {*this, ShallowTag{}}; }

    void clear() {
        this->sample_count = 0;
        if (this->data_type().is_variable()) this->cached_byte_size = 0;
    }

    /// @brief detaches the data buffer for reuse after a shallow_copy. If the buffer is
    /// shared (use_count > 1), allocates a fresh buffer of the same capacity. Resets
    /// size to 0 while preserving capacity.
    void detach_buffer() {
        const auto bc = this->byte_cap();
        if (this->buffer && this->buffer.use_count() > 1) {
            this->buffer = alloc(bc);
            this->cached_byte_cap = bc;
        }
        this->sample_count = 0;
        this->cached_byte_size = 0;
    }

    /// @brief copies src into this series, reusing its buffer capacity to avoid
    /// allocation. Holders on emit hot paths should prefer this over deep_copy()
    /// (SY-4506). A buffer shared via shallow_copy() is replaced, not written in
    /// place.
    void copy_from(const Series &src) {
        const auto bs = src.byte_size();
        const auto bc = this->buffer == nullptr ? 0 : this->byte_cap();
        if (this->buffer == nullptr || this->buffer.use_count() > 1 || bc < bs) {
            this->buffer = alloc(bs);
            this->cached_byte_cap = bs;
        } else
            this->cached_byte_cap = bc;
        if (bs > 0) memcpy(this->buffer.get(), src.buffer.get(), bs);
        this->type = src.type;
        this->sample_count = src.sample_count;
        this->cached_byte_size = src.data_type().is_variable() ? bs : 0;
        this->capacity = this->type.is_variable()
                           ? src.sample_count
                           : this->cached_byte_cap / this->type.density();
        this->time_range = src.time_range;
        this->alignment = src.alignment;
    }

    void resize(size_t new_size) {
        if (this->data_type().is_variable()) {
            throw std::runtime_error(
                "resize not supported for variable-size data types"
            );
        }
        this->ensure_exclusive();
        if (new_size > this->capacity) {
            const auto density = this->data_type().density();
            auto new_data = alloc(new_size * density);
            if (this->sample_count > 0) {
                memcpy(
                    new_data.get(),
                    this->buffer.get(),
                    this->sample_count * density
                );
            }
            this->buffer = std::move(new_data);
            this->capacity = new_size;
            this->cached_byte_cap = new_size * density;
        }
        this->sample_count = new_size;
    }

    /// @brief writes data to the series while performing any necessary type casting
    /// @param data the data to write
    /// @param size the number of samples to write
    /// @returns the number of samples written
    template<typename T>
    size_t write_casted(const T *data, const size_t size) {
        static_assert(std::is_arithmetic_v<T>, "T must be a numeric type");
        this->ensure_exclusive();
        const auto count = std::min(size, this->cap() - this->size());
        if (count == 0) return 0;

        const auto inferred_type = DataType::infer<T>();
        if (inferred_type == this->data_type()) {
            memcpy(
                this->buffer.get() + this->size() * this->data_type().density(),
                data,
                count * this->data_type().density()
            );
        } else {
            auto *dest = this->buffer.get() +
                         this->size() * this->data_type().density();
            if (this->data_type() == FLOAT64_T)
                cast_to_type<double>(dest, data, count);
            else if (this->data_type() == FLOAT32_T)
                cast_to_type<float>(dest, data, count);
            else if (this->data_type() == INT64_T)
                cast_to_type<int64_t>(dest, data, count);
            else if (this->data_type() == INT32_T)
                cast_to_type<int32_t>(dest, data, count);
            else if (this->data_type() == INT16_T)
                cast_to_type<int16_t>(dest, data, count);
            else if (this->data_type() == INT8_T)
                cast_to_type<int8_t>(dest, data, count);
            else if (this->data_type() == UINT64_T)
                cast_to_type<uint64_t>(dest, data, count);
            else if (this->data_type() == UINT32_T)
                cast_to_type<uint32_t>(dest, data, count);
            else if (this->data_type() == UINT16_T)
                cast_to_type<uint16_t>(dest, data, count);
            else if (this->data_type() == UINT8_T)
                cast_to_type<uint8_t>(dest, data, count);
            else
                throw std::runtime_error(
                    "Unsupported data type for casting: " + this->data_type().name()
                );
        }
        this->sample_count += count;
        return count;
    }

    /// @brief writes a vector to the series while performing any necessary type
    /// casting
    /// @param data the vector to write
    /// @returns the number of samples written
    template<typename T>
    size_t write_casted(const std::vector<T> &data) {
        return write_casted(data.data(), data.size());
    }

    /// @brief writes the data from another series to this series
    /// @param other the series to write from
    /// @returns the number of samples written
    /// @throws std::runtime_error if the data types don't match
    size_t write(const Series &other) {
        this->ensure_exclusive();
        const size_t byte_count = std::min(
            other.byte_size(),
            this->byte_cap() - this->byte_size()
        );
        memcpy(this->buffer.get() + this->byte_size(), other.buffer.get(), byte_count);
        const auto count = byte_count / this->data_type().density();
        this->sample_count += count;
        return count;
    }

    /// @brief Calculates the average of all values in the series
    /// @returns The average value as the specified numeric type
    /// @throws std::runtime_error if the series is empty or if the data type is not
    /// numeric
    template<typename T>
    [[nodiscard]] T avg() const {
        static_assert(
            std::is_arithmetic_v<T>,
            "Template argument must be a numeric type"
        );

        if (this->empty())
            throw std::runtime_error("Cannot calculate average of empty series");

        if (this->data_type().is_variable())
            throw std::runtime_error("Cannot calculate average of non-numeric series");

        T sum = 0;
        const auto size = this->size();

        if (this->data_type() == FLOAT64_T) {
            const auto *data_ptr = reinterpret_cast<const double *>(this->buffer.get());
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == FLOAT32_T) {
            const auto *data_ptr = reinterpret_cast<const float *>(this->buffer.get());
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == INT64_T) {
            const auto *data_ptr = reinterpret_cast<const int64_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == INT32_T) {
            const auto *data_ptr = reinterpret_cast<const int32_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == INT16_T) {
            const auto *data_ptr = reinterpret_cast<const int16_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == INT8_T) {
            const auto *data_ptr = reinterpret_cast<const int8_t *>(this->buffer.get());
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == UINT64_T) {
            const auto *data_ptr = reinterpret_cast<const uint64_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == UINT32_T) {
            const auto *data_ptr = reinterpret_cast<const uint32_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == UINT16_T) {
            const auto *data_ptr = reinterpret_cast<const uint16_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == UINT8_T) {
            const auto *data_ptr = reinterpret_cast<const uint8_t *>(
                this->buffer.get()
            );
            for (size_t i = 0; i < size; i++)
                sum += static_cast<T>(data_ptr[i]);
        } else if (this->data_type() == TIMESTAMP_T) {
            throw std::runtime_error("Cannot calculate average of timestamp series");
        } else {
            throw std::runtime_error(
                "Unsupported data type for average: " + this->data_type().name()
            );
        }

        return sum / static_cast<T>(size);
    }

    /// @brief fills the series with data from the given binary reader. Reads until
    /// the series is full or the reader is exhausted, whichever comes first. Returns
    /// the total number of samples read.
    size_t fill_from(binary::Reader &reader) {
        this->ensure_exclusive();
        auto n_read = reader.read(this->data() + this->byte_size(), this->byte_cap());
        this->cached_byte_size += n_read;
        if (this->data_type().is_variable()) {
            this->sample_count = 0;
            const auto *ptr = this->buffer.get();
            const auto *end = ptr + this->byte_size();
            while (ptr + VARIABLE_LENGTH_PREFIX_SIZE <= end) {
                const uint32_t len = read_u32_le(ptr);
                ptr += 4 + len;
                ++this->sample_count;
            }
        } else
            this->sample_count += n_read / this->data_type().density();
        return n_read;
    }
};

/// @brief MultiSeries holds multiple series for accumulating data from a channel.
/// This matches Go's telem.MultiSeries pattern for handling multiple data arrivals
/// before consumption.
struct MultiSeries {
    /// @brief accumulated series.
    std::vector<Series> series;

    /// @brief Append adds a series to the accumulation.
    void append(Series s) { series.push_back(std::move(s)); }

    /// @brief Clear removes all accumulated series.
    void clear() { series.clear(); }

    /// @brief Empty returns true if no series are accumulated.
    [[nodiscard]] bool empty() const { return series.empty(); }

    /// @brief Size returns the number of accumulated series.
    [[nodiscard]] size_t size() const { return series.size(); }
};

}
