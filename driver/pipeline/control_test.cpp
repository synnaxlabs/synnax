// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest
#include "gtest/gtest.h"

#include "driver/pipeline/control.h"

struct MockStreamerConfig {
    std::shared_ptr<std::vector<synnax::Frame> > reads;
    std::shared_ptr<std::vector<freighter::Error> > read_errors;
    freighter::Error close_err;
};

class MockStreamer final : public pipeline::Streamer {
public:
    MockStreamerConfig config;
    size_t current_read = 0;

    explicit MockStreamer(
        MockStreamerConfig config
    ) : config(std::move(config)) {
    }

    std::pair<synnax::Frame, freighter::Error> read() override {
        if (current_read >= config.reads->size()) {
            // block "indefinitely"
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            return {synnax::Frame(0), freighter::NIL};
        }
        auto fr = std::move(config.reads->at(current_read));
        auto err = config.read_errors->at(current_read);
        current_read++;
        return {std::move(fr), freighter::NIL};
    }

    freighter::Error close() override { return config.close_err; }
};

class MockStreamerFactory final : public pipeline::StreamerFactory {
public:
    std::vector<freighter::Error> open_errors;
    std::shared_ptr<std::vector<MockStreamerConfig> > configs;
    StreamerConfig config;
    size_t streamer_opens = 0;

    MockStreamerFactory(
        std::vector<freighter::Error> open_errors,
        std::shared_ptr<std::vector<MockStreamerConfig> > configs
    ) : open_errors(std::move(open_errors)), configs(std::move(configs)) {
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, freighter::Error> openStreamer(
        synnax::StreamerConfig config) override {
        this->streamer_opens++;
        this->config = config;
        /// try to grab the next config
        size_t idx = this->streamer_opens - 1;
        if (this->streamer_opens > this->configs->size())
            idx = this->configs->size() - 1;
        // try to grab the first error. if not, freighter nil
        auto err = this->streamer_opens > this->open_errors.size()
                       ? freighter::NIL
                       : this->open_errors.at(this->streamer_opens - 1);
        if (err) return {nullptr, err};
        return {
            std::make_unique<MockStreamer>((*this->configs)[idx]),
            freighter::NIL
        };
    }
};

class MockSink final : public pipeline::Sink {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    std::shared_ptr<std::vector<freighter::Error> > write_errors;
    freighter::Error stop_err;

    MockSink() : writes(std::make_shared<std::vector<synnax::Frame> >()),
                 write_errors(std::make_shared<std::vector<freighter::Error> >()) {
    }

    freighter::Error write(synnax::Frame frame) override {
        if (frame.size() == 0) return freighter::NIL;
        this->writes->push_back(std::move(frame));
        // try to grab and remove the firste rror. if not, friehgter nil
        if (this->write_errors->empty()) return freighter::NIL;
        auto err = this->write_errors->front();
        this->write_errors->erase(this->write_errors->begin());
        return err;
    }

    void stopped_with_err(const freighter::Error &err) override {
        this->stop_err = err;
    }
};

TEST(ControlPipeline, testHappyPath) {
    auto fr_1 = synnax::Frame(1);
    fr_1.add(1, synnax::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.add(1, synnax::Series(2.0));
    auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    auto read_errors = std::make_shared<std::vector<freighter::Error> >(
        std::vector<freighter::Error>{
            freighter::NIL,
            freighter::NIL,
        });
    auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector<freighter::Error>{},
        std::make_shared<std::vector<MockStreamerConfig> >(
            std::vector<MockStreamerConfig>{
                MockStreamerConfig{
                    reads,
                    read_errors,
                    freighter::NIL
                }
            })
    );
    auto sink = std::make_shared<MockSink>();
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
    auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector<freighter::Error>{
            freighter::UNKNOWN
        },
        std::make_shared<std::vector<MockStreamerConfig> >()
    );
    auto sink = std::make_shared<MockSink>();
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
    fr_1.add(1, synnax::Series(1.0));
    auto fr_2 = synnax::Frame(1);
    fr_2.add(1, synnax::Series(2.0));
    auto reads = std::make_shared<std::vector<synnax::Frame> >();
    reads->push_back(std::move(fr_1));
    reads->push_back(std::move(fr_2));
    auto read_errors = std::make_shared<std::vector<freighter::Error> >(
        std::vector<freighter::Error>{
            freighter::NIL,
            freighter::NIL,
        });
    auto streamer_config = synnax::StreamerConfig{.channels = {1}};
    auto streamer_factory = std::make_shared<MockStreamerFactory>(
        std::vector<freighter::Error>{
            freighter::UNREACHABLE,
            freighter::UNREACHABLE,
            freighter::NIL
        },
        std::make_shared<std::vector<MockStreamerConfig> >(
            std::vector<MockStreamerConfig>{
                MockStreamerConfig{
                    reads,
                    read_errors,
                    freighter::NIL,
                },
                MockStreamerConfig{
                    reads,
                    read_errors,
                    freighter::NIL,
                },
                MockStreamerConfig{
                    reads,
                    read_errors,
                    freighter::NIL
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
            .base_interval = synnax::MICROSECOND * 10
        }
    );

    control.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    control.stop();
    ASSERT_EQ(streamer_factory->streamer_opens, 3);
    ASSERT_EQ(sink->writes->size(), 2);
}
