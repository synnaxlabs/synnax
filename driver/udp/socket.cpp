// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <system_error>
#include <utility>

#include "asio/buffer.hpp"
#include "asio/error.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/address.hpp"
#include "asio/ip/udp.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/errors.h"
#include "driver/transport/transport.h"
#include "driver/udp/icmp.h"
#include "driver/udp/socket.h"

namespace driver::udp {
namespace {

std::string to_string(const asio::ip::udp::endpoint &endpoint) {
    return endpoint.address().to_string() + ":" + std::to_string(endpoint.port());
}

/// @brief returns a transport error of the native error ec, whose message is context
/// followed by the message of ec.
x::errors::Error translate(const std::string &context, const std::error_code &ec) {
    if (ec == asio::error::host_not_found ||
        ec == asio::error::host_not_found_try_again || ec == asio::error::no_data ||
        ec == asio::error::no_recovery)
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    const auto native = transport::from_asio(ec);
    if (transport::matches(
            native,
            {std::errc::host_unreachable,
             std::errc::network_unreachable,
             std::errc::network_down,
             std::errc::address_not_available,
             std::errc::no_such_device,
             std::errc::no_buffer_space,
             std::errc::timed_out}
        ) ||
        native == transport::HOST_DOWN)
        return transport::error(transport::UNREACHABLE_ERROR, context, native);
    if (transport::matches(
            native,
            {std::errc::address_in_use,
             std::errc::permission_denied,
             std::errc::operation_not_permitted,
             std::errc::invalid_argument,
             std::errc::address_family_not_supported,
             std::errc::message_size}
        ))
        return transport::error(transport::CONFIG_ERROR, context, native);
    return transport::error(transport::CRITICAL_ERROR, context, native);
}

/// @brief MAX_V4_DATAGRAM and MAX_V6_DATAGRAM are the most bytes that one datagram
/// carries. An IPv4 packet length counts its 20-byte header, and an IPv6 payload length
/// does not.
constexpr std::size_t MAX_V4_DATAGRAM = 65507;
constexpr std::size_t MAX_V6_DATAGRAM = 65527;

/// @brief cancels the pending operations of socket.
void cancel(asio::ip::udp::socket &socket) {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): it fails only when closed.
    socket.cancel(ignored);
}

}

std::pair<std::unique_ptr<Socket>, x::errors::Error>
Socket::open(const Address &addr, const x::telem::TimeSpan timeout) {
    if (!addr.remote_host.empty() && addr.remote_port == 0)
        return {
            nullptr,
            x::errors::Error(
                transport::CONFIG_ERROR,
                "a remote port is required with a remote host"
            )
        };
    if (addr.remote_host.empty() && addr.remote_port != 0)
        return {
            nullptr,
            x::errors::Error(
                transport::CONFIG_ERROR,
                "a remote host is required with a remote port"
            )
        };
    std::optional<asio::ip::address> local;
    if (!addr.local_address.empty()) {
        std::error_code ec;
        local = asio::ip::make_address(addr.local_address, ec);
        if (ec)
            return {
                nullptr,
                x::errors::Error(
                    transport::CONFIG_ERROR,
                    "'" + addr.local_address + "' is not an IP address"
                )
            };
    }
    auto s = std::unique_ptr<Socket>(new Socket());
    if (!addr.remote_host.empty()) {
        const auto [endpoints, resolved] = transport::resolve<asio::ip::udp>(
            addr.remote_host,
            addr.remote_port,
            timeout
        );
        if (resolved)
            return {
                nullptr,
                translate("failed to resolve " + addr.remote_host, resolved)
            };
        for (const auto &e: endpoints)
            if (!local || e.address().is_v6() == local->is_v6()) {
                s->remote = e;
                break;
            }
        if (!s->remote)
            return {
                nullptr,
                x::errors::Error(
                    transport::CONFIG_ERROR,
                    "remote host " + addr.remote_host +
                        " has no address of the IP version of local address " +
                        addr.local_address
                )
            };
    }
    asio::ip::udp::endpoint bound(asio::ip::udp::v4(), addr.port);
    if (local)
        bound = {*local, addr.port};
    else if (s->remote)
        bound = {s->remote->protocol(), addr.port};
    std::error_code ec;
    if (s->socket.open(bound.protocol(), ec))
        return {nullptr, translate("failed to open a UDP socket", ec)};
    if (const auto ignored = icmp::ignore(s->socket.native_handle()))
        return {nullptr, translate("failed to open a UDP socket", ignored)};
    if (s->socket.bind(bound, ec))
        return {nullptr, translate("failed to bind UDP port " + to_string(bound), ec)};
    return {std::move(s), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Socket::read(const x::telem::TimeSpan timeout) {
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    while (true) {
        const auto done = transport::run_for(
            this->ctx,
            deadline - x::telem::TimeStamp::now(),
            [this](transport::Handler handler) {
                this->socket.async_receive_from(
                    asio::buffer(this->buffer),
                    this->sender,
                    std::move(handler)
                );
            },
            [this] { cancel(this->socket); }
        );
        if (done.timed_out()) return {{}, x::errors::NIL};
        if (done.ec)
            return {{}, translate("failed to receive a UDP datagram", done.ec)};
        if (done.size > 0 &&
            (!this->remote || this->sender.address() == this->remote->address()))
            return {
                {.data = {this->buffer.data(), done.size}, .time = done.time},
                x::errors::NIL
            };
        // A receive completes at once while datagrams are queued, so a flood of skipped
        // datagrams never lets it time out.
        if (x::telem::TimeStamp::now() >= deadline) return {{}, x::errors::NIL};
    }
}

x::errors::Error Socket::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (!this->remote)
        return x::errors::Error(
            transport::CONFIG_ERROR,
            "no remote host is set to send to"
        );
    const auto max = this->remote->address().is_v4() ? MAX_V4_DATAGRAM
                                                     : MAX_V6_DATAGRAM;
    if (data.size() > max)
        return x::errors::Error(
            transport::CONFIG_ERROR,
            "failed to send to " + to_string(*this->remote) + ": " +
                std::to_string(data.size()) + " bytes exceed the " +
                std::to_string(max) + " bytes of a datagram"
        );
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this, data](transport::Handler handler) {
            this->socket.async_send_to(
                asio::buffer(data.data(), data.size()),
                *this->remote,
                std::move(handler)
            );
        },
        [this] { cancel(this->socket); }
    );
    if (!done.timed_out() && !done.ec) return x::errors::NIL;
    return translate(
        "failed to send to " + to_string(*this->remote),
        done.timed_out() ? std::make_error_code(std::errc::timed_out) : done.ec
    );
}
}
