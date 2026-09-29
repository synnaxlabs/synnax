// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>

#include "gtest/gtest.h"
#include <termios.h>

#include "x/cpp/test/test.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/serial/scan.h"

namespace driver::serial {
/// A pty has no IOSSIOSPEED, so a rate outside the termios table must fail loud.
TEST_F(PtyTest, RejectsACustomBaudRateThePortCannotSet) {
    auto p = this->props();
    p.baud_rate = 250000;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
}

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

TEST(Scan, ListsCallOutDevicesSortedByPath) {
    const auto root = std::filesystem::path(::testing::TempDir()) / "macos_scan";
    std::filesystem::remove_all(root);
    std::filesystem::create_directories(root / "dev");
    for (const auto *name:
         {"cu.usbserial-1", "tty.usbserial-1", "cu.Bluetooth-Incoming-Port", "null"})
        std::ofstream(root / "dev" / name).put('\n');
    const auto ports = ASSERT_NIL_P(scan(root));
    ASSERT_EQ(ports.size(), 2);
    EXPECT_EQ(ports[0].path, (root / "dev" / "cu.Bluetooth-Incoming-Port").string());
    EXPECT_EQ(ports[0].name, "Bluetooth-Incoming-Port");
    EXPECT_EQ(ports[1].path, (root / "dev" / "cu.usbserial-1").string());
    EXPECT_EQ(ports[1].name, "usbserial-1");
}

TEST(Scan, FailsWhenTheDeviceDirectoryIsMissing) {
    ASSERT_OCCURRED_AS_P(scan("/synnax-missing-root"), SCAN_ERROR);
}

TEST(Scan, ListsTheHostsPorts) {
    ASSERT_NIL_P(scan());
}
}
