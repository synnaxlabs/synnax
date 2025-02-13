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

class MockSink final : public plugins::FrameSink {
public:
    std::vector<synnax::Frame> written_frames;
    std::vector<std::pair<
            std::vector<synnax::ChannelKey>,
            std::vector<synnax::Authority> >
    > authority_calls;

    xerrors::Error write(synnax::Frame &frame) override {
        written_frames.push_back(std::move(frame));
        return xerrors::NIL;
    }

    xerrors::Error set_authority(
        const std::vector<synnax::ChannelKey> &keys,
        const std::vector<synnax::Authority> &
        authorities
    ) override {
        authority_calls.emplace_back(keys, authorities);
        return xerrors::NIL;
    }
};


TEST(Sequence, basic) {
    // Read pipeline
    synnax::Channel read_channel;
    read_channel.key = 2;
    read_channel.name = "read_channel";
    read_channel.data_type = telem::FLOAT64;
    auto fr_1 = synnax::Frame(read_channel.key, telem::Series(1.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    const auto read_errors = std::make_shared<std::vector<xerrors::Error> >(
        std::vector{xerrors::NIL, xerrors::NIL,});
    const auto streamer_config = synnax::StreamerConfig{.channels = {read_channel.key}};
    const auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig> >(std::vector{
            pipeline::mock::StreamerConfig{
                reads,
                read_errors,
                xerrors::NIL
            },
        }));
    auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
        streamer_factory, std::vector{read_channel}
    );

    // Write pipeline
    synnax::Channel write_channel;
    write_channel.key = 1;
    write_channel.name = "write_channel";
    write_channel.data_type = telem::FLOAT64;
    auto mock_sink = std::make_shared<MockSink>();
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
    const auto start_err = seq.start();

    ASSERT_FALSE(start_err) << start_err;
    const auto next_err = seq.next();
    ASSERT_FALSE(next_err) << next_err;
    ASSERT_EVENTUALLY_EQ_F([&]-> size_t {
        auto _ = seq.next();
        return mock_sink->written_frames.size();
    }, 1);
    const auto stop_err = seq.end();
    ASSERT_FALSE(stop_err) << stop_err;
    ASSERT_EQ(mock_sink->written_frames[0].channels->at(0), write_channel.key);
}


// We need to explicitly define a main function here instead of using gtest_main
// because otherwise the lua interpreters main function will get executed instead.
int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
