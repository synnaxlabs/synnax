// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external.
#include "gtest/gtest.h"

extern "C" {
#include <lualib.h>
}

/// internal.
#include "driver/sequence/plugins/plugins.h"
#include "driver/pipeline/mock/pipeline.h"

TEST(ChannelReceive, Basic) {
    synnax::Channel ch;
    ch.key = 1;
    ch.name = "my_channel";
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, synnax::Series(1.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    const auto read_errors = std::make_shared<std::vector<freighter::Error> >(
        std::vector{
            freighter::NIL,
            freighter::NIL,
        });
    const auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector<freighter::Error>{},
        std::make_shared<std::vector<MockStreamerConfig> >(
            std::vector{
                MockStreamerConfig{
                    reads,
                    read_errors,
                    freighter::NIL
                }
            })
    );
    auto plugin = plugins::ChannelReceive(streamer_factory, std::vector{ch});
    auto L = luaL_newstate();
    luaL_openlibs(L);
    plugin.before_all(L);
    plugin.before_next(L);
    // assert that the lua state has a variable named "my_channel" with the value 1.0
    ASSERT_EQ(lua_getglobal(L, "my_channel"), LUA_TNUMBER);
    ASSERT_EQ(lua_tonumber(L, -1), 1.0);
    lua_close(L);
    plugin.after_all(L);
}
