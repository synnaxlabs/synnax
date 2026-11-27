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
#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/serial/list_linux.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace fs = std::filesystem;

namespace {
/// @brief ListLinux builds a sysfs tree and a device tree in a temporary folder.
class ListLinux : public ::testing::Test {
protected:
    fs::path root = fs::temp_directory_path() /
                    ("synnax-list-" + x::uuid::create().to_string());
    fs::path sys = this->root / "sys";
    fs::path dev = this->root / "dev";

    void SetUp() override {
        fs::create_directories(this->sys / "class" / "tty");
        fs::create_directories(this->dev);
    }

    void TearDown() override { fs::remove_all(this->root); }

    /// @brief writes text to the file at path, and makes its folder.
    static void write(const fs::path &path, const std::string &text) {
        fs::create_directories(path.parent_path());
        std::ofstream(path) << text << "\n";
    }

    /// @brief adds the tty name whose device is the folder device under sys/devices,
    /// and makes its node in dev. A UART port has a type, which is 0 when it holds no
    /// UART.
    void
    tty(const std::string &name,
        const fs::path &device,
        const std::optional<std::string> &type = std::nullopt) {
        const auto dir = this->sys / "devices" / device;
        const auto tty = this->sys / "class" / "tty" / name;
        fs::create_directories(dir);
        fs::create_directories(tty);
        fs::create_directory_symlink(dir, tty / "device");
        if (type) write(tty / "type", *type);
        write(this->dev / name, "");
    }

    /// @brief adds the USB device at the folder device under sys/devices, with product
    /// when it has one.
    void usb(const fs::path &device, const std::optional<std::string> &product) {
        write(this->sys / "devices" / device / "idVendor", "0403");
        if (product) write(this->sys / "devices" / device / "product", *product);
    }
};
}

/// @brief it should list each tty with a device, name it by its by-id link when it has
/// one, and give the product of its USB device as its description.
TEST_F(ListLinux, ListsPortsByLinkWithProduct) {
    this->usb("usb1/1-1", "FT232R USB UART");
    this->tty("ttyUSB0", "usb1/1-1/1-1:1.0/ttyUSB0");
    this->usb("usb1/1-2", std::nullopt);
    this->tty("ttyACM0", "usb1/1-2/1-2:1.0");
    this->tty("ttyS0", "pnp0/00:05/00:05:0/00:05:0.0", "4");
    fs::create_directories(this->sys / "class" / "tty" / "tty0");
    const auto link = this->dev / "serial" / "by-id" / "usb-FTDI_FT232R-if00-port0";
    fs::create_directories(link.parent_path());
    fs::create_symlink(this->dev / "ttyUSB0", link);
    const auto entries = ASSERT_NIL_P(list(this->sys, this->dev));
    EXPECT_EQ(
        entries,
        (std::vector<Entry>{
            {.path = link.string(), .description = "FT232R USB UART"},
            {.path = (this->dev / "ttyACM0").string()},
            {.path = (this->dev / "ttyS0").string()},
        })
    );
}

/// @brief it should leave out a UART port in which the kernel found no UART.
TEST_F(ListLinux, LeavesOutEmptyUARTPorts) {
    this->tty("ttyS1", "platform/serial8250/serial8250:0/serial8250:0.1", "0");
    EXPECT_TRUE(ASSERT_NIL_P(list(this->sys, this->dev)).empty());
}

/// @brief it should leave out a port with no node in the device tree, such as a port of
/// the host in a container.
TEST_F(ListLinux, LeavesOutPortsWithNoNode) {
    this->tty("ttyUSB1", "usb1/1-3/1-3:1.0/ttyUSB1");
    fs::remove(this->dev / "ttyUSB1");
    EXPECT_TRUE(ASSERT_NIL_P(list(this->sys, this->dev)).empty());
}

/// @brief it should fail when the sysfs tree is missing.
TEST_F(ListLinux, FailsWhenSysfsIsMissing) {
    const auto [entries, err] = list(this->root / "missing", this->dev);
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_TRUE(err.data().starts_with("failed to list serial ports: ")) << err.data();
}
}
