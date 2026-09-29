// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/serial_port.hpp"

#include "client/cpp/serial/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::serial {
/// @brief an open serial port. Calls must not overlap, but any thread may make them.
class Port {
    std::string path;
    asio::io_context ctx{1};
    asio::serial_port device{ctx};
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );

    explicit Port(std::string path): path(std::move(path)) {}

public:
    /// @brief opens the port at props.port and applies the settings in props.
    /// @returns UNREACHABLE_ERROR when the port cannot be opened, and CONFIG_ERROR
    /// when a setting is invalid or the port or platform cannot apply it.
    static std::pair<std::unique_ptr<Port>, x::errors::Error>
    open(const synnax::serial::Properties &props);

    /// @brief blocks until bytes arrive or the timeout elapses.
    /// @returns the bytes that arrived, or an empty chunk on timeout.
    /// UNREACHABLE_ERROR when the port fails, as when its device is unplugged.
    std::pair<transport::Chunk, x::errors::Error> read(x::telem::TimeSpan timeout);

    /// @brief blocks until all of data is written or the timeout elapses.
    /// @returns UNREACHABLE_ERROR when the port fails or the timeout elapses first.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout);

    /// @brief closes the port. Reads and writes after close fail.
    void close();
};
}
