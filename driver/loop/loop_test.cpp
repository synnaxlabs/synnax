// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "driver/loop/loop.h"

// @brief it should correctly wait for an expended number of requests.
TEST(LoopTest, testPreciseTimer)
{
    loop::Timer timer;
    for(int i = 0; i < 50; i ++){
        auto start = std::chrono::high_resolution_clock::now();
        timer.exactSleep(std::chrono::nanoseconds(100000));
        auto end = std::chrono::high_resolution_clock::now();
        auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start).count();
        LOG(INFO) << "Elapsed time: " << elapsed/10000000 << " ns";
    }
}