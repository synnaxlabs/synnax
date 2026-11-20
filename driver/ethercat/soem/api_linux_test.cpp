// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
/// @brief it should find no USB adapter for the loopback or a missing interface.
TEST(ProdAPILinux, FindsNoUSBAdapterForLoopbackOrAMissingInterface) {
    ProdAPI api;
    EXPECT_FALSE(api.on_usb("lo"));
    EXPECT_FALSE(api.on_usb("no-such-interface"));
}
}
