// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <condition_variable>
#include <deque>
#include <map>
#include <mutex>
#include <set>
#include <vector>

#include "asio/error.hpp"
#include "asio/executor_work_guard.hpp"
#include "asio/post.hpp"

#include "driver/udp/model.h"

namespace driver::udp::model {
namespace {
struct Datagram {
    std::string data;
    asio::ip::udp::endpoint from;
};

/// @brief a receive that waits for a datagram. The work guard keeps the io_context
/// running, as a real socket's registration with the reactor does.
struct Receive {
    std::span<std::uint8_t> buffer;
    asio::ip::udp::endpoint *sender;
    Handler handler;
    asio::executor_work_guard<asio::io_context::executor_type> work;
};

/// @returns the any address of address's IP version.
asio::ip::address any_address(const asio::ip::address &address) {
    if (address.is_v6()) return asio::ip::address_v6::any();
    return asio::ip::address_v4::any();
}
}

/// @brief the datagrams queued for one bound socket.
struct Mailbox {
    asio::ip::udp::endpoint at;
    /// @brief true for a peer's socket on another host.
    bool remote = false;
    bool shared = false;
    std::set<asio::ip::address> groups;
    std::deque<Datagram> queue;
    asio::io_context *ctx = nullptr;
    std::optional<Receive> receive;

    /// @brief completes the pending receive when a datagram is queued. The caller
    /// holds the network's lock.
    void complete() {
        if (!this->receive.has_value() || this->queue.empty()) return;
        const auto datagram = std::move(this->queue.front());
        this->queue.pop_front();
        const auto n = std::min(this->receive->buffer.size(), datagram.data.size());
        std::copy_n(datagram.data.begin(), n, this->receive->buffer.begin());
        *this->receive->sender = datagram.from;
        asio::post(*this->ctx, [handler = std::move(this->receive->handler), n] {
            handler({}, n);
        });
        this->receive.reset();
    }
};

struct State {
    std::mutex mu;
    std::condition_variable cv;
    std::set<asio::ip::address> local{asio::ip::address_v4::loopback()};
    std::map<std::string, asio::ip::address> names;
    std::vector<std::shared_ptr<Mailbox>> bound;
    std::uint16_t next_port = 49152;
    std::size_t lose = 0;
    std::size_t duplicate = 0;
    std::size_t calls = 0;
    std::map<std::size_t, std::error_code> failures;

    /// @brief counts one stack call.
    /// @returns the error that fail_at scheduled for the call.
    std::error_code call() {
        std::lock_guard lock(this->mu);
        this->calls++;
        const auto failure = this->failures.find(this->calls);
        if (failure == this->failures.end()) return {};
        return failure->second;
    }

    /// @brief binds a mailbox at the endpoint. The caller holds mu.
    std::pair<std::shared_ptr<Mailbox>, std::error_code>
    bind(asio::ip::udp::endpoint at, const bool shared) {
        if (at.port() == 0) at.port(this->next_port++);
        for (const auto &other: this->bound) {
            const auto overlaps = !other->remote && other->at.port() == at.port() &&
                                  (other->at.address() == at.address() ||
                                   other->at.address().is_unspecified() ||
                                   at.address().is_unspecified());
            if (overlaps && !(shared && other->shared))
                return {nullptr, asio::error::address_in_use};
        }
        auto mailbox = std::make_shared<Mailbox>();
        mailbox->at = at;
        mailbox->shared = shared;
        this->bound.push_back(mailbox);
        return {mailbox, {}};
    }

    /// @brief delivers data from one endpoint to another, losing or duplicating it as
    /// the test asked. The caller holds mu.
    void deliver(
        const std::string &data,
        const asio::ip::udp::endpoint &from,
        const asio::ip::udp::endpoint &to
    ) {
        if (this->lose > 0) {
            this->lose--;
            return;
        }
        std::size_t copies = 1;
        if (this->duplicate > 0) {
            this->duplicate--;
            copies = 2;
        }
        std::vector<std::shared_ptr<Mailbox>> targets;
        if (to.address().is_multicast()) {
            for (const auto &m: this->bound)
                if (m->at.port() == to.port() && m->groups.contains(to.address()))
                    targets.push_back(m);
        } else {
            const auto exact = std::ranges::find_if(this->bound, [&to](const auto &m) {
                return m->at == to;
            });
            const auto any = std::ranges::find_if(this->bound, [&to](const auto &m) {
                return m->at.port() == to.port() &&
                       m->at.address() == any_address(to.address());
            });
            if (exact != this->bound.end())
                targets.push_back(*exact);
            else if (any != this->bound.end() && this->local.contains(to.address()))
                targets.push_back(*any);
        }
        for (const auto &m: targets) {
            for (std::size_t i = 0; i < copies; i++)
                m->queue.push_back({.data = data, .from = from});
            m->complete();
        }
        this->cv.notify_all();
    }

    void unbind(const std::shared_ptr<Mailbox> &mailbox) {
        std::erase(this->bound, mailbox);
    }
};

namespace {
class Stack final : public udp::Stack {
    asio::io_context &ctx;
    std::shared_ptr<State> net;
    std::optional<asio::ip::udp> protocol;
    bool shared = false;
    std::shared_ptr<Mailbox> mailbox;

    void post(Handler handler, const std::error_code &ec, const std::size_t n = 0) {
        asio::post(this->ctx, [handler = std::move(handler), ec, n] {
            handler(ec, n);
        });
    }

    /// @returns the address that a datagram from this socket carries as its source.
    [[nodiscard]] asio::ip::udp::endpoint source() const {
        auto from = this->mailbox->at;
        if (from.address().is_unspecified())
            from.address(asio::ip::address_v4::loopback());
        return from;
    }

public:
    Stack(asio::io_context &ctx, std::shared_ptr<State> net):
        ctx(ctx), net(std::move(net)) {}

    ~Stack() override { this->close(); }

    std::pair<std::vector<asio::ip::udp::endpoint>, std::error_code>
    resolve(const std::string &host, const std::uint16_t port) override {
        if (const auto ec = this->net->call()) return {{}, ec};
        {
            std::lock_guard lock(this->net->mu);
            if (const auto named = this->net->names.find(host);
                named != this->net->names.end())
                return {{{named->second, port}}, {}};
        }
        std::error_code ec;
        const auto address = asio::ip::make_address(host, ec);
        if (ec) return {{}, asio::error::host_not_found};
        return {{{address, port}}, {}};
    }

    std::error_code open(const asio::ip::udp &protocol) override {
        if (const auto ec = this->net->call()) return ec;
        this->protocol = protocol;
        return {};
    }

    std::error_code share() override {
        if (const auto ec = this->net->call()) return ec;
        if (!this->protocol) return asio::error::bad_descriptor;
        this->shared = true;
        return {};
    }

    std::error_code bind(const asio::ip::udp::endpoint &local) override {
        if (const auto ec = this->net->call()) return ec;
        if (!this->protocol) return asio::error::bad_descriptor;
        if (this->mailbox != nullptr) return asio::error::invalid_argument;
        std::lock_guard lock(this->net->mu);
        if (!local.address().is_unspecified() &&
            !this->net->local.contains(local.address()))
            return std::make_error_code(std::errc::address_not_available);
        auto [mailbox, ec] = this->net->bind(local, this->shared);
        if (ec) return ec;
        mailbox->ctx = &this->ctx;
        this->mailbox = mailbox;
        return {};
    }

    std::error_code join(
        const asio::ip::address &group,
        const std::optional<asio::ip::address> &iface
    ) override {
        if (const auto ec = this->net->call()) return ec;
        if (this->mailbox == nullptr) return asio::error::bad_descriptor;
        std::lock_guard lock(this->net->mu);
        if (iface && !this->net->local.contains(*iface))
            return asio::error::no_such_device;
        this->mailbox->groups.insert(group);
        return {};
    }

    void async_receive_from(
        std::span<std::uint8_t> buffer,
        asio::ip::udp::endpoint &sender,
        Handler handler
    ) override {
        if (const auto ec = this->net->call())
            return this->post(std::move(handler), ec);
        if (this->mailbox == nullptr)
            return this->post(std::move(handler), asio::error::bad_descriptor);
        std::lock_guard lock(this->net->mu);
        this->mailbox->receive.emplace(
            Receive{
                .buffer = buffer,
                .sender = &sender,
                .handler = std::move(handler),
                .work = asio::make_work_guard(this->ctx),
            }
        );
        this->mailbox->complete();
    }

    void async_send_to(
        std::span<const std::uint8_t> data,
        const asio::ip::udp::endpoint &to,
        Handler handler
    ) override {
        if (const auto ec = this->net->call())
            return this->post(std::move(handler), ec);
        if (this->mailbox == nullptr)
            return this->post(std::move(handler), asio::error::bad_descriptor);
        std::lock_guard lock(this->net->mu);
        this->net->deliver(std::string(data.begin(), data.end()), this->source(), to);
        this->post(std::move(handler), {}, data.size());
    }

    void cancel() override {
        if (this->mailbox == nullptr) return;
        std::lock_guard lock(this->net->mu);
        if (!this->mailbox->receive.has_value()) return;
        this->post(
            std::move(this->mailbox->receive->handler),
            asio::error::operation_aborted
        );
        this->mailbox->receive.reset();
    }

    void close() override {
        this->cancel();
        this->protocol.reset();
        if (this->mailbox == nullptr) return;
        std::lock_guard lock(this->net->mu);
        this->net->unbind(this->mailbox);
        this->mailbox.reset();
    }
};
}

Peer::Peer(std::shared_ptr<State> net, asio::ip::udp::endpoint at):
    net(std::move(net)), at(std::move(at)) {}

asio::ip::udp::endpoint Peer::endpoint() const {
    return this->at;
}

void Peer::send(const std::string &data, const asio::ip::udp::endpoint &to) const {
    std::lock_guard lock(this->net->mu);
    this->net->deliver(data, this->at, to);
}

std::optional<std::pair<std::string, asio::ip::udp::endpoint>>
Peer::receive(const x::telem::TimeSpan timeout) const {
    std::unique_lock lock(this->net->mu);
    const auto mailbox = std::ranges::find_if(this->net->bound, [this](const auto &m) {
        return m->at == this->at;
    });
    if (mailbox == this->net->bound.end()) return std::nullopt;
    const auto box = *mailbox;
    if (!this->net->cv.wait_for(lock, timeout.chrono(), [&box] {
            return !box->queue.empty();
        }))
        return std::nullopt;
    auto datagram = std::move(box->queue.front());
    box->queue.pop_front();
    return std::pair{std::move(datagram.data), datagram.from};
}

void Peer::close() const {
    std::lock_guard lock(this->net->mu);
    std::erase_if(this->net->bound, [this](const auto &m) {
        return m->remote && m->at == this->at;
    });
}

Network::Network(): state(std::make_shared<State>()) {}

void Network::add(const asio::ip::address &address) const {
    std::lock_guard lock(this->state->mu);
    this->state->local.insert(address);
}

void Network::name(const std::string &host, const asio::ip::address &address) const {
    std::lock_guard lock(this->state->mu);
    this->state->names[host] = address;
}

Peer Network::bind(const asio::ip::udp::endpoint &endpoint) const {
    std::lock_guard lock(this->state->mu);
    auto mailbox = std::make_shared<Mailbox>();
    mailbox->at = endpoint;
    mailbox->remote = true;
    this->state->bound.push_back(mailbox);
    return {this->state, endpoint};
}

void Network::lose(const std::size_t n) const {
    std::lock_guard lock(this->state->mu);
    this->state->lose = n;
}

void Network::duplicate(const std::size_t n) const {
    std::lock_guard lock(this->state->mu);
    this->state->duplicate = n;
}

void Network::fail_at(const std::size_t n, const std::error_code ec) const {
    std::lock_guard lock(this->state->mu);
    this->state->failures[this->state->calls + n] = ec;
}

Stacks Network::stacks() const {
    return [state = this->state](asio::io_context &ctx) {
        return std::make_unique<Stack>(ctx, state);
    };
}
}
