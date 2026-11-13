// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/common/sync.h"

namespace driver::common {
namespace {
/// @brief a resource with an identity group (rack, serial, model) and a configuration
/// group (name, identifier).
struct Resource {
    std::string key;
    synnax::rack::Key rack = 0;
    std::string name;
    std::string serial;
    std::string model;
    std::string identifier;
    std::optional<synnax::status::Status<>> status;

    [[nodiscard]] x::json::json to_json() const {
        return {
            {"key", key},
            {"rack", rack},
            {"name", name},
            {"serial", serial},
            {"model", model},
            {"identifier", identifier},
        };
    }
};

/// @brief an in-memory store of resources that records each write.
struct FakeStore final : Store<Resource> {
    std::vector<Resource> stored;
    std::vector<Resource> identity_updates;
    std::vector<Resource> statuses;
    int creates = 0;
    x::errors::Error create_err;

    [[nodiscard]] std::string match_key(const Resource &r) const override {
        return r.serial;
    }

    [[nodiscard]] Resource
    merge_identity(Resource stored, const Resource &found) const override {
        stored.rack = found.rack;
        stored.serial = found.serial;
        stored.model = found.model;
        return stored;
    }

    std::pair<std::vector<Resource>, x::errors::Error>
    retrieve(const synnax::rack::Key rack) override {
        std::vector<Resource> res;
        for (const auto &r: this->stored)
            if (r.rack == rack) res.push_back(r);
        return {res, x::errors::NIL};
    }

    std::pair<std::optional<Resource>, x::errors::Error>
    find(const Resource &found) override {
        for (const auto &r: this->stored)
            if (r.serial == found.serial) return {r, x::errors::NIL};
        return {std::nullopt, x::errors::NIL};
    }

    x::errors::Error create(std::vector<Resource> &resources) override {
        this->creates++;
        if (this->create_err) return this->create_err;
        for (auto &r: resources) {
            r.key = "key-" + r.serial;
            this->stored.push_back(r);
        }
        return x::errors::NIL;
    }

    x::errors::Error update_identity(const Resource &resource) override {
        this->identity_updates.push_back(resource);
        return x::errors::NIL;
    }

    x::errors::Error set_statuses(const std::vector<Resource> &resources) override {
        this->statuses = resources;
        return x::errors::NIL;
    }
};

const synnax::rack::Key RACK = 1;

Resource found(const std::string &serial, const std::string &model = "9205") {
    return Resource{
        .rack = RACK,
        .name = "NI " + model,
        .serial = serial,
        .model = model
    };
}

/// @brief returns a sync over a fake store, and the store.
std::pair<std::unique_ptr<Sync<Resource>>, FakeStore *>
new_sync(std::vector<Resource> stored = {}) {
    auto store = std::make_unique<FakeStore>();
    store->stored = std::move(stored);
    auto *ptr = store.get();
    return {std::make_unique<Sync<Resource>>(std::move(store), RACK, "[test] "), ptr};
}
}

/// @brief it should create found hardware that no stored resource matches.
TEST(Sync, CreatesNewHardware) {
    auto [sync, store] = new_sync();
    ASSERT_NIL(sync->init());
    ASSERT_NIL(sync->sync({found("01A")}));
    ASSERT_EQ(store->stored.size(), 1);
    ASSERT_EQ(sync->get("01A")->key, "key-01A");
    ASSERT_EQ(store->statuses.at(0).status->message, "Device present");
}

/// @brief it should write only the identity group of tracked hardware that changed,
/// and keep the configuration group as stored.
TEST(Sync, UpdatesIdentityAndKeepsConfiguration) {
    auto stored = found("01A");
    stored.key = "key-01A";
    stored.name = "Thermocouples";
    stored.identifier = "tc";
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->init());
    ASSERT_NIL(sync->sync({found("01A", "9211")}));
    ASSERT_EQ(store->creates, 0);
    ASSERT_EQ(store->identity_updates.size(), 1);
    const auto &update = store->identity_updates.at(0);
    ASSERT_EQ(update.model, "9211");
    ASSERT_EQ(update.name, "Thermocouples");
    ASSERT_EQ(update.identifier, "tc");
}

/// @brief it should not write a tracked resource whose identity did not change.
TEST(Sync, SkipsUnchangedIdentity) {
    auto stored = found("01A");
    stored.key = "key-01A";
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->init());
    ASSERT_NIL(sync->sync({found("01A")}));
    ASSERT_TRUE(store->identity_updates.empty());
}

/// @brief it should mark tracked hardware that a scan no longer finds as disconnected,
/// and never delete it.
TEST(Sync, MarksMissingHardwareDisconnected) {
    auto stored = found("01A");
    stored.key = "key-01A";
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->init());
    ASSERT_NIL(sync->sync({}));
    ASSERT_EQ(store->stored.size(), 1);
    ASSERT_EQ(store->statuses.size(), 1);
    ASSERT_EQ(store->statuses.at(0).status->variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(store->statuses.at(0).status->message, "Device disconnected");
}

/// @brief it should move hardware from another rack whose status is stale.
TEST(Sync, TakesOverStaleHardwareFromAnotherRack) {
    auto stored = found("01A");
    stored.key = "key-01A";
    stored.rack = 2;
    stored.status = synnax::status::Status<>{
        .time = x::telem::TimeStamp::now() - TAKEOVER_THRESHOLD * 2,
    };
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->sync({found("01A")}));
    ASSERT_EQ(store->identity_updates.at(0).rack, RACK);
}

/// @brief it should leave hardware on another rack that reported its status recently.
TEST(Sync, KeepsHardwareOnAnActiveRack) {
    auto stored = found("01A");
    stored.key = "key-01A";
    stored.rack = 2;
    stored.status = synnax::status::Status<>{.time = x::telem::TimeStamp::now()};
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->sync({found("01A")}));
    ASSERT_TRUE(store->identity_updates.empty());
    ASSERT_FALSE(sync->get("01A").has_value());
    ASSERT_TRUE(store->statuses.empty());
}

/// @brief it should create hardware again on the next scan after a create fails.
TEST(Sync, RetriesFailedCreate) {
    auto [sync, store] = new_sync();
    store->create_err = x::errors::Error(x::errors::VALIDATION, "rejected");
    ASSERT_OCCURRED_AS(sync->sync({found("01A")}), x::errors::VALIDATION);
    ASSERT_FALSE(sync->get("01A").has_value());
    store->create_err = x::errors::NIL;
    ASSERT_NIL(sync->sync({found("01A")}));
    ASSERT_EQ(store->creates, 2);
    ASSERT_TRUE(sync->get("01A").has_value());
}

/// @brief it should keep the configuration that a set signal reports.
TEST(Sync, KeepsConfigurationFromSetSignal) {
    auto stored = found("01A");
    stored.key = "key-01A";
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->init());
    stored.name = "Renamed";
    sync->on_set(stored);
    ASSERT_NIL(sync->sync({found("01A", "9211")}));
    ASSERT_EQ(store->identity_updates.at(0).name, "Renamed");
}

/// @brief it should stop tracking a resource that a delete signal reports.
TEST(Sync, ForgetsDeletedResource) {
    auto stored = found("01A");
    stored.key = "key-01A";
    auto [sync, store] = new_sync({stored});
    ASSERT_NIL(sync->init());
    sync->on_delete(std::string("key-01A"));
    ASSERT_FALSE(sync->get("01A").has_value());
}
}
