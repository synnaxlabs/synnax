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
#include "asio/ip/multicast.hpp"

#include "driver/transport/transport.h"
#include "driver/udp/stack.h"

namespace driver::udp {
AsioStack::AsioStack(asio::io_context &ctx): socket(ctx) {}

std::pair<std::vector<asio::ip::udp::endpoint>, std::error_code> AsioStack::resolve(
    const std::string &host,
    const std::uint16_t port,
    const x::telem::TimeSpan timeout
) {
    return transport::abandon_after(timeout, [host, port] {
        asio::io_context ctx;
        asio::ip::udp::resolver resolver(ctx);
        std::error_code ec;
        const auto results = resolver.resolve(host, std::to_string(port), ec);
        std::vector<asio::ip::udp::endpoint> endpoints;
        for (const auto &result: results)
            endpoints.push_back(result.endpoint());
        return std::pair{endpoints, ec};
    });
}

std::error_code AsioStack::open(const asio::ip::udp &protocol) {
    std::error_code ec;
    return this->socket.open(protocol, ec);
}

std::error_code AsioStack::share() {
    std::error_code ec;
    return this->socket.set_option(asio::socket_base::reuse_address(true), ec);
}

std::error_code AsioStack::bind(const asio::ip::udp::endpoint &local) {
    std::error_code ec;
    return this->socket.bind(local, ec);
}

std::error_code AsioStack::join(
    const asio::ip::address &group,
    const std::optional<asio::ip::address> &iface
) {
    std::error_code ec;
    if (!iface.has_value())
        return this->socket.set_option(asio::ip::multicast::join_group(group), ec);
    if (group.is_v4())
        return this->socket.set_option(
            asio::ip::multicast::join_group(group.to_v4(), iface->to_v4()),
            ec
        );
    return this->socket.set_option(
        asio::ip::multicast::join_group(group.to_v6(), iface->to_v6().scope_id()),
        ec
    );
}

void AsioStack::async_receive_from(
    std::span<std::uint8_t> buffer,
    asio::ip::udp::endpoint &sender,
    Handler handler
) {
    this->socket.async_receive_from(
        asio::buffer(buffer.data(), buffer.size()),
        sender,
        std::move(handler)
    );
}

void AsioStack::async_send_to(
    std::span<const std::uint8_t> data,
    const asio::ip::udp::endpoint &to,
    Handler handler
) {
    this->socket
        .async_send_to(asio::buffer(data.data(), data.size()), to, std::move(handler));
}

void AsioStack::cancel() {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): it fails only when closed.
    this->socket.cancel(ignored);
}

void AsioStack::close() {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): a failed close still frees it.
    this->socket.close(ignored);
}
}
