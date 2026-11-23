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
#include "asio/ip/tcp.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::tcp {
/// @brief Client is a TCP connection to a device. A failed connection stays closed, and
/// each later read or write returns the error that closed it. It is not safe for
/// concurrent use.
class Client final : public transport::Transport {
    /// @brief address is the host and port, which errors name.
    std::string address;
    asio::io_context ctx{1};
    asio::ip::tcp::socket socket{ctx};
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );
    /// @brief failure is the error that closed the connection.
    x::errors::Error failure;

    explicit Client(std::string address): address(std::move(address)) {}

    /// @brief closes the connection, and returns and keeps the error of ec.
    x::errors::Error fail(const std::string &context, const std::error_code &ec);

public:
    /// @brief resolves host and connects to it on port, with keepalive probes that drop
    /// a connection whose peer stops answering.
    /// @param timeout bounds the resolve and the connect together.
    /// @returns CONFIG_ERROR when host is empty or port is 0, and UNREACHABLE_ERROR
    /// when the host does not resolve, no address accepts the connection, or timeout
    /// elapses first.
    static std::pair<std::unique_ptr<Client>, x::errors::Error>
    connect(const std::string &host, std::uint16_t port, x::telem::TimeSpan timeout);

    std::pair<transport::Chunk, x::errors::Error>
    read(x::telem::TimeSpan timeout) override;

    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout) override;
};
}
