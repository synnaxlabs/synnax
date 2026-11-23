// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

namespace synnax::icd {
namespace {
ICD status_icd(const std::string &name) {
    x::json::Parser parser(
        x::json::json{
            {"key", x::uuid::create().to_string()},
            {"name", name},
            {"messages",
             {{{"key", x::uuid::create().to_string()},
               {"name", "Status"},
               {"payload",
                {{"format", "json"},
                 {"match",
                  {{"type", "string"}, {"pointer", "/kind"}, {"value", "status"}}},
                 {"fields",
                  {{{"key", x::uuid::create().to_string()},
                    {"name", "pressure"},
                    {"scale", 2},
                    {"offset", 0},
                    {"units", "psi"},
                    {"pointer", "/pressure"},
                    {"type", "float"},
                    {"enum_values", {{{"label", "HIGH"}, {"value", 100}}}},
                    {"default", 0}}}}}}}}},
        }
    );
    auto ic = ICD::parse(parser);
    EXPECT_NIL(parser.error());
    return ic;
}
}

/// @brief it should create an ICD and retrieve it by key with every field intact.
TEST(ICDClient, CreatesAndRetrievesByKey) {
    const auto client = new_test_client();
    auto ic = status_icd("Rig");
    const auto sent = ic.to_json();
    ASSERT_NIL(client.icds.create(ic));
    const auto retrieved = ASSERT_NIL_P(client.icds.retrieve(ic.key));
    ASSERT_EQ(retrieved.to_json(), sent);
}

/// @brief it should refuse to create an ICD whose key exists.
TEST(ICDClient, RejectsDuplicateKeyOnCreate) {
    const auto client = new_test_client();
    auto ic = status_icd("Rig");
    ASSERT_NIL(client.icds.create(ic));
    auto duplicate = ic;
    const auto err = client.icds.create(duplicate);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "icd " + ic.key.to_string() + " already exists");
}

/// @brief it should retrieve several ICDs by key.
TEST(ICDClient, RetrievesManyByKey) {
    const auto client = new_test_client();
    auto first = status_icd("First");
    auto second = status_icd("Second");
    ASSERT_NIL(client.icds.create(first));
    ASSERT_NIL(client.icds.create(second));
    const auto icds = ASSERT_NIL_P(client.icds.retrieve({first.key, second.key}));
    std::set<std::string> names;
    for (const auto &ic: icds)
        names.insert(ic.name);
    ASSERT_EQ(names, (std::set<std::string>{"First", "Second"}));
}

/// @brief it should retrieve no ICDs for no keys.
TEST(ICDClient, RetrievesNothingForNoKeys) {
    const auto client = new_test_client();
    const auto icds = ASSERT_NIL_P(client.icds.retrieve(std::vector<Key>{}));
    ASSERT_TRUE(icds.empty());
}

/// @brief it should return NOT_FOUND for a key that no ICD has.
TEST(ICDClient, RetrieveMissingKeyNotFound) {
    const auto client = new_test_client();
    const auto key = x::uuid::create();
    const auto [_, err] = client.icds.retrieve(key);
    ASSERT_OCCURRED_AS(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no ICD where key = \"" + key.to_string() + "\"");
}
}
