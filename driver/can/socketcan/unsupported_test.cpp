// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/socketcan/socketcan.h"

namespace driver::can::socketcan {
TEST(SocketCANUnsupported, ReportsThePlatform) {
    const auto backend = load();
    const auto [channels, scan_err] = backend->scan();
    ASSERT_MATCHES(scan_err, UNSUPPORTED_ERROR);
    EXPECT_EQ(scan_err.data, "SocketCAN runs only on Linux");
    synnax::can::Properties props;
    props.backend = synnax::can::BACKEND_SOCKETCAN;
    props.channel = "can0";
    const auto [bus, open_err] = backend->open(props);
    ASSERT_MATCHES(open_err, UNSUPPORTED_ERROR);
    EXPECT_EQ(bus, nullptr);
}
}
