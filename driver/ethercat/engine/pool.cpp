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
Pool::Pool(master::Factory factory): factory(std::move(factory)) {}

std::shared_ptr<Engine> Pool::acquire(
    const std::string &interface_name,
    const std::string &backend
) {
    std::lock_guard lock(this->mu);
    const std::string key = backend == "igh" ? "igh" : interface_name;
    const auto it = this->engines.find(key);
    if (it != this->engines.end()) return it->second;
    auto master = this->factory(interface_name, backend);
    auto eng = std::make_shared<Engine>(std::move(master));
    this->engines[key] = eng;
    return eng;
}

bool Pool::is_active(const std::string &interface) const {
    std::lock_guard lock(this->mu);
    const auto it = this->engines.find(interface);
    return it != this->engines.end() && it->second->running();
}

std::vector<SlaveInfo> Pool::get_slaves(const std::string &interface) const {
    std::lock_guard lock(this->mu);
    const auto it = this->engines.find(interface);
    if (it != this->engines.end()) return it->second->master->slaves();
    return {};
}
}