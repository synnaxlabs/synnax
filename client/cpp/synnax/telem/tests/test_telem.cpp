/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "synnax/telem/telem.h"

/// std.

using namespace Synnax::Telem;

/// @brief - it should initialize a timestamp from a long.
TEST(TimeStampTests, testContructor)
{
    auto ts = TimeStamp(5);
    ASSERT_EQ(ts.value, 5);
}
