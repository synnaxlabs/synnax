// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std.
#include <memory>

extern "C" {
#include <lauxlib.h>
#include <lua.h>
#include <lualib.h>
}
#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

/// internal.
#include "driver/sequence/plugins/plugins.h"

using json = nlohmann::json;

/// @brief it should apply JSON variables as Lua globals.
TEST(JSONPluginTest, BasicVariableApplication) {
    lua_State *L = luaL_newstate();
    luaL_openlibs(L);

    const json test_data = {
        {"number", 42.5},
        {"string", "hello"},
        {"boolean", true},
        {"array", {1, 2, 3}},
        {"nested", {{"value", 123}}}
    };

    plugins::JSON source(test_data);
    ASSERT_EQ(source.before_all(L), x::errors::NIL);

    lua_getglobal(L, "number");
    ASSERT_TRUE(lua_isnumber(L, -1));
    ASSERT_EQ(lua_tonumber(L, -1), 42.5);
    lua_pop(L, 1);

    lua_getglobal(L, "string");
    ASSERT_TRUE(lua_isstring(L, -1));
    ASSERT_STREQ(lua_tostring(L, -1), "hello");
    lua_pop(L, 1);

    lua_getglobal(L, "boolean");
    ASSERT_TRUE(lua_isboolean(L, -1));
    ASSERT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 1);

    lua_getglobal(L, "array");
    ASSERT_TRUE(lua_istable(L, -1));
    for (int i = 1; i <= 3; i++) {
        lua_rawgeti(L, -1, i);
        ASSERT_TRUE(lua_isnumber(L, -1));
        ASSERT_EQ(lua_tonumber(L, -1), i);
        lua_pop(L, 1);
    }
    lua_pop(L, 1);

    lua_getglobal(L, "nested");
    ASSERT_TRUE(lua_istable(L, -1));
    lua_getfield(L, -1, "value");
    ASSERT_TRUE(lua_isnumber(L, -1));
    ASSERT_EQ(lua_tonumber(L, -1), 123);
    lua_pop(L, 2);

    lua_close(L);
}

/// @brief it should reject non-object JSON as invalid.
TEST(JSONPluginTest, InvalidJSON) {
    lua_State *L = luaL_newstate();
    luaL_openlibs(L);
    const json invalid_json = json::array();
    plugins::JSON plugin(invalid_json);
    ASSERT_FALSE(plugin.before_all(L).ok());
    lua_close(L);
}
