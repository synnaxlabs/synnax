// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest.
#include <gtest/gtest.h>

/// Local headers.
#include "driver/breaker/breaker.h"

// @brief it should correctly wait for an expended number of requests.
TEST(BreakerTests, testBreaker)
{
    auto b = breaker::Breaker(breaker::Config{"my-breaker", synnax::TimeSpan(1), 1, 1});
    ASSERT_TRUE(b.wait());
    ASSERT_FALSE(b.wait());
}