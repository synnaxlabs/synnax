// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/can/types.gen.h"

#include "driver/can/backends/backends.h"

namespace driver::can::backends {
TEST(Backends, LoadsEveryBackendWithoutCrashing) {
    const auto backends = load();
    for (const auto *name:
         {synnax::can::BACKEND_SOCKETCAN,
          synnax::can::BACKEND_PCAN,
          synnax::can::BACKEND_GS_USB,
          synnax::can::BACKEND_SLCAN,
          synnax::can::BACKEND_CANLIB,
          synnax::can::BACKEND_NIXNET}) {
        ASSERT_TRUE(backends.contains(name)) << name;
        const auto [channels, err] = backends.at(name)->scan();
        if (err) { EXPECT_FALSE(err.message().empty()) << name; }
    }
    EXPECT_EQ(backends.size(), 6);
}
}
