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
#include <thread>

#include "asio/buffer.hpp"
#include "asio/connect.hpp"
#include "asio/write.hpp"

#include "driver/tcp/client.h"

namespace driver::tcp {
namespace {
using Clock = std::chrono::steady_clock;
}

Client::Client(const synnax::tcp::Properties &props, const Config &config):
    host(props.host), port(props.port), config(config), backoff(config.min_backoff) {}

std::string Client::address() const {
    return this->host + ":" + std::to_string(this->port);
}

x::errors::Error Client::drop(const x::errors::Error &err) {
    std::error_code ignored;
    this->socket.close(ignored);
    this->next_attempt = Clock::now() + this->backoff.chrono();
    this->backoff = 2 * this->backoff;
    if (this->backoff > this->config.max_backoff)
        this->backoff = this->config.max_backoff;
    return err;
}

x::errors::Error Client::connect() {
    asio::ip::tcp::resolver::results_type endpoints;
    const auto resolved = transport::run_for(
        this->ctx,
        this->config.connect_timeout,
        [this, &endpoints](auto handler) {
            this->resolver.async_resolve(
                this->host,
                std::to_string(this->port),
                [&endpoints, handler](
                    const std::error_code &ec,
                    asio::ip::tcp::resolver::results_type results
                ) {
                    endpoints = std::move(results);
                    handler(ec, 0);
                }
            );
        },
        [this] { this->resolver.cancel(); }
    );
    if (resolved.timed_out())
        return this->drop(
            x::errors::Error(
                transport::UNREACHABLE_ERROR,
                "timed out resolving " + this->host
            )
        );
    if (resolved.ec)
        return this->drop(
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to resolve " + this->host,
                resolved.ec
            )
        );
    const auto connected = transport::run_for(
        this->ctx,
        this->config.connect_timeout,
        [this, &endpoints](auto handler) {
            asio::async_connect(
                this->socket,
                endpoints,
                [handler](const std::error_code &ec, const asio::ip::tcp::endpoint &) {
                    handler(ec, 0);
                }
            );
        },
        [this] {
            std::error_code ignored;
            this->socket.close(ignored);
        }
    );
    if (connected.timed_out())
        return this->drop(
            x::errors::Error(
                transport::UNREACHABLE_ERROR,
                "timed out connecting to " + this->address()
            )
        );
    if (connected.ec)
        return this->drop(
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to connect to " + this->address(),
                connected.ec
            )
        );
    std::error_code ec;
    this->socket.set_option(asio::ip::tcp::no_delay(true), ec);
    if (!ec) this->socket.set_option(asio::socket_base::keep_alive(true), ec);
    if (ec)
        return this->drop(
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to configure the connection to " + this->address(),
                ec
            )
        );
    this->backoff = this->config.min_backoff;
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
    if (!this->socket.is_open()) {
        const auto wait = this->next_attempt - Clock::now();
        if (wait > timeout.chrono()) {
            std::this_thread::sleep_for(timeout.chrono());
            return {{}, x::errors::NIL};
        }
        if (wait > Clock::duration::zero()) std::this_thread::sleep_for(wait);
        return {{}, this->connect()};
    }
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this](auto handler) {
            this->socket.async_read_some(
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
    if (done.ec)
        return {
            {},
            this->drop(
                transport::error(
                    transport::UNREACHABLE_ERROR,
                    "connection to " + this->address() + " dropped",
                    done.ec
                )
            )
        };
    return {
        {.data = {this->buffer.data(), done.size}, .time = done.time},
        x::errors::NIL
    };
}

x::errors::Error Client::write(
    const std::span<const std::uint8_t> data,
    const x::telem::TimeSpan timeout
) {
    if (!this->socket.is_open()) {
        if (Clock::now() < this->next_attempt)
            return x::errors::Error(
                transport::UNREACHABLE_ERROR,
                "not connected to " + this->address()
            );
        if (auto err = this->connect()) return err;
    }
    const auto done = transport::run_for(
        this->ctx,
        timeout,
        [this, data](auto handler) {
            asio::async_write(
                this->socket,
                asio::buffer(data.data(), data.size()),
                std::move(handler)
            );
        },
        [this] {
            std::error_code ignored;
            this->socket.cancel(ignored);
        }
    );
    if (done.timed_out())
        return this->drop(
            x::errors::Error(
                transport::UNREACHABLE_ERROR,
                "timed out writing to " + this->address()
            )
        );
    if (done.ec)
        return this->drop(
            transport::error(
                transport::UNREACHABLE_ERROR,
                "failed to write to " + this->address(),
                done.ec
            )
        );
    return x::errors::NIL;
}

void Client::close() {
    std::error_code ignored;
    this->socket.close(ignored);
}
}
