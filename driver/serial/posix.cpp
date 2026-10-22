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
#include <sys/file.h>
#include <sys/ioctl.h>

#include "driver/serial/errors.h"
#include "driver/serial/native.h"
#include "driver/serial/posix.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
std::error_code last_error() {
    return {errno, std::generic_category()};
}
}

x::errors::Error
translate(const std::string &context, const std::error_code &reported) {
    using transport::any_of;
    const auto ec = transport::from_asio(reported);
    if (ec == std::errc::device_or_resource_busy)
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            context + ": the port is in use by another program"
        );
    // A tty reads end of file once its device hangs up.
    if (ec == asio::error::eof)
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    if (any_of(
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
    if (any_of(
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

namespace native {
x::errors::Error failed(const std::string &setting, const std::string &port) {
    return translate("failed to set " + setting + " on " + port, last_error());
}

x::errors::Error lock(const Handle handle, const std::string &port) {
    const int fd = static_cast<int>(handle);
    // flock stops programs that lock the port, and root, which TIOCEXCL lets through.
    // TIOCEXCL stops every other open by a program without root.
    if (::flock(fd, LOCK_EX | LOCK_NB) != 0) {
        auto ec = last_error();
        if (ec == std::errc::operation_would_block)
            ec = std::make_error_code(std::errc::device_or_resource_busy);
        return translate("failed to open serial port " + port, ec);
    }
    if (::ioctl(fd, TIOCEXCL) != 0)
        return translate("failed to open serial port " + port, last_error());
    return x::errors::NIL;
}

void unlock(const Handle handle) {
    // A pseudo-terminal keeps TIOCEXCL after its last close while its controller is
    // open, so the port clears it.
    ::ioctl(static_cast<int>(handle), TIOCNXCL);
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    int queued = 0;
    if (::ioctl(static_cast<int>(handle), TIOCOUTQ, &queued) != 0)
        return {0, last_error()};
    return {static_cast<std::size_t>(queued), {}};
}

std::pair<std::size_t, std::size_t>
strip(std::uint8_t &marked, const std::span<std::uint8_t> data) {
    std::size_t size = 0;
    std::size_t marks = 0;
    for (const auto b: data) {
        if (marked == 0 && b == 0xFF) {
            marked = 1;
            continue;
        }
        if (marked == 1 && b == 0x00) {
            marked = 2;
            continue;
        }
        if (marked == 2) {
            marks++;
            marked = 0;
            continue;
        }
        if (marked == 1 && b != 0xFF) data[size++] = 0xFF;
        marked = 0;
        data[size++] = b;
    }
    return {size, marks};
}

std::size_t overruns_since(
    std::optional<std::uint32_t> &last,
    const std::optional<std::uint32_t> count
) {
    if (!count.has_value()) return 0;
    const auto previous = last.value_or(*count);
    last = count;
    return static_cast<std::uint32_t>(*count - previous);
}

std::error_code watch(const Handle handle, Line &line) {
    auto [count, ec] = overrun_count(static_cast<int>(handle));
    if (ec) return ec;
    line.overruns = count;
    return {};
}

std::pair<Received, std::error_code>
receive(const Handle handle, Line &line, const std::span<std::uint8_t> data) {
    const auto [count, ec] = overrun_count(static_cast<int>(handle));
    if (ec) return {{}, ec};
    const auto [size, errors] = strip(line.marked, data);
    return {
        {.size = size,
         .errors = errors,
         .overruns = overruns_since(line.overruns, count)},
        {}
    };
}
}
}
