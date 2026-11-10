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
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

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

/// @brief gives the tty a device link, as the kernel does for a tty that hardware
/// backs.
void link_device(const std::filesystem::path &root, const std::string &name) {
    const auto dir = root / "sys" / "class" / "tty" / name;
    std::filesystem::create_directories(dir);
    const auto device = root / "sys" / "devices" / name;
    std::filesystem::create_directories(device);
    std::filesystem::create_directory_symlink(device, dir / "device");
}
}

TEST(Scan, ListsByIdLinksFirstAndHidesTheDevicesTheyReach) {
    const auto root = std::filesystem::path(::testing::TempDir()) / "linux_scan";
    std::filesystem::remove_all(root);
    const auto dev = root / "dev";
    for (const auto *name: {"ttyUSB0", "ttyUSB1", "ttyACM0", "ttyS0", "ttyS1", "tty0"})
        touch(dev / name);
    for (const auto *name: {"ttyUSB0", "ttyUSB1", "ttyACM0", "ttyS0", "ttyS1"})
        link_device(root, name);
    std::filesystem::create_directories(root / "sys" / "class" / "tty" / "tty0");
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

TEST(Scan, FindsEveryTtyThatHardwareBacks) {
    const auto root = std::filesystem::path(::testing::TempDir()) / "linux_scan_any";
    std::filesystem::remove_all(root);
    const auto dev = root / "dev";
    for (const auto *name: {"ttyAMA0", "ttymxc1", "ttyTHS2", "ttyXRUSB0", "ttyGS0"}) {
        touch(dev / name);
        link_device(root, name);
    }
    touch(dev / "ttyp0");
    std::filesystem::create_directories(root / "sys" / "class" / "tty" / "ttyp0");
    link_device(root, "ttyUSB7");
    const auto ports = ASSERT_NIL_P(scan(root));
    std::vector<std::string> names;
    for (const auto &p: ports)
        names.push_back(p.name);
    EXPECT_EQ(
        names,
        (std::vector<
            std::string>{"ttyAMA0", "ttyGS0", "ttyTHS2", "ttyXRUSB0", "ttymxc1"})
    );
}

TEST(Scan, FailsWhenTheDeviceDirectoryIsMissing) {
    ASSERT_OCCURRED_AS_P(scan("/synnax-missing-root"), SCAN_ERROR);
}

TEST(Scan, ListsTheHostsPorts) {
    ASSERT_NIL_P(scan());
}
}
