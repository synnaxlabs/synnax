// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/key/key.h"

namespace driver::rack::key {
namespace {
const Config FAST{
    .renew_lead = 0.5,
    .retry_base_interval = 50 * x::telem::MILLISECOND,
    .retry_max_interval = 100 * x::telem::MILLISECOND,
};

/// @brief records the tokens that a Renewer saves, and fails the first fail_count
/// saves.
struct Saved {
    std::mutex mu;
    std::vector<std::string> tokens;
    int fail_count = 0;
    int attempts = 0;

    Save save() {
        return [this](const std::string &token) {
            std::lock_guard lock(this->mu);
            if (this->attempts++ < this->fail_count)
                return x::errors::Error(x::errors::VALIDATION, "disk full");
            this->tokens.push_back(token);
            return x::errors::NIL;
        };
    }

    size_t count() {
        std::lock_guard lock(this->mu);
        return this->tokens.size();
    }

    std::string last() {
        std::lock_guard lock(this->mu);
        return this->tokens.back();
    }
};

/// @brief returns a client that logs in as a new rack with a key that expires after
/// lifetime, or never when lifetime is empty.
std::shared_ptr<synnax::Synnax>
rack_client(const std::optional<x::telem::TimeSpan> &lifetime) {
    const auto admin = new_test_client();
    const auto rack = ASSERT_NIL_P(admin.racks.create("key renewal"));
    auto expires = x::telem::TIME_STAMP_MAX;
    if (lifetime) expires = x::telem::TimeStamp::now() + *lifetime;
    const auto created = ASSERT_NIL_P(
        admin.api_keys.create("renewed", synnax::rack::resource_id(rack.key), expires)
    );
    auto cfg = test_client_config();
    cfg.api_key = created.token;
    return std::make_shared<synnax::Synnax>(cfg);
}

/// @brief returns the message of the key problem that renewer reports, or "" for none.
std::string health_message(const Renewer &renewer) {
    const auto health = renewer.health();
    if (!health) return "";
    return health->message;
}

x::errors::Error log_in(const std::string &token) {
    auto cfg = test_client_config();
    cfg.api_key = token;
    return synnax::Synnax(cfg).auth->authenticate();
}
}

/// @brief it should rotate the key before it expires, save the new token, and keep the
/// client logged in with it.
TEST(KeyRenewerTest, RotatesBeforeExpiry) {
    const auto client = rack_client(4 * x::telem::SECOND);
    ASSERT_NIL(client->auth->authenticate());
    const auto old = client->auth->current_session().credential;
    Saved saved;
    const Renewer renewer(FAST, client, saved.save());
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        saved.count(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(50)
    );
    ASSERT_NIL(log_in(saved.last()));
    ASSERT_NIL(client->auth->logout());
    ASSERT_NIL(client->auth->authenticate());
    ASSERT_EQ(client->auth->current_session().credential, old);
    ASSERT_EVENTUALLY_EQ(health_message(renewer), "");
}

/// @brief it should never rotate a key that never expires.
TEST(KeyRenewerTest, KeepsKeyThatNeverExpires) {
    const auto client = rack_client(std::nullopt);
    Saved saved;
    const Renewer renewer(FAST, client, saved.save());
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    ASSERT_EQ(saved.count(), 0);
    ASSERT_EQ(health_message(renewer), "");
}

/// @brief it should report a failed save and try the save again until it succeeds.
TEST(KeyRenewerTest, RetriesFailedSave) {
    const auto client = rack_client(2 * x::telem::SECOND);
    Saved saved;
    saved.fail_count = 3;
    const Renewer renewer(
        Config{
            .renew_lead = 0.5,
            .retry_base_interval = 300 * x::telem::MILLISECOND,
            .retry_max_interval = 300 * x::telem::MILLISECOND,
        },
        client,
        saved.save()
    );
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        health_message(renewer),
        std::string("Failed to save API key"),
        std::chrono::seconds(3),
        std::chrono::milliseconds(20)
    );
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        saved.count(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(50)
    );
    ASSERT_EVENTUALLY_EQ(health_message(renewer), "");
    ASSERT_NIL(log_in(saved.last()));
}

/// @brief it should warn before expiry, and not rotate, when it cannot save a key.
TEST(KeyRenewerTest, WarnsWhenItCannotSave) {
    const auto client = rack_client(2 * x::telem::SECOND);
    ASSERT_NIL(client->auth->authenticate());
    const auto credential = client->auth->current_session().credential;
    const Renewer renewer(FAST, client, nullptr);
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        health_message(renewer),
        std::string("API key expires soon"),
        std::chrono::seconds(3),
        std::chrono::milliseconds(20)
    );
    ASSERT_EQ(renewer.health()->variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(client->auth->current_session().credential, credential);
}

/// @brief it should stop without waiting for the key to expire.
TEST(KeyRenewerTest, StopsPromptly) {
    const auto client = rack_client(x::telem::HOUR);
    Saved saved;
    Renewer renewer(FAST, client, saved.save());
    const auto start = std::chrono::steady_clock::now();
    renewer.stop();
    ASSERT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(5));
    ASSERT_EQ(saved.count(), 0);
}
}
