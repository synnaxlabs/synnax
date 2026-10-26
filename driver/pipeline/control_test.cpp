// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <future>

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/test/test.h"

#include "driver/pipeline/control.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::pipeline {
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
    const auto streamer_factory = std::make_shared<mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<mock::StreamerConfig>>(
            std::vector{mock::StreamerConfig{reads, read_errors, x::errors::NIL}}
        )
    );
    const auto sink = std::make_shared<mock::Sink>();
    auto control = Control(
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
    const auto streamer_factory = std::make_shared<mock::StreamerFactory>(
        std::vector{x::errors::UNKNOWN},
        std::make_shared<std::vector<mock::StreamerConfig>>()
    );
    const auto sink = std::make_shared<mock::Sink>();
    auto control = Control(
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

namespace {
/// @brief asserts that the pipeline retries a streamer open failing twice with
/// open_err, then succeeds.
void assert_open_retries(const x::errors::Error &open_err) {
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
    const auto streamer_factory = std::make_shared<mock::StreamerFactory>(
        std::vector{open_err, open_err, x::errors::NIL},
        std::make_shared<std::vector<mock::StreamerConfig>>(std::vector{
            mock::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL,
            },
            mock::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL,
            },
            mock::StreamerConfig{reads, read_errors, x::errors::NIL}
        })
    );
    const auto sink = std::make_shared<mock::Sink>();
    auto control = Control(
        streamer_factory,
        streamer_config,
        sink,
        x::breaker::Config{
            .base_interval = x::telem::MICROSECOND * 10,
            .max_retries = 2,
        }
    );

    control.start();
    ASSERT_EVENTUALLY_EQ(
        streamer_factory->streamer_opens.load(std::memory_order_acquire),
        3
    );
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    control.stop();
}
}

/// @brief it should retry opening streamer on unreachable error and succeed.
TEST(ControlPipeline, testOpenRetrySuccessful) {
    assert_open_retries(freighter::UNREACHABLE);
}

/// @brief it should wait out an unlicensed Core and succeed once it is licensed.
TEST(ControlPipeline, testOpenRetryUnlicensed) {
    assert_open_retries(synnax::errors::LICENSE_MISSING);
}
namespace {
/// @brief a streamer factory whose open blocks until the test releases it.
class GatedStreamerFactory final : public StreamerFactory {
    std::shared_ptr<mock::StreamerFactory> inner;
    std::shared_future<void> gate;

public:
    GatedStreamerFactory(
        std::shared_ptr<mock::StreamerFactory> inner,
        std::shared_future<void> gate
    ):
        inner(std::move(inner)), gate(std::move(gate)) {}

    std::pair<std::unique_ptr<Streamer>, x::errors::Error>
    open_streamer(const synnax::framer::StreamerConfig config) override {
        this->gate.wait();
        return this->inner->open_streamer(config);
    }
};
}

/// @brief it should block wait_until_open until the streamer is open.
TEST(ControlPipeline, WaitUntilOpenBlocksUntilStreamerOpens) {
    std::promise<void> release;
    const auto factory = std::make_shared<GatedStreamerFactory>(
        mock::simple_streamer_factory(
            {1},
            std::make_shared<std::vector<x::telem::Frame>>()
        ),
        release.get_future().share()
    );
    auto control = Control(
        factory,
        synnax::framer::StreamerConfig{.channels = {1}},
        std::make_shared<mock::Sink>(),
        x::breaker::Config{}
    );
    ASSERT_TRUE(control.start());
    auto waited = std::async(std::launch::async, [&] { control.wait_until_open(); });
    ASSERT_EQ(
        waited.wait_for(std::chrono::milliseconds(50)),
        std::future_status::timeout
    );
    release.set_value();
    ASSERT_EQ(waited.wait_for(std::chrono::seconds(5)), std::future_status::ready);
    control.stop();
}

/// @brief it should release wait_until_open when the streamer fails to open for good.
TEST(ControlPipeline, WaitUntilOpenReturnsOnPermanentOpenFailure) {
    const auto sink = std::make_shared<mock::Sink>();
    auto control = Control(
        std::make_shared<mock::StreamerFactory>(
            std::vector{x::errors::UNKNOWN},
            std::make_shared<std::vector<mock::StreamerConfig>>()
        ),
        synnax::framer::StreamerConfig{},
        sink,
        x::breaker::Config{}
    );
    ASSERT_TRUE(control.start());
    control.wait_until_open();
    control.stop();
    ASSERT_MATCHES(sink->stop_err, x::errors::UNKNOWN);
}
}
