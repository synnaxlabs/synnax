// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/engine/pool.h"

namespace ethercat::engine {

Pool::Pool(std::unique_ptr<master::Manager> manager): manager(std::move(manager)) {}

std::vector<master::Info> Pool::enumerate() const {
    if (this->manager == nullptr) return {};
    return this->manager->enumerate();
}

std::pair<std::shared_ptr<Engine>, xerrors::Error>
Pool::acquire_unlocked(const std::string &key) {
    const auto it = this->engines.find(key);
    if (it != this->engines.end()) return {it->second, xerrors::NIL};
    auto [m, err] = this->manager->create(key);
    if (err) return {nullptr, err};
    auto eng = std::make_shared<Engine>(std::move(m));
    this->engines[key] = eng;
    return {eng, xerrors::NIL};
}

std::pair<std::shared_ptr<Engine>, xerrors::Error>
Pool::acquire(const std::string &key) {
    std::lock_guard lock(this->mu);
    return this->acquire_unlocked(key);
}

bool Pool::is_active(const std::string &key) const {
    std::lock_guard lock(this->mu);
    const auto it = this->engines.find(key);
    return it != this->engines.end() && it->second->running();
}

std::vector<SlaveInfo> Pool::get_slaves(const std::string &key) const {
    std::lock_guard lock(this->mu);
    const auto it = this->engines.find(key);
    if (it != this->engines.end()) return it->second->slaves();
    return {};
}

std::pair<std::vector<SlaveInfo>, xerrors::Error>
Pool::discover_slaves(const std::string &key) {
    std::lock_guard lock(this->mu);
    auto [engine, err] = this->acquire_unlocked(key);
    if (err) return {{}, err};
    if (!engine->running())
        if (auto init_err = engine->ensure_initialized()) return {{}, init_err};
    return {engine->slaves(), xerrors::NIL};
}

}
