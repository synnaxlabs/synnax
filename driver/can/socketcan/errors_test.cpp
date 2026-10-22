// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <system_error>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/socketcan/socketcan.h"

namespace driver::can::socketcan {
TEST(SocketCANTranslate, TreatsAGoneOrDownInterfaceAsUnreachable) {
    for (const int code: {ENODEV, ENXIO, ENETDOWN})
        ASSERT_MATCHES(translate(code, "can0"), UNREACHABLE_ERROR);
}

TEST(SocketCANTranslate, KeepsTheBusOpenWhenTheTransmitQueueIsFull) {
    const auto err = translate(ENOBUFS, "can0: failed to send a frame");
    ASSERT_MATCHES(err, TEMPORARY_HARDWARE_ERROR);
    EXPECT_FALSE(err.matches(UNREACHABLE_ERROR));
    EXPECT_EQ(
        err.data,
        "can0: failed to send a frame: " + std::generic_category().message(ENOBUFS)
    );
}

TEST(SocketCANTranslate, TreatsARefusedSocketAsAConfigError) {
    for (const int code: {EAFNOSUPPORT, EPERM, EINVAL})
        ASSERT_MATCHES(translate(code, "can0"), CONFIG_ERROR);
}

TEST(SocketCANTranslate, TreatsAnyOtherErrorAsCritical) {
    ASSERT_MATCHES(translate(EBADF, "can0"), CRITICAL_HARDWARE_ERROR);
}
}
