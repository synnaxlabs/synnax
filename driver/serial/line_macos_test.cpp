// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cstdint>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/line.h"
#include "driver/transport/errors.h"

namespace driver::serial::line {
namespace {
/// @returns settings with every flag that apply writes set, and a speed of 9600 baud.
Settings dirty() {
    Settings s{};
    s.c_iflag = IGNPAR | ISTRIP | IXON | IXOFF | ICRNL;
    s.c_oflag = OPOST;
    s.c_cflag = CS5 | PARENB | PARODD | CSTOPB | CRTSCTS | CREAD | CLOCAL;
    s.c_lflag = ICANON;
    s.c_cc[VMIN] = 1;
    ::cfsetspeed(&s, B9600);
    return s;
}
}

/// @brief it should apply seven data bits, odd parity, two stop bits, and hardware flow
/// control.
TEST(Line, AppliesSevenDataBitsOddParityTwoStopBitsAndHardwareFlowControl) {
    synnax::serial::Properties p;
    p.baud_rate = 19200;
    p.data_bits = 7;
    p.parity = synnax::serial::PARITY_ODD_;
    p.stop_bits = synnax::serial::STOP_BITS_TWO;
    p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(
        s.c_cflag,
        static_cast<tcflag_t>(CS7 | PARENB | PARODD | CSTOPB | CRTSCTS)
    );
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(INPCK | PARMRK));
    EXPECT_EQ(::cfgetispeed(&s), static_cast<speed_t>(B19200));
    EXPECT_EQ(::cfgetospeed(&s), static_cast<speed_t>(B19200));
}

/// @brief it should clear parity, the second stop bit, and flow control for the
/// defaults.
TEST(Line, ClearsParityStopBitsAndFlowControlForTheDefaults) {
    auto s = dirty();
    ASSERT_NIL(apply(synnax::serial::Properties{}, s));
    EXPECT_EQ(s.c_cflag, static_cast<tcflag_t>(CS8 | CREAD | CLOCAL));
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(INPCK | PARMRK | ICRNL));
    EXPECT_EQ(::cfgetospeed(&s), static_cast<speed_t>(B9600));
}

/// @brief it should keep the flags that it does not write.
TEST(Line, KeepsTheFlagsItDoesNotOwn) {
    auto s = dirty();
    ASSERT_NIL(apply(synnax::serial::Properties{}, s));
    EXPECT_EQ(s.c_oflag, static_cast<tcflag_t>(OPOST));
    EXPECT_EQ(s.c_lflag, static_cast<tcflag_t>(ICANON));
    EXPECT_EQ(s.c_cc[VMIN], static_cast<cc_t>(1));
}

/// @brief it should apply each character size from 5 to 8 data bits.
TEST(Line, AppliesEachCharacterSize) {
    const std::array<std::pair<std::uint8_t, tcflag_t>, 4> sizes{
        {{5, CS5}, {6, CS6}, {7, CS7}, {8, CS8}}
    };
    for (const auto &[bits, size]: sizes) {
        synnax::serial::Properties p;
        p.data_bits = bits;
        auto s = dirty();
        ASSERT_NIL(apply(p, s));
        EXPECT_EQ(s.c_cflag & CSIZE, size);
    }
}

/// @brief it should apply software flow control.
TEST(Line, AppliesSoftwareFlowControl) {
    synnax::serial::Properties p;
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(INPCK | PARMRK | IXON | IXOFF | ICRNL));
    EXPECT_EQ(s.c_cflag & CRTSCTS, static_cast<tcflag_t>(0));
}

/// @brief it should apply even parity.
TEST(Line, AppliesEvenParity) {
    synnax::serial::Properties p;
    p.parity = synnax::serial::PARITY_EVEN_;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.c_cflag & (PARENB | PARODD), static_cast<tcflag_t>(PARENB));
}

/// @brief it should leave the speed unchanged for a baud rate outside the termios
/// table.
TEST(Line, LeavesTheSpeedForABaudRateOutsideTheTermiosTable) {
    synnax::serial::Properties p;
    p.baud_rate = 250000;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(::cfgetispeed(&s), static_cast<speed_t>(B9600));
    EXPECT_EQ(::cfgetospeed(&s), static_cast<speed_t>(B9600));
}

/// @brief it should refuse mark and space parity.
TEST(Line, RejectsMarkAndSpaceParity) {
    synnax::serial::Properties p;
    Settings s{};
    for (const auto *parity:
         {synnax::serial::PARITY_MARK_, synnax::serial::PARITY_SPACE_}) {
        p.parity = parity;
        const auto err = apply(p, s);
        ASSERT_MATCHES(err, transport::CONFIG_ERROR);
        EXPECT_EQ(err.data, "macOS does not support mark or space parity");
    }
}

/// @brief it should refuse 1.5 stop bits.
TEST(Line, RejectsOneAndAHalfStopBits) {
    synnax::serial::Properties p;
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    Settings s{};
    const auto err = apply(p, s);
    ASSERT_MATCHES(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "macOS does not support 1.5 stop bits");
}
}
