// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/test/test.h"

#include "driver/arinc429/ddc/backend.h"

namespace driver::arinc429::ddc {
TEST(FromCard, KeepsAWordThatArrivedWithOddParity) {
    const auto word = from_card(0x918C440D & 0x7FFFFFFF);
    EXPECT_EQ(word, codec::arinc429::Word(0x918C440D));
    EXPECT_TRUE(word.parity_valid());
}

TEST(FromCard, RestoresTheEvenParityOfAWordFlaggedAsAnError) {
    const auto word = from_card(0x918C440D | 0x80000000);
    EXPECT_EQ(word, codec::arinc429::Word(0x118C440D));
    EXPECT_FALSE(word.parity_valid());
}

TEST(ToCard, ClearsTheParityBitForTheCardToGenerate) {
    EXPECT_EQ(to_card(codec::arinc429::Word(0x918C440D)), 0x118C440D);
}

TEST(Backend, RejectsAnUnknownSpeed) {
    Backend b;
    const auto [ch, err] = b.open({.speed = "medium"}, Direction::RECEIVE);
    ASSERT_OCCURRED_AS(err, errors::CONFIGURATION_ERROR);
    EXPECT_EQ(err.data, "unknown ARINC 429 speed medium");
}

TEST(Backend, FailsToOpenWithoutTheSDK) {
    Backend b;
    const auto [ch, err] = b.open({}, Direction::RECEIVE);
    ASSERT_OCCURRED_AS(err, x::lib::LOAD_ERROR);
    EXPECT_EQ(err.data, "DDC DD-42992 ARINC 429 SDK library is not installed.");
}

TEST(Backend, FailsToListWithoutTheSDK) {
    Backend b;
    ASSERT_OCCURRED_AS_P(b.list(), x::lib::LOAD_ERROR);
}
}
