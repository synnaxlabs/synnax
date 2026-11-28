// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "absl/log/log.h"

#include "client/cpp/status/status.h"
#include "x/cpp/thread/thread.h"

#include "driver/rack/key/key.h"

namespace driver::rack::key {
namespace {
const std::string LOGIN_HINT = "Run synnax-driver login to give the Driver a new key.";

bool never_expires(const synnax::auth::apikey::APIKey &key) {
    return key.expires == x::telem::TIME_STAMP_MAX;
}

std::string expiry_text(const synnax::auth::apikey::APIKey &key) {
    return "The key expires at " + key.expires.iso8601() + ".";
}
}

Renewer::Renewer(const Config &cfg, std::shared_ptr<synnax::Synnax> client, Save save):
    cfg(cfg),
    client(std::move(client)),
    save(std::move(save)),
    breaker(x::breaker::Config{.name = "api_key"}) {
    this->breaker.start();
    this->thread = std::thread([this] { this->run(); });
}

Renewer::~Renewer() {
    this->stop();
}

void Renewer::stop() {
    this->breaker.stop();
    if (this->thread.joinable()) this->thread.join();
}

std::optional<status::Health> Renewer::health() const {
    std::lock_guard lock(this->mu);
    return this->current;
}

void Renewer::set_health(std::optional<status::Health> health) {
    std::lock_guard lock(this->mu);
    this->current = std::move(health);
}

void Renewer::back_off(const size_t failures) {
    const x::breaker::Config retry{
        .base_interval = this->cfg.retry_base_interval,
        .scale = 2,
        .max_interval = this->cfg.retry_max_interval,
    };
    this->breaker.wait_for(retry.interval(failures));
}

std::optional<synnax::auth::apikey::APIKey> Renewer::own_key() {
    for (size_t failures = 1; this->breaker.running(); failures++) {
        auto [keys, err] = this->client->api_keys.retrieve();
        if (!err) {
            const auto credential = this->client->auth->current_session().credential;
            const auto it = std::ranges::find_if(keys, [&](const auto &k) {
                return k.key == credential;
            });
            if (it != keys.end()) return *it;
            err = x::errors::Error(
                x::errors::NOT_FOUND,
                "the key that the Driver logged in with was not found"
            );
        }
        LOG(WARNING) << "[api_key] failed to retrieve the API key: " << err;
        this->back_off(failures);
    }
    return std::nullopt;
}

std::optional<synnax::auth::apikey::APIKey>
Renewer::rotate(const synnax::auth::apikey::APIKey &key) {
    std::optional<synnax::auth::apikey::New> rotated;
    for (size_t failures = 1; this->breaker.running(); failures++) {
        if (!rotated) {
            auto [res, err] = this->client->api_keys.rotate();
            if (err.matches(synnax::auth::ERR_INVALID_CREDENTIALS)) {
                LOG(ERROR) << "[api_key] the API key expired before it was rotated";
                this->set_health(
                    status::Health{
                        .variant = synnax::status::VARIANT_ERROR,
                        .message = "API key expired",
                        .description = "The Driver could not rotate its API key "
                                       "before it expired. " +
                                       LOGIN_HINT,
                    }
                );
                return std::nullopt;
            }
            if (err) {
                LOG(WARNING) << "[api_key] failed to rotate the API key: " << err;
                this->set_health(
                    status::Health{
                        .variant = synnax::status::VARIANT_WARNING,
                        .message = "Failed to rotate API key",
                        .description = err.message() + ". The Driver tries again. " +
                                       expiry_text(key),
                    }
                );
                this->back_off(failures);
                continue;
            }
            rotated = res;
            failures = 0;
        }
        // The Core revoked the old key, so the new key must reach the saved state.
        if (const auto err = this->save(rotated->token)) {
            LOG(ERROR) << "[api_key] failed to save the rotated API key: " << err;
            this->set_health(
                status::Health{
                    .variant = synnax::status::VARIANT_ERROR,
                    .message = "Failed to save API key",
                    .description = err.message() +
                                   ". The Driver rotated its API key and tries again "
                                   "to save it. If the Driver stops first, it cannot "
                                   "log in. " +
                                   LOGIN_HINT,
                }
            );
            this->back_off(std::max<size_t>(failures, 1));
            continue;
        }
        LOG(INFO) << "[api_key] rotated the API key. The new key expires at "
                  << rotated->key.expires.iso8601();
        this->set_health(std::nullopt);
        return rotated->key;
    }
    return std::nullopt;
}

void Renewer::run() {
    x::thread::set_name("api_key");
    auto key = this->own_key();
    while (key && !never_expires(*key)) {
        const auto lifetime = key->expires - key->created;
        const auto renew_at = key->expires - lifetime * this->cfg.renew_lead;
        this->breaker.wait_for(renew_at - x::telem::TimeStamp::now());
        if (!this->breaker.running()) return;
        if (!this->save) {
            LOG(WARNING) << "[api_key] the API key expires at "
                         << key->expires.iso8601();
            this->set_health(
                status::Health{
                    .variant = synnax::status::VARIANT_WARNING,
                    .message = "API key expires soon",
                    .description =
                        expiry_text(*key) +
                        " The Driver renews only a key that it saved "
                        "itself, not one from its configuration file, "
                        "SYNNAX_DRIVER_API_KEY, or --api-key. Replace the key, "
                        "or run synnax-driver login to give the Driver a "
                        "key that it renews.",
                }
            );
            return;
        }
        key = this->rotate(*key);
    }
}
}
