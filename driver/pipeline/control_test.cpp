// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/pipeline/control.h"
#include "driver/pipeline/mock/pipeline.h"

TEST(ControlPipeline, testHappyPath) {
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, telem::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.emplace(1, telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    const auto read_errors = std::make_shared<std::vector<xerrors::Error>>(std::vector{
        xerrors::NIL,
        xerrors::NIL,
    });
    const auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(std::vector{
            pipeline::mock::StreamerConfig{reads, read_errors, xerrors::NIL}
        })
    );
    const auto sink = std::make_shared<pipeline::mock::Sink>();
    auto control = pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        breaker::Config{}
    );
    control.start();
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    control.stop();
}

TEST(ControlPipeline, testUnknownErrOnOpen) {
    const auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector{xerrors::UNKNOWN},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>()
    );
    const auto sink = std::make_shared<pipeline::mock::Sink>();
    auto control = pipeline::Control(
        streamer_factory,
        synnax::StreamerConfig{},
        sink,
        breaker::Config{}
    );
    control.start();
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 0);
    control.stop();
    ASSERT_TRUE(sink->stop_err.matches(xerrors::UNKNOWN));
}

TEST(ControlPipeline, testOpenRetrySuccessful) {
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, telem::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.emplace(1, telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    const auto read_errors = std::make_shared<std::vector<xerrors::Error>>(std::vector{
        xerrors::NIL,
        xerrors::NIL,
    });
    const auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector{freighter::UNREACHABLE, freighter::UNREACHABLE, xerrors::NIL},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(std::vector{
            pipeline::mock::StreamerConfig{
                reads,
                read_errors,
                xerrors::NIL,
            },
            pipeline::mock::StreamerConfig{
                reads,
                read_errors,
                xerrors::NIL,
            },
            pipeline::mock::StreamerConfig{reads, read_errors, xerrors::NIL}
        })
    );
    const auto sink = std::make_shared<pipeline::mock::Sink>();
    auto control = pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 2,
        }
    );

    control.start();
    ASSERT_EVENTUALLY_EQ(streamer_factory->streamer_opens, 3);
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    control.stop();
}
