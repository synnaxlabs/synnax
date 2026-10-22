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

#include "client/cpp/tcp/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/tcp/errors.h"
#include "driver/tcp/socket.h"
#include "driver/transport/transport.h"

namespace driver::tcp {
/// @brief configures how a client connects.
struct Config {
    /// @brief the longest that resolving the host and connecting to it may take.
    x::telem::TimeSpan connect_timeout = 2 * x::telem::SECOND;
    /// @brief makes the client's socket. Tests substitute a model of the network.
    Sockets sockets = [](asio::io_context &ctx) {
        return std::make_unique<AsioSocket>(ctx);
    };
};

/// @brief a TCP client connection to a device. A failed connection stays closed, and
/// every later read or write returns the error that closed it. Calls must not overlap,
/// but any thread may make them. Errors map through translate.
class Client {
    std::string host;
    std::uint16_t port;
    Config config;
    asio::io_context ctx{1};
    std::unique_ptr<Socket> socket;
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );
    x::errors::Error failure = x::errors::NIL;

    Client(const synnax::tcp::Properties &props, Config config);

    [[nodiscard]] std::string address() const;
    x::errors::Error connect();
    x::errors::Error fail(const std::string &context, const std::error_code &ec);

public:
    /// @brief the pause after which a reader drops a partial frame. TCP never loses
    /// bytes, so a partial frame is never dropped at a pause.
    static inline const x::telem::TimeSpan FRAME_GAP = x::telem::TimeSpan::max();

    /// @brief resolves props.host and connects to it on props.port, with keepalive
    /// probes that drop a connection whose peer stops answering.
    /// @returns CONFIG_ERROR when props has no host or port, and UNREACHABLE_ERROR when
    /// the connection fails or Config::connect_timeout elapses first.
    static std::pair<std::unique_ptr<Client>, x::errors::Error>
    open(const synnax::tcp::Properties &props, const Config &config = Config{});

    /// @brief blocks until bytes arrive or the timeout elapses.
    /// @returns the bytes that arrived, or an empty chunk on timeout. UNREACHABLE_ERROR
    /// when the connection drops.
    std::pair<transport::Chunk, x::errors::Error> read(x::telem::TimeSpan timeout);

    /// @brief blocks until all of data is written or the timeout elapses.
    /// @returns UNREACHABLE_ERROR when the connection drops or the timeout elapses
    /// first, which closes the connection.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout);

    /// @brief closes the connection. Later reads and writes fail.
    void close();
};
}
