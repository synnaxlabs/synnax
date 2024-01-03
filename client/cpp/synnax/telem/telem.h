// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Std.
#include <unordered_map>
#include <string>
#include <any>

namespace synnax {

/// @brief Holds the name and properties of a datatype.
class DataType {
public:
    DataType() = default;

    explicit DataType(std::string data_type) {
        setDataType(data_type);
    }

    void setDataType(std::string data_type) {
        if (!DENSITIES.count(data_type)) {
            if (NAMES.count(data_type)) {
                data_type = NAMES[data_type];
            } else throw std::runtime_error("Tried to create unknown datatype " + data_type);
        }
        value = data_type;
    }

    /// @property Gets type name.
    [[nodiscard]] std::string name() const { return value; }

    /// @property Essentially how many bytes in memory the datatype holds.
    [[nodiscard]] uint32_t density() const { return DENSITIES[value]; }

    /////////////////////////////////// COMPARISON ///////////////////////////////////

    bool operator==(const DataType &other) const { return value == other.value; }

    bool operator!=(const DataType &other) const { return value != other.value; }

private:
    /// @brief Holds the id of the data type
    std::string value;
    /// @brief Maps the data type to the 'density' of
    /// the object.
    static inline std::unordered_map<std::string, uint32_t> DENSITIES = {
            {"float64",   8},
            {"float32",   4},
            {"int8",      1},
            {"int16",     2},
            {"int32",     4},
            {"int64",     8},
            {"uint8",     1},
            {"uint16",    2},
            {"uint32",    4},
            {"uint64",    8},
            {"timestamp", 8},
            {"string",    0},
            {"uuid",      16}
    };

    /// @brief Maps the data type id to name
    static inline std::unordered_map<std::string, std::string> NAMES = {
            {typeid(int).name(),            "int32"},
            {typeid(double).name(),         "float64"},
            {typeid(float).name(),          "float32"},
            {typeid(long).name(),           "int64"},
            {typeid(short).name(),          "int16"},
            {typeid(char).name(),           "int8"},
            {typeid(unsigned int).name(),   "uint32"},
            {typeid(unsigned long).name(),  "uint64"},
            {typeid(unsigned short).name(), "uint16"},
            {typeid(unsigned char).name(),  "uint8"},
            {typeid(std::string).name(),    "string"},
    };

};

/// @brief representation of a float64 data type in a Synnax cluster.
const DataType FLOAT64 = DataType("float64");
/// @brief representation of a float32 data type in a Synnax cluster.
const DataType FLOAT32 = DataType("float32");
/// @brief representation of a int8 data type in a Synnax cluster.
const DataType INT8 = DataType("int8");
/// @brief representation of a int16 data type in a Synnax cluster.
const DataType INT16 = DataType("int16");
/// @brief representation of a int32 data type in a Synnax cluster.
const DataType INT32 = DataType("int32");
/// @brief representation of a int64 data type in a Synnax cluster.
const DataType INT64 = DataType("int64");
/// @brief representation of a timestamp data type in a Synnax cluster.
const DataType TIMESTAMP = DataType("timestamp");
/// @brief representation of a uint8 data type in a Synnax cluster.
const DataType UINT8 = DataType("uint8");
/// @brief representation of a uint16 data type in a Synnax cluster.
const DataType UINT16 = DataType("uint16");
/// @brief representation of a uint32 data type in a Synnax cluster.
const DataType UINT32 = DataType("uint32");
/// @brief representation of a uint64 data type in a Synnax cluster.
const DataType UINT64 = DataType("uint64");
/// @brief representation of a uuid data type in a Synnax cluster.
const DataType STRING = DataType("string");

class TimeSpan {
public:
    /// @property value holds the internal, primitive value of the timespan.
    long value;

    /// @brief Constructs a timespan from the given long, interpreting it as a nanosecond-precision
    /// timespan.
    explicit TimeSpan(long i) : value(i) {}

    ///////////////////////////////////// COMPARISON /////////////////////////////////////

    bool operator==(const TimeSpan &other) const { return value == other.value; }

    bool operator!=(const TimeSpan &other) const { return value != other.value; }

    bool operator<(const TimeSpan &other) const { return value < other.value; }

    bool operator>(const TimeSpan &other) const { return value > other.value; }

    bool operator<=(const TimeSpan &other) const { return value <= other.value; }

    bool operator>=(const TimeSpan &other) const { return value >= other.value; }

    //////////////////////////////////// ADDITION /////////////////////////////////////

    TimeSpan operator+(const TimeSpan &other) const { return TimeSpan(value + other.value); }

    friend TimeSpan operator+(const long &lhs, const TimeSpan &rhs) { return TimeSpan(lhs + rhs.value); }

    TimeSpan operator+(const long &other) const { return TimeSpan(value + other); }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeSpan &other) const { return TimeSpan(value - other.value); }

    friend TimeSpan operator-(const long &lhs, const TimeSpan &rhs) { return TimeSpan(lhs - rhs.value); }

    TimeSpan operator-(const long &other) const { return TimeSpan(value - other); }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    TimeSpan operator*(const TimeSpan &other) const { return TimeSpan(value * other.value); }

    friend TimeSpan operator*(const long &lhs, const TimeSpan &rhs) { return TimeSpan(lhs * rhs.value); }

    TimeSpan operator*(const long &other) const { return TimeSpan(value * other); }

    ////////////////////////////////// DIVISION /////////////////////////////////

    TimeSpan operator/(const TimeSpan &other) const { return TimeSpan(value / other.value); }

    friend TimeSpan operator/(const long &lhs, const TimeSpan &rhs) { return TimeSpan(lhs / rhs.value); }

    TimeSpan operator/(const long &other) const { return TimeSpan(value / other); }
};

/// @brief represents a 64-bit nanosecond-precision, UNIX Epoch UTC timestamp.
class TimeStamp {
public:
    /// @property value holds the internal, primitive value of the timestamp.
    long value;

    TimeStamp() = default;

    /// @brief Constructs a timestamp from the given long, interpreting it as a nanosecond-precision UTC
    /// timestamp.
    explicit TimeStamp(long value) : value(value) {}

    /// @brief interprets the given TimeSpan as a TimeStamp.
    explicit TimeStamp(TimeSpan ts) : value(ts.value) {}

    TimeStamp static now() {
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

    TimeStamp operator+(const TimeStamp &other) const { return TimeStamp(value + other.value); }

    friend TimeStamp operator+(const long &lhs, const TimeStamp &rhs) { return TimeStamp(lhs + rhs.value); }

    TimeStamp operator+(const TimeSpan &other) const { return TimeStamp(value + other.value); }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    TimeSpan operator-(const TimeStamp &other) const { return TimeSpan(value - other.value); }

    friend TimeSpan operator-(const long &lhs, const TimeStamp &rhs) { return TimeSpan(lhs - rhs.value); }

    TimeSpan operator-(const TimeSpan &other) const { return TimeSpan(value - other.value); }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    TimeStamp operator*(const TimeStamp &other) const { return TimeStamp(value * other.value); }

    TimeStamp operator/(const TimeStamp &other) const { return TimeStamp(value / other.value); }

    TimeStamp operator%(const TimeStamp &other) const { return TimeStamp(value % other.value); }

    TimeStamp operator+=(const TimeStamp &other) { return TimeStamp(value += other.value); }

    TimeStamp operator-=(const TimeStamp &other) { return TimeStamp(value -= other.value); }

    TimeStamp operator*=(const TimeStamp &other) { return TimeStamp(value *= other.value); }

    TimeStamp operator/=(const TimeStamp &other) { return TimeStamp(value /= other.value); }

    TimeStamp operator%=(const TimeStamp &other) { return TimeStamp(value %= other.value); }
};

class TimeRange {
public:
    TimeStamp start;
    TimeStamp end;

    TimeRange() = default;

    /// @brief constructs a TimeRange from the given start and end timestamps.
    TimeRange(TimeStamp start, TimeStamp end) : start(TimeStamp(start)), end(end) {}

    /// @brief returns true if the given timestamp is within the range, start inclusive, end exclusive.
    [[nodiscard]] bool contains(TimeStamp time) const { return start <= time && time < end; }

    /// @brief returns true if the TimeRange contains the given TimeRange. If the two time ranges are equal,
    /// returns true. In this case, the two time ranges contain each other.
    [[nodiscard]] bool contains(TimeRange tr) const { return tr.start >= start && tr.end <= end; }

    bool operator==(const TimeRange &other) const { return start == other.start && end == other.end; }
};

class Rate {
public:
    float value;

    explicit Rate(float i) : value(i) {}

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

    friend Rate operator+(const float &lhs, const Rate &rhs) { return Rate(lhs + rhs.value); }

    Rate operator+(const float &other) const { return Rate(value + other); }

    /////////////////////////////////// SUBTRACTION ///////////////////////////////////

    Rate operator-(const Rate &other) const { return Rate(value - other.value); }

    friend Rate operator-(const float &lhs, const Rate &rhs) { return Rate(lhs - rhs.value); }

    Rate operator-(const float &other) const { return Rate(value - other); }

    ////////////////////////////////// MULTIPLICATION /////////////////////////////////

    Rate operator*(const Rate &other) const { return Rate(value * other.value); }

    friend Rate operator*(const float &lhs, const Rate &rhs) { return Rate(lhs * rhs.value); }

    Rate operator*(const float &other) const { return Rate(value * other); }
};

/// @brief a single hertz. Can be made into many hertz through multiplication
/// e.g. 55 * HZ = 55 hertz.
static const Rate HZ = Rate(1);
/// @brief a single kilohertz. Can be made into many kilohertz through multiplication
/// e.g. 55 * KHZ = 55 kilohertz.
static const Rate KHZ = 1000 * HZ;
/// @brief a single megahertz. Can be made into many megahertz through multiplication
/// e.g. 55 * MHZ = 55 megahertz.
static const Rate MHZ = 1000 * KHZ;

/// @brief a single nanosecond. Can be made into many nanoseconds through multiplication
/// e.g. 55 * NANOSECOND = 55 nanoseconds.
const TimeSpan NANOSECOND = TimeSpan(1);
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

