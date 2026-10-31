// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <memory>
#include <mutex>
#include <unordered_map>
#include <unordered_set>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

namespace driver::task {
/// @brief writes task statuses to the Core and keeps the latest status of each task, so
/// it can send them again. Safe for concurrent use.
class StatusCache {
    /// @brief the latest status of one task, with no command key.
    struct Entry {
        /// @brief guards status and orders the task's writes, so a re-send never lands
        /// after a newer status.
        std::mutex mu;
        synnax::task::Status status;
        /// @brief true once the task has left this Driver. Set without mu, so a drop
        /// never waits on a write.
        std::atomic<bool> dropped = false;
    };

    const synnax::status::Client client;
    /// @brief guards entries and released. Never held during a write, so a task never
    /// waits on the write of another task.
    std::mutex mu;
    std::unordered_map<synnax::task::Key, std::shared_ptr<Entry>> entries;
    /// @brief the tasks that left this Driver and were not claimed again.
    std::unordered_set<synnax::task::Key> released;

    /// @brief returns the entry of the task of status, creating it with status if
    /// absent. Returns null for a released task.
    std::shared_ptr<Entry> entry_for(const synnax::task::Status &status);

public:
    explicit StatusCache(synnax::status::Client client);

    /// @brief writes the status and keeps it as the latest status of its task, unless
    /// the task left this Driver.
    /// @returns the error of the write, or NIL for a task that left this Driver.
    [[nodiscard]] x::errors::Error set(synnax::task::Status &status);

    /// @brief sends the latest status of each task again, stamped now and with no
    /// command key. Waits for a write of a task's own status in progress, then sends
    /// the latest status, so a failed write is repaired too. Writes one task at a time,
    /// so a task waits on at most one re-send write.
    /// @returns the first error of the writes. An error does not stop the writes of the
    /// other tasks.
    [[nodiscard]] x::errors::Error resend();

    /// @brief forgets the latest status of the task and discards its later statuses, so
    /// that neither a re-send nor a late report writes one. Call it once the task has
    /// left this Driver. Never waits on a write.
    void drop(const synnax::task::Key &key);

    /// @brief writes and keeps the statuses of the task again after drop. Call it
    /// before the task runs on this Driver.
    void claim(const synnax::task::Key &key);
};
}
