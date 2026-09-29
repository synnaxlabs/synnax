// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <fstream>
#include <set>
#include <string>

#include <linux/serial.h>
#include <sys/ioctl.h>

#include "driver/serial/line.h"
#include "driver/serial/native.h"
#include "driver/serial/posix.h"
#include "driver/serial/scan.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
const std::array<tcflag_t, 4> SIZES{CS5, CS6, CS7, CS8};
}

namespace line {
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings) {
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE_AND_HALF)
        return {transport::CONFIG_ERROR, "Linux does not support 1.5 stop bits"};
    settings.c_cflag &= ~(CBAUD | (CBAUD << IBSHIFT));
    settings.c_cflag |= BOTHER | (BOTHER << IBSHIFT);
    settings.c_ispeed = props.baud_rate;
    settings.c_ospeed = props.baud_rate;
    settings.c_cflag &= ~CSIZE;
    settings.c_cflag |= SIZES.at(props.data_bits - 5);
    settings.c_cflag &= ~(PARENB | PARODD | CMSPAR);
    settings.c_iflag &= ~(IGNPAR | PARMRK | INPCK);
    const auto &parity = props.parity;
    if (parity == synnax::serial::PARITY_NONE_) {
        settings.c_iflag |= IGNPAR;
    } else {
        settings.c_iflag |= INPCK;
        settings.c_cflag |= PARENB;
    }
    if (parity == synnax::serial::PARITY_ODD_ || parity == synnax::serial::PARITY_MARK_)
        settings.c_cflag |= PARODD;
    if (parity == synnax::serial::PARITY_MARK_ ||
        parity == synnax::serial::PARITY_SPACE_)
        settings.c_cflag |= CMSPAR;
    if (props.stop_bits == synnax::serial::STOP_BITS_TWO)
        settings.c_cflag |= CSTOPB;
    else
        settings.c_cflag &= ~CSTOPB;
    settings.c_iflag &= ~(IXON | IXOFF);
    settings.c_cflag &= ~CRTSCTS;
    if (props.flow_control == synnax::serial::FLOW_CONTROL_SOFTWARE)
        settings.c_iflag |= IXON | IXOFF;
    if (props.flow_control == synnax::serial::FLOW_CONTROL_HARDWARE)
        settings.c_cflag |= CRTSCTS;
    return x::errors::NIL;
}
}

namespace native {
x::errors::Error
configure(const Handle handle, const synnax::serial::Properties &props) {
    const int fd = static_cast<int>(handle);
    line::Settings settings{};
    if (::ioctl(fd, TCGETS2, &settings) != 0)
        return failed("the line settings", props.port);
    if (auto err = line::apply(props, settings)) return err;
    if (::ioctl(fd, TCSETS2, &settings) != 0)
        return failed("the line settings", props.port);
    if (props.rs485) {
        serial_rs485 conf{};
        conf.flags = SER_RS485_ENABLED | SER_RS485_RTS_ON_SEND;
        if (::ioctl(fd, TIOCSRS485, &conf) != 0)
            return failed("RS-485 mode", props.port);
    }
    return x::errors::NIL;
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
