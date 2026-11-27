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
#include <optional>
#include <ostream>
#include <string>
#include <thread>

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/log/log.h"
#include "x/cpp/telem/telem.h"

#include "driver/rack/status/status.h"

namespace driver::rack::key {
/// @brief settings for the renewal of the Driver's API key.
struct Config {
    /// @brief the part of the key's lifetime that is left when the Driver rotates it.
    double renew_lead = 0.2;
    /// @brief the wait after the first failed rotation or save.
    x::telem::TimeSpan retry_base_interval = 10 * x::telem::SECOND;
    /// @brief the longest wait between attempts.
    x::telem::TimeSpan retry_max_interval = 10 * x::telem::MINUTE;

    template<typename Parser>
    void override(Parser &p) {
        this->renew_lead = p.field("renew_lead", this->renew_lead);
        this->retry_base_interval = x::telem::SECOND *
                                    p.field(
                                        "renew_retry_base_interval",
                                        this->retry_base_interval.seconds()
                                    );
        this->retry_max_interval = x::telem::SECOND *
                                   p.field(
                                       "renew_retry_max_interval",
                                       this->retry_max_interval.seconds()
                                   );
        if (this->renew_lead <= 0 || this->renew_lead >= 1)
            p.field_err("renew_lead", "must be between 0 and 1");
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "  " << x::log::SHALE() << "API key renew lead" << x::log::RESET() << ": "
           << cfg.renew_lead << "\n"
           << "  " << x::log::SHALE() << "API key renew retry" << x::log::RESET()
           << ": " << cfg.retry_base_interval << " to " << cfg.retry_max_interval;
        return os;
    }
};

/// @brief saves the token of a rotated key where the Driver loads its key from.
using Save = std::function<x::errors::Error(const std::string &token)>;

/// @brief rotates the API key that a client logged in with before the key expires,
/// and reports problems with the key as rack health. A key that never expires is never
/// rotated.
class Renewer {
    Config cfg;
    std::shared_ptr<synnax::Synnax> client;
    /// @brief null when the Driver cannot save a new key, so it only warns before the
    /// key expires.
    Save save;
    x::breaker::Breaker breaker;
    std::thread thread;
    mutable std::mutex mu;
    std::optional<status::Health> current;

    void run();
    void set_health(std::optional<status::Health> health);
    /// @brief returns the key the client logged in with, retrying until it succeeds.
    /// Returns nothing when stopped first.
    std::optional<synnax::auth::apikey::APIKey> own_key();
    /// @brief rotates the key, saves the new one, and returns it. Retries until it
    /// succeeds. Returns nothing when stopped first or when the key has expired.
    std::optional<synnax::auth::apikey::APIKey>
    rotate(const synnax::auth::apikey::APIKey &key);
    /// @brief waits the backoff interval after the given number of failures.
    void back_off(size_t failures);

public:
    /// @brief starts the renewer. save is null when the Driver did not load its key
    /// from its persisted state.
    Renewer(Config cfg, std::shared_ptr<synnax::Synnax> client, Save save);

    ~Renewer();

    /// @brief stops renewal and waits for it to exit. Idempotent.
    void stop();

    Renewer(const Renewer &) = delete;
    Renewer &operator=(const Renewer &) = delete;

    /// @brief returns what the rack status reports about the key, or nothing when the
    /// key is fine. Safe to call from any thread.
    [[nodiscard]] std::optional<status::Health> health() const;
};
}
