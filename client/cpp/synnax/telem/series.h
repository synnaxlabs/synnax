#pragma once

/// Local hdrs.
#include "synnax/telem/telem.h"

// std.
#include <string>
#include <vector>
#include <cstddef>
#include <typeinfo>

namespace Synnax { namespace Telem {

/// @brief Series type, able to hold generic types under the hood.
class Series
{
public:
    Series(std::vector<std::any> vals)
    {
        // Use typeinfo to get name of type. 
        data_type.setDataType(vals[0].type().name());
        data = vals;
    }

    std::vector<std::any>& getRaw()
    {
        return data;
    }

    DataType &getDataType()
    {
        return data_type;
    }

private:
    /// @brief Holds what type of data is being used.
    DataType data_type;

    /// @brief Holds the data. 
    std::vector<std::any> data;
};

}};