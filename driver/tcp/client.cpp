// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <span>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "asio/buffer.hpp"
#include "asio/error.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/tcp/client.h"
#include "driver/tcp/keepalive.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::tcp {
namespace {
using Handler = std::function<void(const std::error_code &, std::size_t)>;

/// @brief HOST_DOWN is the native error of a host on the local network that stops
/// answering, which no std::errc names.
#ifdef _WIN32
constexpr int HOST_DOWN = WSAEHOSTDOWN;
#else
constexpr int HOST_DOWN = EHOSTDOWN;
#endif

/// @brief returns a transport error of the native error ec, whose message is context
/// followed by the message of ec.
x::errors::Error translate(const std::string &context, const std::error_code &ec) {
    if (ec == asio::error::eof || ec == asio::error::host_not_found ||
        ec == asio::error::host_not_found_try_again || ec == asio::error::no_data ||
        ec == asio::error::no_recovery)
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    const auto native = transport::from_asio(ec);
    if (native == std::error_code(HOST_DOWN, std::system_category()) ||
        transport::matches(
            native,
            {std::errc::connection_refused,
             std::errc::connection_reset,
             std::errc::connection_aborted,
             std::errc::not_connected,
             std::errc::broken_pipe,
             std::errc::timed_out,
             std::errc::host_unreachable,
             std::errc::network_unreachable,
             std::errc::network_down,
             std::errc::network_reset,
             std::errc::address_not_available}
        ))
        return transport::error(transport::UNREACHABLE_ERROR, context, native);
    if (transport::matches(
            native,
            {std::errc::permission_denied,
             std::errc::operation_not_permitted,
             std::errc::invalid_argument,
             std::errc::address_family_not_supported,
             std::errc::no_protocol_option,
             std::errc::operation_not_supported}
        ))
        return transport::error(transport::CONFIG_ERROR, context, native);
    return transport::error(transport::CRITICAL_ERROR, context, native);
}

/// @brief cancels the pending operations of socket.
void cancel(asio::ip::tcp::socket &socket) {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): it fails only when closed.
    socket.cancel(ignored);
}

/// @brief closes socket, which aborts its pending operations.
void close(asio::ip::tcp::socket &socket) {
    std::error_code ignored;
    // NOLINTNEXTLINE(bugprone-unused-return-value): a failed close still frees it.
    socket.close(ignored);
}

/// @brief resolves host on port through getaddrinfo, which cannot be cancelled, so a
/// resolve that times out finishes on its own thread.
std::pair<std::vector<asio::ip::tcp::endpoint>, std::error_code> resolve(
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
}

x::errors::Error Client::fail(const std::string &context, const std::error_code &ec) {
    close(this->socket);
    this->failure = translate(context, ec);
    return this->failure;
}

std::pair<std::unique_ptr<Client>, x::errors::Error> Client::connect(
    const std::string &host,
    const std::uint16_t port,
    const x::telem::TimeSpan timeout
) {
    if (host.empty())
        return {
            nullptr,
            x::errors::Error(transport::CONFIG_ERROR, "the host is empty")
        };
    if (port == 0)
        return {nullptr, x::errors::Error(transport::CONFIG_ERROR, "the port is 0")};
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    auto client = std::unique_ptr<Client>(
        new Client(host + ":" + std::to_string(port))
    );
    const auto [endpoints, resolved] = resolve(host, port, timeout);
    if (resolved) return {nullptr, client->fail("failed to resolve " + host, resolved)};
    auto &socket = client->socket;
    std::error_code ec = asio::error::host_not_found;
    for (const auto &endpoint: endpoints) {
        const auto done = transport::run_for(
            client->ctx,
            deadline - x::telem::TimeStamp::now(),
            [&socket, &endpoint](Handler handler) {
                socket.async_connect(
                    endpoint,
                    [handler = std::move(handler)](const std::error_code &e) {
                        handler(e, 0);
                    }
                );
            },
            [&socket] { close(socket); }
        );
        ec = done.timed_out() ? std::make_error_code(std::errc::timed_out) : done.ec;
        if (!ec || done.timed_out()) break;
        close(socket);
    }
    const auto context = "failed to connect to " + client->address;
    if (ec) return {nullptr, client->fail(context, ec)};
    if (socket.set_option(asio::ip::tcp::no_delay(true), ec))
        return {nullptr, client->fail(context, ec)};
    if (const auto err = keepalive::enable(socket.native_handle()))
        return {nullptr, client->fail(context, err)};
    return {std::move(client), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Client::read(const x::telem::TimeSpan timeout) {
    if (this->failure) return {{}, this->failure};
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this](Handler handler) {
            this->socket.async_read_some(
                asio::buffer(this->buffer.data(), this->buffer.size()),
                std::move(handler)
            );
        },
        [this] { cancel(this->socket); }
    );
    if (done.timed_out()) return {{}, x::errors::NIL};
    if (done.ec)
        return {{}, this->fail("failed to read from " + this->address, done.ec)};
    return {
        {.data = {this->buffer.data(), done.size}, .time = done.time},
        x::errors::NIL,
    };
}

x::errors::Error Client::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (this->failure) return this->failure;
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    for (std::size_t written = 0; written < data.size();) {
        const auto rest = data.subspan(written);
        const auto done = transport::run_for(
            this->ctx,
            deadline - x::telem::TimeStamp::now(),
            [this, rest](Handler handler) {
                this->socket.async_write_some(
                    asio::buffer(rest.data(), rest.size()),
                    std::move(handler)
                );
            },
            [this] { cancel(this->socket); }
        );
        if (done.timed_out() || done.ec)
            return this->fail(
                "failed to write to " + this->address,
                done.timed_out() ? std::make_error_code(std::errc::timed_out) : done.ec
            );
        written += done.size;
    }
    return x::errors::NIL;
}
}
