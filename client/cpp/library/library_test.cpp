// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <variant>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::library {
namespace {
/// @brief creates a library that holds an enum and a CAN message whose binary field
/// names its values with the enum.
Library create_can_library(const std::string &name) {
    EnumEntry state;
    state.key = x::uuid::create();
    state.name = "state";
    state.values = {{.value = 0, .name = "idle"}, {.value = 1, .name = "running"}};

    BinaryField speed;
    speed.key = x::uuid::create();
    speed.name = "speed";
    speed.scale = 0.5;
    speed.units = "rpm";
    speed.enumeration = state.key;
    speed.start_bit = 8;
    speed.bit_length = 16;
    speed.byte_order = BYTE_ORDER_BIG_ENDIAN;
    speed.signed_ = true;

    MessageEntry status;
    status.key = x::uuid::create();
    status.name = "status";
    status.payload = BinaryPayload{
        .length = 8,
        .identifier = CanIdentifier{.id = 0x123},
        .fields = {speed},
    };

    return Library{.name = name, .entries = {state, status}};
}
}

/// @brief it should create a library and retrieve its enum and message entries.
TEST(LibraryTests, testCreateAndRetrieve) {
    const auto client = new_test_client();
    auto lib = create_can_library("create_retrieve");
    ASSERT_NIL(client.libraries.create(lib));
    ASSERT_FALSE(lib.key.is_nil());
    const auto retrieved = ASSERT_NIL_P(client.libraries.retrieve(lib.key));
    ASSERT_EQ(retrieved.key, lib.key);
    ASSERT_EQ(retrieved.name, "create_retrieve");
    ASSERT_EQ(retrieved.entries.size(), 2);

    const auto &state = std::get<EnumEntry>(retrieved.entries[0]);
    const auto &original_state = std::get<EnumEntry>(lib.entries[0]);
    ASSERT_EQ(state.key, original_state.key);
    ASSERT_EQ(state.name, "state");
    ASSERT_EQ(state.values.size(), 2);
    ASSERT_EQ(state.values[1].value, 1);
    ASSERT_EQ(state.values[1].name, "running");

    const auto &status = std::get<MessageEntry>(retrieved.entries[1]);
    ASSERT_EQ(status.name, "status");
    const auto &payload = std::get<BinaryPayload>(status.payload);
    ASSERT_TRUE(payload.identifier.has_value());
    const auto &id = std::get<CanIdentifier>(*payload.identifier);
    ASSERT_EQ(id.id, 0x123);
    ASSERT_FALSE(id.extended);
    ASSERT_EQ(payload.length, 8);
    ASSERT_EQ(payload.fields.size(), 1);
    const auto &speed = payload.fields[0];
    ASSERT_EQ(speed.name, "speed");
    ASSERT_EQ(speed.scale, 0.5);
    ASSERT_EQ(speed.units, "rpm");
    ASSERT_EQ(speed.enumeration, original_state.key);
    ASSERT_EQ(speed.start_bit, 8);
    ASSERT_EQ(speed.bit_length, 16);
    ASSERT_EQ(speed.byte_order, BYTE_ORDER_BIG_ENDIAN);
    ASSERT_TRUE(speed.signed_);
    ASSERT_FALSE(speed.float_);
}

/// @brief it should create multiple libraries and retrieve them by their keys.
TEST(LibraryTests, testCreateAndRetrieveMany) {
    const auto client = new_test_client();
    std::vector libs = {
        create_can_library("many_1"),
        create_can_library("many_2"),
    };
    ASSERT_NIL(client.libraries.create(libs));
    ASSERT_FALSE(libs[0].key.is_nil());
    ASSERT_FALSE(libs[1].key.is_nil());
    const std::vector keys = {libs[0].key, libs[1].key};
    const auto retrieved = ASSERT_NIL_P(client.libraries.retrieve(keys));
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should delete a library.
TEST(LibraryTests, testDelete) {
    const auto client = new_test_client();
    auto lib = create_can_library("to_delete");
    ASSERT_NIL(client.libraries.create(lib));
    ASSERT_NIL(client.libraries.del(lib.key));
    ASSERT_OCCURRED_AS_P(client.libraries.retrieve(lib.key), x::errors::NOT_FOUND);
}

/// @brief it should delete multiple libraries.
TEST(LibraryTests, testDeleteMany) {
    const auto client = new_test_client();
    auto lib_1 = create_can_library("del_many_1");
    auto lib_2 = create_can_library("del_many_2");
    ASSERT_NIL(client.libraries.create(lib_1));
    ASSERT_NIL(client.libraries.create(lib_2));
    ASSERT_NIL(client.libraries.del(std::vector{lib_1.key, lib_2.key}));
    ASSERT_OCCURRED_AS_P(client.libraries.retrieve(lib_1.key), x::errors::NOT_FOUND);
    ASSERT_OCCURRED_AS_P(client.libraries.retrieve(lib_2.key), x::errors::NOT_FOUND);
}

/// @brief it should return a not found error for a library that does not exist.
TEST(LibraryTests, testRetrieveNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.libraries.retrieve(x::uuid::create()),
        x::errors::NOT_FOUND
    );
}
}
