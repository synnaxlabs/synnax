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
constexpr uint64_t NANOSECOND = 1;
constexpr uint64_t MICROSECOND = NANOSECOND * 1e3;
constexpr uint64_t MILLISECOND = MICROSECOND * 1e3;
constexpr uint64_t SECOND = MILLISECOND * 1e3;
constexpr uint64_t MINUTE = SECOND * 60;
constexpr uint64_t HOUR = MINUTE * 60;
constexpr uint64_t DAY = HOUR * 24;
} // namespace _priv

#define ASSERT_TYPE_SIZE(type, size) \
    static_assert(sizeof(type) == size, "synnax only supports compilation environments with " #size " bit " #type "s")

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
    uint8_t // UINT8
>;

template<typename T>
[[nodiscard]] T cast_numeric_sample_value(const NumericSampleValue value) {
    if (std::holds_alternative<T>(value))
        return std::get<T>(value);
    return std::visit([](auto &&arg) -> T {
        return static_cast<T>(arg);
    }, value);
}

constexpr size_t FLOAT64_INDEX = 0;
constexpr size_t FLOAT32_INDEX = 1;
constexpr size_t INT64_INDEX = 2;
constexpr size_t INT32_INDEX = 3;
constexpr size_t INT16_INDEX = 4;
constexpr size_t INT8_INDEX = 5;
constexpr size_t UINT64_INDEX = 6;
constexpr size_t UINT32_INDEX = 7;
constexpr size_t UINT16_INDEX = 8;
constexpr size_t UINT8_INDEX = 9;
constexpr size_t STRING_INDEX = 10;

/// @brief Holds the name and properties of a datatype.
class DataType {
public:
    DataType() = default;

    /// @brief Holds the id of the data type
    std::string value;

    explicit DataType(std::string data_type) {
        if (!DENSITIES.count(data_type)) {
            if (!NAMES.count(data_type))
                throw std::runtime_error(
                    "Tried to create unknown datatype " + data_type);
            data_type = NAMES[data_type];
        }
        value = data_type;
    }

    /// @returns the data type corresponding to the given type.
    template<typename T>
    DataType static infer(const DataType &dt = DataType("")) {
        if (dt != DataType("")) return dt;
        const auto type_index = std::type_index(typeid(T));
        if (!TYPE_INDEXES.count(type_index))
            throw std::runtime_error(
                "failed to infer data type for " + std::string(typeid(T).name()));
        return DataType(TYPE_INDEXES[type_index]);
    }


    /// @property Gets type name.
    [[nodiscard]] std::string name() const { return value; }

    /// @property Essentially how many bytes in memory the datatype holds.
    [[nodiscard]] uint32_t density() const { return DENSITIES[value]; }

    [[nodiscard]] bool is_variable() const {
        return value == "string" || value == "json";
    }

    /// @brief Checks if this data type matches another data type.
    /// @param other The data type to compare against
    /// @returns true if the data types match, false otherwise
    [[nodiscard]] bool matches(const DataType &other) const {
        if (value.empty() || other.value.empty()) return true;
        return *this == other;
    }

    /// @brief Checks if this data type matches a string data type identifier.
    /// @param other The data type string to compare against
    /// @returns true if the data types match, false otherwise
    [[nodiscard]] bool matches(const std::string &other) const {
        if (value.empty() || other.empty()) return true;
        return value == other;
    }

    /// @brief Checks if this data type matches any of the provided data types.
    /// @param others Vector of data types to compare against
    /// @returns true if this data type matches any in the vector, false otherwise
    [[nodiscard]] bool matches(const std::vector<DataType> &others) const {
        if (value.empty()) return true;
        for (const auto &other: others) if (matches(other)) return true;
        return false;
    }

    /// @brief Casts a numeric sample value to the type corresponding to this data type
    /// @param value The numeric sample value to cast
    /// @returns A new numeric sample value of the appropriate type
    /// @throws std::runtime_error if the data type is not numeric
    [[nodiscard]] NumericSampleValue cast(const NumericSampleValue& value) const {
        if (this->value == "float64") return cast_numeric_sample_value<double>(value);
        if (this->value == "float32") return cast_numeric_sample_value<float>(value);
        if (this->value == "int64") return cast_numeric_sample_value<int64_t>(value);
        if (this->value == "int32") return cast_numeric_sample_value<int32_t>(value);
        if (this->value == "int16") return cast_numeric_sample_value<int16_t>(value);
        if (this->value == "int8") return cast_numeric_sample_value<int8_t>(value);
        if (this->value == "uint64") return cast_numeric_sample_value<uint64_t>(value);
        if (this->value == "uint32") return cast_numeric_sample_value<uint32_t>(value);
        if (this->value == "uint16") return cast_numeric_sample_value<uint16_t>(value);
        if (this->value == "uint8") return cast_numeric_sample_value<uint8_t>(value);
        throw std::runtime_error("Cannot cast non-numeric data type: " + this->value);
    }

    /////////////////////////////////// COMPARISON H///////////////////////////////////

    bool operator==(const DataType &other) const { return value == other.value; }

    bool operator!=(const DataType &other) const { return value != other.value; }

    ////////////////////////////////// OSTREAM /////////////////////////////////

    friend std::ostream &operator<<(std::ostream &os, const DataType &dt) {
        os << dt.value;
        return os;
    }

private:
    /// @brief Maps the data type to the 'density' of
    /// the object.
    inline static std::unordered_map<std::string, uint32_t> DENSITIES = {
        {"", 0},
        {"float64", 8},
        {"float32", 4},
        {"int8", 1},
        {"int16", 2},
        {"int32", 4},
        {"int64", 8},
        {"uint8", 1},
        {"uint16", 2},
        {"uint32", 4},
        {"uint64", 8},
        {"uint128", 16},
        {"timestamp", 8},
        {"uuid", 16},
        {"string", 0},
        {"json", 0},
    };

    /// @brief stores a map of C++ type indexes to their corresponding synnax data
    /// type identifiers.
    inline static std::unordered_map<std::type_index, std::string> TYPE_INDEXES = {
        {std::type_index(typeid(float)), "float32"},
        {std::type_index(typeid(double)), "float64"},
        {std::type_index(typeid(char)), "int8"},
        {std::type_index(typeid(std::int8_t)), "int8"},
        {std::type_index(typeid(short)), "int16"},
        {std::type_index(typeid(std::int16_t)), "int16"},
        {std::type_index(typeid(int)), "int32"},
        {std::type_index(typeid(std::int32_t)), "int32"},
        {std::type_index(typeid(long)), sizeof(long) == 8 ? "int64" : "int32"},
        {std::type_index(typeid(long long)), "int64"},
        {std::type_index(typeid(std::int64_t)), "int64"},
        {std::type_index(typeid(unsigned char)), "uint8"},
        {std::type_index(typeid(std::uint8_t)), "uint8"},
        {std::type_index(typeid(unsigned short)), "uint16"},
        {std::type_index(typeid(std::uint16_t)), "uint16"},
        {std::type_index(typeid(unsigned int)), "uint32"},
        {std::type_index(typeid(std::uint32_t)), "uint32"},
        {
            std::type_index(typeid(unsigned long)),
            sizeof(unsigned long) == 8 ? "uint64" : "uint32"
        },
        {std::type_index(typeid(unsigned long long)), "uint64"},
        {std::type_index(typeid(std::uint64_t)), "uint64"},
        {std::type_index(typeid(std::string)), "string"},
    };

    /// @brief Maps the data type id to name
    inline static std::unordered_map<std::string, std::string> NAMES = {
        {typeid(double).name(), "float64"},
        {typeid(float).name(), "float32"},
        {typeid(char).name(), "int8"},
        {typeid(std::int8_t).name(), "int8"},
        {typeid(short).name(), "int16"},
        {typeid(std::int16_t).name(), "int16"},
        {typeid(int).name(), "int32"},
        {typeid(std::int32_t).name(), "int32"},
        {typeid(long long).name(), "int64"},
        {typeid(std::int64_t).name(), "int64"},
        {typeid(unsigned char).name(), "uint8"},
        {typeid(std::uint8_t).name(), "uint8"},
        {typeid(unsigned short).name(), "uint16"},
        {typeid(std::uint16_t).name(), "uint16"},
        {typeid(unsigned int).name(), "uint32"},
        {typeid(unsigned long long).name(), "uint64"},
        {typeid(std::string).name(), "string"},
    };
};

/// @brief
const auto DATA_TYPE_UNKNOWN = DataType("");
/// @brief identifier for a fixed-size float64 data type in a Synnax cluster.
const auto FLOAT64_T = DataType("float64");
/// @brief identifier for a fixed-size float32 data type in a Synnax cluster.
const auto FLOAT32_T = DataType("float32");
/// @brief identifier for a fixed-size int8 data type in a Synnax cluster.
const auto INT8_T = DataType("int8");
/// @brief identifier for a fixed-size int16 data type in a Synnax cluster.
const auto INT16_T = DataType("int16");
/// @brief identifier for a fixed-size int32 data type in a Synnax cluster.
const auto INT32_T = DataType("int32");
/// @brief identifier for a fixed-size int64 data type in a Synnax cluster.
const auto INT64_T = DataType("int64");
/// @brief identifier for a fixed-size timestamp data type in a Synnax cluster.
const auto TIMESTAMP_T = DataType("timestamp");
/// @brief identifier for a fixed-size uint8 data type in a Synnax cluster.
const auto UINT8_T = DataType("uint8");
/// @brief identifier for a fixed-size uint16 data type in a Synnax cluster.
const auto UINT16_T = DataType("uint16");
/// @brief identifier for a fixed-size uint32 data type in a Synnax cluster.
const auto UINT32_T = DataType("uint32");
/// @brief identifier for a fixed-size uint64 data type in a Synnax cluster.
const auto UINT64_T = DataType("uint64");
/// @brief identifier for a fixed-size uint128 data type in a Synnax cluster (16 bytes).
const auto UINT128_T = DataType("uint128");
/// @brief identifier for a fixed-size UUID data type in a Synnax cluster (16 bytes).
const auto UUID_T = DataType("uuid");
/// @brief identifier for a newline separated, variable-length string data type in a
/// Synnax cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const auto STRING_T = DataType("string");
/// @brief identifier for a newline separated, stringified JSON data type in a Synnax
/// cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const auto JSON_T = DataType("json");

class TimeSpan {
public:
    /// @property value holds the internal, primitive value of the timespan.
    std::uint64_t value;

    TimeSpan() = default;

    /// @brief Constructs a timespan from the given unsigned long long, interpreting it as a nanosecond-precision
    /// timespan.
    explicit TimeSpan(const std::uint64_t i) : value(i) {
    }

    explicit TimeSpan(
        const std::chrono::duration<std::int64_t, std::nano> &duration) : value(
        duration.count()) {
    }

    static TimeSpan days(const double &days) {
        return TimeSpan(static_cast<std::uint64_t>(days * _priv::DAY));
    }

    static TimeSpan hours(const double &hours) {
        return TimeSpan(static_cast<std::uint64_t>(hours * _priv::HOUR));
    }

    static TimeSpan minutes(const double &minutes) {
        return TimeSpan(static_cast<std::uint64_t>(minutes * _priv::MINUTE));
    }

    static TimeSpan seconds(const double &seconds) {
        return TimeSpan(static_cast<std::uint64_t>(seconds * _priv::SECOND));
    }

    ///////////////////////////////////// COMPARISON /////////////////////////////////////

    bool operator==(const TimeSpan &other) const { return value == other.value; }

    bool operator==(const std::uint64_t &other) const { return value == other; }

    bool operator!=(const std::uint64_t &other) const { return value != other; }

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

    TimeSpan operator+(const std::uint64_t &other) const {
        return TimeSpan(value + other);
    }

    friend TimeSpan operator+(const unsigned long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs + rhs.value);
    }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeSpan &other) const {
        return TimeSpan(value - other.value);
    }

    friend TimeSpan operator-(const unsigned long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    TimeSpan operator-(const unsigned long long &other) const {
        return TimeSpan(value - other);
    }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    TimeSpan operator*(const TimeSpan &other) const {
        return TimeSpan(value * other.value);
    }

    TimeSpan operator*(const float &other) const { return TimeSpan(value * other); }

    friend TimeSpan operator*(const unsigned long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs * rhs.value);
    }

    TimeSpan operator*(const unsigned long long &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const int &other) const { return TimeSpan(value * other); }

    TimeSpan operator*(const unsigned int &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const double &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const long &other) const { return TimeSpan(value * other); }

    TimeSpan operator*(const unsigned long &other) const {
        return TimeSpan(value * other);
    }

    TimeSpan operator*(const long long &other) const {
        return TimeSpan(value * other);
    }


    ////////////////////////////////// DIVISION /////////////////////////////////

    TimeSpan operator/(const TimeSpan &other) const {
        return TimeSpan(value / other.value);
    }

    friend TimeSpan operator/(const unsigned long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs / rhs.value);
    }

    TimeSpan operator/(const unsigned long long &other) const {
        return TimeSpan(value / other);
    }

    ////////////////////////////////// MODULO /////////////////////////////////

    TimeSpan operator%(const TimeSpan &other) const {
        return TimeSpan(value % other.value);
    }

    friend TimeSpan operator%(const unsigned long long &lhs, const TimeSpan &rhs) {
        return TimeSpan(lhs % rhs.value);
    }

    TimeSpan operator%(const unsigned long long &other) const {
        return TimeSpan(value % other);
    }

    [[nodiscard]] TimeSpan truncate(const TimeSpan &other) const {
        return TimeSpan(value / other.value * other.value);
    }

    [[nodiscard]] TimeSpan delta(const TimeSpan &other) const {
        if (other > *this) return other - *this;
        return *this - other;
    }

    [[nodiscard]] double days() const {
        return static_cast<double>(value) / _priv::DAY;
    }

    [[nodiscard]] double hours() const {
        return static_cast<double>(value) / _priv::HOUR;
    }

    [[nodiscard]] double minutes() const {
        return static_cast<double>(value) / _priv::MINUTE;
    }

    [[nodiscard]] double seconds() const {
        return static_cast<double>(value) / _priv::SECOND;
    }

    [[nodiscard]] double milliseconds() const {
        return static_cast<double>(value) / _priv::MILLISECOND;
    }

    [[nodiscard]] double microseconds() const {
        return static_cast<double>(value) / _priv::MICROSECOND;
    }


    ////////////////////////////////// OSTREAM /////////////////////////////////

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
        const auto seconds = (total_seconds.value - total_minutes.value) / _priv::SECOND;
        const auto milliseconds = (total_milliseconds.value - total_seconds.value) / _priv::MILLISECOND;
        const auto microseconds = (total_microseconds.value - total_milliseconds.value) / _priv::MICROSECOND;
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


    [[nodiscard]] std::chrono::nanoseconds chrono() const {
        return std::chrono::nanoseconds(value);
    }

};

/// @brief represents a 64-bit nanosecond-precision, UNIX Epoch UTC timestamp.
class TimeStamp {
public:
    /// @property value holds the internal, primitive value of the timestamp.
    std::uint64_t value;

    TimeStamp() = default;

    /// @brief Constructs a timestamp from the given unsigned long long, interpreting it as a nanosecond-precision UTC
    /// timestamp.
    explicit TimeStamp(const std::uint64_t value) : value(value) {
    }

    /// @brief interprets the given TimeSpan as a TimeStamp.
    explicit TimeStamp(const TimeSpan ts) : value(ts.value) {
    }

    TimeStamp static now() {
        // note that on some machines, high res clock refs system_clock and on others it references
        // steady_clock. This could create a problem so we should probably use system_clock.
        return TimeStamp(std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count());
    }


    ///////////////////////////////////// COMPARISON /////////////////////////////////////

    bool operator==(const TimeStamp &other) const { return value == other.value; }

    bool operator!=(const TimeStamp &other) const { return value != other.value; }

    bool operator<(const TimeStamp &other) const { return value < other.value; }

    bool operator>(const TimeStamp &other) const { return value > other.value; }

    bool operator<=(const TimeStamp &other) const { return value <= other.value; }

    bool operator>=(const TimeStamp &other) const { return value >= other.value; }

    //////////////////////////////////// ADDITION /////////////////////////////////////

    TimeStamp operator+(const TimeStamp &other) const {
        return TimeStamp(value + other.value);
    }

    friend TimeStamp
    operator+(const unsigned long long &lhs, const TimeStamp &rhs) {
        return TimeStamp(lhs + rhs.value);
    }

    TimeStamp operator+(const TimeSpan &other) const {
        return TimeStamp(value + other.value);
    }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeStamp &other) const {
        return TimeSpan(value - other.value);
    }

    friend TimeSpan operator-(const unsigned long long &lhs, const TimeStamp &rhs) {
        return TimeSpan(lhs - rhs.value);
    }

    TimeSpan operator-(const TimeSpan &other) const {
        return TimeSpan(value - other.value);
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

    TimeRange(const std::uint64_t start, const std::uint64_t end) : start(start),
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
public:
    float value;

    explicit Rate(const float i) : value(i) {
    }

    explicit Rate(const int i) : value(i) {
    }

    explicit Rate(const double i) : value(i) {
    }

    explicit Rate(const TimeSpan period) : value(1 / period.seconds()) {
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

    [[nodiscard]] TimeSpan period() const { return TimeSpan(1 / value * 1e9); }

    ////////////////////////////////// DIVISION /////////////////////////////////

    Rate operator/(const Rate &other) const { return Rate(value / other.value); }

    Rate operator/(const float &other) const { return Rate(value / other); }

    Rate operator/(const int &other) const { return Rate(value / other); }

    Rate operator/(const unsigned int &other) const { return Rate(value / other); }

    Rate operator/(const double &other) const { return Rate(value / other); }

    Rate operator/(const long &other) const { return Rate(value / other); }

    Rate operator/(const size_t &other) const { return Rate(value / other); }
};

/// @brief a single hertz. Can be made into many hertz through multiplication
/// e.g. 55 * HZ = 55 hertz.
const auto HZ = Rate(1);
/// @brief a single kilohertz. Can be made into many kilohertz through multiplication
/// e.g. 55 * KHZ = 55 kilohertz.
const Rate KHZ = 1000 * HZ;
/// @brief a single megahertz. Can be made into many megahertz through multiplication
/// e.g. 55 * MHZ = 55 megahertz.
const Rate MHZ = 1000 * KHZ;

/// @brief a single nanosecond. Can be made into many nanoseconds through multiplication
/// e.g. 55 * NANOSECOND = 55 nanoseconds.
const auto NANOSECOND = TimeSpan(1);
/// @brief a single microsecond. Can be made into many microseconds through multiplication
/// e.g. 55 * MICROSECOND = 55 microseconds.
const TimeSpan MICROSECOND = NANOSECOND * 1000;
/// @brief a single millisecond. Can be made into many milliseconds through multiplication
/// e.g. 55 * MILLISECOND = 55 milliseconds.
const TimeSpan MILLISECOND = MICROSECOND * 1000;
/// @brief a single second. Can be made into many seconds through multiplication
/// e.g. 55 * SECOND = 55 seconds.
const TimeSpan SECOND = MILLISECOND * 1000;
/// @brief a single minute. Can be made into many minutes through multiplication
/// e.g. 55 * MINUTE = 55 minutes.
const TimeSpan MINUTE = SECOND * 60;
/// @brief a single hour. Can be made into many hours through multiplication
/// e.g. 55 * HOUR = 55 hours.
const TimeSpan HOUR = MINUTE * 60;
/// @brief a single day. Can be made into many days through multiplication
/// e.g. 55 * DAY = 55 days.
const TimeSpan DAY = HOUR * 24;
}
