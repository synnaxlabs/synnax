// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <limits>
#include <thread>
#include <utility>
#include <vector>

#include "asio/error.hpp"
#include "asio/executor_work_guard.hpp"
#include "asio/post.hpp"

#include "driver/tcp/model.h"

namespace driver::tcp::model {
namespace {
/// @brief keeps an io_context running while an operation is pending, as a real
/// socket's registration with the reactor does.
using Work = asio::executor_work_guard<asio::io_context::executor_type>;

/// @brief an operation that waits for the network.
struct Pending {
    Handler handler;
    Work work;
};
}

struct Pipe {
    std::mutex mu;
    std::condition_variable cv;
    asio::io_context &ctx;
    std::deque<std::uint8_t> to_client;
    std::deque<std::uint8_t> to_peer;
    bool client_closed = false;
    bool peer_closed = false;
    std::size_t read_size;
    std::size_t write_size;

    struct Read {
        std::span<std::uint8_t> buffer;
        Pending pending;
    };
    std::optional<Read> read;

    Pipe(
        asio::io_context &ctx,
        const std::size_t read_size,
        const std::size_t write_size
    ):
        ctx(ctx), read_size(read_size), write_size(write_size) {}

    void post(Handler handler, const std::error_code &ec, const std::size_t n) const {
        asio::post(this->ctx, [handler = std::move(handler), ec, n] {
            handler(ec, n);
        });
    }

    /// @brief completes the pending read when bytes are queued or the peer closed.
    /// The caller holds mu.
    void complete_read() {
        if (!this->read.has_value()) return;
        if (!this->to_client.empty()) {
            const auto n = std::min(
                {this->read->buffer.size(), this->read_size, this->to_client.size()}
            );
            std::copy_n(this->to_client.begin(), n, this->read->buffer.begin());
            this->to_client.erase(this->to_client.begin(), this->to_client.begin() + n);
            this->post(std::move(this->read->pending.handler), {}, n);
        } else if (this->peer_closed)
            this->post(std::move(this->read->pending.handler), asio::error::eof, 0);
        else
            return;
        this->read.reset();
    }
};

struct State {
    std::mutex mu;
    std::condition_variable cv;
    std::map<std::string, asio::ip::address> names;
    std::set<std::string> hung;
    std::map<asio::ip::tcp::endpoint, std::deque<std::shared_ptr<Peer>>> listeners;
    std::set<asio::ip::tcp::endpoint> dropped;
    std::size_t calls = 0;
    std::map<std::size_t, std::error_code> failures;
    std::size_t read_size = std::numeric_limits<std::size_t>::max();
    std::size_t write_size = std::numeric_limits<std::size_t>::max();
    std::size_t configured = 0;

    /// @brief counts one socket call.
    /// @returns the error that fail_at scheduled for the call.
    std::error_code call() {
        std::lock_guard lock(this->mu);
        this->calls++;
        const auto failure = this->failures.find(this->calls);
        if (failure == this->failures.end()) return {};
        return failure->second;
    }
};

namespace {
class Socket final : public tcp::Socket {
    asio::io_context &ctx;
    std::shared_ptr<State> net;
    std::shared_ptr<Pipe> pipe;
    std::optional<Pending> connecting;

    void post(Handler handler, const std::error_code &ec, const std::size_t n = 0) {
        asio::post(this->ctx, [handler = std::move(handler), ec, n] {
            handler(ec, n);
        });
    }

public:
    Socket(asio::io_context &ctx, std::shared_ptr<State> net):
        ctx(ctx), net(std::move(net)) {}

    ~Socket() override { this->close(); }

    std::pair<std::vector<asio::ip::tcp::endpoint>, std::error_code> resolve(
        const std::string &host,
        const std::uint16_t port,
        const x::telem::TimeSpan timeout
    ) override {
        if (const auto ec = this->net->call()) return {{}, ec};
        std::unique_lock lock(this->net->mu);
        if (this->net->hung.contains(host)) {
            lock.unlock();
            std::this_thread::sleep_for(timeout.chrono());
            return {{}, std::make_error_code(std::errc::timed_out)};
        }
        if (const auto named = this->net->names.find(host);
            named != this->net->names.end())
            return {{{named->second, port}}, {}};
        std::error_code ec;
        const auto address = asio::ip::make_address(host, ec);
        if (ec) return {{}, asio::error::host_not_found};
        return {{{address, port}}, {}};
    }

    void
    async_connect(const asio::ip::tcp::endpoint &endpoint, Handler handler) override {
        if (const auto ec = this->net->call())
            return this->post(std::move(handler), ec);
        std::lock_guard lock(this->net->mu);
        if (const auto listener = this->net->listeners.find(endpoint);
            listener != this->net->listeners.end()) {
            this->pipe = std::make_shared<Pipe>(
                this->ctx,
                this->net->read_size,
                this->net->write_size
            );
            listener->second.push_back(std::make_shared<Peer>(this->pipe));
            this->net->cv.notify_all();
            return this->post(std::move(handler), {});
        }
        if (this->net->dropped.contains(endpoint)) {
            this->connecting.emplace(
                Pending{
                    .handler = std::move(handler),
                    .work = asio::make_work_guard(this->ctx),
                }
            );
            return;
        }
        this->post(std::move(handler), asio::error::connection_refused);
    }

    std::error_code configure() override {
        if (const auto ec = this->net->call()) return ec;
        if (this->pipe == nullptr) return asio::error::bad_descriptor;
        std::lock_guard lock(this->net->mu);
        this->net->configured++;
        return {};
    }

    void async_read_some(std::span<std::uint8_t> buffer, Handler handler) override {
        if (const auto ec = this->net->call())
            return this->post(std::move(handler), ec);
        if (this->pipe == nullptr)
            return this->post(std::move(handler), asio::error::bad_descriptor);
        std::lock_guard lock(this->pipe->mu);
        this->pipe->read.emplace(
            Pipe::Read{
                .buffer = buffer,
                .pending = {
                    .handler = std::move(handler),
                    .work = asio::make_work_guard(this->ctx),
                },
            }
        );
        this->pipe->complete_read();
    }

    void
    async_write_some(std::span<const std::uint8_t> data, Handler handler) override {
        if (const auto ec = this->net->call())
            return this->post(std::move(handler), ec);
        if (this->pipe == nullptr)
            return this->post(std::move(handler), asio::error::bad_descriptor);
        std::lock_guard lock(this->pipe->mu);
        if (this->pipe->peer_closed)
            return this->post(std::move(handler), asio::error::connection_reset);
        const auto n = std::min(data.size(), this->pipe->write_size);
        this->pipe->to_peer
            .insert(this->pipe->to_peer.end(), data.begin(), data.begin() + n);
        this->pipe->cv.notify_all();
        this->post(std::move(handler), {}, n);
    }

    void cancel() override {
        if (this->connecting.has_value()) {
            this->post(
                std::move(this->connecting->handler),
                asio::error::operation_aborted
            );
            this->connecting.reset();
        }
        if (this->pipe == nullptr) return;
        std::lock_guard lock(this->pipe->mu);
        if (!this->pipe->read.has_value()) return;
        this->post(
            std::move(this->pipe->read->pending.handler),
            asio::error::operation_aborted
        );
        this->pipe->read.reset();
    }

    void close() override {
        this->cancel();
        if (this->pipe == nullptr) return;
        {
            std::lock_guard lock(this->pipe->mu);
            this->pipe->client_closed = true;
            this->pipe->cv.notify_all();
        }
        this->pipe.reset();
    }
};
}

Peer::Peer(std::shared_ptr<Pipe> pipe): pipe(std::move(pipe)) {}

void Peer::send(const std::string &data) const {
    std::lock_guard lock(this->pipe->mu);
    this->pipe->to_client.insert(this->pipe->to_client.end(), data.begin(), data.end());
    this->pipe->complete_read();
}

std::string Peer::receive(const std::size_t n, const x::telem::TimeSpan timeout) const {
    std::unique_lock lock(this->pipe->mu);
    this->pipe->cv.wait_for(lock, timeout.chrono(), [this, n] {
        return this->pipe->to_peer.size() >= n;
    });
    const auto size = std::min(n, this->pipe->to_peer.size());
    std::string out(this->pipe->to_peer.begin(), this->pipe->to_peer.begin() + size);
    this->pipe->to_peer.erase(
        this->pipe->to_peer.begin(),
        this->pipe->to_peer.begin() + size
    );
    return out;
}

void Peer::close() const {
    std::lock_guard lock(this->pipe->mu);
    this->pipe->peer_closed = true;
    this->pipe->complete_read();
}

bool Peer::closed() const {
    std::lock_guard lock(this->pipe->mu);
    return this->pipe->client_closed;
}

Network::Network(): state(std::make_shared<State>()) {}

void Network::name(const std::string &host, const asio::ip::address &address) const {
    std::lock_guard lock(this->state->mu);
    this->state->names[host] = address;
}

void Network::hang(const std::string &host) const {
    std::lock_guard lock(this->state->mu);
    this->state->hung.insert(host);
}

void Network::listen(const asio::ip::tcp::endpoint &endpoint) const {
    std::lock_guard lock(this->state->mu);
    this->state->listeners[endpoint];
}

void Network::unlisten(const asio::ip::tcp::endpoint &endpoint) const {
    std::lock_guard lock(this->state->mu);
    this->state->listeners.erase(endpoint);
}

void Network::drop(const asio::ip::tcp::endpoint &endpoint) const {
    std::lock_guard lock(this->state->mu);
    this->state->dropped.insert(endpoint);
}

void Network::fail_at(const std::size_t n, const std::error_code ec) const {
    std::lock_guard lock(this->state->mu);
    this->state->failures[this->state->calls + n] = ec;
}

void Network::read_size(const std::size_t n) const {
    std::lock_guard lock(this->state->mu);
    this->state->read_size = n;
}

void Network::write_size(const std::size_t n) const {
    std::lock_guard lock(this->state->mu);
    this->state->write_size = n;
}

std::shared_ptr<Peer> Network::accept(
    const asio::ip::tcp::endpoint &endpoint,
    const x::telem::TimeSpan timeout
) const {
    std::unique_lock lock(this->state->mu);
    const auto ready = this->state->cv.wait_for(lock, timeout.chrono(), [&] {
        const auto listener = this->state->listeners.find(endpoint);
        return listener != this->state->listeners.end() && !listener->second.empty();
    });
    if (!ready) return nullptr;
    auto &queue = this->state->listeners[endpoint];
    auto peer = queue.front();
    queue.pop_front();
    return peer;
}

std::size_t Network::configured() const {
    std::lock_guard lock(this->state->mu);
    return this->state->configured;
}

Sockets Network::sockets() const {
    return [state = this->state](asio::io_context &ctx) {
        return std::make_unique<Socket>(ctx, state);
    };
}
}
