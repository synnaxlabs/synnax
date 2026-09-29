// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"
#include <termios.h>

#include "x/cpp/test/test.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"

namespace driver::serial {
TEST_F(PtyTest, ReadsWhatThePeerWrites) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const auto before = x::telem::TimeStamp::now();
    this->peer_write("hello");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "hello");
    EXPECT_GE(chunk.time, before);
    EXPECT_LE(chunk.time, x::telem::TimeStamp::now());
}

TEST_F(PtyTest, AnswersAQueryInARoundTrip) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_NIL(port->write(bytes("MEAS:VOLT?\n"), x::telem::SECOND));
    EXPECT_EQ(this->peer_read(11), "MEAS:VOLT?\n");
    this->peer_write("12.5\n");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "12.5\n");
}

TEST_F(PtyTest, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(port->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_F(PtyTest, ReadsAgainAfterATimeout) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    ASSERT_TRUE(ASSERT_NIL_P(port->read(10 * x::telem::MILLISECOND)).data.empty());
    this->peer_write("late");
    const auto chunk = ASSERT_NIL_P(port->read(x::telem::SECOND));
    EXPECT_EQ(as_string(chunk.data), "late");
}

TEST_F(PtyTest, AppliesTheLineSettings) {
    auto p = this->props();
    p.baud_rate = 19200;
    p.data_bits = 7;
    p.parity = synnax::serial::PARITY_ODD;
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

TEST_F(PtyTest, RejectsOneAndAHalfStopBits) {
    auto p = this->props();
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
}

TEST_F(PtyTest, RejectsRS485OnAPortWithoutIt) {
    auto p = this->props();
    p.rs485 = true;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
}

TEST_F(PtyTest, ReportsTheDeviceAsUnreachableWhenThePeerCloses) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    this->close_peer();
    ASSERT_OCCURRED_AS_P(port->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
}

TEST_F(PtyTest, FailsToReadAfterClose) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    port->close();
    ASSERT_OCCURRED_AS_P(port->read(x::telem::SECOND), transport::UNREACHABLE_ERROR);
}

TEST(Port, FailsToOpenAMissingPort) {
    synnax::serial::Properties p;
    p.port = "/dev/synnax-missing-port";
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::UNREACHABLE_ERROR);
}

TEST(Port, RejectsInvalidSettingsBeforeOpening) {
    const auto invalid = [](auto edit) {
        synnax::serial::Properties p;
        p.port = "/dev/synnax-missing-port";
        edit(p);
        return Port::open(p);
    };
    using P = synnax::serial::Properties;
    ASSERT_OCCURRED_AS_P(invalid([](P &p) { p.port = ""; }), transport::CONFIG_ERROR);
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.baud_rate = 0; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.data_bits = 9; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.parity = "bogus"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.stop_bits = "3"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) { p.flow_control = "bogus"; }),
        transport::CONFIG_ERROR
    );
    ASSERT_OCCURRED_AS_P(
        invalid([](P &p) {
            p.rs485 = true;
            p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
        }),
        transport::CONFIG_ERROR
    );
}
}
