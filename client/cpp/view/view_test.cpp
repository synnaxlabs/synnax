// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::view {
/// @brief it should correctly create a view.
TEST(ViewTests, testCreateView) {
    const auto client = new_test_client();
    auto v = View{
        .name = "test_view",
        .type = "lineplot",
        .query = {{"channels", x::json::json::array({"ch-1", "ch-2"})}},
    };
    ASSERT_NIL(client.views.create(v));
    ASSERT_FALSE(v.key.is_nil());
    ASSERT_EQ(v.name, "test_view");
}

/// @brief it should correctly retrieve a view by key.
TEST(ViewTests, testRetrieveView) {
    const auto client = new_test_client();
    auto v = View{
        .name = "retrieve_test",
        .type = "table",
        .query = {{"columns", x::json::json::array({"col-1"})}},
    };
    ASSERT_NIL(client.views.create(v));
    const auto retrieved = ASSERT_NIL_P(client.views.retrieve(v.key));
    ASSERT_EQ(retrieved.key, v.key);
    ASSERT_EQ(retrieved.name, "retrieve_test");
    ASSERT_EQ(retrieved.type, "table");
}

/// @brief it should correctly retrieve multiple views by keys.
TEST(ViewTests, testRetrieveMultipleViews) {
    const auto client = new_test_client();
    auto v1 = View{
        .name = "multi_1",
        .type = "lineplot",
        .query = {{"channels", x::json::json::array({"ch-1"})}},
    };
    auto v2 = View{
        .name = "multi_2",
        .type = "table",
        .query = {{"columns", x::json::json::array({"col-1"})}},
    };
    ASSERT_NIL(client.views.create(v1));
    ASSERT_NIL(client.views.create(v2));
    const std::vector keys = {v1.key, v2.key};
    const auto views = ASSERT_NIL_P(client.views.retrieve(keys));
    ASSERT_EQ(views.size(), 2);
}

/// @brief it should correctly create multiple views at once.
TEST(ViewTests, testCreateMultipleViews) {
    const auto client = new_test_client();
    std::vector views = {
        View{
            .name = "batch_1",
            .type = "lineplot",
            .query = {{"channels", x::json::json::array({"ch-1"})}},
        },
        View{
            .name = "batch_2",
            .type = "table",
            .query = {{"columns", x::json::json::array({"col-1"})}},
        },
    };
    ASSERT_NIL(client.views.create(views));
    ASSERT_FALSE(views[0].key.is_nil());
    ASSERT_FALSE(views[1].key.is_nil());
    for (const auto &v: views) {
        const auto retrieved = ASSERT_NIL_P(client.views.retrieve(v.key));
        ASSERT_EQ(retrieved.name, v.name);
        ASSERT_EQ(retrieved.type, v.type);
    }
}

/// @brief it should correctly delete a view.
TEST(ViewTests, testDeleteView) {
    const auto client = new_test_client();
    auto v = View{
        .name = "to_delete",
        .type = "lineplot",
        .query = {},
    };
    ASSERT_NIL(client.views.create(v));
    ASSERT_NIL(client.views.del(v.key));
    ASSERT_OCCURRED_AS_P(client.views.retrieve(v.key), x::errors::NOT_FOUND);
}

/// @brief it should correctly delete multiple views.
TEST(ViewTests, testDeleteMultipleViews) {
    const auto client = new_test_client();
    auto v1 = View{
        .name = "del_multi_1",
        .type = "lineplot",
        .query = {},
    };
    auto v2 = View{
        .name = "del_multi_2",
        .type = "table",
        .query = {},
    };
    ASSERT_NIL(client.views.create(v1));
    ASSERT_NIL(client.views.create(v2));
    const std::vector keys = {v1.key, v2.key};
    ASSERT_NIL(client.views.del(keys));
    ASSERT_OCCURRED_AS_P(client.views.retrieve(v1.key), x::errors::NOT_FOUND);
    ASSERT_OCCURRED_AS_P(client.views.retrieve(v2.key), x::errors::NOT_FOUND);
}
}
