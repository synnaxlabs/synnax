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

TEST_F(PtyTest, RejectsMarkAndSpaceParity) {
    auto p = this->props();
    p.parity = synnax::serial::PARITY_MARK;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
    p.parity = synnax::serial::PARITY_SPACE;
    ASSERT_OCCURRED_AS_P(Port::open(p), transport::CONFIG_ERROR);
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
