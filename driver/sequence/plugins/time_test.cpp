// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/sequence/plugins/plugins.h"

extern "C" {
#include <lualib.h>
}

/// @brief it should correctly return the elapsed sequence time based on the current
/// time.
TEST(TimePluginTest, testElapsed) {
    auto current_time = telem::TimeSpan(0);
    auto now = [&current_time]() -> telem::TimeStamp {
        return telem::TimeStamp(current_time);
    };

    lua_State *L = luaL_newstate();
    luaL_openlibs(L);

    auto plugin = plugins::Time(now);

    // Initialize the plugin with the Lua state
    ASSERT_EQ(plugin.before_all(L), xerrors::NIL);

    // Test before_next updates elapsed time correctly
    current_time = telem::SECOND * 1;
    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);

    // Check that Lua global variables were set correctly
    lua_getglobal(L, "elapsed_time");
    EXPECT_EQ(lua_type(L, -1), LUA_TNUMBER);
    EXPECT_EQ(lua_tonumber(L, -1), 1.0); // 1.0 seconds

    current_time = telem::SECOND * 5;
    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);
    lua_getglobal(L, "elapsed_time");
    EXPECT_EQ(lua_type(L, -1), LUA_TNUMBER);
    EXPECT_EQ(lua_tonumber(L, -1), 5.0); // 5.0 seconds

    lua_close(L);
}

/// @brief it should track iteration count across multiple executions.
TEST(TimePluginTest, testIteration) {
    lua_State *L = luaL_newstate();
    luaL_openlibs(L);

    auto plugin = plugins::Time();

    ASSERT_EQ(plugin.before_all(L), xerrors::NIL);

    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);
    lua_getglobal(L, "iteration");
    EXPECT_EQ(lua_type(L, -1), LUA_TNUMBER);
    EXPECT_EQ(lua_tointeger(L, -1), 1);
    lua_pop(L, 1);

    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);
    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);
    lua_getglobal(L, "iteration");
    EXPECT_EQ(lua_type(L, -1), LUA_TNUMBER);
    EXPECT_EQ(lua_tointeger(L, -1), 3);

    lua_close(L);
}

/// @brief it should check if elapsed time is within a specified range.
TEST(TimePluginTest, testElapsedWithin) {
    auto current_time = telem::TimeSpan(0);
    auto now = [&current_time]() -> telem::TimeStamp {
        return telem::TimeStamp(current_time);
    };

    lua_State *L = luaL_newstate();
    luaL_openlibs(L);

    auto plugin = plugins::Time(now);

    ASSERT_EQ(plugin.before_all(L), xerrors::NIL);

    current_time = telem::SECOND * 3;
    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);

    lua_getglobal(L, "elapsed_time_within");
    lua_pushnumber(L, 1); // start time
    lua_pushnumber(L, 5); // end time
    lua_call(L, 2, 1);
    EXPECT_EQ(lua_type(L, -1), LUA_TBOOLEAN);
    EXPECT_TRUE(lua_toboolean(L, -1)); // 3 seconds is within 1-5 seconds
    lua_pop(L, 1);

    lua_getglobal(L, "elapsed_time_within");
    lua_pushnumber(L, 4); // start time
    lua_pushnumber(L, 6); // end time
    lua_call(L, 2, 1);
    EXPECT_EQ(lua_type(L, -1), LUA_TBOOLEAN);
    EXPECT_FALSE(lua_toboolean(L, -1)); // 3 seconds is not within 4-6 seconds
    lua_pop(L, 1);

    current_time = telem::SECOND * 7;
    ASSERT_EQ(plugin.before_next(L), xerrors::NIL);

    lua_getglobal(L, "elapsed_time_within");
    lua_pushnumber(L, 5); // start time
    lua_pushnumber(L, 10); // end time
    lua_call(L, 2, 1);
    EXPECT_EQ(lua_type(L, -1), LUA_TBOOLEAN);
    EXPECT_TRUE(lua_toboolean(L, -1)); // 7 seconds is within 5-10 seconds

    lua_close(L);
}
