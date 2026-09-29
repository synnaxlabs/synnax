// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cerrno>

#include <IOKit/serial/ioss.h>
#include <sys/ioctl.h>
#include <termios.h>

#include "driver/serial/native.h"
#include "driver/serial/scan.h"

namespace driver::serial {
namespace native {
std::error_code set_flow_control(const Handle handle, const FlowControl mode) {
    const int fd = static_cast<int>(handle);
    termios tio{};
    if (::tcgetattr(fd, &tio) != 0) return {errno, std::generic_category()};
    tio.c_iflag &= ~(IXON | IXOFF);
    tio.c_cflag &= ~CRTSCTS;
    if (mode == FlowControl::SOFTWARE) tio.c_iflag |= IXON | IXOFF;
    if (mode == FlowControl::HARDWARE) tio.c_cflag |= CRTSCTS;
    if (::tcsetattr(fd, TCSANOW, &tio) != 0) return {errno, std::generic_category()};
    return {};
}

std::error_code set_baud_rate(const Handle handle, const std::uint32_t rate) {
    speed_t speed = rate;
    if (::ioctl(static_cast<int>(handle), IOSSIOSPEED, &speed) != 0)
        return {errno, std::generic_category()};
    return {};
}

std::error_code set_mark_space_parity(Handle, bool) {
    return std::make_error_code(std::errc::not_supported);
}

std::error_code enable_rs485(Handle) {
    return std::make_error_code(std::errc::not_supported);
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    int queued = 0;
    if (::ioctl(static_cast<int>(handle), TIOCOUTQ, &queued) != 0)
        return {0, {errno, std::generic_category()}};
    return {static_cast<std::size_t>(queued), {}};
}
}

std::pair<std::vector<Info>, x::errors::Error> scan(const std::filesystem::path &root) {
    const auto dev = root / "dev";
    std::error_code ec;
    std::filesystem::directory_iterator it(dev, ec);
    if (ec)
        return {
            {},
            x::errors::Error(
                SCAN_ERROR,
                "failed to list " + dev.string() + ": " + ec.message()
            )
        };
    std::vector<Info> ports;
    for (const auto &entry: it) {
        const auto name = entry.path().filename().string();
        if (name.starts_with("cu."))
            ports.push_back({.path = entry.path().string(), .name = name.substr(3)});
    }
    std::ranges::sort(ports, {}, &Info::path);
    return {ports, x::errors::NIL};
}
}
