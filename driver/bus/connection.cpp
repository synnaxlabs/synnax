// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "driver/bus/connection.h"

namespace driver::bus {
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
    if (this->conn.current == nullptr) {
        auto [t, err] = this->conn.open();
        if (err) return {nullptr, err};
        this->conn.current = std::move(t);
        this->conn.opened++;
    }
    return {this->conn.current.get(), x::errors::NIL};
}

void Connection::Guard::close() {
    this->conn.current.reset();
}

std::pair<std::shared_ptr<Connection>, x::errors::Error> Connections::acquire(
    const std::string &key,
    const x::json::json &settings,
    const Opener &open
) {
    std::lock_guard lock(this->mu);
    auto &entry = this->entries[key];
    if (auto conn = entry.conn.lock()) {
        if (entry.settings != settings)
            return {
                nullptr,
                x::errors::Error(
                    transport::CONFIG_ERROR,
                    "another task has device " + key +
                        " open with different settings. Stop or reconfigure it"
                )
            };
        return {std::move(conn), x::errors::NIL};
    }
    auto conn = std::make_shared<Connection>(open);
    entry = {.conn = conn, .settings = settings};
    return {std::move(conn), x::errors::NIL};
}

Acquire acquirer(
    std::shared_ptr<Connections> connections,
    std::string key,
    x::json::json settings,
    Opener open
) {
    return
        [connections = std::move(connections),
         key = std::move(key),
         settings = std::move(settings),
         open = std::move(open)] { return connections->acquire(key, settings, open); };
}
}
