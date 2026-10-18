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

namespace synnax::view {
namespace {
const Base &base(const View &v) {
    return std::visit([](const auto &variant) -> const Base & { return variant; }, v);
}

View range_view(const std::string &name) {
    RangeView v;
    v.name = name;
    return v;
}

View status_view(const std::string &name, std::vector<std::string> variants = {}) {
    StatusView v;
    v.name = name;
    v.query.variants = std::move(variants);
    return v;
}
}

/// @brief it should correctly create a view.
TEST(ViewTests, testCreateView) {
    const auto client = new_test_client();
    auto v = range_view("test_view");
    ASSERT_NIL(client.views.create(v));
    ASSERT_FALSE(base(v).key.is_nil());
    ASSERT_EQ(base(v).name, "test_view");
}

/// @brief it should correctly retrieve a view by key.
TEST(ViewTests, testRetrieveView) {
    const auto client = new_test_client();
    auto v = status_view("retrieve_test", {"error"});
    ASSERT_NIL(client.views.create(v));
    const auto retrieved = ASSERT_NIL_P(client.views.retrieve(base(v).key));
    ASSERT_EQ(base(retrieved).key, base(v).key);
    ASSERT_EQ(base(retrieved).name, "retrieve_test");
    const auto *status = std::get_if<StatusView>(&retrieved);
    ASSERT_NE(status, nullptr);
    ASSERT_EQ(status->query.variants, std::vector<std::string>{"error"});
}

/// @brief it should correctly retrieve multiple views by keys.
TEST(ViewTests, testRetrieveMultipleViews) {
    const auto client = new_test_client();
    auto v1 = range_view("multi_1");
    auto v2 = status_view("multi_2");
    ASSERT_NIL(client.views.create(v1));
    ASSERT_NIL(client.views.create(v2));
    const std::vector keys = {base(v1).key, base(v2).key};
    const auto views = ASSERT_NIL_P(client.views.retrieve(keys));
    ASSERT_EQ(views.size(), 2);
}

/// @brief it should correctly create multiple views at once.
TEST(ViewTests, testCreateMultipleViews) {
    const auto client = new_test_client();
    std::vector views = {range_view("batch_1"), status_view("batch_2")};
    ASSERT_NIL(client.views.create(views));
    ASSERT_FALSE(base(views[0]).key.is_nil());
    ASSERT_FALSE(base(views[1]).key.is_nil());
    for (const auto &v: views) {
        const auto retrieved = ASSERT_NIL_P(client.views.retrieve(base(v).key));
        ASSERT_EQ(base(retrieved).name, base(v).name);
        ASSERT_EQ(retrieved.index(), v.index());
    }
}

/// @brief it should correctly delete a view.
TEST(ViewTests, testDeleteView) {
    const auto client = new_test_client();
    auto v = range_view("to_delete");
    ASSERT_NIL(client.views.create(v));
    ASSERT_NIL(client.views.del(base(v).key));
    ASSERT_OCCURRED_AS_P(client.views.retrieve(base(v).key), x::errors::NOT_FOUND);
}

/// @brief it should correctly delete multiple views.
TEST(ViewTests, testDeleteMultipleViews) {
    const auto client = new_test_client();
    auto v1 = range_view("del_multi_1");
    auto v2 = status_view("del_multi_2");
    ASSERT_NIL(client.views.create(v1));
    ASSERT_NIL(client.views.create(v2));
    const std::vector keys = {base(v1).key, base(v2).key};
    ASSERT_NIL(client.views.del(keys));
    ASSERT_OCCURRED_AS_P(client.views.retrieve(base(v1).key), x::errors::NOT_FOUND);
    ASSERT_OCCURRED_AS_P(client.views.retrieve(base(v2).key), x::errors::NOT_FOUND);
}
}
