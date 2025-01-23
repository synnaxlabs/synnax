// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstdint>
#include <string>
#include <iostream>
#include <typeindex>
#include <unordered_map>

namespace synnax {
// private namespace for internal constants
namespace _priv {
const uint64_t NANOSECOND = 1;
const uint64_t MICROSECOND = NANOSECOND * 1e3;
const uint64_t MILLISECOND = MICROSECOND * 1e3;
const uint64_t SECOND = MILLISECOND * 1e3;
const uint64_t MINUTE = SECOND * 60;
const uint64_t HOUR = MINUTE * 60;
const uint64_t DAY = HOUR * 24;
} // namespace _priv


/// @brief Holds the name and properties of a datatype.
class DataType {
public:
    DataType() : value("") {
    }

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
    DataType static infer() {
        if (!TYPE_INDEXES.count(std::type_index(typeid(T))))
            return DataType("");
        return DataType(TYPE_INDEXES[std::type_index(typeid(T))]);
    }

    /// @property Gets type name.
    [[nodiscard]] std::string name() const { return value; }

    /// @property Essentially how many bytes in memory the datatype holds.
    [[nodiscard]] uint32_t density() const { return DENSITIES[value]; }

    [[nodiscard]] bool is_variable() const {
        return value == "string" || value == "json";
    }

    /////////////////////////////////// COMPARISON ///////////////////////////////////

    bool operator==(const DataType &other) const { return value == other.value; }

    bool operator!=(const DataType &other) const { return value != other.value; }

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

    /// @brief stores a map of C++ type indexes to their correspondign synnax data
    /// type identifiers.
    inline static std::unordered_map<std::type_index, std::string> TYPE_INDEXES = {
        {std::type_index(typeid(int)), "int32"},
        {std::type_index(typeid(double)), "float64"},
        {std::type_index(typeid(float)), "float32"},
        {std::type_index(typeid(long long)), "int64"},
        {std::type_index(typeid(short)), "int16"},
        {std::type_index(typeid(char)), "int8"},
        {std::type_index(typeid(unsigned int)), "int32"},
        {std::type_index(typeid(unsigned long long)), "uint64"},
        {std::type_index(typeid(unsigned short)), "uint16"},
        {std::type_index(typeid(unsigned char)), "uint8"},
        {std::type_index(typeid(std::string)), "string"},
    };

    /// @brief Maps the data type id to name
    inline static std::unordered_map<std::string, std::string> NAMES = {
        {typeid(int).name(), "int32"},
        {typeid(double).name(), "float64"},
        {typeid(float).name(), "float32"},
        {typeid(long long).name(), "int64"},
        {typeid(short).name(), "int16"},
        {typeid(char).name(), "int8"},
        {typeid(unsigned int).name(), "uint32"},
        {typeid(unsigned long long).name(), "uint64"},
        {typeid(unsigned short).name(), "uint16"},
        {typeid(unsigned char).name(), "uint8"},
        {typeid(std::string).name(), "string"},
    };
};

/// @brief
const auto DATA_TYPE_UNKNOWN = DataType("");
/// @brief identifier for a fixed-size float64 data type in a Synnax cluster.
const auto FLOAT64 = DataType("float64");
/// @brief identifier for a fixed-size float32 data type in a Synnax cluster.
const auto FLOAT32 = DataType("float32");
/// @brief identifier for a fixed-size int8 data type in a Synnax cluster.
const auto INT8 = DataType("int8");
/// @brief identifier for a fixed-size int16 data type in a Synnax cluster.
const auto INT16 = DataType("int16");
/// @brief identifier for a fixed-size int32 data type in a Synnax cluster.
const auto INT32 = DataType("int32");
/// @brief identifier for a fixed-size int64 data type in a Synnax cluster.
const auto INT64 = DataType("int64");
/// @brief identifier for a fixed-size timestamp data type in a Synnax cluster.
const auto TIMESTAMP = DataType("timestamp");
/// @brief identifier for a fixed-size uint8 data type in a Synnax cluster.
const auto SY_UINT8 = DataType("uint8");
/// @brief identifier for a fixed-size uint16 data type in a Synnax cluster.
const auto SY_UINT16 = DataType("uint16");
/// @brief identifier for a fixed-size uint32 data type in a Synnax cluster.
const auto UINT32 = DataType("uint32");
/// @brief identifier for a fixed-size uint64 data type in a Synnax cluster.
const auto UINT64 = DataType("uint64");
/// @brief identifier for a fixed-size uint128 data type in a Synnax cluster (16 bytes).
const auto UINT128 = DataType("uint128");
/// @brief identifier for a fixed-size UUID data type in a Synnax cluster (16 bytes).
const auto UUID = DataType("uuid");
/// @brief identifier for a newline separated, variable-length string data type in a
/// Synnax cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const auto STRING = DataType("string");
/// @brief identifier for a newline separated, stringified JSON data type in a Synnax
/// cluster. Note that variable-length data types have reduced performance and
/// restricted use within a Synnax cluster.
const auto JSON = DataType("json");

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
        return TimeSpan((value / other.value) * other.value);
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

    friend std::ostream &operator<<(std::ostream &os, const TimeSpan &ts) {
        const auto total_days = ts.truncate(TimeSpan(_priv::DAY));
        const auto total_hours = ts.truncate(TimeSpan(_priv::HOUR));
        const auto total_minutes = ts.truncate(TimeSpan(_priv::MINUTE));
        const auto total_seconds = ts.truncate(TimeSpan(_priv::SECOND));
        const auto total_milliseconds = ts.truncate(TimeSpan(_priv::MILLISECOND));
        const auto total_microseconds = ts.truncate(TimeSpan(_priv::MICROSECOND));
        const auto total_nanoseconds = ts;
        const auto days = total_days;
        const auto hours = total_hours - total_days;
        const auto minutes = total_minutes - total_hours;
        const auto seconds = total_seconds - total_minutes;
        const auto milliseconds = total_milliseconds - total_seconds;
        const auto microseconds = total_microseconds - total_milliseconds;
        const auto nanoseconds = total_nanoseconds - total_microseconds;

        if (total_days != 0) os << days.days() << "d ";
        if (total_hours != 0) os << hours.hours() << "h ";
        if (total_minutes != 0) os << minutes.minutes() << "m ";
        if (total_seconds != 0) os << seconds.seconds() << "s ";
        if (total_milliseconds != 0) os << milliseconds.milliseconds() << "ms ";
        if (total_microseconds != 0) os << microseconds.microseconds() << "us ";
        if (total_nanoseconds != 0) os << nanoseconds.value << "ns";
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
};
