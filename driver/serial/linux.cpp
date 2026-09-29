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
#include <fstream>
#include <set>
#include <string>

#include <asm/termbits.h>
#include <linux/serial.h>
#include <sys/ioctl.h>

#include "driver/serial/native.h"
#include "driver/serial/scan.h"

namespace driver::serial {
namespace native {
namespace {
std::error_code last_error() {
    return {errno, std::generic_category()};
}

template<typename Edit>
std::error_code edit_termios(const Handle handle, Edit &&edit) {
    const int fd = static_cast<int>(handle);
    termios2 tio{};
    if (::ioctl(fd, TCGETS2, &tio) != 0) return last_error();
    edit(tio);
    if (::ioctl(fd, TCSETS2, &tio) != 0) return last_error();
    return {};
}
}

std::error_code set_flow_control(const Handle handle, const FlowControl mode) {
    return edit_termios(handle, [mode](termios2 &tio) {
        tio.c_iflag &= ~(IXON | IXOFF);
        tio.c_cflag &= ~CRTSCTS;
        if (mode == FlowControl::SOFTWARE) tio.c_iflag |= IXON | IXOFF;
        if (mode == FlowControl::HARDWARE) tio.c_cflag |= CRTSCTS;
    });
}

std::error_code set_baud_rate(const Handle handle, const std::uint32_t rate) {
    return edit_termios(handle, [rate](termios2 &tio) {
        tio.c_cflag &= ~(CBAUD | (CBAUD << IBSHIFT));
        tio.c_cflag |= BOTHER | (BOTHER << IBSHIFT);
        tio.c_ispeed = rate;
        tio.c_ospeed = rate;
    });
}

std::error_code set_mark_space_parity(const Handle handle, const bool mark) {
    return edit_termios(handle, [mark](termios2 &tio) {
        tio.c_iflag &= ~(IGNPAR | PARMRK);
        tio.c_iflag |= INPCK;
        tio.c_cflag |= PARENB | CMSPAR;
        if (mark)
            tio.c_cflag |= PARODD;
        else
            tio.c_cflag &= ~PARODD;
    });
}

std::error_code enable_rs485(const Handle handle) {
    serial_rs485 conf{};
    conf.flags = SER_RS485_ENABLED | SER_RS485_RTS_ON_SEND;
    if (::ioctl(static_cast<int>(handle), TIOCSRS485, &conf) != 0) return last_error();
    return {};
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    int queued = 0;
    if (::ioctl(static_cast<int>(handle), TIOCOUTQ, &queued) != 0)
        return {0, last_error()};
    return {static_cast<std::size_t>(queued), {}};
}
}

namespace {
/// @brief ttyS devices exist for every UART slot the kernel reserves. A slot without a
/// UART reports type 0 (PORT_UNKNOWN).
bool has_uart(const std::filesystem::path &root, const std::string &name) {
    std::ifstream type(root / "sys" / "class" / "tty" / name / "type");
    int value = -1;
    if (!(type >> value)) return true;
    return value != 0;
}

bool is_port(const std::string &name) {
    return name.starts_with("ttyS") || name.starts_with("ttyUSB") ||
           name.starts_with("ttyACM");
}

x::errors::Error
list_failed(const std::filesystem::path &dir, const std::error_code &ec) {
    return x::errors::Error(
        SCAN_ERROR,
        "failed to list " + dir.string() + ": " + ec.message()
    );
}
}

std::pair<std::vector<Info>, x::errors::Error> scan(const std::filesystem::path &root) {
    const auto dev = root / "dev";
    const auto by_id = dev / "serial" / "by-id";
    std::vector<Info> links;
    std::set<std::filesystem::path> linked;
    std::error_code ec;
    if (std::filesystem::exists(by_id, ec)) {
        std::filesystem::directory_iterator it(by_id, ec);
        if (ec) return {{}, list_failed(by_id, ec)};
        for (const auto &entry: it) {
            std::error_code dangling;
            const auto target = std::filesystem::canonical(entry.path(), dangling);
            if (dangling) continue;
            linked.insert(target);
            links.push_back({
                .path = entry.path().string(),
                .name = entry.path().filename().string(),
            });
        }
    }
    std::filesystem::directory_iterator it(dev, ec);
    if (ec) return {{}, list_failed(dev, ec)};
    std::vector<Info> devices;
    for (const auto &entry: it) {
        const auto name = entry.path().filename().string();
        if (!is_port(name)) continue;
        std::error_code missing;
        if (linked.contains(std::filesystem::canonical(entry.path(), missing)))
            continue;
        if (name.starts_with("ttyS") && !has_uart(root, name)) continue;
        devices.push_back({.path = entry.path().string(), .name = name});
    }
    std::ranges::sort(links, {}, &Info::path);
    std::ranges::sort(devices, {}, &Info::path);
    links.insert(links.end(), devices.begin(), devices.end());
    return {links, x::errors::NIL};
}
}
