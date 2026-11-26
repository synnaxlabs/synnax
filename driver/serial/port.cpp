// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <string>
#include <system_error>
#include <thread>

#include "asio/buffer.hpp"
#include "asio/write.hpp"

#include "driver/serial/errors.h"
#include "driver/serial/port.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @brief DRAIN_TIMEOUT is how long a closing port waits for its written bytes to
/// leave.
const auto DRAIN_TIMEOUT = x::telem::SECOND;

/// @brief cancels the pending operations of device.
void cancel(asio::serial_port &device) {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): it fails only when closed.
    device.cancel(ignored);
}
}

native::Handle Port::handle() {
    return static_cast<native::Handle>(this->device.native_handle());
}

std::pair<std::unique_ptr<Port>, x::errors::Error>
Port::open(const synnax::serial::Properties &props) {
    if (props.port.empty())
        return {
            nullptr,
            x::errors::Error(transport::CONFIG_ERROR, "serial port path is required")
        };
    auto port = std::unique_ptr<Port>(new Port(props.port));
    std::error_code ec;
    if (port->device.open(props.port, ec))
        return {nullptr, translate("failed to open serial port " + props.port, ec)};
    if (auto err = native::lock(port->handle(), props.port)) return {nullptr, err};
    port->locked = true;
    if (auto err = native::configure(port->handle(), props)) return {nullptr, err};
    if (const auto watched = native::watch(port->handle(), port->line))
        return {
            nullptr,
            translate("failed to open serial port " + props.port, watched)
        };
    return {std::move(port), x::errors::NIL};
}

void Port::close() {
    if (this->locked) native::unlock(this->handle());
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): a failed close still frees it.
    this->device.close(ignored);
}

x::errors::Error Port::fail(const x::errors::Error &err) {
    this->close();
    this->failure = err;
    return this->failure;
}

Port::~Port() {
    if (!this->device.is_open()) return;
    // Closing a port discards the output that it has not yet sent.
    const x::telem::Stopwatch sw;
    while (this->locked && sw.elapsed() < DRAIN_TIMEOUT) {
        const auto [queued, ec] = native::queued_output(this->handle());
        if (ec || queued == 0) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
    this->close();
}

std::pair<transport::Chunk, x::errors::Error>
Port::read(const x::telem::TimeSpan timeout) {
    if (this->failure) return {{}, this->failure};
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this](transport::Handler handler) {
            this->device.async_read_some(
                asio::buffer(this->buffer.data(), this->buffer.size()),
                std::move(handler)
            );
        },
        [this] { cancel(this->device); }
    );
    if (done.timed_out()) return {{}, x::errors::NIL};
    const auto context = "failed to read from serial port " + this->path;
    if (done.ec) return {{}, this->fail(translate(context, done.ec))};
    const auto [received, ec] = native::receive(
        this->handle(),
        this->line,
        {this->buffer.data(), done.size}
    );
    if (ec) return {{}, this->fail(translate(context, ec))};
    return {
        {.data = {this->buffer.data(), received.size},
         .time = done.time,
         .errors = received.errors,
         .overruns = received.overruns},
        x::errors::NIL,
    };
}

x::errors::Error Port::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (this->failure) return this->failure;
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this, data](transport::Handler handler) {
            asio::async_write(
                this->device,
                asio::buffer(data.data(), data.size()),
                std::move(handler)
            );
        },
        [this] { cancel(this->device); }
    );
    const auto context = "failed to write to serial port " + this->path;
    if (done.timed_out())
        return this->fail(
            translate(context, std::make_error_code(std::errc::timed_out))
        );
    if (done.ec) return this->fail(translate(context, done.ec));
    return x::errors::NIL;
}
}
