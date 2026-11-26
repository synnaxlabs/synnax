// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include <termios.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/transport/errors.h"

namespace driver::serial {
/// @brief it should refuse a baud rate outside the termios table on a terminal that
/// cannot set it.
TEST_F(PtyTest, RejectsACustomBaudRateThePortCannotSet) {
    auto p = this->props();
    p.baud_rate = 250000;
    const auto [port, err] = Port::open(p);
    ASSERT_OCCURRED_AS(err, transport::CONFIG_ERROR);
    EXPECT_TRUE(
        err.data.starts_with("failed to set baud rate 250000 on " + p.port + ": ")
    ) << err.data;
}

/// @brief it should apply the line settings to the terminal.
TEST_F(PtyTest, AppliesTheLineSettings) {
    auto p = this->props();
    p.baud_rate = 19200;
    p.data_bits = 7;
    p.parity = synnax::serial::PARITY_ODD_;
    p.stop_bits = synnax::serial::STOP_BITS_TWO;
    p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
    const auto port = ASSERT_NIL_P(Port::open(p));
    const auto tio = this->termios();
    EXPECT_EQ(cfgetospeed(&tio), static_cast<speed_t>(B19200));
    EXPECT_EQ(tio.c_cflag & CSIZE, static_cast<tcflag_t>(CS7));
    EXPECT_TRUE(tio.c_cflag & PARENB);
    EXPECT_TRUE(tio.c_cflag & PARODD);
    EXPECT_TRUE(tio.c_cflag & CSTOPB);
    EXPECT_TRUE(tio.c_cflag & CRTSCTS);
    EXPECT_FALSE(tio.c_iflag & (IXON | IXOFF));
}

/// @brief it should apply software flow control and no parity to the terminal.
TEST_F(PtyTest, AppliesSoftwareFlowControlAndNoParity) {
    auto p = this->props();
    p.baud_rate = 115200;
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    const auto port = ASSERT_NIL_P(Port::open(p));
    const auto tio = this->termios();
    EXPECT_EQ(cfgetospeed(&tio), static_cast<speed_t>(B115200));
    EXPECT_EQ(tio.c_cflag & CSIZE, static_cast<tcflag_t>(CS8));
    EXPECT_FALSE(tio.c_cflag & PARENB);
    EXPECT_FALSE(tio.c_cflag & CSTOPB);
    EXPECT_FALSE(tio.c_cflag & CRTSCTS);
    EXPECT_TRUE(tio.c_iflag & IXON);
    EXPECT_TRUE(tio.c_iflag & IXOFF);
}
}
