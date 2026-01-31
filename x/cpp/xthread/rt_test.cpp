// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"
#include "x/cpp/xthread/rt.h"

TEST(RTConfigTest, DefaultConfig) {
    xthread::RTConfig cfg;
    EXPECT_FALSE(cfg.enabled);
    EXPECT_EQ(cfg.priority, xthread::DEFAULT_RT_PRIORITY);
    EXPECT_EQ(cfg.cpu_affinity, xthread::CPU_AFFINITY_NONE);
    EXPECT_FALSE(cfg.lock_memory);
}

TEST(RTConfigTest, ApplyEmptyConfig) {
    xthread::RTConfig cfg;
    ASSERT_NIL(xthread::apply_rt_config(cfg));
}

TEST(RTConfigTest, ApplyWithRTEnabled) {
    xthread::RTConfig cfg;
    cfg.enabled = true;
    cfg.priority = 50;
    ASSERT_NIL(xthread::apply_rt_config(cfg));
}

TEST(RTConfigTest, HasRTSupportReturns) {
    [[maybe_unused]] bool supported = xthread::has_rt_support();
}
