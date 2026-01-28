// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

extern "C" {
#include <lualib.h>
}

/// internal.
#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/plugins.h"

namespace driver::sequence::plugins {
/// @brief it should receive channel values and expose them as Lua globals.
TEST(ChannelReceive, Basic) {
    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "my_channel";
    ch.data_type = x::telem::FLOAT64_T;
    auto fr_1 = x::telem::Frame(1);
    fr_1.emplace(1, x::telem::Series(1.0, x::telem::FLOAT64_T));
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    reads->push_back(std::move(fr_1));
    const auto factory = pipeline::mock::simple_streamer_factory(
        {ch.key},
        reads
    );
    auto plugin = ChannelReceive(factory, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);
    plugin.before_all(L);
    auto check_writes = [&]() {
        plugin.before_next(L);
        return lua_getglobal(L, "my_channel");
    };
    ASSERT_EVENTUALLY_EQ_F(check_writes, LUA_TNUMBER);
    ASSERT_EQ(lua_tonumber(L, -1), 1.0);
    lua_close(L);
    plugin.after_all(L);
}

/// @brief it should safely handle stop being called before start.
TEST(ChannelReceive, StopBeforeStart) {
    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "my_channel";
    ch.data_type = x::telem::FLOAT64_T;
    const auto factory = pipeline::mock::simple_streamer_factory(
        {ch.key},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    auto plugin = ChannelReceive(factory, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);
    // Stopping before starting should be safe
    plugin.after_all(L);
    lua_close(L);
}

/// @brief it should safely handle being started twice.
TEST(ChannelReceive, DoubleStart) {
    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "my_channel";
    ch.data_type = x::telem::FLOAT64_T;
    const auto factory = pipeline::mock::simple_streamer_factory(
        {ch.key},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    auto plugin = ChannelReceive(factory, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);
    // Starting twice should be safe
    plugin.before_all(L);
    plugin.before_all(L);
    plugin.after_all(L);
    lua_close(L);
}

/// @brief it should safely handle being stopped twice.
TEST(ChannelReceive, DoubleStop) {
    synnax::channel::Channel ch;
    ch.key = 1;
    ch.name = "my_channel";
    ch.data_type = x::telem::FLOAT64_T;
    const auto factory = pipeline::mock::simple_streamer_factory(
        {ch.key},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    auto plugin = ChannelReceive(factory, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);
    plugin.before_all(L);
    // Stopping twice should be safe
    plugin.after_all(L);
    plugin.after_all(L);
    lua_close(L);
}
}