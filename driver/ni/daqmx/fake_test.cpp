// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ni/daqmx/fake.h"
#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::daqmx {
TEST(FakeAPI, RecordsCallsThroughSugaredAPI) {
    auto fake = std::make_shared<FakeAPI>();
    SugaredAPI dmx(fake);
    ASSERT_FALSE(dmx.CreateLinScale("scale_0", 2.0, 1.0, DAQmx_Val_Volts, "Pascals"));
    const auto calls = fake->calls_to("CreateLinScale");
    ASSERT_EQ(calls.size(), 1);
    EXPECT_EQ(calls[0]["args"]["name"], "scale_0");
    EXPECT_EQ(calls[0]["args"]["slope"], 2.0);
}
}
