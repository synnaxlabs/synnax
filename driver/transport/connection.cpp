// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/transport/connection.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::transport {
Connection::Connection(Config cfg):
    cfg(std::move(cfg)), backoff(this->cfg.min_backoff) {}

Connection::Guard::Guard(Connection &conn, const std::uint64_t ticket): conn(conn) {
    std::unique_lock lock(conn.mu);
    conn.cv.wait(lock, [&] { return conn.serving == ticket; });
}

Connection::Guard::~Guard() {
    {
        std::lock_guard lock(this->conn.mu);
        if (this->healthy) this->conn.backoff = this->conn.cfg.min_backoff;
        this->conn.serving++;
    }
    this->conn.cv.notify_all();
}

std::pair<Transport *, x::errors::Error> Connection::Guard::transport() {
    std::lock_guard lock(this->conn.mu);
    if (this->conn.current == nullptr) return {nullptr, this->conn.err};
    this->healthy = true;
    return {this->conn.current.get(), x::errors::NIL};
}

void Connection::Guard::fail(const x::errors::Error &err) {
    std::lock_guard lock(this->conn.mu);
    this->healthy = false;
    this->conn.current.reset();
    this->conn.failed(err);
}

std::uint64_t Connection::Guard::opens() const {
    std::lock_guard lock(this->conn.mu);
    return this->conn.opened;
}

Connection::Guard Connection::lock() {
    std::uint64_t ticket = 0;
    {
        std::lock_guard lock(this->mu);
        ticket = this->next_ticket++;
    }
    this->open();
    return Guard(*this, ticket);
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
        }
    }
    this->cv.notify_all();
}

void Connection::failed(const x::errors::Error &err) {
    this->err = err;
    this->next_open = this->cfg.now() + this->backoff.chrono();
    this->backoff = std::min(this->backoff * 2, this->cfg.max_backoff);
}

std::chrono::steady_clock::time_point Connection::retry_at() {
    std::lock_guard lock(this->mu);
    return this->next_open;
}

std::pair<std::shared_ptr<Connection>, x::errors::Error> Connections::acquire(
    const std::string &key,
    const x::json::json &settings,
    Connection::Config cfg
) {
    std::lock_guard lock(this->mu);
    auto &entry = this->entries[key];
    if (auto conn = entry.conn.lock()) {
        if (entry.settings != settings)
            return {
                nullptr,
                x::errors::Error(
                    CONFIG_ERROR,
                    "another task has device " + key + " open with other settings"
                ),
            };
        return {std::move(conn), x::errors::NIL};
    }
    auto conn = std::make_shared<Connection>(std::move(cfg));
    entry = {.conn = conn, .settings = settings};
    return {std::move(conn), x::errors::NIL};
}
}
