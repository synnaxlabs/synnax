// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <system_error>

#include "asio/error.hpp"

#include "driver/udp/socket.h"

namespace driver::udp {
namespace {
x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(transport::CONFIG_ERROR, message);
}

std::string to_string(const asio::ip::udp::endpoint &endpoint) {
    return endpoint.address().to_string() + ":" + std::to_string(endpoint.port());
}

asio::ip::udp protocol_of(const asio::ip::address &address) {
    return address.is_v6() ? asio::ip::udp::v6() : asio::ip::udp::v4();
}
}

Socket::Socket(const Config &config): stack(config.stacks(this->ctx)) {}

std::pair<std::unique_ptr<Socket>, x::errors::Error>
Socket::open(const synnax::udp::Properties &props, const Config &config) {
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
    std::optional<asio::ip::address> iface;
    if (!props.local_address.empty()) {
        iface = asio::ip::make_address(props.local_address, ec);
        if (ec)
            return {
                nullptr,
                invalid("'" + props.local_address + "' is not an IP address")
            };
    }
    if (group && iface && group->is_v6() != iface->is_v6())
        return {
            nullptr,
            invalid(
                "local address " + props.local_address + " and multicast group " +
                props.multicast_group + " must use the same IP version"
            )
        };
    std::optional<asio::ip::udp> protocol;
    if (iface) protocol = protocol_of(*iface);
    if (group) protocol = protocol_of(*group);
    auto s = std::unique_ptr<Socket>(new Socket(config));
    if (!props.remote_host.empty()) {
        const auto [endpoints, resolved] = s->stack->resolve(
            props.remote_host,
            props.remote_port
        );
        if (resolved)
            return {
                nullptr,
                translate("failed to resolve " + props.remote_host, resolved)
            };
        const auto remote = std::ranges::find_if(endpoints, [&protocol](const auto &e) {
            return !protocol || e.protocol() == *protocol;
        });
        if (remote == endpoints.end())
            return {
                nullptr,
                invalid(
                    "remote host " + props.remote_host +
                    " has no address of the local address's IP version"
                )
            };
        s->remote = *remote;
        protocol = remote->protocol();
    }
    const auto port = std::to_string(props.port);
    // Linux delivers group datagrams only to sockets bound to the any address, so a
    // multicast socket uses the local address to pick the interface it joins on.
    const auto local = iface && !group ? asio::ip::udp::endpoint(*iface, props.port)
                                       : asio::ip::udp::endpoint(
                                             protocol.value_or(asio::ip::udp::v4()),
                                             props.port
                                         );
    if (const auto opened = s->stack->open(local.protocol()))
        return {nullptr, translate("failed to open a UDP socket", opened)};
    if (group)
        if (const auto shared = s->stack->share())
            return {nullptr, translate("failed to share UDP port " + port, shared)};
    if (const auto bound = s->stack->bind(local))
        return {
            nullptr,
            translate("failed to bind UDP port " + to_string(local), bound)
        };
    if (group)
        if (const auto joined = s->stack->join(*group, iface))
            return {
                nullptr,
                translate(
                    "failed to join multicast group " + props.multicast_group,
                    joined
                )
            };
    return {std::move(s), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Socket::read(const x::telem::TimeSpan timeout) {
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    while (true) {
        const auto done = transport::run_for(
            this->ctx,
            deadline - x::telem::TimeStamp::now(),
            [this](Handler handler) {
                this->stack->async_receive_from(
                    this->buffer,
                    this->sender,
                    std::move(handler)
                );
            },
            [this] { this->stack->cancel(); }
        );
        if (done.timed_out()) return {{}, x::errors::NIL};
        // An ICMP port unreachable reply to an earlier send surfaces here on some
        // platforms. It says nothing about what this socket receives.
        if (done.ec == asio::error::connection_refused ||
            done.ec == asio::error::connection_reset)
            continue;
        if (done.ec)
            return {{}, translate("failed to receive a UDP datagram", done.ec)};
        if (this->remote && this->sender != *this->remote) continue;
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
        [this, data](Handler handler) {
            this->stack->async_send_to(data, *this->remote, std::move(handler));
        },
        [this] { this->stack->cancel(); }
    );
    const auto context = "failed to send to " + to_string(*this->remote);
    if (done.timed_out())
        return translate(context, std::make_error_code(std::errc::timed_out));
    if (done.ec) return translate(context, done.ec);
    return x::errors::NIL;
}

void Socket::close() {
    this->stack->close();
}
}
