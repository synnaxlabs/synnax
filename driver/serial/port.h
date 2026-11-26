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

#include "driver/serial/native.h"
#include "driver/transport/transport.h"

namespace driver::serial {
/// @brief Port is an open serial port. Its errors map through translate. A failed port
/// closes at once, and each later read or write returns the error that closed it. It is
/// not safe for concurrent use.
class Port final : public transport::Transport {
    /// @brief path is the path of the port, which errors name.
    std::string path;
    asio::io_context ctx{1};
    asio::serial_port device{ctx};
    native::Line line;
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );
    /// @brief failure is the error that closed the port.
    x::errors::Error failure;

    explicit Port(std::string path): path(std::move(path)) {}

    [[nodiscard]] native::Handle handle();

    /// @brief lets other programs open the port, then closes it.
    void close();

    /// @brief closes the port, and returns and keeps err.
    x::errors::Error fail(const x::errors::Error &err);

public:
    /// @brief opens the port at props.port, locks it against other programs, and
    /// applies the settings in props.
    /// @returns UNREACHABLE_ERROR when the port is missing or in use, and CONFIG_ERROR
    /// when a setting is invalid or the port or platform cannot apply it.
    static std::pair<std::unique_ptr<Port>, x::errors::Error>
    open(const synnax::serial::Properties &props);

    /// @brief waits up to one second for written bytes to leave a port that has not
    /// failed, then closes it.
    ~Port() override;

    Port(const Port &) = delete;
    Port &operator=(const Port &) = delete;

    std::pair<transport::Chunk, x::errors::Error>
    read(x::telem::TimeSpan timeout) override;

    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout) override;
};
}
