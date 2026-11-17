// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/errors/errors.h"
#include "driver/ethercat/engine/pool.h"

namespace driver::ethercat::engine {

Pool::Pool(
    std::unique_ptr<master::Manager> manager,
    SlaveConfigs slave_configs,
    std::shared_ptr<x::thread::rt::Manager> rt_manager,
    const x::telem::TimeSpan start_timeout
):
    manager(std::move(manager)),
    rt_manager(std::move(rt_manager)),
    slave_configs(std::move(slave_configs)),
    start_timeout(start_timeout) {}

std::vector<master::Info> Pool::enumerate() const {
    if (this->manager == nullptr) return {};
    return this->manager->enumerate();
}

x::errors::Error Pool::check_available() const {
    if (this->manager != nullptr) return x::errors::NIL;
    return driver::errors::missing_lib(PCAP_LIBRARY_INFO);
}

std::pair<std::shared_ptr<Engine>, x::errors::Error>
Pool::acquire(const std::string &key) {
    std::lock_guard lock(this->mu);
    const auto it = this->engines.find(key);
    if (it != this->engines.end()) return {it->second, x::errors::NIL};
    if (auto err = this->check_available()) return {nullptr, err};
    auto [m, err] = this->manager->create(key);
    if (err) return {nullptr, err};
    engine::Config cfg;
    cfg.slave_configs = [slave_configs = this->slave_configs, key, name = m->name()] {
        return slave_configs(key, name);
    };
    cfg.start_timeout = this->start_timeout;
    cfg.rt_manager = this->rt_manager;
    auto eng = std::make_shared<Engine>(key, std::move(m), cfg);
    this->engines[key] = eng;
    return {eng, x::errors::NIL};
}

std::vector<slave::Health> Pool::health(const std::string &key) const {
    std::shared_ptr<Engine> engine;
    {
        std::lock_guard lock(this->mu);
        const auto it = this->engines.find(key);
        if (it == this->engines.end()) return {};
        engine = it->second;
    }
    return engine->health();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Pool::discover(const std::string &key) {
    auto [engine, err] = this->acquire(key);
    if (err) return {{}, err};
    return engine->discover();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Pool::rediscover(const std::string &key) {
    auto [engine, err] = this->acquire(key);
    if (err) return {{}, err};
    return engine->rediscover();
}

}
