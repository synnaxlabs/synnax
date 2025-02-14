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
#include "x/cpp/xtest/xtest.h"

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
    ch.data_type = telem::FLOAT64_T;
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, telem::Series(1.0, telem::FLOAT64_T));
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    const auto read_errors = std::make_shared<std::vector<xerrors::Error> >(
        std::vector{
            xerrors::NIL,
            xerrors::NIL,
        });
    const auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig> >(
            std::vector{
                pipeline::mock::StreamerConfig{
                    reads,
                    read_errors,
                    xerrors::NIL
                }
            })
    );
    auto plugin = plugins::ChannelReceive(streamer_factory, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);
    plugin.before_all(L);
    ASSERT_EVENTUALLY_EQ_F([&] {
        plugin.before_next(L);
        return lua_getglobal(L, "my_channel");
    }, LUA_TNUMBER);
    ASSERT_EQ(lua_tonumber(L, -1), 1.0);
    lua_close(L);
    plugin.after_all(L);
}

// We need to explicitly define a main function here instead of using gtest_main
// because otherwise the lua interpreters main function will get executed instead.
int main(int argc, char** argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
