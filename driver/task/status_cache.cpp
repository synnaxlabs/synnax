// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <vector>

#include "driver/task/status_cache.h"

namespace driver::task {
StatusCache::StatusCache(synnax::status::Client client): client(std::move(client)) {}

std::shared_ptr<StatusCache::Entry>
StatusCache::entry_for(const synnax::task::Status &status) {
    std::lock_guard lock{this->mu};
    if (this->released.contains(status.details.task)) return nullptr;
    auto &entry = this->entries[status.details.task];
    if (entry != nullptr) return entry;
    // Filled before resend can reach it, so a re-send never sends an empty status.
    entry = std::make_shared<Entry>();
    entry->status = status;
    entry->status.details.cmd.clear();
    return entry;
}

x::errors::Error StatusCache::set(synnax::task::Status &status) {
    const auto entry = this->entry_for(status);
    if (entry == nullptr) return x::errors::NIL;
    std::lock_guard lock{entry->mu};
    // drop can run between the lookup and the lock.
    if (entry->dropped) return x::errors::NIL;
    entry->status = status;
    entry->status.details.cmd.clear();
    return this->client.set<synnax::task::StatusDetails>(status);
}

x::errors::Error StatusCache::resend() {
    std::vector<std::shared_ptr<Entry>> entries;
    {
        std::lock_guard lock{this->mu};
        entries.reserve(this->entries.size());
        for (const auto &[_, entry]: this->entries)
            entries.push_back(entry);
    }
    x::errors::Error first_err = x::errors::NIL;
    for (const auto &entry: entries) {
        std::lock_guard lock{entry->mu};
        // drop can run between the copy and the lock.
        if (entry->dropped) continue;
        auto status = entry->status;
        status.time = x::telem::TimeStamp::now();
        if (const auto err = this->client.set<synnax::task::StatusDetails>(status);
            err && !first_err)
            first_err = err;
    }
    return first_err;
}

void StatusCache::drop(const synnax::task::Key &key) {
    std::lock_guard lock{this->mu};
    this->released.insert(key);
    const auto it = this->entries.find(key);
    if (it == this->entries.end()) return;
    it->second->dropped = true;
    this->entries.erase(it);
}

void StatusCache::claim(const synnax::task::Key &key) {
    std::lock_guard lock{this->mu};
    this->released.erase(key);
}
}
