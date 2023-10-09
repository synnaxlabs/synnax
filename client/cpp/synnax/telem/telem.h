#pragma once


#include <unordered_map>
#include <string>
#include <any>

namespace Synnax {
    namespace Telem {

/// @brief Holds the name and properties of a datatype. 
        class DataType {
        public:
            /// @brief Holds the id of the data type
            std::string value;

            DataType() = default;

            DataType(std::string data_type_) {
                setDataType(data_type_);
            }

            void setDataType(std::string data_type_) {
                if (!DENSITIES.count(data_type_)) {
                    throw std::runtime_error("Tried to create an unknown datatype.");
                }
                value = data_type_;
            }

            /// @property Gets type name.
            std::string name() {
                return NAMES[value];
            }

            /// @property Essentially how many bytes in memory the datatype holds.
            int density() {
                return DENSITIES[value];
            }

        private:


            /// @brief Maps the data type to the 'density' of
            /// the object.
            static std::unordered_map<std::string, int> DENSITIES;

            /// @brief Maps the data type id to name
            static std::unordered_map<std::string, std::string> NAMES;
        };

        std::unordered_map<std::string, int> DataType::DENSITIES =
                {
                        {typeid(int).name(), 4}
                };

        std::unordered_map<std::string, std::string> DataType::NAMES =
                {
                        {typeid(int).name(), "int"}
                };

        /// @brief represents a 64-bit nanosecond-precision UTC timestamp.
        class TimeStamp {
        public:
            /// @property value holds the internal, primitive value of the timestamp.
            long value;

            /// @brief Constructs a timestamp from the given long, interpreting it as a nanosecond-precision UTC
            /// timestamp.
            TimeStamp(long value) : value(value) {}

            /// @brief Constructs a TimeStamp from another TimeStamp.
            TimeStamp(const TimeStamp &other) : value(other.value) {}

            bool operator==(const TimeStamp &other) const { return value == other.value; }

            bool operator!=(const TimeStamp &other) const { return value != other.value; }

            bool operator<(const TimeStamp &other) const { return value < other.value; }

            bool operator>(const TimeStamp &other) const { return value > other.value; }

            bool operator<=(const TimeStamp &other) const { return value <= other.value; }

            bool operator>=(const TimeStamp &other) const { return value >= other.value; }

            TimeStamp operator+(const TimeStamp &other) const { return TimeStamp(value + other.value); }

            TimeStamp operator-(const TimeStamp &other) const { return TimeStamp(value - other.value); }

            TimeStamp operator*(const TimeStamp &other) const { return TimeStamp(value * other.value); }

            TimeStamp operator/(const TimeStamp &other) const { return TimeStamp(value / other.value); }

            TimeStamp operator%(const TimeStamp &other) const { return TimeStamp(value % other.value); }

            TimeStamp operator+=(const TimeStamp &other) { return TimeStamp(value += other.value); }

            TimeStamp operator-=(const TimeStamp &other) { return TimeStamp(value -= other.value); }

            TimeStamp operator*=(const TimeStamp &other) { return TimeStamp(value *= other.value); }

            TimeStamp operator/=(const TimeStamp &other) { return TimeStamp(value /= other.value); }

            TimeStamp operator%=(const TimeStamp &other) { return TimeStamp(value %= other.value); }

            TimeStamp operator++() { return TimeStamp(++value); }

            TimeStamp operator--() { return TimeStamp(--value); }

            TimeStamp operator++(int) { return TimeStamp(value++); }

            TimeStamp operator--(int) { return TimeStamp(value--); }
        };


        class TimeRange {
        public:
            TimeStamp start;
            TimeStamp end;

            /// @brief constructs a TimeRange from the given start and end timestamps.
            TimeRange(TimeStamp start, TimeStamp end) : start(TimeStamp(start)), end(end) {}

            bool operator==(const TimeRange &other) const { return start == other.start && end == other.end; }

            /// @brief returns true if the given timestamp is within the range, start inclusive, end exclusive.
            bool contains(TimeStamp time) {
                return start <= time && time < end;
            }

            /// @brief returns true if the TimeRange contains the given TimeRange. If the two time ranges are equal,
            /// returns true. In this case, the two time ranges contain each other.
            bool contains(TimeRange tr) {
                return tr.start >= start && tr.end <= end;
            }
        };

        class Rate {
        public:
            explicit Rate(float i) : value(i) {
            }

            float value;
        };

        class TimeSpan {
        public:
            long value;

            explicit TimeSpan(long i) : value(i) {
            }

            TimeSpan operator+(const TimeSpan &other) const { return TimeSpan(value + other.value); }

            TimeSpan operator-(const TimeSpan &other) const { return TimeSpan(value - other.value); }

            TimeSpan operator*(const TimeSpan &other) const { return TimeSpan(value * other.value); }

            TimeSpan operator*(const long &other) const { return TimeSpan(value * other); }

            TimeSpan operator/(const TimeSpan &other) const { return TimeSpan(value / other.value); }
        };

        static const TimeSpan NANOSECOND = TimeSpan(1);
        static const TimeSpan MICROSECOND = NANOSECOND * 1000;
        static const TimeSpan MILLISECOND = MICROSECOND * 1000;
        static const TimeSpan SECOND = MILLISECOND * 1000;
        static const TimeSpan MINUTE = SECOND * 60;
        static const TimeSpan HOUR = MINUTE * 60;
        static const TimeSpan DAY = HOUR * 24;
    }


};

