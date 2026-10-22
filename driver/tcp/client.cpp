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

#include "asio/error.hpp"

#include "driver/tcp/client.h"

namespace driver::tcp {
Client::Client(const synnax::tcp::Properties &props, Config config):
    host(props.host), port(props.port), config(std::move(config)) {
    this->socket = this->config.sockets(this->ctx);
}

std::string Client::address() const {
    return this->host + ":" + std::to_string(this->port);
}

x::errors::Error Client::fail(const std::string &context, const std::error_code &ec) {
    this->socket->close();
    this->failure = translate(context, ec);
    return this->failure;
}

x::errors::Error Client::connect() {
    const auto deadline = x::telem::TimeStamp::now() + this->config.connect_timeout;
    const auto [endpoints, resolved] = this->socket->resolve(
        this->host,
        this->port,
        this->config.connect_timeout
    );
    if (resolved) return this->fail("failed to resolve " + this->host, resolved);
    std::error_code ec = asio::error::host_not_found;
    for (const auto &endpoint: endpoints) {
        const auto done = transport::run_for(
            this->ctx,
            deadline - x::telem::TimeStamp::now(),
            [this, &endpoint](Handler handler) {
                this->socket->async_connect(endpoint, std::move(handler));
            },
            [this] { this->socket->close(); }
        );
        ec = done.timed_out() ? std::make_error_code(std::errc::timed_out) : done.ec;
        if (!ec || done.timed_out()) break;
        this->socket->close();
    }
    const auto context = "failed to connect to " + this->address();
    if (ec) return this->fail(context, ec);
    if (const auto configured = this->socket->configure())
        return this->fail(context, configured);
    return x::errors::NIL;
}

std::pair<std::unique_ptr<Client>, x::errors::Error>
Client::open(const synnax::tcp::Properties &props, const Config &config) {
    if (props.host.empty())
        return {
            nullptr,
            x::errors::Error(transport::CONFIG_ERROR, "TCP host is required")
        };
    if (props.port == 0)
        return {
            nullptr,
            x::errors::Error(transport::CONFIG_ERROR, "TCP port is required")
        };
    auto client = std::unique_ptr<Client>(new Client(props, config));
    if (auto err = client->connect()) return {nullptr, err};
    return {std::move(client), x::errors::NIL};
}

std::pair<transport::Chunk, x::errors::Error>
Client::read(const x::telem::TimeSpan timeout) {
    if (this->failure) return {{}, this->failure};
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this](Handler handler) {
            this->socket->async_read_some(this->buffer, std::move(handler));
        },
        [this] { this->socket->cancel(); }
    );
    if (done.timed_out()) return {{}, x::errors::NIL};
    if (done.ec)
        return {{}, this->fail("failed to read from " + this->address(), done.ec)};
    return {
        {.data = {this->buffer.data(), done.size}, .time = done.time},
        x::errors::NIL
    };
}

x::errors::Error Client::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (this->failure) return this->failure;
    const auto context = "failed to write to " + this->address();
    const auto deadline = x::telem::TimeStamp::now() + timeout;
    for (std::size_t written = 0; written < data.size();) {
        const auto done = transport::run_for(
            this->ctx,
            deadline - x::telem::TimeStamp::now(),
            [this, rest = data.subspan(written)](Handler handler) {
                this->socket->async_write_some(rest, std::move(handler));
            },
            [this] { this->socket->cancel(); }
        );
        if (done.timed_out())
            return this->fail(context, std::make_error_code(std::errc::timed_out));
        if (done.ec) return this->fail(context, done.ec);
        written += done.size;
    }
    return x::errors::NIL;
}

void Client::close() {
    this->socket->close();
}
}
