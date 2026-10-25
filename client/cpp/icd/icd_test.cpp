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

namespace synnax::icd {
namespace {
/// @brief creates an ICD that holds a CAN message with one binary field.
ICD create_can_icd(const std::string &name) {
    BinaryField speed;
    speed.key = x::uuid::create();
    speed.name = "speed";
    speed.scale = 0.5;
    speed.units = "rpm";
    speed.start_bit = 8;
    speed.bit_length = 16;
    speed.byte_order = BYTE_ORDER_BIG_ENDIAN;
    speed.signed_ = true;

    Message status;
    status.key = x::uuid::create();
    status.name = "status";
    status.payload = BinaryPayload{
        .length = 8,
        .identifier = CanIdentifier{.id = 0x123},
        .fields = {speed},
    };

    return ICD{.name = name, .messages = {status}};
}
}

/// @brief it should create an ICD and retrieve its messages.
TEST(ICDTests, testCreateAndRetrieve) {
    const auto client = new_test_client();
    auto ic = create_can_icd("create_retrieve");
    ASSERT_NIL(client.icds.create(ic));
    ASSERT_FALSE(ic.key.is_nil());
    const auto retrieved = ASSERT_NIL_P(client.icds.retrieve(ic.key));
    ASSERT_EQ(retrieved.key, ic.key);
    ASSERT_EQ(retrieved.name, "create_retrieve");
    ASSERT_EQ(retrieved.messages.size(), 1);

    const auto &status = retrieved.messages[0];
    ASSERT_EQ(status.key, ic.messages[0].key);
    ASSERT_EQ(status.name, "status");
    const auto &payload = std::get<BinaryPayload>(status.payload);
    ASSERT_TRUE(payload.identifier.has_value());
    const auto &id = std::get<CanIdentifier>(payload.identifier.value());
    ASSERT_EQ(id.id, 0x123);
    ASSERT_FALSE(id.extended);
    ASSERT_EQ(payload.length, 8);
    ASSERT_EQ(payload.fields.size(), 1);
    const auto &speed = payload.fields[0];
    ASSERT_EQ(speed.name, "speed");
    ASSERT_EQ(speed.scale, 0.5);
    ASSERT_EQ(speed.units, "rpm");
    ASSERT_EQ(speed.start_bit, 8);
    ASSERT_EQ(speed.bit_length, 16);
    ASSERT_EQ(speed.byte_order, BYTE_ORDER_BIG_ENDIAN);
    ASSERT_TRUE(speed.signed_);
    ASSERT_FALSE(speed.float_);
}

/// @brief it should create multiple ICDs and retrieve them by their keys.
TEST(ICDTests, testCreateAndRetrieveMany) {
    const auto client = new_test_client();
    std::vector ics = {
        create_can_icd("many_1"),
        create_can_icd("many_2"),
    };
    ASSERT_NIL(client.icds.create(ics));
    ASSERT_FALSE(ics[0].key.is_nil());
    ASSERT_FALSE(ics[1].key.is_nil());
    const std::vector keys = {ics[0].key, ics[1].key};
    const auto retrieved = ASSERT_NIL_P(client.icds.retrieve(keys));
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should replace an ICD when the replacement carries the hash it read.
TEST(ICDTests, testReplaceWithReadHash) {
    const auto client = new_test_client();
    auto ic = create_can_icd("replace");
    ASSERT_NIL(client.icds.create(ic));
    ASSERT_FALSE(ic.hash.empty());
    ic.name = "replaced";
    ASSERT_NIL(client.icds.create(ic));
    const auto retrieved = ASSERT_NIL_P(client.icds.retrieve(ic.key));
    ASSERT_EQ(retrieved.name, "replaced");
}

/// @brief it should reject a replacement whose hash is not the stored hash.
TEST(ICDTests, testReplaceStale) {
    const auto client = new_test_client();
    auto ic = create_can_icd("stale");
    ASSERT_NIL(client.icds.create(ic));
    ic.hash = "stale";
    ASSERT_OCCURRED_AS(
        client.icds.create(ic),
        x::errors::Error("sy.library.stale", "")
    );
}

/// @brief it should delete an ICD.
TEST(ICDTests, testDelete) {
    const auto client = new_test_client();
    auto ic = create_can_icd("to_delete");
    ASSERT_NIL(client.icds.create(ic));
    ASSERT_NIL(client.icds.del(ic.key));
    ASSERT_OCCURRED_AS_P(client.icds.retrieve(ic.key), x::errors::NOT_FOUND);
}

/// @brief it should delete multiple ICDs.
TEST(ICDTests, testDeleteMany) {
    const auto client = new_test_client();
    auto ic_1 = create_can_icd("del_many_1");
    auto ic_2 = create_can_icd("del_many_2");
    ASSERT_NIL(client.icds.create(ic_1));
    ASSERT_NIL(client.icds.create(ic_2));
    ASSERT_NIL(client.icds.del(std::vector{ic_1.key, ic_2.key}));
    ASSERT_OCCURRED_AS_P(client.icds.retrieve(ic_1.key), x::errors::NOT_FOUND);
    ASSERT_OCCURRED_AS_P(client.icds.retrieve(ic_2.key), x::errors::NOT_FOUND);
}

/// @brief it should return a not found error for an ICD that does not exist.
TEST(ICDTests, testRetrieveNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(client.icds.retrieve(x::uuid::create()), x::errors::NOT_FOUND);
}
}
