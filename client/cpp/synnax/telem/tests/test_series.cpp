/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "synnax/telem/series.h"

/// std.
#include <iostream>

/// @brief create basic int frame
TEST(FrameTests, testString)
{
    std::vector<std::any> vals;
    vals.push_back(5);
    Synnax::Telem::Series s{vals};
    std::vector<std::any> raw_vals = s.getRaw();
    auto type_name = s.getDataType().name();
    ASSERT_EQ(type_name, "int");
}