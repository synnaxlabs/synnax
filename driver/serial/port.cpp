// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <initializer_list>
#include <string>
#include <system_error>
#include <thread>
#include <type_traits>

#include "asio/buffer.hpp"
#include "asio/write.hpp"

#include "driver/serial/native.h"
#include "driver/serial/port.h"

namespace driver::serial {
namespace {
bool one_of(
    const std::string &value,
    const std::initializer_list<const char *> options
) {
    return std::ranges::any_of(options, [&value](const char *o) { return value == o; });
}

x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(transport::CONFIG_ERROR, message);
}

x::errors::Error validate(const synnax::serial::Properties &p) {
    if (p.port.empty()) return invalid("serial port path is required");
    if (p.baud_rate == 0) return invalid("baud rate must be greater than zero");
    if (p.data_bits < 5 || p.data_bits > 8)
        return invalid(
            "data bits must be 5, 6, 7, or 8, got " + std::to_string(p.data_bits)
        );
    if (!one_of(
            p.parity,
            {synnax::serial::PARITY_NONE_,
             synnax::serial::PARITY_EVEN_,
             synnax::serial::PARITY_ODD_,
             synnax::serial::PARITY_MARK_,
             synnax::serial::PARITY_SPACE_}
        ))
        return invalid("unknown parity '" + p.parity + "'");
    if (!one_of(
            p.stop_bits,
            {synnax::serial::STOP_BITS_ONE,
             synnax::serial::STOP_BITS_ONE_AND_HALF,
             synnax::serial::STOP_BITS_TWO}
        ))
        return invalid("unknown stop bits '" + p.stop_bits + "'");
    if (!one_of(
            p.flow_control,
            {synnax::serial::FLOW_CONTROL_NONE,
             synnax::serial::FLOW_CONTROL_SOFTWARE,
             synnax::serial::FLOW_CONTROL_HARDWARE}
        ))
        return invalid("unknown flow control '" + p.flow_control + "'");
    if (p.rs485 && p.flow_control == synnax::serial::FLOW_CONTROL_HARDWARE)
        return invalid(
            "RS-485 mode drives RTS, so it cannot use hardware flow control"
        );
    return x::errors::NIL;
}

template<typename NativeHandle>
native::Handle to_native(const NativeHandle handle) {
    if constexpr (std::is_pointer_v<NativeHandle>)
        return reinterpret_cast<native::Handle>(handle);
    else
        return static_cast<native::Handle>(handle);
}
}

std::pair<std::unique_ptr<Port>, x::errors::Error>
Port::open(const synnax::serial::Properties &props) {
    if (auto err = validate(props)) return {nullptr, err};
    auto port = std::unique_ptr<Port>(new Port(props.port));
    std::error_code ec;
    port->device.open(props.port, ec);
    if (ec)
        return {
            nullptr,
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to open serial port " + props.port,
                ec
            )
        };
    if (auto err = native::configure(to_native(port->device.native_handle()), props))
        return {nullptr, err};
    return {std::move(port), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Port::read(const x::telem::TimeSpan timeout) {
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this](auto handler) {
            this->device.async_read_some(
                asio::buffer(this->buffer),
                std::move(handler)
            );
        },
        [this] {
            std::error_code ignored;
            this->device.cancel(ignored);
        }
    );
    if (done.timed_out()) return {{}, x::errors::NIL};
    if (done.ec)
        return {
            {},
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to read from serial port " + this->path,
                done.ec
            )
        };
    return {
        {.data = {this->buffer.data(), done.size}, .time = done.time},
        x::errors::NIL
    };
}

x::errors::Error Port::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this, data](auto handler) {
            asio::async_write(
                this->device,
                asio::buffer(data.data(), data.size()),
                std::move(handler)
            );
        },
        [this] {
            std::error_code ignored;
            this->device.cancel(ignored);
        }
    );
    if (done.timed_out())
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            "timed out writing to serial port " + this->path
        );
    if (done.ec)
        return transport::error(
            transport::UNREACHABLE_ERROR,
            "failed to write to serial port " + this->path,
            done.ec
        );
    return x::errors::NIL;
}

void Port::close(const x::telem::TimeSpan timeout) {
    if (!this->device.is_open()) return;
    // Closing a port discards the output that it has not yet sent.
    const auto handle = to_native(this->device.native_handle());
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    while (x::telem::TimeStamp::now() < deadline) {
        const auto [queued, ec] = native::queued_output(handle);
        if (ec || queued == 0) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
    std::error_code ignored;
    this->device.close(ignored);
}
}
