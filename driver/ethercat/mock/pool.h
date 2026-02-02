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
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/mock/master.h"

namespace ethercat::mock {

/// @brief mock implementation of engine::Pool for testing.
class Pool {
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Master>> masters;
    std::unordered_map<std::string, std::shared_ptr<engine::Engine>> engines;
    std::vector<master::Info> master_infos;
    xerrors::Error inject_acquire_err;

public:
    Pool() = default;

    /// @brief configures a mock master for the given key.
    void configure_master(const std::string &key, std::shared_ptr<Master> master) {
        std::lock_guard lock(this->mu);
        this->masters[key] = std::move(master);
        this->master_infos.push_back({key, ""});
    }

    /// @brief injects an error to be returned by acquire().
    void inject_acquire_error(const xerrors::Error &err) {
        this->inject_acquire_err = err;
    }

    /// @brief clears any injected acquire error.
    void clear_injected_errors() { this->inject_acquire_err = xerrors::NIL; }

    /// @brief returns configured master infos.
    [[nodiscard]] std::vector<master::Info> enumerate() const {
        std::lock_guard lock(this->mu);
        return this->master_infos;
    }

    /// @brief acquires or creates an engine for the specified master.
    std::pair<std::shared_ptr<engine::Engine>, xerrors::Error>
    acquire(const std::string &key) {
        std::lock_guard lock(this->mu);
        if (this->inject_acquire_err) return {nullptr, this->inject_acquire_err};

        auto eng_it = this->engines.find(key);
        if (eng_it != this->engines.end()) return {eng_it->second, xerrors::NIL};

        auto master_it = this->masters.find(key);
        if (master_it == this->masters.end())
            return {
                nullptr,
                xerrors::Error(
                    MASTER_INIT_ERROR,
                    "no mock master configured for key: " + key
                )
            };

        auto eng = std::make_shared<engine::Engine>(master_it->second);
        this->engines[key] = eng;
        return {eng, xerrors::NIL};
    }

    /// @brief checks if a key has an active (running) engine.
    [[nodiscard]] bool is_active(const std::string &key) const {
        std::lock_guard lock(this->mu);
        auto it = this->engines.find(key);
        return it != this->engines.end() && it->second->running();
    }

    /// @brief returns cached slave information from a key's mock master.
    [[nodiscard]] std::vector<SlaveInfo> get_slaves(const std::string &key) const {
        std::lock_guard lock(this->mu);
        auto it = this->masters.find(key);
        if (it != this->masters.end()) return it->second->slaves();
        return {};
    }

    /// @brief returns the mock master for a key (for test verification).
    [[nodiscard]] std::shared_ptr<Master> get_master(const std::string &key) const {
        std::lock_guard lock(this->mu);
        auto it = this->masters.find(key);
        if (it != this->masters.end()) return it->second;
        return nullptr;
    }

    /// @brief returns the engine for a key (for test verification).
    [[nodiscard]] std::shared_ptr<engine::Engine>
    get_engine(const std::string &key) const {
        std::lock_guard lock(this->mu);
        auto it = this->engines.find(key);
        if (it != this->engines.end()) return it->second;
        return nullptr;
    }
};

}
