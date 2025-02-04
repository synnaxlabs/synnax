// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest
#include "gtest/gtest.h"

#include "driver/pipeline/acquisition.h"

class MockWriter final : public pipeline::Writer {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    freighter::Error close_err;
    int return_false_ok_on;

    explicit MockWriter(
        std::shared_ptr<std::vector<synnax::Frame> > writes,
        freighter::Error close_err = freighter::NIL,
        int return_false_ok_on = -1
    ) : writes(std::move(writes)),
        close_err(close_err),
        return_false_ok_on(return_false_ok_on) {
    }

    bool write(synnax::Frame &fr) override {
        if (this->writes->size() == this->return_false_ok_on) return false;
        this->writes->push_back(std::move(fr));
        return true;
    }

    freighter::Error close() override {
        return this->close_err;
    }
};

class MockWriterFactory final : public pipeline::WriterFactory {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    std::vector<freighter::Error> open_errors;
    std::vector<freighter::Error> close_errors;
    std::vector<int> return_false_ok_on;
    WriterConfig config;
    size_t writer_opens;

    explicit MockWriterFactory(
        std::shared_ptr<std::vector<synnax::Frame> > writes,
        std::vector<freighter::Error> open_errors = {},
        std::vector<freighter::Error> close_errors = {},
        std::vector<int> return_false_ok_on = {}
    ) : writes(
            std::move(writes)), open_errors(std::move(open_errors)),
        close_errors(std::move(close_errors)),
        return_false_ok_on(std::move(return_false_ok_on)),
        config(),
        writer_opens(0) {
    }

    std::pair<std::unique_ptr<pipeline::Writer>, freighter::Error> openWriter(
        const WriterConfig &config) override {
            this->writer_opens++;
            this->config = config;
            auto err = this->open_errors.empty()
                           ? freighter::NIL
                           : this->open_errors.front();
            if (!this->open_errors.empty())
                this->open_errors.erase(
                    this->open_errors.begin());
            auto close_err = this->close_errors.empty()
                                 ? freighter::NIL
                                 : this->close_errors.front();
            if (!this->close_errors.empty())
                this->close_errors.erase(
                    this->close_errors.begin());
            auto return_false_ok_on = this->return_false_ok_on.empty()
                                          ? -1
                                          : this->return_false_ok_on.front();
            if (!this->return_false_ok_on.empty())
                this->return_false_ok_on.erase(
                    this->return_false_ok_on.begin());
            auto writer = std::make_unique<MockWriter>(
                this->writes, close_err, return_false_ok_on);
            return {std::move(writer), err};
    }
};

class MockSource : public pipeline::Source {
public:
    synnax::TimeStamp start_ts;

    explicit MockSource(synnax::TimeStamp start_ts) : start_ts(start_ts) {
    }

    std::pair<Frame, freighter::Error> read() override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto fr = Frame(1);
        fr.add(1, synnax::Series(start_ts));
        return {std::move(fr), freighter::Error()};
    }
};


/// @brief it should correctly resolve the start timestamp for the pipeline from the
/// first frame written.
TEST(AcquisitionPipeline, testStartResolution) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    auto start_ts = synnax::TimeStamp::now();
    auto source = std::make_shared<MockSource>(start_ts);
    synnax::WriterConfig writer_config{.channels = {1}};
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(600));
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
    ASSERT_EQ(writes->at(0).at<std::uint64_t>(1, 0), start_ts.value);
    ASSERT_EQ(mock_factory->config.start.value, start_ts.value);
}

/// @brief it should correclty retry opening the writer when an unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{
            freighter::UNREACHABLE, freighter::UNREACHABLE, freighter::NIL
        });
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 3,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_GE(writes->size(), 1);
}

/// @brief it should not retry when a non-unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{
            freighter::Error(synnax::UNAUTHORIZED_ERROR), freighter::NIL
        }
    );
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 3,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should retry opening the writer when write returns false and the
/// error is unreachable.
TEST(AcquisitionPipeline, testWriteRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{},
        std::vector{freighter::UNREACHABLE},
        std::vector<int>{1}
    );
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 1,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    pipeline.stop();
    ASSERT_EQ(mock_factory->writer_opens, 2);
    ASSERT_GE(writes->size(), 3);
}

/// @brief it should not retry opening the writer when write returns false and the
/// error is not unreachable.
TEST(AcquisitionPipeline, testWriteRetryUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{},
        std::vector{freighter::Error(synnax::UNAUTHORIZED_ERROR)},
        std::vector<int>{0}
    );
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 1,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    pipeline.stop();
    ASSERT_EQ(mock_factory->writer_opens, 1);
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, testStartAlreadyStartedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
}

/// @brief it should not stop the pipeline if it has already been stopped.
TEST(AcquisitionPipeline, testStopAlreadyStoppedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
}
