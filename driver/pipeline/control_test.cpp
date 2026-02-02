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

#include "driver/pipeline/control.h"
#include "driver/pipeline/mock/pipeline.h"

/// @brief it should read frames from streamer and write to sink.
TEST(ControlPipeline, testHappyPath) {
    auto fr_1 = x::telem::Frame(1);
    fr_1.emplace(1, x::telem::Series(1.0));
    auto fr_2 = x::telem::Frame(1);
    fr_2.emplace(1, x::telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    const auto read_errors = std::make_shared<std::vector<x::errors::Error>>(
        std::vector{
            x::errors::NIL,
            x::errors::NIL,
        }
    );
    const auto streamer_config = synnax::framer::StreamerConfig{.channels = {1}};
    const auto
        streamer_factory = std::make_shared<driver::pipeline::mock::StreamerFactory>(
            std::vector<x::errors::Error>{},
            std::make_shared<std::vector<driver::pipeline::mock::StreamerConfig>>(
                std::vector{driver::pipeline::mock::StreamerConfig{
                    reads,
                    read_errors,
                    x::errors::NIL
                }}
            )
        );
    const auto sink = std::make_shared<driver::pipeline::mock::Sink>();
    auto control = driver::pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        x::breaker::Config{}
    );
    control.start();
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    control.stop();
}

/// @brief it should stop and report error when streamer open fails with unknown error.
TEST(ControlPipeline, testUnknownErrOnOpen) {
    const auto
        streamer_factory = std::make_shared<driver::pipeline::mock::StreamerFactory>(
            std::vector{x::errors::UNKNOWN},
            std::make_shared<std::vector<driver::pipeline::mock::StreamerConfig>>()
        );
    const auto sink = std::make_shared<driver::pipeline::mock::Sink>();
    auto control = driver::pipeline::Control(
        streamer_factory,
        synnax::framer::StreamerConfig{},
        sink,
        x::breaker::Config{}
    );
    control.start();
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 0);
    control.stop();
    ASSERT_MATCHES(sink->stop_err, x::errors::UNKNOWN);
}

/// @brief it should retry opening streamer on unreachable error and succeed.
TEST(ControlPipeline, testOpenRetrySuccessful) {
    auto fr_1 = x::telem::Frame(1);
    fr_1.emplace(1, x::telem::Series(1.0));
    auto fr_2 = x::telem::Frame(1);
    fr_2.emplace(1, x::telem::Series(2.0));
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    const auto read_errors = std::make_shared<std::vector<x::errors::Error>>(
        std::vector{
            x::errors::NIL,
            x::errors::NIL,
        }
    );
    const auto streamer_config = synnax::framer::StreamerConfig{.channels = {1}};
    const auto streamer_factory = std::make_shared<
        driver::pipeline::mock::StreamerFactory>(
        std::vector{
            freighter::ERR_UNREACHABLE,
            freighter::ERR_UNREACHABLE,
            x::errors::NIL
        },
        std::make_shared<
            std::vector<driver::pipeline::mock::StreamerConfig>>(std::vector{
            driver::pipeline::mock::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL,
            },
            driver::pipeline::mock::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL,
            },
            driver::pipeline::mock::StreamerConfig{reads, read_errors, x::errors::NIL}
        })
    );
    const auto sink = std::make_shared<driver::pipeline::mock::Sink>();
    auto control = driver::pipeline::Control(
        streamer_factory,
        streamer_config,
        sink,
        x::breaker::Config{
            .base_interval = x::telem::MICROSECOND * 10,
            .max_retries = 2,
        }
    );

    control.start();
    ASSERT_EVENTUALLY_EQ(streamer_factory->streamer_opens, 3);
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    control.stop();
}
