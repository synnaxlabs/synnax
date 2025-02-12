// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/rack/rack.h"
#include "gtest/gtest.h"

TEST(CConfig, testDefault) {
    breaker::Breaker brk;
    auto c_err = rack::Config::clear_persisted_state(0, nullptr);
    ASSERT_FALSE(c_err) << c_err;
    auto [cfg, err] = rack::Config::load(0, nullptr, brk);
    ASSERT_TRUE(err) << err;
}