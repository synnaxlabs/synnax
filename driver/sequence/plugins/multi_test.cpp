// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/sequence/plugins/plugins.h"

class MockPlugin : public plugins::Plugin {
public:
    std::vector<std::string> calls;
    bool should_error = false;
    std::string error_on;

    xerrors::Error before_all(lua_State *L) override {
        calls.emplace_back("before_all");
        if (should_error && error_on == "before_all")
            return xerrors::Error("mock error");
        return xerrors::NIL;
    }

    xerrors::Error after_all(lua_State *L) override {
        calls.emplace_back("after_all");
        if (should_error && error_on == "after_all")
            return xerrors::Error("mock error");
        return xerrors::NIL;
    }

    xerrors::Error before_next(lua_State *L) override {
        calls.emplace_back("before_next");
        if (should_error && error_on == "before_next")
            return xerrors::Error("mock error");
        return xerrors::NIL;
    }

    xerrors::Error after_next(lua_State *L) override {
        calls.emplace_back("after_next");
        if (should_error && error_on == "after_next")
            return xerrors::Error("mock error");
        return xerrors::NIL;
    }
};

/// @brief it should call plugin methods in correct order across all plugins.
TEST(MultiPlugin, testCallOrder) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2};

    auto multi = plugins::MultiPlugin(plugins);

    ASSERT_EQ(multi.before_all(nullptr), xerrors::NIL);
    ASSERT_EQ(multi.before_next(nullptr), xerrors::NIL);
    ASSERT_EQ(multi.after_next(nullptr), xerrors::NIL);
    ASSERT_EQ(multi.after_all(nullptr), xerrors::NIL);

    std::vector<std::string> expected =
        {"before_all", "before_next", "after_next", "after_all"};
    ASSERT_EQ(plugin1->calls, expected);
    ASSERT_EQ(plugin2->calls, expected);
}

/// @brief it should propagate errors from before_all.
TEST(MultiPlugin, testErrorPropagationBeforeAll) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    plugin2->should_error = true;
    plugin2->error_on = "before_all";

    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2};
    auto multi = plugins::MultiPlugin(plugins);

    auto err = multi.before_all(nullptr);
    ASSERT_NE(err, xerrors::NIL);
    ASSERT_EQ(plugin1->calls.size(), 1);
    ASSERT_EQ(plugin2->calls.size(), 1);
}

/// @brief it should propagate errors from after_all.
TEST(MultiPlugin, testErrorPropagationAfterAll) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    plugin2->should_error = true;
    plugin2->error_on = "after_all";

    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2};
    auto multi = plugins::MultiPlugin(plugins);

    auto err = multi.after_all(nullptr);
    ASSERT_NE(err, xerrors::NIL);
    ASSERT_EQ(plugin1->calls.size(), 1);
    ASSERT_EQ(plugin2->calls.size(), 1);
}

/// @brief it should propagate errors from before_next.
TEST(MultiPlugin, testErrorPropagationBeforeNext) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    plugin2->should_error = true;
    plugin2->error_on = "before_next";

    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2};
    auto multi = plugins::MultiPlugin(plugins);

    auto err = multi.before_next(nullptr);
    ASSERT_NE(err, xerrors::NIL);
    ASSERT_EQ(plugin1->calls.size(), 1);
    ASSERT_EQ(plugin2->calls.size(), 1);
}

/// @brief it should propagate errors from after_next.
TEST(MultiPlugin, testErrorPropagationAfterNext) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    plugin2->should_error = true;
    plugin2->error_on = "after_next";

    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2};
    auto multi = plugins::MultiPlugin(plugins);

    auto err = multi.after_next(nullptr);
    ASSERT_NE(err, xerrors::NIL);
    ASSERT_EQ(plugin1->calls.size(), 1);
    ASSERT_EQ(plugin2->calls.size(), 1);
}

/// @brief it should call after_all on all plugins even if one returns an error
TEST(MultiPlugin, testAfterAllCallsAllPlugins) {
    auto plugin1 = std::make_shared<MockPlugin>();
    auto plugin2 = std::make_shared<MockPlugin>();
    auto plugin3 = std::make_shared<MockPlugin>();

    // Make the middle plugin return an error
    plugin2->should_error = true;
    plugin2->error_on = "after_all";

    std::vector<std::shared_ptr<plugins::Plugin>> plugins = {plugin1, plugin2, plugin3};
    auto multi = plugins::MultiPlugin(plugins);

    auto err = multi.after_all(nullptr);
    ASSERT_NE(err, xerrors::NIL);

    // Verify that all plugins had after_all called
    ASSERT_EQ(plugin1->calls.size(), 1);
    ASSERT_EQ(plugin1->calls[0], "after_all");

    ASSERT_EQ(plugin2->calls.size(), 1);
    ASSERT_EQ(plugin2->calls[0], "after_all");

    ASSERT_EQ(plugin3->calls.size(), 1);
    ASSERT_EQ(plugin3->calls[0], "after_all");
}
