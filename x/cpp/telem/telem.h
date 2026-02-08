// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cmath>
#include <cstdint>
#include <functional>
#include <iostream>
#include <string>
#include <typeindex>
#include <unordered_map>
#include <variant>
#include <vector>

#include <google/protobuf/struct.pb.h>

namespace x::telem {
// private namespace for internal constants
namespace _priv {
constexpr int64_t NANOSECOND = 1;
constexpr int64_t MICROSECOND = NANOSECOND * 1e3;
constexpr int64_t MILLISECOND = MICROSECOND * 1e3;
constexpr int64_t SECOND = MILLISECOND * 1e3;
constexpr int64_t MINUTE = SECOND * 60;
constexpr int64_t HOUR = MINUTE * 60;
constexpr int64_t DAY = HOUR * 24;
}

/// @brief timespan is a nanosecond-precision time duration.
class TimeSpan {
    /// @property value holds the internal, primitive value of the timespan.
    std::int64_t value;

public:
    TimeSpan(): value(0) {}

    /// @brief returns the absolute value of the timespan.
    [[nodiscard]] TimeSpan abs() const { return TimeSpan(std::abs(value)); }

    /// @brief Constructs a timespan from the given int64, interpreting it as a
    /// nanosecond-precision timespan.
    explicit TimeSpan(const std::int64_t i): value(i) {}

    /// @brief returns a new TimeSpan from the given chrono duration.
    explicit TimeSpan(const std::chrono::duration<std::int64_t, std::nano> &duration):
        value(duration.count()) {}

    /// @brief returns the number of nanoseconds in the timespan.
    [[nodiscard]] std::int64_t nanoseconds() const { return this->value; }

    bool operator==(const std::int64_t &other) const { return value == other; }

    bool operator!=(const std::int64_t &other) const { return value != other; }

    bool operator<(const int64_t &other) const { return value < other; }

    bool operator>(const int64_t &other) const { return value > other; }

    TimeSpan operator+(const std::int64_t &other) const {
        return TimeSpan(value + other);
    }

    friend TimeSpan operator-(const std::int64_t &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    friend TimeSpan operator+(const std::int64_t &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs + rhs.value);
    }

    TimeSpan operator-(const std::int64_t &other) const {
        return TimeSpan(value - other);
    }

    friend TimeSpan operator*(const std::int64_t &lhs, const TimeSpan &rhs) {
        return TimeSpan(rhs.value * lhs);
    }

    TimeSpan operator*(const std::int64_t &other) const {
        return TimeSpan(other * value);
    }

    friend TimeSpan operator%(const std::int64_t &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs % rhs.value);
    }

    TimeSpan operator%(const std::int64_t &other) const {
        return TimeSpan(value % other);
    }

    TimeSpan operator+=(const int64_t &other) {
        this->value += other;
        return *this;
    }

    TimeSpan operator-=(const int64_t &other) {
        this->value -= other;
        return *this;
    }

    TimeSpan operator*=(const int64_t &other) {
        this->value *= other;
        return *this;
    }

    TimeSpan operator/=(const int64_t &other) {
        this->value /= other;
        return *this;
    }

    TimeSpan operator%=(const int64_t &other) {
        this->value %= other;
        return *this;
    }

    friend TimeSpan operator/(const std::int64_t &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs / rhs.value);
    }

    TimeSpan operator/(const std::int64_t &other) const {
        return TimeSpan(value / other);
    }

    bool operator==(const TimeSpan &other) const { return value == other.value; }
    bool operator!=(const TimeSpan &other) const { return value != other.value; }
    bool operator<(const TimeSpan &other) const { return value < other.value; }
    bool operator>(const TimeSpan &other) const { return value > other.value; }
    bool operator<=(const TimeSpan &other) const { return value <= other.value; }
    bool operator>=(const TimeSpan &other) const { return value >= other.value; }

    TimeSpan operator+(const TimeSpan &other) const {
        return TimeSpan(value + other.value);
    }

    TimeSpan operator+=(const TimeSpan &other) {
        this->value += other.value;
        return *this;
    }

    TimeSpan operator-(const TimeSpan &other) const {
        return TimeSpan(value - other.value);
    }

    TimeSpan operator/(const TimeSpan &other) const {
        return TimeSpan(value / other.value);
    }

    TimeSpan operator*(const TimeSpan &other) const {
        return TimeSpan(value * other.value);
    }

    TimeSpan operator%(const TimeSpan &other) const {
        return TimeSpan(value % other.value);
    }

    TimeSpan operator*(const size_t &other) const {
        return TimeSpan(value * static_cast<std::int64_t>(other));
    }

    TimeSpan operator*(const float &other) const {
        return TimeSpan(static_cast<std::int64_t>(static_cast<double>(value) * other));
    }

    TimeSpan operator*(const int &other) const { return TimeSpan(value * other); }

    TimeSpan operator*(const unsigned int &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const double &other) const {
        return TimeSpan(static_cast<std::int64_t>(static_cast<double>(value) * other));
    }

    [[nodiscard]] TimeSpan truncate(const TimeSpan &other) const {
        if (other == 0) return *this;
        return TimeSpan(value / other.value * other.value);
    }

    [[nodiscard]] TimeSpan delta(const TimeSpan &other) const {
        if (other > *this) return other - *this;
        return *this - other;
    }

    /// @brief returns the exact number of days in the timespan as double-precision
    /// floating point.
    [[nodiscard]] double days() const {
        return static_cast<double>(value) / _priv::DAY;
    }

    /// @brief returns the exact number of hours in the timespan as double-precision
    /// floating point value.
    [[nodiscard]] double hours() const {
        return static_cast<double>(value) / _priv::HOUR;
    }

    /// @brief returns the exact number of minutes in the timespan as double-precision
    /// floating point value.
    [[nodiscard]] double minutes() const {
        return static_cast<double>(value) / _priv::MINUTE;
    }

    /// @brief returns the exact number of seconds in the timespan as double-precision
    /// floating point value.
    [[nodiscard]] double seconds() const {
        return static_cast<double>(value) / _priv::SECOND;
    }

    /// @brief returns the exact number of milliseconds in the timespan as a
    /// double-precision floating point value.
    [[nodiscard]] double milliseconds() const {
        return static_cast<double>(value) / _priv::MILLISECOND;
    }

    /// @brief returns the exact number of microseconds in the timespan as a double
    /// precision floating point value.
    [[nodiscard]] double microseconds() const {
        return static_cast<double>(value) / _priv::MICROSECOND;
    }

    /// @brief returns a pretty-printed string representation of the timespan.
    [[nodiscard]] std::string to_string() const {
        const auto total_days = this->truncate(TimeSpan(_priv::DAY));
        const auto total_hours = this->truncate(TimeSpan(_priv::HOUR));
        const auto total_minutes = this->truncate(TimeSpan(_priv::MINUTE));
        const auto total_seconds = this->truncate(TimeSpan(_priv::SECOND));
        const auto total_milliseconds = this->truncate(TimeSpan(_priv::MILLISECOND));
        const auto total_microseconds = this->truncate(TimeSpan(_priv::MICROSECOND));
        const auto total_nanoseconds = this->value;

        const auto days = total_days.value / _priv::DAY;
        const auto hours = (total_hours.value - total_days.value) / _priv::HOUR;
        const auto minutes = (total_minutes.value - total_hours.value) / _priv::MINUTE;
        const auto seconds = (total_seconds.value - total_minutes.value) /
                             _priv::SECOND;
        const auto milliseconds = (total_milliseconds.value - total_seconds.value) /
                                  _priv::MILLISECOND;
        const auto microseconds = (total_microseconds.value -
                                   total_milliseconds.value) /
                                  _priv::MICROSECOND;
        const auto nanoseconds = total_nanoseconds - total_microseconds.value;

        std::string out;
        if (days != 0) out += std::to_string(days) + "d ";
        if (hours != 0) out += std::to_string(hours) + "h ";
        if (minutes != 0) out += std::to_string(minutes) + "m ";
        if (seconds != 0) out += std::to_string(seconds) + "s ";
        if (milliseconds != 0) out += std::to_string(milliseconds) + "ms ";
        if (microseconds != 0) out += std::to_string(microseconds) + "us ";
        if (nanoseconds != 0) out += std::to_string(nanoseconds) + "ns";
        if (out.empty()) return "0ns";
        if (out.back() == ' ') out.pop_back();
        return out;
    }

    friend std::ostream &operator<<(std::ostream &os, const TimeSpan &ts) {
        os << ts.to_string();
        return os;
    }

    /// @brief returns the timespan as a chrono duration.
    [[nodiscard]] std::chrono::nanoseconds chrono() const {
        return std::chrono::nanoseconds(value);
    }

    /// @brief a zero nanosecond timespan.
    static TimeSpan ZERO() { return TimeSpan(0); }

    /// @brief the maximum representable timespan.
    static TimeSpan max() { return TimeSpan(std::numeric_limits<int64_t>::max()); }

    /// @brief the minimum representable timespan.
    static TimeSpan min() { return TimeSpan(std::numeric_limits<int64_t>::min()); }
};

/// @brief represents a 64-bit nanosecond-precision, UNIX Epoch UTC timestamp.
class TimeStamp {
    /// @property value holds the internal, primitive value of the timestamp.
    std::int64_t value;

public:
    TimeStamp() = default;

    /// @brief Constructs a timestamp from the given interpreting it as a
    /// nanosecond-precision UTC timestamp.
    explicit TimeStamp(const std::int64_t value): value(value) {}

    /// @brief returns the number of nanoseconds in the timestamp.
    [[nodiscard]] std::int64_t nanoseconds() const { return this->value; }

    /// @brief interprets the given TimeSpan as a TimeStamp.
    explicit TimeStamp(const TimeSpan ts): value(ts.nanoseconds()) {}

    /// @brief the maximum representable timestamp.
    static TimeStamp max() { return TimeStamp(std::numeric_limits<int64_t>::max()); }

    /// @brief the minimum representable timestamp.
    static TimeStamp min() { return TimeStamp(std::numeric_limits<int64_t>::min()); }

    TimeStamp static now() {
        // note that on some machines, hig-res clock refs system_clock and on others
        // it references steady_clock. This could create a problem so we should
        // probably use system_clock.
        return TimeStamp(
            std::chrono::duration_cast<std::chrono::nanoseconds>(
                std::chrono::system_clock::now().time_since_epoch()
            )
                .count()
        );
    }

    TimeStamp static midpoint(const TimeStamp &start, const TimeStamp &end) {
        return TimeStamp(start + (end - start) / 2);
    }

    bool operator==(const TimeStamp &other) const { return value == other.value; }

    bool operator!=(const TimeStamp &other) const { return value != other.value; }

    bool operator<(const TimeStamp &other) const { return value < other.value; }

    bool operator>(const TimeStamp &other) const { return value > other.value; }

    bool operator<=(const TimeStamp &other) const { return value <= other.value; }

    bool operator>=(const TimeStamp &other) const { return value >= other.value; }

    TimeStamp operator+(const TimeSpan &other) const {
        return TimeStamp(value + other.nanoseconds());
    }

    TimeSpan operator-(const TimeStamp &other) const {
        return TimeSpan(value - other.value);
    }

    TimeStamp operator-(const TimeSpan &other) const {
        return TimeStamp(value - other.nanoseconds());
    }

    TimeStamp operator*(const TimeStamp &other) const {
        return TimeStamp(value * other.value);
    }

    TimeStamp operator/(const TimeStamp &other) const {
        return TimeStamp(value / other.value);
    }

    TimeStamp operator%(const TimeStamp &other) const {
        return TimeStamp(value % other.value);
    }

    TimeStamp operator+=(const TimeStamp &other) {
        return TimeStamp(value += other.value);
    }

    TimeStamp operator-=(const TimeStamp &other) {
        return TimeStamp(value -= other.value);
    }

    TimeStamp operator*=(const TimeStamp &other) {
        return TimeStamp(value *= other.value);
    }

    TimeStamp operator/=(const TimeStamp &other) {
        return TimeStamp(value /= other.value);
    }

    TimeStamp operator%=(const TimeStamp &other) {
        return TimeStamp(value %= other.value);
    }

    TimeStamp operator+(const TimeStamp &other) const {
        return TimeStamp(value + other.value);
    }

    bool operator==(const int &other) const { return value == other; }

    bool operator!=(const int &other) const { return value != other; }

    friend TimeStamp operator+(const std::int64_t &lhs, const TimeStamp &rhs) {
        return TimeStamp(lhs + rhs.value);
    }

    friend TimeSpan operator-(const std::int64_t &lhs, const TimeStamp &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    friend std::ostream &operator<<(std::ostream &os, const TimeStamp &ts) {
        os << ts.value;
        return os;
    }
};

inline TimeSpan since(const TimeStamp stamp) {
    return TimeStamp::now() - stamp;
}

class TimeRange {
public:
    TimeStamp start;
    TimeStamp end;

    TimeRange() = default;

    /// @brief constructs a TimeRange from the given start and end timestamps.
    TimeRange(const TimeStamp start, const TimeStamp end): start(start), end(end) {}

    TimeRange(const std::int64_t start, const std::int64_t end):
        start(start), end(end) {}

    /// @brief returns true if the given timestamp is within the range, start
    /// inclusive, end exclusive.
    [[nodiscard]] bool contains(const TimeStamp time) const {
        return start <= time && time < end;
    }

    /// @brief returns true if the TimeRange contains the given TimeRange. If the
    /// two time ranges are equal, returns true. In this case, the two time ranges
    /// contain each other.
    [[nodiscard]] bool contains(const TimeRange tr) const {
        return tr.start >= start && tr.end <= end;
    }

    bool operator==(const TimeRange &other) const {
        return start == other.start && end == other.end;
    }

    bool operator!=(const TimeRange &other) const {
        return start != other.start || end != other.end;
    }
};

/// @brief A stopwatch for measuring elapsed time using a monotonic clock.
/// This class provides a simple interface for timing code execution and returns
/// results as TimeSpan for consistency with other time-related utilities.
class Stopwatch {
    std::chrono::steady_clock::time_point start_;

public:
    /// @brief Constructs a Stopwatch and starts timing immediately.
    Stopwatch(): start_(std::chrono::steady_clock::now()) {}

    /// @brief Returns the elapsed time since construction or last reset.
    [[nodiscard]] TimeSpan elapsed() const {
        const auto now = std::chrono::steady_clock::now();
        const auto ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
                            now - this->start_
        )
                            .count();
        return TimeSpan(ns);
    }

    /// @brief Resets the stopwatch to start timing from now.
    void reset() { this->start_ = std::chrono::steady_clock::now(); }
};

class Rate {
    float value;

public:
    [[nodiscard]] float hz() const { return this->value; }

    Rate() = default;

    explicit Rate(const float i): value(i) {}

    explicit Rate(const int i): value(static_cast<float>(i)) {}

    explicit Rate(const double i): value(static_cast<float>(i)) {}

    explicit Rate(const TimeSpan period):
        value(static_cast<float>(1 / period.seconds())) {}

    bool operator==(const Rate &other) const { return value == other.value; }

    bool operator!=(const Rate &other) const { return value != other.value; }

    bool operator<(const Rate &other) const { return value < other.value; }

    bool operator>(const Rate &other) const { return value > other.value; }

    bool operator<=(const Rate &other) const { return value <= other.value; }

    bool operator>=(const Rate &other) const { return value >= other.value; }

    Rate operator+(const Rate &other) const { return Rate(value + other.value); }

    Rate operator-(const Rate &other) const { return Rate(value - other.value); }

    Rate operator*(const Rate &other) const { return Rate(value * other.value); }

    size_t operator/(const Rate &other) const { return value / other.value; }

    friend Rate operator+(const float &lhs, const Rate &rhs) {
        return Rate(lhs + rhs.value);
    }
    Rate operator+(const float &other) const { return Rate(value + other); }

    friend Rate operator-(const float &lhs, const Rate &rhs) {
        return Rate(lhs - rhs.value);
    }

    Rate operator-(const float &other) const { return Rate(value - other); }

    friend Rate operator*(const float &lhs, const Rate &rhs) {
        return Rate(lhs * rhs.value);
    }

    Rate operator*(const float &other) const { return Rate(value * other); }

    Rate operator/(const float &other) const { return Rate(value / other); }

    Rate operator/(const int &other) const {
        return Rate(value / static_cast<float>(other));
    }

    Rate operator/(const size_t &other) const {
        return Rate(value / static_cast<float>(other));
    }

    [[nodiscard]] TimeSpan period() const {
        return TimeSpan(std::llround(static_cast<double>(_priv::SECOND) / value));
    }

    friend std::ostream &operator<<(std::ostream &os, const Rate &r) {
        os << r.value << " Hz";
        return os;
    }
};

/// @brief a single hertz
inline const auto HERTZ = Rate(1);
/// @brief a single kilohertz
inline const Rate KILOHERTZ = 1000 * HERTZ;
/// @brief a single megahertz
inline const Rate MEGAHERTZ = 1000 * KILOHERTZ;
/// @brief a single nanosecond.
inline const auto NANOSECOND = TimeSpan(1);
/// @brief a single microsecond.
inline const TimeSpan MICROSECOND = NANOSECOND * 1000;
/// @brief a single millisecond.
inline const TimeSpan MILLISECOND = MICROSECOND * 1000;
/// @brief a single second.
inline const TimeSpan SECOND = MILLISECOND * 1000;
/// @brief a single minute.
inline const TimeSpan MINUTE = SECOND * 60;
/// @brief a single hour.
inline const TimeSpan HOUR = MINUTE * 60;
/// @brief a single day.
inline const TimeSpan DAY = HOUR * 24;

#define ASSERT_TYPE_SIZE(type, size)                                                   \
    static_assert(                                                                     \
        sizeof(type) == size,                                                          \
        "synnax only supports compilation environments with " #size " bit " #type "s"  \
    )

// Do a check to ensure that our compilation environment uses sizes that match
// those expected by Synnax.
ASSERT_TYPE_SIZE(float, 4);
ASSERT_TYPE_SIZE(double, 8);
ASSERT_TYPE_SIZE(int64_t, 8);
ASSERT_TYPE_SIZE(int32_t, 4);
ASSERT_TYPE_SIZE(int16_t, 2);
ASSERT_TYPE_SIZE(int8_t, 1);
ASSERT_TYPE_SIZE(uint64_t, 8);
ASSERT_TYPE_SIZE(uint32_t, 4);
ASSERT_TYPE_SIZE(uint16_t, 2);
ASSERT_TYPE_SIZE(uint8_t, 1);

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
    TimeStamp, // TIMESTAMP
    std::string // STRING
    >;

template<typename T>
[[nodiscard]] T cast(const SampleValue &value) {
    if (std::holds_alternative<T>(value)) return std::get<T>(value);
    if constexpr (std::is_same_v<T, std::string>) {
        return std::visit(
            []<typename IT>(IT &&arg) -> std::string {
                if constexpr (std::is_same_v<std::decay_t<IT>, std::string>)
                    return arg;
                else if constexpr (std::is_same_v<std::decay_t<IT>, TimeStamp>)
                    return std::to_string(arg.nanoseconds());
                else
                    return std::to_string(arg);
            },
            value
        );
    }
    if constexpr (std::is_same_v<T, TimeStamp>) {
        return std::visit(
            []<typename IT>(IT &&arg) -> TimeStamp {
                if constexpr (std::is_arithmetic_v<std::decay_t<IT>>) {
                    return TimeStamp(static_cast<std::int64_t>(arg));
                } else if constexpr (std::is_same_v<std::decay_t<IT>, std::string>) {
                    try {
                        return TimeStamp(std::stoll(arg));
                    } catch (...) {
                        throw std::runtime_error(
                            "failed to convert string to TimeStamp"
                        );
                    }
                }
                throw std::runtime_error("invalid type conversion to TimeStamp");
            },
            value
        );
    }
    if (std::holds_alternative<TimeStamp>(value)) {
        if constexpr (std::is_arithmetic_v<T>)
            return static_cast<T>(std::get<TimeStamp>(value).nanoseconds());
    }
    if (std::holds_alternative<std::string>(value)) {
        const auto &str = std::get<std::string>(value);
        if constexpr (std::is_arithmetic_v<T>) {
            try {
                if constexpr (std::is_floating_point_v<T>)
                    return static_cast<T>(std::stod(str));
                else if constexpr (std::is_signed_v<T>)
                    return static_cast<T>(std::stoll(str));
                else
                    return static_cast<T>(std::stoull(str));
            } catch (...) {
                throw std::runtime_error("failed to convert string to numeric type");
            }
        }
    }
    return std::visit(
        []<typename IT>(IT &&arg) -> T {
            if constexpr (std::is_arithmetic_v<T> &&
                          std::is_arithmetic_v<std::decay_t<IT>>)
                return static_cast<T>(arg);
            throw std::runtime_error("invalid type conversion");
        },
        value
    );
}

[[nodiscard]] inline void *cast_to_void_ptr(const SampleValue &value) {
    return std::visit(
        []<typename T>(const T &arg) -> void * {
            if constexpr (std::is_same_v<T, std::string>)
                return const_cast<void *>(static_cast<const void *>(arg.data()));
            else
                return const_cast<void *>(static_cast<const void *>(&arg));
        },
        value
    );
}

[[nodiscard]] inline std::string to_string(const SampleValue &value) {
    return cast<std::string>(value);
}

/// @brief Converts a google::protobuf::Value to SampleValue.
/// @param v The protobuf Value to convert.
/// @returns The SampleValue representation.
/// @note Struct and list values are not supported and will return a default double(0).
[[nodiscard]] inline SampleValue from_proto(const google::protobuf::Value &v) {
    switch (v.kind_case()) {
        case google::protobuf::Value::kNumberValue:
            return v.number_value();
        case google::protobuf::Value::kStringValue:
            return v.string_value();
        case google::protobuf::Value::kBoolValue:
            return v.bool_value() ? static_cast<uint8_t>(1) : static_cast<uint8_t>(0);
        case google::protobuf::Value::kNullValue:
        case google::protobuf::Value::kStructValue:
        case google::protobuf::Value::kListValue:
        case google::protobuf::Value::KIND_NOT_SET:
        default:
            return 0.0;
    }
}

/// @brief Converts a SampleValue to google::protobuf::Value.
/// @param sv The SampleValue to convert.
/// @param v Pointer to the protobuf Value to populate.
inline void to_proto(const SampleValue &sv, google::protobuf::Value *v) {
    std::visit(
        [v]<typename T>(const T &val) {
            if constexpr (std::is_same_v<T, std::string>)
                v->set_string_value(val);
            else if constexpr (std::is_same_v<T, TimeStamp>)
                v->set_number_value(static_cast<double>(val.nanoseconds()));
            else
                v->set_number_value(static_cast<double>(val));
        },
        sv
    );
}

using NowFunc = std::function<TimeStamp()>;

namespace _priv {
const std::string UNKNOWN_T;
const std::string FLOAT64_T = "float64";
const std::string FLOAT32_T = "float32";
const std::string INT8_T = "int8";
const std::string INT16_T = "int16";
const std::string INT32_T = "int32";
const std::string INT64_T = "int64";
const std::string TIMESTAMP_T = "timestamp";
const std::string UINT8_T = "uint8";
const std::string UINT16_T = "uint16";
const std::string UINT32_T = "uint32";
const std::string UINT64_T = "uint64";
const std::string UUID_T = "uuid";
const std::string STRING_T = "string";
const std::string JSON_T = "json";
const std::vector VARIABLE_TYPES = {JSON_T, STRING_T};
}

/// @brief Holds the name and properties of a datatype.
class DataType {
    /// @brief Holds the id of the data type
    std::string value;
    size_t density_ = 0;

public:
    DataType() = default;

    /// @brief constructs a data type from the provided string.
    explicit DataType(std::string data_type): value(std::move(data_type)) {
        const auto cached_density_iter = DENSITIES.find(value);
        if (cached_density_iter != DENSITIES.end())
            this->density_ = cached_density_iter->second;
    }

    /// @brief Infers the data type from a given C++ type along with an optional
    /// override.
    /// @returns the inferred data type if the override is not provided, otherwise
    /// returns the override.
    template<typename T>
    DataType static infer(const DataType &override = DataType(_priv::UNKNOWN_T)) {
        if (override != _priv::UNKNOWN_T) return override;
        const auto type_index = std::type_index(typeid(T));
        const auto it = TYPE_INDEXES.find(type_index);
        if (it != TYPE_INDEXES.end()) return DataType(it->second);
        throw std::runtime_error("cannot infer data type from unknown C++ type");
    }

    /// @brief Infers the data type from a given sample value.
    DataType static infer(const SampleValue &value) {
        return std::visit(
            []<typename IT>(IT &&) -> DataType {
                using T = std::decay_t<IT>;
                if constexpr (std::is_same_v<T, double>)
                    return DataType(_priv::FLOAT64_T);
                if constexpr (std::is_same_v<T, float>)
                    return DataType(_priv::FLOAT32_T);
                if constexpr (std::is_same_v<T, int64_t>)
                    return DataType(_priv::INT64_T);
                if constexpr (std::is_same_v<T, int32_t>)
                    return DataType(_priv::INT32_T);
                if constexpr (std::is_same_v<T, int16_t>)
                    return DataType(_priv::INT16_T);
                if constexpr (std::is_same_v<T, int8_t>) return DataType(_priv::INT8_T);
                if constexpr (std::is_same_v<T, uint64_t>)
                    return DataType(_priv::UINT64_T);
                if constexpr (std::is_same_v<T, uint32_t>)
                    return DataType(_priv::UINT32_T);
                if constexpr (std::is_same_v<T, uint16_t>)
                    return DataType(_priv::UINT16_T);
                if constexpr (std::is_same_v<T, uint8_t>)
                    return DataType(_priv::UINT8_T);
                if constexpr (std::is_same_v<T, TimeStamp>)
                    return DataType(_priv::TIMESTAMP_T);
                if constexpr (std::is_same_v<T, std::string>)
                    return DataType(_priv::STRING_T);
                return DataType(_priv::UNKNOWN_T);
            },
            value
        );
    }

    /// @property Gets type name.
    [[nodiscard]] std::string name() const { return value; }

    /// @property how many bytes in memory the data type holds.
    [[nodiscard]] size_t density() const { return this->density_; }

    [[nodiscard]] bool is_variable() const {
        return this->matches(_priv::VARIABLE_TYPES);
    }

    /// @brief Checks if this data type matches any of the provided data types.
    /// @param others Vector of data types to compare against
    /// @returns true if this data type matches any in the vector, false otherwise
    [[nodiscard]] bool matches(const std::vector<DataType> &others) const {
        for (const auto &other: others)
            if (other == *this) return true;
        return false;
    }

    [[nodiscard]] bool matches(const std::vector<std::string> &others) const {
        for (const auto &other: others)
            if (other == this->value) return true;
        return false;
    }

    /// @brief Casts a numeric sample value to the type corresponding to this data
    /// type
    /// @param value The numeric sample value to cast
    /// @returns A new numeric sample value of the appropriate type
    /// @throws std::runtime_error if the data type is not numeric
    [[nodiscard]] SampleValue cast(const SampleValue &value) const {
        if (*this == _priv::FLOAT64_T) return cast<double>(value);
        if (*this == _priv::FLOAT32_T) return cast<float>(value);
        if (*this == _priv::INT64_T) return cast<int64_t>(value);
        if (*this == _priv::INT32_T) return cast<int32_t>(value);
        if (*this == _priv::INT16_T) return cast<int16_t>(value);
        if (*this == _priv::INT8_T) return cast<int8_t>(value);
        if (*this == _priv::UINT64_T) return cast<uint64_t>(value);
        if (*this == _priv::UINT32_T) return cast<uint32_t>(value);
        if (*this == _priv::UINT16_T) return cast<uint16_t>(value);
        if (*this == _priv::UINT8_T) return cast<uint8_t>(value);
        if (*this == _priv::TIMESTAMP_T) return cast<TimeStamp>(value);
        if (this->is_variable()) return cast<std::string>(value);
        throw std::runtime_error(
            "cannot cast sample value to unknown data type " + this->value
        );
    }

    /// @brief Casts a void pointer to a sample value of the appropriate type.
    /// @param value The void pointer to cast
    /// @param value_type - The data type of the value
    /// @returns A new sample value of the appropriate type
    /// @throws std::runtime_error if the data type is not numeric
    SampleValue cast(const void *value, const DataType &value_type) const {
        if (value_type == _priv::FLOAT64_T)
            return this->cast(*static_cast<const double *>(value));
        if (value_type == _priv::FLOAT32_T)
            return this->cast(*static_cast<const float *>(value));
        if (value_type == _priv::INT64_T)
            return this->cast(*static_cast<const int64_t *>(value));
        if (value_type == _priv::INT32_T)
            return this->cast(*static_cast<const int32_t *>(value));
        if (value_type == _priv::INT16_T)
            return this->cast(*static_cast<const int16_t *>(value));
        if (value_type == _priv::INT8_T)
            return this->cast(*static_cast<const int8_t *>(value));
        if (value_type == _priv::UINT8_T)
            return this->cast(*static_cast<const uint8_t *>(value));
        if (value_type == _priv::UINT16_T)
            return this->cast(*static_cast<const uint16_t *>(value));
        if (value_type == _priv::UINT32_T)
            return this->cast(*static_cast<const uint32_t *>(value));
        if (value_type == _priv::UINT64_T)
            return this->cast(*static_cast<const uint64_t *>(value));
        if (value_type == _priv::TIMESTAMP_T)
            return this->cast(*static_cast<const TimeStamp *>(value));
        if (value_type == _priv::STRING_T)
            return this->cast(*static_cast<const std::string *>(value));
        if (value_type == _priv::JSON_T)
            return this->cast(*static_cast<const std::string *>(value));
        throw std::runtime_error(
            "cannot cast sample value to unknown data type " + this->value
        );
    }

    bool operator==(const DataType &other) const { return value == other.value; }

    bool operator==(const std::string &other) const { return value == other; }

    bool operator!=(const DataType &other) const { return value != other.value; }

    bool operator!=(const std::string &other) const { return value != other; }

    bool operator<(const DataType &other) const { return value < other.value; }

    bool operator<(const std::string &other) const { return value < other; }

    bool operator>(const DataType &other) const { return value > other.value; }

    bool operator>(const std::string &other) const { return value > other; }

    bool operator<=(const DataType &other) const { return value <= other.value; }

    bool operator<=(const std::string &other) const { return value <= other; }

    bool operator>=(const DataType &other) const { return value >= other.value; }

    bool operator>=(const std::string &other) const { return value >= other; }

    /// @brief Concatenates this DataType with another DataType
    /// @param other The DataType to concatenate with
    /// @returns A string with the concatenated values
    std::string operator+(const DataType &other) const { return value + other.value; }

    /// @brief Concatenates this DataType with a string
    /// @param other The string to concatenate with
    /// @returns A string with the concatenated values
    std::string operator+(const std::string &other) const { return value + other; }

    /// @brief Friend operator to allow string + DataType concatenation
    /// @param lhs The string on the left side of the + operator
    /// @param rhs The DataType on the right side of the + operator
    /// @returns A string with the concatenated values
    friend std::string operator+(const std::string &lhs, const DataType &rhs) {
        return lhs + rhs.value;
    }

    friend std::ostream &operator<<(std::ostream &os, const DataType &dt) {
        os << dt.value;
        return os;
    }

    // Add hash support for DataType
    friend struct std::hash<DataType>;

private:
    inline static std::unordered_map<std::string, size_t> DENSITIES = {
        {_priv::FLOAT64_T, 8},
        {_priv::FLOAT32_T, 4},
        {_priv::INT8_T, 1},
        {_priv::INT16_T, 2},
        {_priv::INT32_T, 4},
        {_priv::INT64_T, 8},
        {_priv::UINT8_T, 1},
        {_priv::UINT16_T, 2},
        {_priv::UINT32_T, 4},
        {_priv::UINT64_T, 8},
        {_priv::TIMESTAMP_T, 8},
        {_priv::UUID_T, 16},
        {_priv::STRING_T, 0},
        {_priv::JSON_T, 0},
    };

    /// @brief stores a map of C++ type indexes to their corresponding synnax data
    /// type identifiers.
    inline static std::unordered_map<std::type_index, std::string> TYPE_INDEXES = {
        {std::type_index(typeid(float)), _priv::FLOAT32_T},
        {std::type_index(typeid(double)), _priv::FLOAT64_T},
        {std::type_index(typeid(char)), _priv::INT8_T},
        {std::type_index(typeid(std::int8_t)), _priv::INT8_T},
        {std::type_index(typeid(short)), _priv::INT16_T},
        {std::type_index(typeid(std::int16_t)), _priv::INT16_T},
        {std::type_index(typeid(int)), _priv::INT32_T},
        {std::type_index(typeid(std::int32_t)), _priv::INT32_T},
        {std::type_index(typeid(long)),
         sizeof(long) == 8 ? _priv::INT64_T : _priv::INT32_T},
        {std::type_index(typeid(long long)), _priv::INT64_T},
        {std::type_index(typeid(std::int64_t)), _priv::INT64_T},
        {std::type_index(typeid(unsigned char)), _priv::UINT8_T},
        {std::type_index(typeid(std::uint8_t)), _priv::UINT8_T},
        {std::type_index(typeid(unsigned short)), _priv::UINT16_T},
        {std::type_index(typeid(std::uint16_t)), _priv::UINT16_T},
        {std::type_index(typeid(unsigned int)), _priv::UINT32_T},
        {std::type_index(typeid(std::uint32_t)), _priv::UINT32_T},
        {std::type_index(typeid(unsigned long)),
         sizeof(unsigned long) == 8 ? _priv::UINT64_T : _priv::UINT32_T},
        {std::type_index(typeid(unsigned long long)), _priv::UINT64_T},
        {std::type_index(typeid(std::uint64_t)), _priv::UINT64_T},
        {std::type_index(typeid(std::string)), _priv::STRING_T},
        {std::type_index(typeid(TimeStamp)), _priv::TIMESTAMP_T},
    };
};

// Alignment is two array index values that can be used to represent
// the location of a sample within an array of arrays. For example, if you have two
// arrays that have 50 elements each, and you want the 15th element of the second array,
// you would use NewAlignment(1, 15). The first index is called the 'domain index' and
// the second index is called the 'sample index'. The domain index is the index of the
// array, and the sample index is the index of the sample within that array.
//
// You may think a better design is to just use a single number that overflows the
// arrays before it, i.e., the value of our previous example would be 50 + 14 = 64.
// However, this requires us to know the size of all arrays, which is not always
// possible.
//
// While not as meaningful as a single number, Alignment is a uint64 that guarantees
// that a larger value is, in fact, 'positionally' after a smaller value. This is useful
// for ordering samples correctly.
class Alignment {
    std::uint64_t value;

public:
    explicit Alignment(const std::uint64_t value = 0): value(value) {}

    Alignment(const std::uint32_t domain_index, const std::uint32_t sample_index):
        value(static_cast<std::uint64_t>(domain_index) << 32 | sample_index) {}

    /// @returns the value of the Alignment as a uint64_t.
    [[nodiscard]] std::uint64_t uint64() const { return this->value; }

    /// @returns the domain index of the Alignment. This is the index in
    /// the array of arrays.
    [[nodiscard]] std::uint32_t domain_index() const {
        return static_cast<uint32_t>(this->value >> 32);
    }

    /// @returns the sample index of the Alignment. This is the index
    /// inside a particular array.
    [[nodiscard]] std::uint32_t sample_index() const {
        return static_cast<uint32_t>(this->value);
    }

    bool operator==(const Alignment &other) const { return value == other.value; }

    bool operator!=(const Alignment &other) const { return value != other.value; }

    bool operator==(const std::uint64_t &other) const { return value == other; }

    bool operator!=(const std::uint64_t &other) const { return value != other; }
};

/// Note for future editors of these types, using `inline const` is dangerous as it
/// causes problems with density lookups.

/// @brief identifier for an unknown data type in a Synnax cluster.
const DataType UNKNOWN_T(_priv::UNKNOWN_T);
/// @brief identifier for a fixed-size float64 data type in a Synnax cluster.
const DataType FLOAT64_T(_priv::FLOAT64_T);
/// @brief identifier for a fixed-size float32 data type in a Synnax cluster.
const DataType FLOAT32_T(_priv::FLOAT32_T);
/// @brief identifier for a fixed-size int8 data type in a Synnax cluster.
const DataType INT8_T(_priv::INT8_T);
/// @brief identifier for a fixed-size int16 data type in a Synnax cluster.
const DataType INT16_T(_priv::INT16_T);
/// @brief identifier for a fixed-size int32 data type in a Synnax cluster.
const DataType INT32_T(_priv::INT32_T);
/// @brief identifier for a fixed-size int64 data type in a Synnax cluster.
const DataType INT64_T(_priv::INT64_T);
/// @brief identifier for a fixed-size timestamp data type in a Synnax cluster.
const DataType TIMESTAMP_T(_priv::TIMESTAMP_T);
/// @brief identifier for a fixed-size uint8 data type in a Synnax cluster.
const DataType UINT8_T(_priv::UINT8_T);
/// @brief identifier for a fixed-size uint16 data type in a Synnax cluster.
const DataType UINT16_T(_priv::UINT16_T);
/// @brief identifier for a fixed-size uint32 data type in a Synnax cluster.
const DataType UINT32_T(_priv::UINT32_T);
/// @brief identifier for a fixed-size uint64 data type in a Synnax cluster.
const DataType UINT64_T(_priv::UINT64_T);
/// @brief identifier for a fixed-size UUID data type in a Synnax cluster (16
/// bytes).
const DataType UUID_T(_priv::UUID_T);
/// @brief identifier for a newline separated, variable-length string data type in a
/// Synnax cluster. Note that variable-length data types have reduced performance
/// and restricted use within a Synnax cluster.
const DataType STRING_T(_priv::STRING_T);
/// @brief identifier for a newline separated, stringified JSON data type in a
/// Synnax cluster. Note that variable-length data types have reduced performance
/// and restricted use within a Synnax cluster.
const DataType JSON_T(_priv::JSON_T);
}

// Add hash specialization in std namespace
template<>
struct std::hash<x::telem::DataType> {
    size_t operator()(const x::telem::DataType &dt) const noexcept {
        return hash<string>()(dt.value);
    }
};
