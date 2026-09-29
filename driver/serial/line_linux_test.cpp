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

#include "driver/serial/line.h"
#include "driver/transport/errors.h"

namespace driver::serial::line {
namespace {
/// @returns settings with every flag that apply writes set, and a speed of 9600 baud.
Settings dirty() {
    Settings s{};
    s.c_iflag = IGNPAR | PARMRK | INPCK | IXON | IXOFF | ICRNL;
    s.c_oflag = OPOST;
    s.c_cflag = B9600 | (B9600 << IBSHIFT) | CS5 | PARENB | PARODD | CMSPAR | CSTOPB |
                CRTSCTS | CREAD | CLOCAL;
    s.c_lflag = ICANON;
    s.c_cc[VMIN] = 1;
    s.c_ispeed = 9600;
    s.c_ospeed = 9600;
    return s;
}
}

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
        static_cast<tcflag_t>(
            BOTHER | (BOTHER << IBSHIFT) | CS7 | PARENB | PARODD | CSTOPB | CRTSCTS
        )
    );
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(INPCK));
    EXPECT_EQ(s.c_ispeed, 19200u);
    EXPECT_EQ(s.c_ospeed, 19200u);
}

TEST(Line, ClearsParityStopBitsAndFlowControlForTheDefaults) {
    auto s = dirty();
    ASSERT_NIL(apply(synnax::serial::Properties{}, s));
    EXPECT_EQ(
        s.c_cflag,
        static_cast<tcflag_t>(BOTHER | (BOTHER << IBSHIFT) | CS8 | CREAD | CLOCAL)
    );
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(IGNPAR | ICRNL));
    EXPECT_EQ(s.c_ospeed, 9600u);
}

TEST(Line, KeepsTheFlagsItDoesNotOwn) {
    auto s = dirty();
    ASSERT_NIL(apply(synnax::serial::Properties{}, s));
    EXPECT_EQ(s.c_oflag, static_cast<tcflag_t>(OPOST));
    EXPECT_EQ(s.c_lflag, static_cast<tcflag_t>(ICANON));
    EXPECT_EQ(s.c_cc[VMIN], static_cast<cc_t>(1));
}

TEST(Line, AppliesSoftwareFlowControl) {
    synnax::serial::Properties p;
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(IGNPAR | IXON | IXOFF | ICRNL));
    EXPECT_FALSE(s.c_cflag & CRTSCTS);
}

TEST(Line, AppliesEvenParity) {
    synnax::serial::Properties p;
    p.parity = synnax::serial::PARITY_EVEN_;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.c_cflag & (PARENB | PARODD | CMSPAR), static_cast<tcflag_t>(PARENB));
    EXPECT_EQ(s.c_iflag & (IGNPAR | PARMRK | INPCK), static_cast<tcflag_t>(INPCK));
}

TEST(Line, AppliesMarkParity) {
    synnax::serial::Properties p;
    p.parity = synnax::serial::PARITY_MARK_;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(
        s.c_cflag & (PARENB | PARODD | CMSPAR),
        static_cast<tcflag_t>(PARENB | PARODD | CMSPAR)
    );
    EXPECT_EQ(s.c_iflag, static_cast<tcflag_t>(INPCK));
}

TEST(Line, AppliesSpaceParity) {
    synnax::serial::Properties p;
    p.parity = synnax::serial::PARITY_SPACE_;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(
        s.c_cflag & (PARENB | PARODD | CMSPAR),
        static_cast<tcflag_t>(PARENB | CMSPAR)
    );
}

TEST(Line, CarriesABaudRateOutsideTheTermiosTable) {
    synnax::serial::Properties p;
    p.baud_rate = 250000;
    auto s = dirty();
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(
        s.c_cflag & (CBAUD | (CBAUD << IBSHIFT)),
        static_cast<tcflag_t>(BOTHER | (BOTHER << IBSHIFT))
    );
    EXPECT_EQ(s.c_ispeed, 250000u);
    EXPECT_EQ(s.c_ospeed, 250000u);
}

TEST(Line, RejectsOneAndAHalfStopBits) {
    synnax::serial::Properties p;
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    Settings s{};
    const auto err = apply(p, s);
    ASSERT_MATCHES(err, transport::CONFIG_ERROR);
    EXPECT_EQ(err.data, "Linux does not support 1.5 stop bits");
}
}
