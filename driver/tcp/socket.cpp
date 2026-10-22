// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "asio/buffer.hpp"

#include "driver/tcp/keepalive.h"
#include "driver/tcp/socket.h"
#include "driver/transport/transport.h"

namespace driver::tcp {
AsioSocket::AsioSocket(asio::io_context &ctx): socket(ctx) {}

std::pair<std::vector<asio::ip::tcp::endpoint>, std::error_code> AsioSocket::resolve(
    const std::string &host,
    const std::uint16_t port,
    const x::telem::TimeSpan timeout
) {
    return transport::abandon_after(timeout, [host, port] {
        asio::io_context ctx;
        asio::ip::tcp::resolver resolver(ctx);
        std::error_code ec;
        const auto results = resolver.resolve(host, std::to_string(port), ec);
        std::vector<asio::ip::tcp::endpoint> endpoints;
        for (const auto &result: results)
            endpoints.push_back(result.endpoint());
        return std::pair{endpoints, ec};
    });
}

void AsioSocket::async_connect(
    const asio::ip::tcp::endpoint &endpoint,
    Handler handler
) {
    this->socket.async_connect(
        endpoint,
        [handler = std::move(handler)](const std::error_code &ec) { handler(ec, 0); }
    );
}

std::error_code AsioSocket::configure() {
    std::error_code ec;
    this->socket.set_option(asio::ip::tcp::no_delay(true), ec);
    if (ec) return ec;
    return keepalive::enable(this->socket.native_handle());
}

void AsioSocket::async_read_some(std::span<std::uint8_t> buffer, Handler handler) {
    this->socket.async_read_some(
        asio::buffer(buffer.data(), buffer.size()),
        std::move(handler)
    );
}

void AsioSocket::async_write_some(std::span<const std::uint8_t> data, Handler handler) {
    this->socket.async_write_some(
        asio::buffer(data.data(), data.size()),
        std::move(handler)
    );
}

void AsioSocket::cancel() {
    std::error_code ignored;
    this->socket.cancel(ignored);
}

void AsioSocket::close() {
    std::error_code ignored;
    this->socket.close(ignored);
}
}
