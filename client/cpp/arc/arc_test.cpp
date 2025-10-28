// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <random>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xtest/xtest.h"

std::mt19937 gen_rand = random_generator(std::move("Arc Tests"));

std::string random_arc_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + "_" + std::to_string(dis(gen_rand));
}

/// @brief it should create an Arc program and assign it a non-zero key.
TEST(TestArc, testCreate) {
    const auto client = new_test_client();
    auto arc = synnax::Arc("test_arc");
    arc.text.set_raw("// Simple Arc program");

    ASSERT_NIL(client.arcs.create(arc));
    ASSERT_EQ(arc.name, "test_arc");
    ASSERT_FALSE(arc.key.empty());
}

/// @brief it should create an Arc program using the convenience method.
TEST(TestArc, testCreateConvenience) {
    const auto client = new_test_client();
    auto [arc, err] = client.arcs.create("convenience_arc");

    ASSERT_NIL(err);
    ASSERT_EQ(arc.name, "convenience_arc");
    ASSERT_FALSE(arc.key.empty());
}

/// @brief it should create multiple Arc programs.
TEST(TestArc, testCreateMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<synnax::Arc>{
        synnax::Arc("arc1"),
        synnax::Arc("arc2"),
        synnax::Arc("arc3"),
    };

    ASSERT_NIL(client.arcs.create(arcs));

    for (const auto &arc : arcs) {
        ASSERT_FALSE(arc.key.empty());
    }
    ASSERT_EQ(arcs[0].name, "arc1");
    ASSERT_EQ(arcs[1].name, "arc2");
    ASSERT_EQ(arcs[2].name, "arc3");
}

/// @brief it should retrieve an Arc program by name.
TEST(TestArc, testRetrieveByName) {
    const auto client = new_test_client();
    auto name = random_arc_name("retrieve_test");
    auto created = synnax::Arc(name);
    ASSERT_NIL(client.arcs.create(created));

    auto [retrieved, err] = client.arcs.retrieve_by_name(name);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, created.key);
    ASSERT_EQ(retrieved.name, name);
}

/// @brief it should retrieve an Arc program by key.
TEST(TestArc, testRetrieveByKey) {
    const auto client = new_test_client();
    auto created = synnax::Arc("key_test");
    ASSERT_NIL(client.arcs.create(created));

    auto [retrieved, err] = client.arcs.retrieve_by_key(created.key);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.key, created.key);
    ASSERT_EQ(retrieved.name, "key_test");
}

/// @brief it should retrieve multiple Arc programs by names.
TEST(TestArc, testRetrieveMany) {
    const auto client = new_test_client();
    auto name1 = random_arc_name("multi1");
    auto name2 = random_arc_name("multi2");
    auto arcs = std::vector<synnax::Arc>{
        synnax::Arc(name1),
        synnax::Arc(name2),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    auto [retrieved, err] = client.arcs.retrieve({name1, name2});

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should retrieve multiple Arc programs by keys.
TEST(TestArc, testRetrieveByKeys) {
    const auto client = new_test_client();
    auto arcs = std::vector<synnax::Arc>{
        synnax::Arc("keys1"),
        synnax::Arc("keys2"),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<std::string> keys = {arcs[0].key, arcs[1].key};
    auto [retrieved, err] = client.arcs.retrieve_by_keys(keys);

    ASSERT_NIL(err);
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should delete an Arc program by key.
TEST(TestArc, testDelete) {
    const auto client = new_test_client();
    auto arc = synnax::Arc("delete_test");
    ASSERT_NIL(client.arcs.create(arc));

    ASSERT_NIL(client.arcs.delete_arc(arc.key));

    // Verify it's deleted
    auto [_, err] = client.arcs.retrieve_by_key(arc.key);
    ASSERT_FALSE(err.ok());
}

/// @brief it should delete multiple Arc programs by keys.
TEST(TestArc, testDeleteMany) {
    const auto client = new_test_client();
    auto arcs = std::vector<synnax::Arc>{
        synnax::Arc("delete1"),
        synnax::Arc("delete2"),
    };
    ASSERT_NIL(client.arcs.create(arcs));

    std::vector<std::string> keys = {arcs[0].key, arcs[1].key};
    ASSERT_NIL(client.arcs.delete_arc(keys));

    // Verify they're deleted by trying to retrieve - should fail
    auto [retrieved, err] = client.arcs.retrieve_by_keys(keys);
    // Server returns error when arcs don't exist
    ASSERT_FALSE(err.ok());
}
