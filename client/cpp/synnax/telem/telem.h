#pragma once


#include <unordered_map>
#include <string>
#include <any>

namespace Synnax {
    namespace Telem {

/// @brief Holds the name and properties of a datatype. 
        class DataType {
        public:
            void setDataType(std::string data_type_) {
                if (!DENSITIES.count(data_type_)) {
                    throw std::runtime_error("Tried to create an unknown datatype.");
                }
                data_type = data_type_;
            }

            /// @property Gets type name.
            std::string name() {
                return NAMES[data_type];
            }

            /// @property Essentially how many bytes in memory the datatype holds.
            int density() {
                return DENSITIES[data_type];
            }

        private:
            /// @brief Holds the id of the data type
            std::string data_type;

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

        class TimeStamp {
        public:
            long value;

            TimeStamp(long value) : value(value) {}
        };

        class TimeRange {
        public:
            TimeStamp end;
            TimeStamp start;

            TimeRange(TimeStamp start, TimeStamp end) : start(start), end(end) {}
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
        };
    }


};

