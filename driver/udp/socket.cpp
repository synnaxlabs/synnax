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
#include <string>
#include <system_error>

#include "asio/buffer.hpp"
#include "asio/ip/multicast.hpp"

#include "driver/udp/socket.h"

namespace driver::udp {
namespace {
using Clock = std::chrono::steady_clock;

x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(transport::CONFIG_ERROR, message);
}

x::errors::Error unreachable(const std::string &context, const std::error_code &ec) {
    return transport::error(transport::UNREACHABLE_ERROR, context, ec);
}

std::string to_string(const asio::ip::udp::endpoint &endpoint) {
    return endpoint.address().to_string() + ":" + std::to_string(endpoint.port());
}
}

std::pair<std::unique_ptr<Socket>, x::errors::Error>
Socket::open(const synnax::udp::Properties &props) {
    if (!props.remote_host.empty() && props.remote_port == 0)
        return {nullptr, invalid("a remote port is required with a remote host")};
    if (props.remote_host.empty() && props.remote_port != 0)
        return {nullptr, invalid("a remote host is required with a remote port")};
    std::error_code ec;
    std::optional<asio::ip::address> group;
    if (!props.multicast_group.empty()) {
        group = asio::ip::make_address(props.multicast_group, ec);
        if (ec || !group->is_multicast())
            return {
                nullptr,
                invalid("'" + props.multicast_group + "' is not a multicast address")
            };
    }
    std::optional<asio::ip::udp> protocol;
    if (group) protocol = group->is_v6() ? asio::ip::udp::v6() : asio::ip::udp::v4();
    auto s = std::unique_ptr<Socket>(new Socket());
    if (!props.remote_host.empty()) {
        asio::ip::udp::resolver resolver(s->ctx);
        const auto service = std::to_string(props.remote_port);
        const auto results = protocol
                               ? resolver
                                     .resolve(*protocol, props.remote_host, service, ec)
                               : resolver.resolve(props.remote_host, service, ec);
        if (ec)
            return {nullptr, unreachable("failed to resolve " + props.remote_host, ec)};
        s->remote = results.begin()->endpoint();
        protocol = s->remote->protocol();
    }
    const auto local = asio::ip::udp::endpoint(
        protocol.value_or(asio::ip::udp::v4()),
        props.port
    );
    s->socket.open(local.protocol(), ec);
    if (ec) return {nullptr, unreachable("failed to open a UDP socket", ec)};
    if (group) {
        s->socket.set_option(asio::socket_base::reuse_address(true), ec);
        if (ec)
            return {
                nullptr,
                unreachable(
                    "failed to share UDP port " + std::to_string(props.port),
                    ec
                )
            };
    }
    s->socket.bind(local, ec);
    if (ec)
        return {
            nullptr,
            unreachable("failed to bind UDP port " + std::to_string(props.port), ec)
        };
    if (group) {
        s->socket.set_option(asio::ip::multicast::join_group(*group), ec);
        if (ec)
            return {
                nullptr,
                unreachable(
                    "failed to join multicast group " + props.multicast_group,
                    ec
                )
            };
    }
    return {std::move(s), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Socket::read(const x::telem::TimeSpan timeout) {
    const auto deadline = Clock::now() + timeout.chrono();
    while (true) {
        const auto remaining = std::max(
            Clock::duration::zero(),
            deadline - Clock::now()
        );
        const auto done = transport::run_for(
            this->ctx,
            x::telem::TimeSpan(
                std::chrono::duration_cast<std::chrono::nanoseconds>(remaining)
            ),
            [this](auto handler) {
                this->socket.async_receive(
                    asio::buffer(this->buffer),
                    std::move(handler)
                );
            },
            [this] {
                std::error_code ignored;
                this->socket.cancel(ignored);
            }
        );
        if (done.timed_out()) return {{}, x::errors::NIL};
        // An ICMP port unreachable reply to an earlier send surfaces here on some
        // platforms. It says nothing about what this socket receives.
        if (done.ec == asio::error::connection_refused ||
            done.ec == asio::error::connection_reset)
            continue;
        if (done.ec)
            return {{}, unreachable("failed to receive a UDP datagram", done.ec)};
        return {
            {.data = {this->buffer.data(), done.size}, .time = done.time},
            x::errors::NIL
        };
    }
}

x::errors::Error Socket::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (!this->remote) return invalid("no remote host is set to send to");
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this, data](auto handler) {
            this->socket.async_send_to(
                asio::buffer(data.data(), data.size()),
                *this->remote,
                std::move(handler)
            );
        },
        [this] {
            std::error_code ignored;
            this->socket.cancel(ignored);
        }
    );
    if (done.timed_out())
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            "timed out sending to " + to_string(*this->remote)
        );
    if (done.ec)
        return unreachable("failed to send to " + to_string(*this->remote), done.ec);
    return x::errors::NIL;
}

void Socket::close() {
    std::error_code ignored;
    this->socket.close(ignored);
}
}
