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
/// every task on the device. A connection is destroyed when the last task releases it.
/// Safe for concurrent use.
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
        auto conn = create();
        entry = {.conn = conn, .settings = settings};
        return {std::move(conn), x::errors::NIL};
    }

private:
    struct Entry {
        std::weak_ptr<T> conn;
        x::json::json settings;
    };

    std::mutex mu;
    std::unordered_map<std::string, Entry> entries;
};

/// @brief acquires a task's shared connection to its device.
/// @returns transport::CONFIG_ERROR when the device is open with other settings.
template<typename T>
using Acquirer = std::function<std::pair<std::shared_ptr<T>, x::errors::Error>()>;
}
