// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/transport/errors.h"

namespace driver::bus {
/// @brief the live connections of an integration's devices, one per device, shared by
/// every task on the device. When the last task releases a connection, the connection
/// is destroyed and the registry forgets the device. Safe for concurrent use.
/// @tparam T the connection type.
template<typename T>
class Registry {
public:
    /// @brief returns the live connection to the device, or creates one.
    /// @param key the device key.
    /// @param settings how the device opens. Every task on the device must use the same
    /// settings.
    /// @param create creates the connection. It must not block on I/O.
    /// @returns transport::CONFIG_ERROR when the device is open with other settings.
    std::pair<std::shared_ptr<T>, x::errors::Error> acquire(
        const std::string &key,
        const x::json::json &settings,
        const std::function<std::shared_ptr<T>()> &create
    ) {
        std::lock_guard lock(this->state->mu);
        auto &entry = this->state->entries[key];
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
        auto created = create();
        auto *raw = created.get();
        // The last release destroys the connection, then forgets the device unless
        // another acquire has already replaced it.
        std::shared_ptr<T> conn(
            raw,
            [created = std::move(created),
             state = std::weak_ptr(this->state),
             key](T *) mutable {
                created.reset();
                const auto s = state.lock();
                if (s == nullptr) return;
                std::lock_guard lock(s->mu);
                if (const auto it = s->entries.find(key);
                    it != s->entries.end() && it->second.conn.expired())
                    s->entries.erase(it);
            }
        );
        entry = {.conn = conn, .settings = settings};
        return {std::move(conn), x::errors::NIL};
    }

    /// @returns the live connection to the device, or nullptr when no task holds one.
    std::shared_ptr<T> find(const std::string &key) {
        std::lock_guard lock(this->state->mu);
        const auto it = this->state->entries.find(key);
        if (it == this->state->entries.end()) return nullptr;
        return it->second.conn.lock();
    }

private:
    struct Entry {
        std::weak_ptr<T> conn;
        x::json::json settings;
    };

    struct State {
        std::mutex mu;
        std::unordered_map<std::string, Entry> entries;
    };

    /// @brief shared with each connection's deleter, which can outlive the registry.
    std::shared_ptr<State> state = std::make_shared<State>();
};

/// @brief acquires a task's shared connection to its device.
/// @returns transport::CONFIG_ERROR when the device is open with other settings.
template<typename T>
using Acquirer = std::function<std::pair<std::shared_ptr<T>, x::errors::Error>()>;
}
