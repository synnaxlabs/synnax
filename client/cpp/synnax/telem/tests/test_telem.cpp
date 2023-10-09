/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "synnax/telem/series.h"

/// std.
#include <iostream>

/// @brief create basic int frame
TEST(TimeStampTests, testContructor)
{
    Synnax::Telem::TimeStamp ts{5};
    ASSERT_EQ(ts.get(), 5);
}