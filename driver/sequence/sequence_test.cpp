// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/channel/channel.h"
#include "client/cpp/framer/framer.h"
#include "x/cpp/test/test.h"

#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/mock/plugins.h"
#include "driver/sequence/plugins/plugins.h"
#include "driver/sequence/sequence.h"

namespace driver::sequence {
/// @brief it should executed a basic sequence.
TEST(Sequence, nominal) {
    synnax::channel::Channel read_channel;
    read_channel.key = 2;
    read_channel.name = "read_channel";
    read_channel.data_type = x::telem::FLOAT64_T;
    auto fr_1 = x::telem::Frame(read_channel.key, x::telem::Series(1.0));
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    reads->push_back(std::move(fr_1));
    auto streamer_factory = pipeline::mock::simple_streamer_factory(
        {read_channel.key},
        reads
    );
    auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
        streamer_factory,
        std::vector{read_channel}
    );

    // Write pipeline
    synnax::channel::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = x::telem::FLOAT64_T;
    auto mock_sink = std::make_shared<plugins::mock::FrameSink>();
    auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
        mock_sink,
        std::vector{write_channel}
    );
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{
            ch_receive_plugin,
            ch_write_plugin
        }
    );

    const auto script = R"(
        if read_channel == nil then
            return
        end
        set("write_channel", read_channel)
    )";

    auto seq = Sequence(plugins, script);
    const auto start_err = seq.begin();
    const auto next_err = seq.next();

    auto check_writes = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    ASSERT_NIL(start_err);
    ASSERT_NIL(next_err);
    ASSERT_EVENTUALLY_GE_F(check_writes, 1);
    ASSERT_NIL(seq.end());
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);
}

/// @brief it should correctly return an error when the script fails to compile.
TEST(Sequence, compileError) {
    const auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{}
    );

    const auto script = R"(
        if read_channel = nil then  -- incorrect equality operator
            return
        end
    )";

    auto seq = Sequence(plugins, script);
    ASSERT_OCCURRED_AS(seq.compile(), COMPILATION_ERROR);
}

/// @brief it should return an error when the caller tries to compare a number with
/// nil
TEST(Sequence, compareNil) {
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{}
    );
    const auto script = R"(
        if 42 > nil then
            return
        end
    )";
    auto seq = Sequence(plugins, script);
    ASSERT_NIL(seq.begin());
    ASSERT_OCCURRED_AS(seq.next(), RUNTIME_ERROR);
    ASSERT_NIL(seq.end());
}

/// @brief it should return an error when trying to set a non-existent channel
TEST(Sequence, channelNotFound) {
    synnax::channel::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = x::telem::FLOAT64_T;
    auto mock_sink = std::make_shared<plugins::mock::FrameSink>();
    auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
        mock_sink,
        std::vector{write_channel}
    );
    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{ch_write_plugin}
    );
    const auto script = R"(
        set("nonexistent_channel", 42)
    )";
    auto seq = Sequence(plugins, script);
    ASSERT_NIL(seq.begin());
    const auto next_err = seq.next();
    ASSERT_MATCHES(next_err, RUNTIME_ERROR);
    EXPECT_NE(next_err.message().find("nonexistent_channel"), std::string::npos);
    EXPECT_NE(next_err.message().find("not found"), std::string::npos);
    ASSERT_NIL(seq.end());
    EXPECT_EQ(mock_sink->writes->size(), 0);
}

/// @brief it should correctly restart and re-execute a sequence several times,
/// including binding correct variable names and functions.
TEST(Sequence, restart) {
    synnax::channel::Channel read_channel;
    read_channel.key = 2;
    read_channel.name = "read_channel";
    read_channel.data_type = x::telem::FLOAT64_T;

    auto fr_1 = x::telem::Frame(read_channel.key, x::telem::Series(1.0));
    auto fr_2 = x::telem::Frame(read_channel.key, x::telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));

    auto streamer_factory = pipeline::mock::simple_streamer_factory(
        {read_channel.key},
        reads
    );
    auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
        streamer_factory,
        std::vector{read_channel}
    );

    synnax::channel::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = x::telem::FLOAT64_T;
    auto mock_sink = std::make_shared<plugins::mock::FrameSink>();
    auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
        mock_sink,
        std::vector{write_channel}
    );

    auto plugins = std::make_shared<plugins::MultiPlugin>(
        std::vector<std::shared_ptr<plugins::Plugin>>{
            ch_receive_plugin,
            ch_write_plugin
        }
    );

    const auto script = R"(
        if read_channel == nil then
            return
        end
        set("write_channel", read_channel)
    )";

    auto seq = Sequence(plugins, script);

    auto check_writes = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    // First execution
    ASSERT_NIL(seq.begin());
    ASSERT_NIL(seq.next());
    ASSERT_EVENTUALLY_GE_F(check_writes, 1);
    ASSERT_NIL(seq.end());
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);

    auto curr_size = mock_sink->writes->size();

    auto check_writes_2 = [&]() -> size_t {
        auto _ = seq.next();
        return mock_sink->writes->size();
    };

    ASSERT_NIL(seq.begin());
    ASSERT_NIL(seq.next());
    ASSERT_EVENTUALLY_GE_F(check_writes_2, curr_size);
    ASSERT_NIL(seq.end());
    ASSERT_EQ(mock_sink->writes->at(0).channels->at(0), write_channel.key);
}
}
