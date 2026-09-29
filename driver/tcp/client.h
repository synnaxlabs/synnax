// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"

#include "client/cpp/tcp/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::tcp {
/// @brief configures how a client connects and reconnects.
struct Config {
    /// @brief the longest one connection attempt may take.
    x::telem::TimeSpan connect_timeout = 2 * x::telem::SECOND;
    /// @brief the wait before reconnecting after a drop. Each failed attempt doubles
    /// it, up to max_backoff, and a successful connection resets it.
    x::telem::TimeSpan min_backoff = 250 * x::telem::MILLISECOND;
    /// @brief the longest wait between reconnection attempts.
    x::telem::TimeSpan max_backoff = 10 * x::telem::SECOND;
};

/// @brief a TCP client connection to a device. When the peer drops, the client
/// reconnects with backoff during later reads and writes. It is not safe for
/// concurrent use.
class Client {
    std::string host;
    std::uint16_t port;
    Config config;
    asio::io_context ctx{1};
    asio::ip::tcp::resolver resolver{ctx};
    asio::ip::tcp::socket socket{ctx};
    std::vector<std::uint8_t> buffer = std::vector<std::uint8_t>(
        transport::BUFFER_SIZE
    );
    std::chrono::steady_clock::time_point next_attempt;
    x::telem::TimeSpan backoff;

    Client(const synnax::tcp::Properties &props, const Config &config);

    [[nodiscard]] std::string address() const;
    x::errors::Error connect();
    x::errors::Error drop(const x::errors::Error &err);

public:
    /// @brief connects to props.host on props.port.
    /// @returns CONFIG_ERROR when props has no host or port, and UNREACHABLE_ERROR
    /// when the first connection attempt fails.
    static std::pair<std::unique_ptr<Client>, x::errors::Error>
    open(const synnax::tcp::Properties &props, const Config &config = Config{});

    /// @brief blocks until bytes arrive or the timeout elapses. While disconnected, it
    /// waits out the backoff within the timeout and then makes one connection attempt,
    /// which may run for up to Config::connect_timeout.
    /// @returns the bytes that arrived, or an empty chunk on timeout, while waiting out
    /// the backoff, or after reconnecting. UNREACHABLE_ERROR when the connection drops
    /// or a connection attempt fails. The stream restarts after a drop, so a reader
    /// must discard any partial message it holds.
    std::pair<transport::Chunk, x::errors::Error> read(x::telem::TimeSpan timeout);

    /// @brief blocks until all of data is written or the timeout elapses. While
    /// disconnected, it makes a connection attempt if the backoff has passed.
    /// @returns UNREACHABLE_ERROR when the client is disconnected, or when the write
    /// fails or times out, which drops the connection.
    x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout);

    /// @brief closes the connection. A later read or write reconnects.
    void close();
};
}
