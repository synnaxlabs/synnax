// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "synnax/synnax.h"

const synnax::Config cfg = {
        .host =  "localhost",
        .port =  9090,
        .secure =  false,
        .username =  "synnax",
        .password =  "seldon"
};

/// @brief it should correctly write a frame of telemetry to the DB.
TEST(FramerTests, testWriteBasic) {
    auto client = synnax::Client(cfg);
    auto [time, tErr] = client.channels.create(
            "time",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr);
    auto [data, dErr] = client.channels.create(
            "data",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr);
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
        .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
        .start = synnax::TimeStamp(10 * synnax::SECOND),
        .authorities = std::vector<synnax::Authority>{synnax::ABSOLUTE},
        .subject = synnax::Subject{.name = "test_writer"},
    });
    ASSERT_FALSE(wErr);
}