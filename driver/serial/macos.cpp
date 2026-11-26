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
#include <cstdint>
#include <optional>
#include <string>
#include <system_error>
#include <utility>

#include <IOKit/serial/ioss.h>
#include <sys/ioctl.h>
#include <termios.h>

#include "driver/serial/line.h"
#include "driver/serial/native.h"
#include "driver/serial/posix.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @brief the rates that termios carries. The port sets any other rate through
/// IOSSIOSPEED.
const std::array<speed_t, 22> STANDARD_RATES{
    B50,    B75,    B110,   B134,   B150,    B200,    B300,   B600,
    B1200,  B1800,  B2400,  B4800,  B7200,   B9600,   B14400, B19200,
    B28800, B38400, B57600, B76800, B115200, B230400,
};

bool standard_rate(const std::uint32_t rate) {
    return std::ranges::find(STANDARD_RATES, static_cast<speed_t>(rate)) !=
           STANDARD_RATES.end();
}

const std::array<tcflag_t, 4> SIZES{CS5, CS6, CS7, CS8};
}

namespace line {
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings) {
    const auto &parity = props.parity;
    if (parity == synnax::serial::PARITY_MARK_ ||
        parity == synnax::serial::PARITY_SPACE_)
        return {transport::CONFIG_ERROR, "macOS does not support mark or space parity"};
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE_AND_HALF)
        return {transport::CONFIG_ERROR, "macOS does not support 1.5 stop bits"};
    if (standard_rate(props.baud_rate))
        ::cfsetspeed(&settings, static_cast<speed_t>(props.baud_rate));
    settings.c_cflag &= ~CSIZE;
    settings.c_cflag |= SIZES.at(props.data_bits - 5);
    settings.c_cflag &= ~(PARENB | PARODD);
    // The port marks parity, framing, and break errors in the input and removes them,
    // so it counts errors that IGNPAR would drop without a trace.
    settings.c_iflag &= ~(IGNPAR | ISTRIP);
    settings.c_iflag |= INPCK | PARMRK;
    if (parity != synnax::serial::PARITY_NONE_) settings.c_cflag |= PARENB;
    if (parity == synnax::serial::PARITY_ODD_) settings.c_cflag |= PARODD;
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
    if (::tcgetattr(fd, &settings) != 0) return failed("the line settings", props.port);
    if (auto err = line::apply(props, settings)) return err;
    if (::tcsetattr(fd, TCSANOW, &settings) != 0)
        return failed("the line settings", props.port);
    if (standard_rate(props.baud_rate)) return x::errors::NIL;
    speed_t speed = props.baud_rate;
    if (::ioctl(fd, IOSSIOSPEED, &speed) != 0)
        return failed("baud rate " + std::to_string(props.baud_rate), props.port);
    return x::errors::NIL;
}

// macOS has no API that reports receive overruns.
std::pair<std::optional<std::uint32_t>, std::error_code> overrun_count(int) {
    return {std::nullopt, {}};
}
}
}
