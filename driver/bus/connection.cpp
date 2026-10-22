// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "driver/bus/connection.h"

namespace driver::bus {
Connection::Connection(Config cfg):
    cfg(std::move(cfg)), backoff(this->cfg.min_backoff) {}

Connection::Guard::Guard(Connection &conn): conn(conn) {
    std::unique_lock lock(conn.mu);
    const auto ticket = conn.next_ticket++;
    conn.cv.wait(lock, [&] { return conn.serving == ticket; });
}

Connection::Guard::~Guard() {
    {
        std::lock_guard lock(this->conn.mu);
        this->conn.serving++;
    }
    this->conn.cv.notify_all();
}

std::pair<Transport *, x::errors::Error> Connection::Guard::transport() {
    std::lock_guard lock(this->conn.mu);
    if (this->conn.current == nullptr) return {nullptr, this->conn.err};
    return {this->conn.current.get(), x::errors::NIL};
}

void Connection::Guard::fail(const x::errors::Error &err) {
    std::unique_ptr<Transport> closed;
    {
        std::lock_guard lock(this->conn.mu);
        closed = std::move(this->conn.current);
        this->conn.failed(err);
    }
}

std::uint64_t Connection::Guard::opens() const {
    std::lock_guard lock(this->conn.mu);
    return this->conn.opened;
}

Connection::Guard Connection::lock() {
    this->open();
    return Guard(*this);
}

void Connection::open() {
    {
        std::unique_lock lock(this->mu);
        this->cv.wait(lock, [this] { return !this->opening; });
        if (this->current != nullptr || this->cfg.now() < this->next_open) return;
        this->opening = true;
    }
    auto [t, err] = this->cfg.open();
    {
        std::lock_guard lock(this->mu);
        this->opening = false;
        if (err)
            this->failed(err);
        else {
            this->current = std::move(t);
            this->opened++;
            this->err = x::errors::NIL;
            this->backoff = this->cfg.min_backoff;
        }
    }
    this->cv.notify_all();
}

void Connection::failed(const x::errors::Error &err) {
    this->err = err;
    this->next_open = this->cfg.now() + this->backoff;
    this->backoff = std::min(this->backoff * 2, this->cfg.max_backoff);
}

x::errors::Error Connection::error() {
    std::lock_guard lock(this->mu);
    return this->err;
}

x::telem::TimeStamp Connection::retry_at() {
    std::lock_guard lock(this->mu);
    return this->next_open;
}

Acquire acquirer(
    std::shared_ptr<Connections> connections,
    std::string key,
    x::json::json settings,
    Opener open
) {
    return [connections = std::move(connections),
            key = std::move(key),
            settings = std::move(settings),
            open = std::move(open)] {
        return connections->acquire(key, settings, [&open] {
            return std::make_shared<Connection>(Connection::Config{.open = open});
        });
    };
}
}
