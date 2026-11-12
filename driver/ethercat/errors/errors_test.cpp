// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::errors {
/// @brief it should put the first letter in upper case and a period at the end.
TEST(SentenceTest, CapitalizesAndEndsWithAPeriod) {
    EXPECT_EQ(sentence("failed to reach OP"), "Failed to reach OP.");
}

/// @brief it should not add a second period.
TEST(SentenceTest, KeepsOnePeriod) {
    EXPECT_EQ(sentence("Check the cable."), "Check the cable.");
}

/// @brief it should return empty text as it is.
TEST(SentenceTest, KeepsEmptyText) {
    EXPECT_EQ(sentence(""), "");
}
}
