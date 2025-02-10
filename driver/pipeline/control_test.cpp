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

/// internal
#include "driver/pipeline/control.h"
#include "driver/pipeline/mock/pipeline.h"

TEST(ControlPipeline, testHappyPath) {
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, telem::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.emplace(1, telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    const auto read_errors = std::make_shared<std::vector<xerrors::Error> >(
        std::vector{
            xerrors::NIL,
            xerrors::NIL,
        });
    const auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<MockStreamerConfig> >(
            std::vector{
                MockStreamerConfig{
                    reads,
                    read_errors,
                    xerrors::NIL
                }
            })
    );
    const auto sink = std::make_shared<MockSink>();
    auto control = pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        breaker::Config{}
    );
    control.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    control.stop();
    ASSERT_EQ(sink->writes->size(), 2);
}

TEST(ControlPipeline, testUnknownErrOnOpen) {
    const auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector{
            freighter::UNKNOWN
        },
        std::make_shared<std::vector<MockStreamerConfig> >()
    );
    const auto sink = std::make_shared<MockSink>();
    auto control = pipeline::Control(
        streamer_factory,
        StreamerConfig{},
        sink,
        breaker::Config{}
    );
    control.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    control.stop();
    ASSERT_EQ(sink->writes->size(), 0);
    ASSERT_TRUE(sink->stop_err.matches(freighter::UNKNOWN));
}

TEST(ControlPipeline, testOpenRetrySuccessful) {
    auto fr_1 = synnax::Frame(1);
    fr_1.emplace(1, telem::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.emplace(1, telem::Series(2.0));
    auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    auto read_errors = std::make_shared<std::vector<xerrors::Error> >(
        std::vector{
            xerrors::NIL,
            xerrors::NIL,
        });
    auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector{
            freighter::UNREACHABLE,
            freighter::UNREACHABLE,
            xerrors::NIL
        },
        std::make_shared<std::vector<MockStreamerConfig> >(
            std::vector{
                MockStreamerConfig{
                    reads,
                    read_errors,
                    xerrors::NIL,
                },
                MockStreamerConfig{
                    reads,
                    read_errors,
                    xerrors::NIL,
                },
                MockStreamerConfig{
                    reads,
                    read_errors,
                    xerrors::NIL
                }
            }
        )
    );
    auto sink = std::make_shared<MockSink>();
    auto control = pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        breaker::Config{
            .max_retries = 2,
            .base_interval = telem::MICROSECOND * 10
        }
    );

    control.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    control.stop();
    ASSERT_EQ(streamer_factory->streamer_opens, 3);
    ASSERT_EQ(sink->writes->size(), 2);
}
