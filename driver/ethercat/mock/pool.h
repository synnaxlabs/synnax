// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/mock/master.h"

namespace ethercat::mock {
/// Mock implementation of engine::Pool for testing.
///
/// Allows pre-configuring mock masters for specific interfaces before tests run.
/// When acquire() is called, the pool creates engines using the configured mock
/// masters.
class Pool {
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Master>> masters;
    std::unordered_map<std::string, std::shared_ptr<engine::Engine>> engines;
    xerrors::Error inject_acquire_err;

public:
    Pool() = default;

    /// Configures a mock master for the given interface.
    /// Must be called before acquire() for that interface.
    void configure_interface(
        const std::string &interface_name,
        std::shared_ptr<Master> master
    ) {
        std::lock_guard lock(this->mu);
        this->masters[interface_name] = std::move(master);
    }

    /// Injects an error to be returned by acquire().
    void inject_acquire_error(const xerrors::Error &err) {
        this->inject_acquire_err = err;
    }

    /// Clears any injected acquire error.
    void clear_injected_errors() { this->inject_acquire_err = xerrors::NIL; }

    /// Acquires or creates an engine for the specified interface.
    std::pair<std::shared_ptr<engine::Engine>, xerrors::Error> acquire(
        const std::string &interface_name,
        telem::Rate rate,
        const std::string &backend = "auto"
    ) {
        std::lock_guard lock(this->mu);
        if (this->inject_acquire_err) return {nullptr, this->inject_acquire_err};

        const std::string key = backend == "igh" ? "igh" : interface_name;

        auto eng_it = this->engines.find(key);
        if (eng_it != this->engines.end()) {
            if (eng_it->second->cfg().cycle_time != rate.period())
                return {
                    nullptr,
                    xerrors::Error(
                        RATE_MISMATCH,
                        "engine already exists with different rate"
                    )
                };
            return {eng_it->second, xerrors::NIL};
        }

        auto master_it = this->masters.find(interface_name);
        if (master_it == this->masters.end())
            return {
                nullptr,
                xerrors::Error(
                    MASTER_INIT_ERROR,
                    "no mock master configured for interface"
                )
            };

        auto eng = std::make_shared<engine::Engine>(
            master_it->second,
            engine::Config(rate.period())
        );
        this->engines[key] = eng;
        return {eng, xerrors::NIL};
    }

    /// Checks if an interface has an active (running) engine.
    bool is_active(const std::string &interface) const {
        std::lock_guard lock(this->mu);
        auto it = this->engines.find(interface);
        return it != this->engines.end() && it->second->running();
    }

    /// Returns cached slave information from an interface's mock master.
    std::vector<SlaveInfo> get_slaves(const std::string &interface) const {
        std::lock_guard lock(this->mu);
        auto it = this->masters.find(interface);
        if (it != this->masters.end()) return it->second->slaves();
        return {};
    }

    /// Returns the mock master for an interface (for test verification).
    std::shared_ptr<Master> get_master(const std::string &interface) const {
        std::lock_guard lock(this->mu);
        auto it = this->masters.find(interface);
        if (it != this->masters.end()) return it->second;
        return nullptr;
    }

    /// Returns the engine for an interface (for test verification).
    std::shared_ptr<engine::Engine> get_engine(const std::string &interface) const {
        std::lock_guard lock(this->mu);
        auto it = this->engines.find(interface);
        if (it != this->engines.end()) return it->second;
        return nullptr;
    }
};
}
