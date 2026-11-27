// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <utility>

#include "driver/bus/holders.h"
#include "driver/errors/errors.h"

namespace driver::bus {
Holders::Claim::Claim(std::shared_ptr<Holders> holders, std::string key):
    holders(std::move(holders)), key(std::move(key)) {}

Holders::Claim::~Claim() {
    std::lock_guard lock(this->holders->mu);
    const auto it = this->holders->held.find(this->key);
    if (--it->second.claims == 0) this->holders->held.erase(it);
}

std::pair<std::unique_ptr<Holders::Claim>, std::optional<std::string>> Holders::take(
    const std::shared_ptr<Holders> &holders,
    const std::string &key,
    const std::string &id,
    const std::string &name
) {
    std::lock_guard lock(holders->mu);
    auto &holder = holders->held[key];
    if (holder.claims > 0 && holder.id != id) return {nullptr, holder.name};
    if (holder.claims == 0) holder = {.id = id, .name = name};
    holder.claims++;
    return {std::unique_ptr<Claim>(new Claim(holders, key)), std::nullopt};
}

Readers::Readers(std::string kind): kind(std::move(kind)) {}

std::pair<std::unique_ptr<Holders::Claim>, x::errors::Error>
Readers::claim(const std::string &device, const synnax::task::Task &task) {
    auto [claim, holder] = Holders::take(
        this->holders,
        device,
        task.key.to_string(),
        task.name
    );
    if (holder.has_value())
        return {
            nullptr,
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "A " + this->kind + " device runs one read task at a time, and the " +
                    "task '" + *holder +
                    "' reads from this device. Add these entries to that task."
            ),
        };
    return {std::move(claim), x::errors::NIL};
}
}
