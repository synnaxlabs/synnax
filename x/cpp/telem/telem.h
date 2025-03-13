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
#include <chrono>
#include <cstdint>
#include <string>
#include <iostream>
#include <typeindex>
#include <unordered_map>
#include <vector>
#include <variant>

namespace telem {
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
    TimeSpan(): value(0) {
    }

    /// @brief returns the absolute value of the timespan.
    [[nodiscard]] TimeSpan abs() const {
        return TimeSpan(std::abs(value));
    }

    /// @brief Constructs a timespan from the given int64, interpreting it as a
    /// nanosecond-precision timespan.
    explicit TimeSpan(const std::int64_t i) : value(i) {
    }

    /// @brief returns a new TimeSpan from the given chrono duration.
    explicit TimeSpan(const std::chrono::duration<std::int64_t, std::nano> &duration) :
        value(duration.count()) {
    }

    /// @brief returns the number of nanoseconds in the timespan.
    [[nodiscard]] std::int64_t nanoseconds() const { return this->value; }

    ///////////////////////////////////// COMPARISON /////////////////////////////////////

    bool operator==(const TimeSpan &other) const { return value == other.value; }

    bool operator==(const std::int64_t &other) const { return value == other; }

    bool operator!=(const std::int64_t &other) const { return value != other; }

    bool operator!=(const TimeSpan &other) const { return value != other.value; }

    bool operator<(const TimeSpan &other) const { return value < other.value; }

    bool operator>(const TimeSpan &other) const { return value > other.value; }

    bool operator<=(const TimeSpan &other) const { return value <= other.value; }

    bool operator>=(const TimeSpan &other) const { return value >= other.value; }

    //////////////////////////////////// ADDITION /////////////////////////////////////

    TimeSpan operator+(const TimeSpan &other) const {
        return TimeSpan(value + other.value);
    }

    TimeSpan operator+=(const TimeSpan &other) {
        this->value += other.value;
        return *this;
    }

    TimeSpan operator+(const std::int64_t &other) const {
        return TimeSpan(value + other);
    }

    friend TimeSpan operator+(const long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs + rhs.value);
    }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeSpan &other) const {
        return TimeSpan(value - other.value);
    }

    friend TimeSpan operator-(const long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    TimeSpan operator-(const long long &other) const {
        return TimeSpan(value - other);
    }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    TimeSpan operator*(const TimeSpan &other) const {
        return TimeSpan(value * other.value);
    }

    TimeSpan operator*(const size_t &other) const {
        return TimeSpan(value * static_cast<std::int64_t>(other));
    }

    TimeSpan operator*(const float &other) const {
        return TimeSpan(static_cast<std::int64_t>(static_cast<double>(value) * other));
    }

    friend TimeSpan operator*(const long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(rhs.value * lhs);
    }

    TimeSpan operator*(const long long &other) const {
        return TimeSpan(other * value);
    }

    TimeSpan operator*(const int &other) const { return TimeSpan(value * other); }

    TimeSpan operator*(const unsigned int &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const double &other) const {
        return TimeSpan(static_cast<std::int64_t>(static_cast<double>(value) * other));
    }

    TimeSpan operator*(const long &other) const { return TimeSpan(value * other); }

    ////////////////////////////////// DIVISION /////////////////////////////////

    TimeSpan operator/(const TimeSpan &other) const {
        return TimeSpan(value / other.value);
    }

    friend TimeSpan operator/(const long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs / rhs.value);
    }

    TimeSpan operator/(const long long &other) const {
        return TimeSpan(value / other);
    }

    ////////////////////////////////// MODULO /////////////////////////////////

    TimeSpan operator%(const TimeSpan &other) const {
        return TimeSpan(value % other.value);
    }

    friend TimeSpan operator%(const long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs % rhs.value);
    }

    TimeSpan operator%(const long long &other) const {
        return TimeSpan(value % other);
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

    /// @brief returns the exact number of microseconds in the timespan as a
    /// double precision floating point value.
    [[nodiscard]] double microseconds() const {
        return static_cast<double>(value) / _priv::MICROSECOND;
    }


    ////////////////////////////////// OSTREAM /////////////////////////////////

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
        const auto microseconds = (total_microseconds.value - total_milliseconds.value)
                                  / _priv::MICROSECOND;
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
};

/// @brief represents a 64-bit nanosecond-precision, UNIX Epoch UTC timestamp.
class TimeStamp {
    /// @property value holds the internal, primitive value of the timestamp.
    std::int64_t value;

public:
    TimeStamp() = default;

    /// @brief Constructs a timestamp from the given interpreting it as a nanosecond-precision UTC
    /// timestamp.
    explicit TimeStamp(const std::int64_t value) : value(value) {
    }

    /// @brief returns the number of nanoseconds in the timestamp.
    [[nodiscard]] std::int64_t nanoseconds() const { return this->value; }

    /// @brief interprets the given TimeSpan as a TimeStamp.
    explicit TimeStamp(const TimeSpan ts) : value(ts.nanoseconds()) {
    }

    TimeStamp static now() {
        // note that on some machines, hig-res clock refs system_clock and on others it references
        // steady_clock. This could create a problem so we should probably use system_clock.
        return TimeStamp(std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count());
    }

    TimeStamp static midpoint(const TimeStamp &start, const TimeStamp &end) {
        return TimeStamp(start + (end - start) / 2);
    }

    ///////////////////////////////////// COMPARISON /////////////////////////////////////

    bool operator==(const TimeStamp &other) const { return value == other.value; }

    bool operator!=(const TimeStamp &other) const { return value != other.value; }

    bool operator<(const TimeStamp &other) const { return value < other.value; }

    bool operator>(const TimeStamp &other) const { return value > other.value; }

    bool operator<=(const TimeStamp &other) const { return value <= other.value; }

    bool operator>=(const TimeStamp &other) const { return value >= other.value; }

    bool operator==(const int &other) const { return value == other; }

    bool operator !=(const int &other) const { return value != other; }

    //////////////////////////////////// ADDITION /////////////////////////////////////

    TimeStamp operator+(const TimeStamp &other) const {
        return TimeStamp(value + other.value);
    }

    friend TimeStamp
    operator+(const long long &lhs, const TimeStamp &rhs) {
        return TimeStamp(lhs + rhs.value);
    }

    TimeStamp operator+(const TimeSpan &other) const {
        return TimeStamp(value + other.nanoseconds());
    }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeStamp &other) const {
        return TimeSpan(value - other.value);
    }

    friend TimeSpan operator-(const long long &lhs, const TimeStamp &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    TimeSpan operator-(const TimeSpan &other) const {
        return TimeSpan(value - other.nanoseconds());
    }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

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
};

class TimeRange {
public:
    TimeStamp start;
    TimeStamp end;

    TimeRange() = default;

    /// @brief constructs a TimeRange from the given start and end timestamps.
    TimeRange(const TimeStamp start, const TimeStamp end) : start(start), end(end) {
    }

    TimeRange(const std::int64_t start, const std::int64_t end) : start(start),
        end(end) {
    }

    /// @brief returns true if the given timestamp is within the range, start inclusive,
    /// end exclusive.
    [[nodiscard]] bool contains(const TimeStamp time) const {
        return start <= time && time < end;
    }

    /// @brief returns true if the TimeRange contains the given TimeRange. If the two
    /// time ranges are equal, returns true. In this case, the two time ranges contain
    /// each other.
    [[nodiscard]] bool contains(const TimeRange tr) const {
        return tr.start >= start && tr.end <= end;
    }

    bool operator==(const TimeRange &other) const {
        return start == other.start && end == other.end;
    }
};

class Rate {
    float value;

public:
    [[nodiscard]] float hz() const { return this->value; }

    explicit Rate(const float i) : value(i) {
    }

    explicit Rate(const int i) : value(static_cast<float>(i)) {
    }

    explicit Rate(const double i) : value(static_cast<float>(i)) {
    }

    explicit Rate(const TimeSpan period) : value(
        static_cast<float>(1 / period.seconds())) {
    }

    Rate() = default;

    //////////////////////////////////// COMPARISON ///////////////////////////////////

    bool operator==(const Rate &other) const { return value == other.value; }

    bool operator!=(const Rate &other) const { return value != other.value; }

    bool operator<(const Rate &other) const { return value < other.value; }

    bool operator>(const Rate &other) const { return value > other.value; }

    bool operator<=(const Rate &other) const { return value <= other.value; }

    bool operator>=(const Rate &other) const { return value >= other.value; }

    //////////////////////////////////// ADDITION ///////////////////////////////////

    Rate operator+(const Rate &other) const { return Rate(value + other.value); }

    friend Rate operator+(const float &lhs, const Rate &rhs) {
        return Rate(lhs + rhs.value);
    }

    Rate operator+(const float &other) const { return Rate(value + other); }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    Rate operator-(const Rate &other) const { return Rate(value - other.value); }

    friend Rate operator-(const float &lhs, const Rate &rhs) {
        return Rate(lhs - rhs.value);
    }

    Rate operator-(const float &other) const { return Rate(value - other); }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    Rate operator*(const Rate &other) const { return Rate(value * other.value); }

    friend Rate operator*(const float &lhs, const Rate &rhs) {
        return Rate(lhs * rhs.value);
    }

    Rate operator*(const float &other) const { return Rate(value * other); }

    [[nodiscard]] TimeSpan period() const {
        return TimeSpan(
            static_cast<std::int64_t>(1 / value * static_cast<double>(_priv::SECOND)));
    }

    ////////////////////////////////// DIVISION /////////////////////////////////

    size_t operator/(const Rate &other) const { return value / other.value; }

    Rate operator/(const float &other) const { return Rate(value / other); }

    Rate operator/(const int &other) const {
        return Rate(value / static_cast<float>(other));
    }

    Rate operator/(const double &other) const { return Rate(value / other); }

    Rate operator/(const size_t &other) const {
        return Rate(value / static_cast<float>(other));
    }
};

/// @brief a single hertz
inline const auto HZ = Rate(1);
/// @brief a single kilohertz
inline const Rate KHZ = 1000 * HZ;
/// @brief a single megahertz
inline const Rate MHZ = 1000 * KHZ;
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

#define ASSERT_TYPE_SIZE(type, size) \
    static_assert(sizeof(type) == size, "synnax only supports compilation environments with " #size " bit " #type "s")

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

using NumericSampleValue = std::variant<
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
    TimeStamp // TIMESTAMP
>;

[[nodiscard]] inline NumericSampleValue narrow_numeric(const SampleValue &value) {
    if (std::holds_alternative<std::string>(value))
        throw std::runtime_error("cannot narrow non-numeric sample value");

    return std::visit([]<typename T>(T &&arg) -> NumericSampleValue {
        if constexpr (std::is_same_v<std::decay_t<T>, TimeStamp>)
            return arg;
        else if constexpr (std::is_same_v<std::decay_t<T>, std::string>)
            throw std::runtime_error("cannot narrow string to numeric sample value");
        else
            return arg;
    }, value);
}

inline SampleValue widen_numeric(const NumericSampleValue &value) {
    return std::visit([]<typename T>(T &&arg) -> SampleValue {
        if constexpr (std::is_same_v<std::decay_t<T>, TimeStamp>)
            return arg;
        else
            return arg;
    }, value);
}

/// @brief Subtracts the second NumericSampleValue from the first
/// @param lhs The left-hand side operand
/// @param rhs The right-hand side operand to subtract
/// @returns A new NumericSampleValue containing the result of the subtraction
/// @throws std::runtime_error if the types are incompatible for subtraction
[[nodiscard]] inline NumericSampleValue subtract(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return std::visit([&rhs]<typename LHS>(LHS &&lhs_val) -> NumericSampleValue {
        return std::visit([&lhs_val]<typename RHS>(RHS &&rhs_val) -> NumericSampleValue {
            using LhsType = std::decay_t<LHS>;
            using RhsType = std::decay_t<RHS>;
            if constexpr (std::is_same_v<LhsType, TimeStamp>) {
                if constexpr (std::is_same_v<RhsType, TimeStamp>)
                    return (lhs_val - rhs_val).nanoseconds();
                else if constexpr (std::is_arithmetic_v<RhsType>)
                    return TimeStamp(lhs_val.nanoseconds() - static_cast<int64_t>(rhs_val));
            } else if constexpr (std::is_same_v<RhsType, TimeStamp>) {
                if constexpr (std::is_arithmetic_v<LhsType>)
                    return TimeStamp(static_cast<int64_t>(lhs_val) - rhs_val.nanoseconds());
            } else if constexpr (std::is_arithmetic_v<LhsType> && std::is_arithmetic_v<RhsType>) {
                if constexpr (std::is_same_v<LhsType, RhsType>)
                    return static_cast<LhsType>(lhs_val - rhs_val);
                using ResultType = std::common_type_t<LhsType, RhsType>;
                return static_cast<ResultType>(lhs_val) - static_cast<ResultType>(rhs_val);
            }
            throw std::runtime_error("incompatible types for subtraction");
        }, rhs);
    }, lhs);
}

/// @brief Adds two NumericSampleValues together
/// @param lhs The left-hand side operand
/// @param rhs The right-hand side operand to add
/// @returns A new NumericSampleValue containing the result of the addition
/// @throws std::runtime_error if the types are incompatible for addition
[[nodiscard]] inline NumericSampleValue add(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return std::visit([&rhs]<typename LHS>(LHS &&lhs_val) -> NumericSampleValue {
        return std::visit([&lhs_val]<typename RHS>(RHS &&rhs_val) -> NumericSampleValue {
            using LhsType = std::decay_t<LHS>;
            using RhsType = std::decay_t<RHS>;
            if constexpr (std::is_same_v<LhsType, TimeStamp>) {
                if constexpr (std::is_same_v<RhsType, TimeStamp>)
                    return TimeStamp(lhs_val.nanoseconds() + rhs_val.nanoseconds());
                else if constexpr (std::is_arithmetic_v<RhsType>)
                    return TimeStamp(lhs_val.nanoseconds() + static_cast<int64_t>(rhs_val));
            } else if constexpr (std::is_same_v<RhsType, TimeStamp>) {
                if constexpr (std::is_arithmetic_v<LhsType>)
                    return TimeStamp(static_cast<int64_t>(lhs_val) + rhs_val.nanoseconds());
            } else if constexpr (std::is_arithmetic_v<LhsType> && std::is_arithmetic_v<RhsType>) {
                if constexpr (std::is_same_v<LhsType, RhsType>)
                    return static_cast<LhsType>(lhs_val + rhs_val);
                using ResultType = std::common_type_t<LhsType, RhsType>;
                return static_cast<ResultType>(lhs_val) + static_cast<ResultType>(rhs_val);
            }
            throw std::runtime_error("incompatible types for addition");
        }, rhs);
    }, lhs);
}

/// @brief Multiplies two NumericSampleValues together
/// @param lhs The left-hand side operand
/// @param rhs The right-hand side operand to multiply
/// @returns A new NumericSampleValue containing the result of the multiplication
/// @throws std::runtime_error if the types are incompatible for multiplication
[[nodiscard]] inline NumericSampleValue multiply(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return std::visit([&rhs]<typename LHS>(LHS &&lhs_val) -> NumericSampleValue {
        return std::visit([&lhs_val]<typename RHS>(RHS &&rhs_val) -> NumericSampleValue {
            using LhsType = std::decay_t<LHS>;
            using RhsType = std::decay_t<RHS>;
            if constexpr (std::is_same_v<LhsType, TimeStamp>) {
                if constexpr (std::is_arithmetic_v<RhsType>)
                    return TimeStamp(lhs_val.nanoseconds() * static_cast<int64_t>(rhs_val));
            } else if constexpr (std::is_same_v<RhsType, TimeStamp>) {
                if constexpr (std::is_arithmetic_v<LhsType>)
                    return TimeStamp(static_cast<int64_t>(lhs_val) * rhs_val.nanoseconds());
            } else if constexpr (std::is_arithmetic_v<LhsType> && std::is_arithmetic_v<RhsType>) {
                // For arithmetic types, if they're the same type, preserve that type
                if constexpr (std::is_same_v<LhsType, RhsType>)
                    return static_cast<LhsType>(lhs_val * rhs_val);
                using ResultType = std::common_type_t<LhsType, RhsType>;
                return static_cast<ResultType>(lhs_val) * static_cast<ResultType>(rhs_val);
            }
            throw std::runtime_error("incompatible types for multiplication");
        }, rhs);
    }, lhs);
}

/// @brief Divides the first NumericSampleValue by the second
/// @param lhs The left-hand side operand (dividend)
/// @param rhs The right-hand side operand (divisor)
/// @returns A new NumericSampleValue containing the result of the division
/// @throws std::runtime_error if the types are incompatible for division or if dividing by zero
[[nodiscard]] inline NumericSampleValue divide(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return std::visit([&rhs]<typename LHS>(LHS &&lhs_val) -> NumericSampleValue {
        return std::visit([&lhs_val]<typename RHS>(RHS &&rhs_val) -> NumericSampleValue {
            using LhsType = std::decay_t<LHS>;
            using RhsType = std::decay_t<RHS>;
            if constexpr (std::is_arithmetic_v<RhsType>)
                if (rhs_val == 0) throw std::runtime_error("division by zero");
            if constexpr (std::is_same_v<RhsType, TimeStamp>)
                if (rhs_val.nanoseconds() == 0) throw std::runtime_error("division by zero");
            if constexpr (std::is_same_v<LhsType, TimeStamp>) {
                if constexpr (std::is_same_v<RhsType, TimeStamp>)
                    return static_cast<double>(lhs_val.nanoseconds()) / static_cast<double>(rhs_val.nanoseconds());
                else if constexpr (std::is_arithmetic_v<RhsType>)
                    return TimeStamp(lhs_val.nanoseconds() / static_cast<int64_t>(rhs_val));
            } else if constexpr (std::is_arithmetic_v<LhsType>) {
                if constexpr (std::is_same_v<RhsType, TimeStamp>)
                    return static_cast<double>(lhs_val) / static_cast<double>(rhs_val.nanoseconds());
                else if constexpr (std::is_arithmetic_v<RhsType>) {
                    if constexpr (std::is_same_v<LhsType, RhsType>)
                        return static_cast<LhsType>(lhs_val / rhs_val);
                    using ResultType = std::common_type_t<LhsType, RhsType>;
                    return static_cast<ResultType>(lhs_val) / static_cast<ResultType>(rhs_val);
                }
            }
            throw std::runtime_error("incompatible types for division");
        }, rhs);
    }, lhs);
}

[[nodiscard]] inline NumericSampleValue operator+(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return add(lhs, rhs);
}

[[nodiscard]] inline NumericSampleValue operator-(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return subtract(lhs, rhs);
}

[[nodiscard]] inline NumericSampleValue operator*(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return multiply(lhs, rhs);
}

[[nodiscard]] inline NumericSampleValue operator/(const NumericSampleValue &lhs, const NumericSampleValue &rhs) {
    return divide(lhs, rhs);
}



template<typename T>
[[nodiscard]] T cast(const SampleValue &value) {
    if (std::holds_alternative<T>(value)) return std::get<T>(value);
    if constexpr (std::is_same_v<T, std::string>) {
        return std::visit([]<typename IT>(IT &&arg) -> std::string {
            if constexpr (std::is_same_v<std::decay_t<IT>, std::string>)
                return arg;
            else if constexpr (std::is_same_v<std::decay_t<IT>, TimeStamp>)
                return std::to_string(arg.nanoseconds());
            else
                return std::to_string(arg);
        }, value);
    }
    if constexpr (std::is_same_v<T, TimeStamp>) {
        return std::visit([]<typename IT>(IT &&arg) -> TimeStamp {
            if constexpr (std::is_arithmetic_v<std::decay_t<IT> >) {
                return TimeStamp(static_cast<std::int64_t>(arg));
            } else if constexpr (std::is_same_v<std::decay_t<IT>,
                std::string>) {
                try {
                    return TimeStamp(std::stoll(arg));
                } catch (...) {
                    throw std::runtime_error("failed to convert string to TimeStamp");
                }
            }
            throw std::runtime_error("invalid type conversion to TimeStamp");
        }, value);
    }
    if (std::holds_alternative<TimeStamp>(value))
        if constexpr (std::is_arithmetic_v<T>)
            return static_cast<T>(std::get<TimeStamp>(value).nanoseconds());
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
    return std::visit([]<typename IT>(IT &&arg) -> T {
        if constexpr (std::is_arithmetic_v<T> && std::is_arithmetic_v<std::decay_t<
                          IT> >)
            return static_cast<T>(arg);
        throw std::runtime_error("invalid type conversion");
    }, value);
}

[[nodiscard]] inline void *cast_to_void_ptr(const SampleValue &value) {
    return std::visit([]<typename T>(const T &arg) -> void *{
        if constexpr (std::is_same_v<T, std::string>)
            return const_cast<void *>(static_cast<const void *>(arg.data()));
        else
            return const_cast<void *>(static_cast<const void *>(&arg));
    }, value);
}



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
const std::string UINT128_T = "uint128";
const std::string UUID_T = "uuid";
const std::string STRING_T = "string";
const std::string JSON_T = "json";
const std::vector VARIABLE_TYPES = {JSON_T, STRING_T};
}

/// @brief Holds the name and properties of a datatype.
class DataType {
    /// @brief Holds the id of the data type
    std::string value;
public:
    DataType() = default;

    /// @brief constructs a data type from the provided string.
    explicit DataType(std::string data_type): value(std::move(data_type)) {
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
        return std::visit([]<typename IT>(IT &&) -> DataType {
            using T = std::decay_t<IT>;
            if constexpr (std::is_same_v<T, double>) return DataType(_priv::FLOAT64_T);
            if constexpr (std::is_same_v<T, float>) return DataType(_priv::FLOAT32_T);
            if constexpr (std::is_same_v<T, int64_t>) return DataType(_priv::INT64_T);
            if constexpr (std::is_same_v<T, int32_t>) return DataType(_priv::INT32_T);
            if constexpr (std::is_same_v<T, int16_t>) return DataType(_priv::INT16_T);
            if constexpr (std::is_same_v<T, int8_t>) return DataType(_priv::INT8_T);
            if constexpr (std::is_same_v<T, uint64_t>) return DataType(_priv::UINT64_T);
            if constexpr (std::is_same_v<T, uint32_t>) return DataType(_priv::UINT32_T);
            if constexpr (std::is_same_v<T, uint16_t>) return DataType(_priv::UINT16_T);
            if constexpr (std::is_same_v<T, uint8_t>) return DataType(_priv::UINT8_T);
            if constexpr (std::is_same_v<T, TimeStamp>)
                return DataType(_priv::TIMESTAMP_T);
            if constexpr (std::is_same_v<T, std::string>)
                return DataType(_priv::STRING_T);
            return DataType(_priv::UNKNOWN_T);
        }, value);
    }

    /// @property Gets type name.
    [[nodiscard]] std::string name() const { return value; }

    /// @property how many bytes in memory the data type holds.
    [[nodiscard]] size_t density() const { return DENSITIES[value]; }

    [[nodiscard]] bool is_variable() const {
        return this->matches(_priv::VARIABLE_TYPES);
    }

    /// @brief Checks if this data type matches any of the provided data types.
    /// @param others Vector of data types to compare against
    /// @returns true if this data type matches any in the vector, false otherwise
    [[nodiscard]] bool matches(const std::vector<DataType> &others) const {
        for (const auto &other: others) if (other == *this) return true;
        return false;
    }

    [[nodiscard]] bool matches(const std::vector<std::string> &others) const {
        for (const auto &other: others) if (other == this->value) return true;
        return false;
    }

    /// @brief Casts a numeric sample value to the type corresponding to this data type
    /// @param value The numeric sample value to cast
    /// @returns A new numeric sample value of the appropriate type
    /// @throws std::runtime_error if the data type is not numeric
    [[nodiscard]] SampleValue cast(const SampleValue &value) const {
        if (*this == _priv::FLOAT64_T) return telem::cast<double>(value);
        if (*this == _priv::FLOAT32_T) return telem::cast<float>(value);
        if (*this == _priv::INT64_T) return telem::cast<int64_t>(value);
        if (*this == _priv::INT32_T) return telem::cast<int32_t>(value);
        if (*this == _priv::INT16_T) return telem::cast<int16_t>(value);
        if (*this == _priv::INT8_T) return telem::cast<int8_t>(value);
        if (*this == _priv::UINT64_T) return telem::cast<uint64_t>(value);
        if (*this == _priv::UINT32_T) return telem::cast<uint32_t>(value);
        if (*this == _priv::UINT16_T) return telem::cast<uint16_t>(value);
        if (*this == _priv::UINT8_T) return telem::cast<uint8_t>(value);
        if (*this == _priv::TIMESTAMP_T) return telem::cast<TimeStamp>(value);
        if (this->is_variable()) return telem::cast<std::string>(value);
        throw std::runtime_error(
            "cannot cast sample value to unknown data type " + this->value);
    }

    /// @brief Casts a void pointer to a sample value of the appropriate type.
    /// @param value The void pointer to cast
    /// @param value_type
    /// @returns A new sample value of the appropriate type
    /// @throws std::runtime_error if the data type is not numeric
    SampleValue cast(const void *value, const DataType &value_type) const {
        if (value_type == _priv::FLOAT64_T) return this->cast(
            *static_cast<const double *>(value));
        if (value_type == _priv::FLOAT32_T) return this->cast(
            *static_cast<const float *>(value));
        if (value_type == _priv::INT64_T) return this->cast(
            *static_cast<const int64_t *>(value));
        if (value_type == _priv::INT32_T) return this->cast(
            *static_cast<const int32_t *>(value));
        if (value_type == _priv::INT16_T) return this->cast(
            *static_cast<const int16_t *>(value));
        if (value_type == _priv::INT8_T) return this->cast(
            *static_cast<const int8_t *>(value));
        if (value_type == _priv::UINT8_T) return this->cast(
            *static_cast<const uint8_t *>(value));
        if (value_type == _priv::UINT16_T) return this->cast(
            *static_cast<const uint16_t *>(value));
        if (value_type == _priv::UINT32_T) return this->cast(
            *static_cast<const uint32_t *>(value));
        if (value_type == _priv::UINT64_T) return this->cast(
            *static_cast<const uint64_t *>(value));
        if (value_type == _priv::TIMESTAMP_T) return this->cast(
            *static_cast<const TimeStamp *>(value));
        if (value_type == _priv::STRING_T) return this->cast(
            *static_cast<const std::string *>(value));
        if (value_type == _priv::JSON_T) return this->cast(
            *static_cast<const std::string *>(value));
        throw std::runtime_error(
            "cannot cast sample value to unknown data type " + this->value);
    }

    /////////////////////////////////// COMPARISON H///////////////////////////////////

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

    ////////////////////////////////// ADDITION OPERATORS /////////////////////////////////

    /// @brief Concatenates this DataType with another DataType
    /// @param other The DataType to concatenate with
    /// @returns A string with the concatenated values
    std::string operator+(const DataType &other) const {
        return value + other.value;
    }

    /// @brief Concatenates this DataType with a string
    /// @param other The string to concatenate with
    /// @returns A string with the concatenated values
    std::string operator+(const std::string &other) const {
        return value + other;
    }

    /// @brief Friend operator to allow string + DataType concatenation
    /// @param lhs The string on the left side of the + operator
    /// @param rhs The DataType on the right side of the + operator
    /// @returns A string with the concatenated values
    friend std::string operator+(const std::string &lhs, const DataType &rhs) {
        return lhs + rhs.value;
    }

    ////////////////////////////////// OSTREAM /////////////////////////////////

    friend std::ostream &operator<<(std::ostream &os, const DataType &dt) {
        os << dt.value;
        return os;
    }

    // Add hash support for DataType
    friend struct std::hash<telem::DataType>;

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
        {_priv::UINT128_T, 16},
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
        {
            std::type_index(typeid(long)),
            sizeof(long) == 8 ? _priv::INT64_T : _priv::INT32_T
        },
        {std::type_index(typeid(long long)), _priv::INT64_T},
        {std::type_index(typeid(std::int64_t)), _priv::INT64_T},
        {std::type_index(typeid(unsigned char)), _priv::UINT8_T},
        {std::type_index(typeid(std::uint8_t)), _priv::UINT8_T},
        {std::type_index(typeid(unsigned short)), _priv::UINT16_T},
        {std::type_index(typeid(std::uint16_t)), _priv::UINT16_T},
        {std::type_index(typeid(unsigned int)), _priv::UINT32_T},
        {std::type_index(typeid(std::uint32_t)), _priv::UINT32_T},
        {
            std::type_index(typeid(unsigned long)),
            sizeof(unsigned long) == 8 ? _priv::UINT64_T : _priv::UINT32_T
        },
        {std::type_index(typeid(unsigned long long)), _priv::UINT64_T},
        {std::type_index(typeid(std::uint64_t)), _priv::UINT64_T},
        {std::type_index(typeid(std::string)), _priv::STRING_T},
        {std::type_index(typeid(TimeStamp)), _priv::TIMESTAMP_T},
    };
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
/// @brief identifier for a fixed-size uint128 data type in a Synnax cluster (16 bytes).
const DataType UINT128_T(_priv::UINT128_T);
/// @brief identifier for a fixed-size UUID data type in a Synnax cluster (16 bytes).
const DataType UUID_T(_priv::UUID_T);
/// @brief identifier for a newline separated, variable-length string data type in a
/// Synnax cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const DataType STRING_T(_priv::STRING_T);
/// @brief identifier for a newline separated, stringified JSON data type in a Synnax
/// cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const DataType JSON_T(_priv::JSON_T);
}

// Add hash specialization in std namespace
template<>
struct std::hash<telem::DataType> {
    size_t operator()(const telem::DataType &dt) const noexcept {
        return hash<string>()(dt.value);
    }
};

