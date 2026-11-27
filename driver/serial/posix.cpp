// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstddef>
#include <string>
#include <system_error>
#include <utility>

#include "asio/error.hpp"
#include "asio/serial_port.hpp"
#include <fcntl.h>
#include <sys/file.h>
#include <sys/ioctl.h>
#include <termios.h>
#include <unistd.h>

#include "driver/serial/device.h"
#include "driver/serial/errors.h"
#include "driver/serial/native.h"
#include "driver/serial/posix.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
std::error_code last_error() {
    return {errno, std::generic_category()};
}

/// @brief refuses the port at fd when another program holds it, then locks it. A macOS
/// pseudo-terminal lets TIOCEXCL stop only root, so a Driver without root opens one
/// that another program holds with TIOCEXCL alone.
x::errors::Error lock(const int fd, const std::string &context) {
    const auto busy = std::make_error_code(std::errc::device_or_resource_busy);
    // Root gets past TIOCEXCL, and the port must not clear a flag that it did not set.
    const auto [held, ec] = native::exclusive(fd);
    if (ec) return translate(context, ec);
    if (held) return translate(context, busy);
    // flock stops programs that lock the port, and root, which TIOCEXCL lets through.
    // TIOCEXCL stops every other open by a program without root.
    if (::flock(fd, LOCK_EX | LOCK_NB) != 0) {
        const auto locked = last_error();
        return translate(
            context,
            locked == std::errc::operation_would_block ? busy : locked
        );
    }
    if (::ioctl(fd, TIOCEXCL) != 0) return translate(context, last_error());
    return x::errors::NIL;
}
}

x::errors::Error
translate(const std::string &context, const std::error_code &reported) {
    const auto ec = transport::from_asio(reported);
    if (ec == std::errc::device_or_resource_busy)
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            context + ": the port is in use by another program"
        );
    // A tty reads end of file once its device hangs up.
    if (ec == asio::error::eof)
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    if (transport::matches(
            ec,
            {std::errc::no_such_file_or_directory,
             std::errc::no_such_device_or_address,
             std::errc::no_such_device,
             std::errc::io_error,
             std::errc::broken_pipe,
             std::errc::bad_file_descriptor,
             std::errc::timed_out}
        ))
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    if (transport::matches(
            ec,
            {std::errc::permission_denied,
             std::errc::operation_not_permitted,
             std::errc::invalid_argument,
             std::errc::inappropriate_io_control_operation,
             std::errc::is_a_directory,
             std::errc::not_a_directory,
             std::errc::filename_too_long,
             std::errc::not_supported}
        ))
        return transport::error(transport::CONFIG_ERROR, context, ec);
    return transport::error(transport::CRITICAL_ERROR, context, ec);
}

namespace device {
x::errors::Error open(asio::serial_port &device, const std::string &port) {
    const auto context = "failed to open serial port " + port;
    // The port opens the descriptor itself because Asio's open changes the line
    // settings before the port can lock it.
    const int fd = ::open(port.c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK | O_CLOEXEC);
    if (fd < 0) return translate(context, last_error());
    std::error_code ec;
    if (device.assign(fd, ec)) {
        ::close(fd);
        return translate(context, ec);
    }
    if (auto err = lock(fd, context)) {
        // NOLINTNEXTLINE(bugprone-unused-return-value): a failed close still frees it.
        device.close(ec);
        return err;
    }
    return x::errors::NIL;
}
}

namespace native {
x::errors::Error failed(const std::string &setting, const std::string &port) {
    return translate("failed to set " + setting + " on " + port, last_error());
}

void discard_output(const Handle handle) {
    ::tcflush(static_cast<int>(handle), TCOFLUSH);
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    int queued = 0;
    if (::ioctl(static_cast<int>(handle), TIOCOUTQ, &queued) != 0)
        return {0, last_error()};
    return {static_cast<std::size_t>(queued), {}};
}
}
}
