// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <random>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/filter/field.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/common/sync.h"
#include "driver/labjack/store.h"

namespace driver::labjack {
namespace {
/// @brief a Core client and a fresh rack on it.
struct Env {
    std::shared_ptr<synnax::Synnax> client;
    synnax::rack::Rack rack;
};

Env create_env() {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    synnax::rack::Rack rack{.name = "labjack_store_rack"};
    EXPECT_NIL(client->racks.create(rack));
    return {std::move(client), std::move(rack)};
}

/// @brief returns a serial number that no other test device has.
std::uint32_t new_serial() {
    static std::mt19937 gen{std::random_device{}()};
    return std::uniform_int_distribution<std::uint32_t>{1, 4'000'000'000}(gen);
}

/// @brief returns the T7 that a scan finds on the rack.
Device found_device(const synnax::rack::Key rack, const std::uint32_t serial) {
    return {
        .rack = rack,
        .name = "T7-" + std::to_string(serial % 10000),
        .serial = serial,
        .connection = synnax::labjack::device::CONNECTION_USB,
        .model = synnax::labjack::device::MODEL_T_7,
        .ports = synnax::labjack::device::T7Ports{},
    };
}

std::pair<Device, x::errors::Error>
retrieve_by_serial(const Env &env, const std::uint32_t serial) {
    return env.client->labjack.devices.retrieve(
        synnax::labjack::device::Where{.serial = serial},
        x::filter::one,
        {.include_status = true}
    );
}
}

/// @brief it should dispatch only the identity fields that changed.
TEST(Store, IdentityActionsOnlyForChangedFields) {
    const auto stored = found_device(1, 470010001);
    auto next = stored;
    next.connection = synnax::labjack::device::CONNECTION_ETHERNET;
    const auto acts = identity_actions(stored, next);
    ASSERT_EQ(acts.size(), 1);
    const auto &set = std::get<synnax::labjack::device::SetPayload>(acts[0]);
    ASSERT_EQ(set.path, std::vector<std::string>{"connection"});
    ASSERT_TRUE(identity_actions(stored, stored).empty());
}

/// @brief it should create the device that a scan finds, with a present status.
TEST(Store, CreatesFoundDevice) {
    auto env = create_env();
    common::Sync<Device> sync(std::make_unique<Store>(env.client), env.rack.key, "");
    const auto serial = new_serial();
    ASSERT_NIL(sync.sync({found_device(env.rack.key, serial)}));
    const auto got = ASSERT_NIL_P(retrieve_by_serial(env, serial));
    ASSERT_EQ(got.rack, env.rack.key);
    ASSERT_EQ(got.status.value().message, "Device present");
    ASSERT_EQ(got.status.value().details.device, got.key);
}

/// @brief it should write a changed connection and keep the configuration that a
/// user wrote.
TEST(Store, KeepsConfigurationOnRescan) {
    auto env = create_env();
    common::Sync<Device> sync(std::make_unique<Store>(env.client), env.rack.key, "");
    const auto serial = new_serial();
    auto found = found_device(env.rack.key, serial);
    ASSERT_NIL(sync.sync({found}));
    const auto stored = ASSERT_NIL_P(retrieve_by_serial(env, serial));
    ASSERT_NIL(env.client->labjack.devices.dispatch(
        stored.key,
        {synnax::labjack::device::set({.path = {"name"}, .value = "Bench T7"}),
         synnax::labjack::device::set({.path = {"identifier"}, .value = "bench"})}
    ));
    found.connection = synnax::labjack::device::CONNECTION_ETHERNET;
    ASSERT_NIL(sync.sync({found}));
    const auto got = ASSERT_NIL_P(retrieve_by_serial(env, serial));
    ASSERT_EQ(got.connection, synnax::labjack::device::CONNECTION_ETHERNET);
    ASSERT_EQ(got.name, "Bench T7");
    ASSERT_EQ(got.identifier, "bench");
}

/// @brief it should mark a device that a scan no longer finds as disconnected.
TEST(Store, MarksMissingDevice) {
    auto env = create_env();
    common::Sync<Device> sync(std::make_unique<Store>(env.client), env.rack.key, "");
    const auto serial = new_serial();
    ASSERT_NIL(sync.sync({found_device(env.rack.key, serial)}));
    ASSERT_NIL(sync.sync({}));
    ASSERT_EQ(
        ASSERT_NIL_P(retrieve_by_serial(env, serial)).status.value().message,
        "Device disconnected"
    );
}

/// @brief it should prefer the row on the rack of the scan when two rows hold the same
/// hardware.
TEST(Store, PrefersRowOnScanRack) {
    auto env = create_env();
    synnax::rack::Rack other{.name = "labjack_store_other_rack"};
    ASSERT_NIL(env.client->racks.create(other));
    const auto serial = new_serial();
    // The row on the scan rack has the older status, so only the rack rule picks it.
    auto here = found_device(env.rack.key, serial);
    auto elsewhere = found_device(other.key, serial);
    ASSERT_NIL(env.client->labjack.devices.create(here));
    ASSERT_NIL(env.client->labjack.devices.create(elsewhere));
    Store store(env.client);
    const auto got = ASSERT_NIL_P(store.find(found_device(env.rack.key, serial)));
    ASSERT_EQ(got.value().key, here.key);
}

/// @brief it should prefer the most recently reported row when no row is on the rack
/// of the scan.
TEST(Store, PrefersMostRecentlyReportedRow) {
    auto env = create_env();
    synnax::rack::Rack first{.name = "labjack_store_first_rack"};
    synnax::rack::Rack second{.name = "labjack_store_second_rack"};
    ASSERT_NIL(env.client->racks.create(first));
    ASSERT_NIL(env.client->racks.create(second));
    const auto serial = new_serial();
    auto older = found_device(first.key, serial);
    auto newer = found_device(second.key, serial);
    ASSERT_NIL(env.client->labjack.devices.create(older));
    ASSERT_NIL(env.client->labjack.devices.create(newer));
    Store store(env.client);
    older.status = synnax::labjack::device::Status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Device present",
        .time = x::telem::TimeStamp::now(),
    };
    ASSERT_NIL(store.set_statuses({older}));
    const auto got = ASSERT_NIL_P(store.find(found_device(env.rack.key, serial)));
    ASSERT_EQ(got.value().key, older.key);
}

/// @brief it should move a device that another rack stopped reporting to the rack of
/// the scan, together with its status.
TEST(Store, TakesOverDeviceFromSilentRack) {
    auto env = create_env();
    synnax::rack::Rack other{.name = "labjack_store_silent_rack"};
    ASSERT_NIL(env.client->racks.create(other));
    const auto serial = new_serial();
    auto stored = found_device(other.key, serial);
    ASSERT_NIL(env.client->labjack.devices.create(stored));
    common::Sync<Device> sync(
        std::make_unique<Store>(env.client),
        env.rack.key,
        "",
        x::telem::TimeSpan(0)
    );
    // The first scan only notes the status time of the other rack.
    ASSERT_NIL(sync.sync({found_device(env.rack.key, serial)}));
    ASSERT_EQ(ASSERT_NIL_P(retrieve_by_serial(env, serial)).rack, other.key);
    ASSERT_NIL(sync.sync({found_device(env.rack.key, serial)}));
    const auto got = ASSERT_NIL_P(retrieve_by_serial(env, serial));
    ASSERT_EQ(got.key, stored.key);
    ASSERT_EQ(got.rack, env.rack.key);
    ASSERT_EQ(got.status.value().details.rack, env.rack.key);
    ASSERT_EQ(got.status.value().message, "Device present");
}
/// @brief it should refuse a task device key that is not a serial number.
TEST(RetrieveTaskDevice, RejectsKeysThatAreNotSerials) {
    const auto client = new_test_client();
    for (const auto *key: {"", "labjack_abc", "4294967296", "+5", " 5", "5 ", "-1"}) {
        SCOPED_TRACE(key);
        ASSERT_OCCURRED_AS_P(retrieve_task_device(client, key), x::errors::VALIDATION);
    }
}

/// @brief it should report a serial number that no device has as not found.
TEST(RetrieveTaskDevice, ReportsMissingSerial) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        retrieve_task_device(client, std::to_string(new_serial())),
        x::errors::NOT_FOUND
    );
}

/// @brief it should pick the most recently reported device when two hold the serial.
TEST(RetrieveTaskDevice, PicksMostRecentlyReportedDevice) {
    auto env = create_env();
    synnax::rack::Rack other{.name = "labjack_task_device_rack"};
    ASSERT_NIL(env.client->racks.create(other));
    const auto serial = new_serial();
    auto older = found_device(env.rack.key, serial);
    auto newer = found_device(other.key, serial);
    ASSERT_NIL(env.client->labjack.devices.create(older));
    ASSERT_NIL(env.client->labjack.devices.create(newer));
    newer.status = synnax::labjack::device::Status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Device present",
        .time = x::telem::TimeStamp::now(),
    };
    ASSERT_NIL(Store(env.client).set_statuses({newer}));
    const auto got = ASSERT_NIL_P(
        retrieve_task_device(*env.client, std::to_string(serial))
    );
    ASSERT_EQ(got.key, newer.key);
}
}
