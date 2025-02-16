// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "client/cpp/channel/channel.h"
#include "client/cpp/framer/framer.h"
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/sequence/sequence.h"
#include "driver/sequence/plugins/plugins.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/mock/plugins.h"

/// @brief it should executed a basic sequence.
TEST(Sequence, nominal) {
    // Read pipeline
    synnax::Channel read_channel;
    read_channel.key = 2;
    read_channel.name = "read_channel";
    read_channel.data_type = telem::FLOAT64_T;
    auto fr_1 = synnax::Frame(read_channel.key, telem::Series(1.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    auto streamer_factory = pipeline::mock::simple_streamer_factory({read_channel.key}, reads);
    auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
        streamer_factory, std::vector{read_channel}
    );

    // Write pipeline
    synnax::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = telem::FLOAT64_T;
    auto mock_sink = std::make_shared<plugins::mock::FrameSink>();
    auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
        mock_sink, std::vector{write_channel});
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin> >{
            ch_receive_plugin, ch_write_plugin
        });


    const auto script = R"(
        if read_channel == nil then
            return 
        end
        set("write_channel", read_channel)
    )";

    auto seq = sequence::Sequence(plugins, script);
    const auto start_err = seq.begin();
    const auto next_err = seq.next();

    auto check_writes = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    ASSERT_FALSE(start_err) << start_err;
    ASSERT_FALSE(next_err) << next_err;
    ASSERT_EVENTUALLY_GE_F(check_writes, 1);
    const auto stop_err = seq.end();
    ASSERT_FALSE(stop_err) << stop_err;
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);
}

/// @brief it should correctly return an error when the script fails to compile.
TEST(Sequence, compileError) {
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin> >{});
    
    // Invalid Lua syntax
    const auto script = R"(
        if read_channel = nil then  -- incorrect equality operator
            return
        end
    )";

    auto seq = sequence::Sequence(plugins, script);
    const auto err = seq.compile();
    ASSERT_TRUE(err);
    ASSERT_TRUE(err.matches(sequence::COMPILATION_ERROR));
}

/// @brief it should return an error when the caller tries to compare a number with
/// nil
TEST(Sequence, compareNil) {
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin> >{});
    const auto script = R"(
        if 42 > nil then
            return
        end
    )";
    auto seq = sequence::Sequence(plugins, script);
    const auto start_err = seq.begin();
    ASSERT_FALSE(start_err) << start_err;
    const auto next_err = seq.next();
    ASSERT_TRUE(next_err) << next_err;
    ASSERT_TRUE(next_err.matches(sequence::RUNTIME_ERROR));
    const auto end_err = seq.end();
    ASSERT_FALSE(end_err) << next_err;
}

/// @brief it should correctly restart and re-execute a sequence several times,
/// including binding correct variable names and functions.
TEST(Sequence, restart) {
    // Setup read pipeline
    synnax::Channel read_channel;
    read_channel.key = 2;
    read_channel.name = "read_channel";
    read_channel.data_type = telem::FLOAT64_T;
    
    auto fr_1 = synnax::Frame(read_channel.key, telem::Series(1.0));
    auto fr_2 = synnax::Frame(read_channel.key, telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    
    auto streamer_factory = pipeline::mock::simple_streamer_factory({read_channel.key}, reads);
    auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
        streamer_factory, std::vector{read_channel}
    );

    // Setup write pipeline
    synnax::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = telem::FLOAT64_T;
    auto mock_sink = std::make_shared<plugins::mock::FrameSink>();
    auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
        mock_sink, std::vector{write_channel});

    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{
            ch_receive_plugin, ch_write_plugin
        });

    const auto script = R"(
        if read_channel == nil then
            return 
        end
        set("write_channel", read_channel)
    )";

    auto seq = sequence::Sequence(plugins, script);

    auto check_writes = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    // First execution
    ASSERT_FALSE(seq.begin());
    ASSERT_FALSE(seq.next());
    ASSERT_EVENTUALLY_GE_F(check_writes, 1);
    ASSERT_FALSE(seq.end());
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);

    auto curr_size = mock_sink->writes->size();
    
    auto check_writes_2 = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    ASSERT_FALSE(seq.begin());
    ASSERT_FALSE(seq.next());
    ASSERT_EVENTUALLY_GE_F(check_writes_2, curr_size);
    ASSERT_FALSE(seq.end());
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);
}
