#pragma once

/// Local hdrs.
#include "synnax/telem/telem.h"
#include "telempb/telem.pb.h"

// std.
#include <string>
#include <vector>
#include <cstddef>
#include <typeinfo>

namespace Synnax {
    namespace Telem {

/// @brief Series type, able to hold generic types under the hood.
        class Series {
        public:
            Series(std::vector<std::any> vals) {
                data_type.setDataType(vals[0].type().name());
                data_type = DataType(vals[0].type().name());
                data = vals;
            }

            Series(const telempb::Series &s) {
                data_type = DataType(s.data_type());
                for (auto &val : s.data()) {
                    data.push_back(val);
                }
            }

            std::vector<std::any> &getRaw() {
                return data;
            }

            DataType &getDataType() {
                return data_type;
            }

            void to_proto(telempb::Series *s) const {
                s->set_data_type(data_type.value);
            }

        private:
            /// @brief Holds what type of data is being used.
            DataType data_type;

            /// @brief Holds the data.
            std::vector<std::any> data;
        };
    }
}