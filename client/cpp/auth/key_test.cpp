// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::auth {
namespace {
synnax::Synnax keyed_client(const std::string &token) {
    auto cfg = test_client_config();
    cfg.api_key = token;
    return synnax::Synnax(cfg);
}

bool holds(const std::vector<APIKey> &keys, const std::string &key) {
    return std::ranges::any_of(keys, [&](const APIKey &k) { return k.key == key; });
}
}

/// @brief it should create a key, log in with it, list it, and delete it.
TEST(TestKeys, Lifecycle) {
    auto client = new_test_client();
    const auto created = ASSERT_NIL_P(client.api_keys.create("cpp"));
    ASSERT_TRUE(created.token.starts_with("syk_"));
    ASSERT_EQ(created.key.name, "cpp");
    ASSERT_EQ(created.key.subject.type, "user");
    const auto keyed = keyed_client(created.token);
    ASSERT_NIL(keyed.auth->authenticate());
    const auto session = keyed.auth->current_session();
    ASSERT_EQ(session.credential, created.key.key);
    ASSERT_EQ(session.subject.key, created.key.subject.key);
    ASSERT_TRUE(holds(ASSERT_NIL_P(client.api_keys.retrieve()), created.key.key));
    ASSERT_NIL(client.api_keys.del({created.key.key}));
    ASSERT_FALSE(holds(ASSERT_NIL_P(client.api_keys.retrieve()), created.key.key));
}

/// @brief it should fail the next request of a client whose key was deleted.
TEST(TestKeys, DeletedKey) {
    auto client = new_test_client();
    const auto created = ASSERT_NIL_P(client.api_keys.create("deleted"));
    const auto keyed = keyed_client(created.token);
    ASSERT_NIL(keyed.auth->authenticate());
    ASSERT_NIL(client.api_keys.del({created.key.key}));
    ASSERT_OCCURRED_AS_P(keyed.api_keys.retrieve(), ERR);
}

/// @brief it should reject a key the Core never issued.
TEST(TestKeys, UnknownKey) {
    const auto keyed = keyed_client("syk_00000000-0000-0000-0000-000000000000secret");
    ASSERT_OCCURRED_AS(keyed.auth->authenticate(), ERR);
}
}
