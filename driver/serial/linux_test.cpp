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
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/serial/scan.h"

namespace driver::serial {
namespace {
void touch(const std::filesystem::path &path) {
    std::filesystem::create_directories(path.parent_path());
    std::ofstream(path).put('\n');
}

void write_type(const std::filesystem::path &root, const std::string &name, int type) {
    const auto dir = root / "sys" / "class" / "tty" / name;
    std::filesystem::create_directories(dir);
    std::ofstream(dir / "type") << type << '\n';
}
}

TEST(Scan, ListsByIdLinksFirstAndHidesTheDevicesTheyReach) {
    const auto root = std::filesystem::path(::testing::TempDir()) / "linux_scan";
    std::filesystem::remove_all(root);
    const auto dev = root / "dev";
    for (const auto *name: {"ttyUSB0", "ttyUSB1", "ttyACM0", "ttyS0", "ttyS1", "tty0"})
        touch(dev / name);
    write_type(root, "ttyS0", 0);
    write_type(root, "ttyS1", 4);
    const auto by_id = dev / "serial" / "by-id";
    std::filesystem::create_directories(by_id);
    std::filesystem::create_symlink("../../ttyUSB0", by_id / "usb-FTDI_FT232R-if00");
    std::filesystem::create_symlink("../../ttyUSB9", by_id / "usb-gone-if00");
    const auto ports = ASSERT_NIL_P(scan(root));
    ASSERT_EQ(ports.size(), 4);
    EXPECT_EQ(ports[0].path, (by_id / "usb-FTDI_FT232R-if00").string());
    EXPECT_EQ(ports[0].name, "usb-FTDI_FT232R-if00");
    EXPECT_EQ(ports[1].path, (dev / "ttyACM0").string());
    EXPECT_EQ(ports[2].path, (dev / "ttyS1").string());
    EXPECT_EQ(ports[3].path, (dev / "ttyUSB1").string());
    EXPECT_EQ(ports[3].name, "ttyUSB1");
}

TEST(Scan, FailsWhenTheDeviceDirectoryIsMissing) {
    ASSERT_OCCURRED_AS_P(scan("/synnax-missing-root"), SCAN_ERROR);
}

TEST(Scan, ListsTheHostsPorts) {
    ASSERT_NIL_P(scan());
}
}
