// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/synnax.h"

#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"

namespace pipeline::mock {
// Configuration for a mock Streamer that allows controlling its behavior in tests.
struct StreamerConfig {
    // A sequence of frames that the Streamer will return on each read() call.
    // When all frames are consumed, the Streamer will block briefly and return
    // empty frames.
    std::shared_ptr<std::vector<synnax::Frame>> reads;

    // A sequence of errors to return alongside frames during read() calls.
    // If provided, each read will return the corresponding error at the same index.
    // If nullptr or index exceeds size, returns NIL error.
    std::shared_ptr<std::vector<xerrors::Error>> read_errors;

    // Error to return when close() is called on the Streamer.
    xerrors::Error close_err;
};

// Mock implementation of pipeline::Streamer for testing.
class Streamer final : public pipeline::Streamer {
public:
    // Configuration controlling this Streamer's behavior
    StreamerConfig config;

    // Tracks the current position in the reads sequence
    size_t current_read = 0;

    explicit Streamer(StreamerConfig config): config(std::move(config)) {}

    std::pair<synnax::Frame, xerrors::Error> read() override {
        if (current_read >= config.reads->size()) {
            // block "indefinitely"
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            return {synnax::Frame(0), xerrors::NIL};
        }
        auto fr = std::move(config.reads->at(current_read));
        auto err = xerrors::NIL;
        if (config.read_errors != nullptr && config.read_errors->size() > current_read)
            err = config.read_errors->at(current_read);
        current_read++;
        return {std::move(fr), err};
    }

    xerrors::Error close() override { return config.close_err; }

    void close_send() override {}
};

// Factory for creating mock Streamers with configurable behavior.
class StreamerFactory final : public pipeline::StreamerFactory {
public:
    // Sequence of errors to return when opening new Streamers.
    // Each call to open_streamer consumes the next error.
    std::vector<xerrors::Error> open_errors;

    // Sequence of configurations for created Streamers.
    // Each new Streamer takes the next config, or the last config if exhausted.
    std::shared_ptr<std::vector<StreamerConfig>> configs;

    // Stores the most recent streamer configuration passed to open_streamer
    synnax::StreamerConfig config;

    // Counts how many times open_streamer has been called
    size_t streamer_opens = 0;

    StreamerFactory(
        std::vector<xerrors::Error> open_errors,
        std::shared_ptr<std::vector<StreamerConfig>> configs
    ):
        open_errors(std::move(open_errors)), configs(std::move(configs)) {}

    std::pair<std::unique_ptr<pipeline::Streamer>, xerrors::Error>
    open_streamer(const synnax::StreamerConfig config) override {
        this->streamer_opens++;
        this->config = config;
        /// try to grab the next config
        size_t idx = this->streamer_opens - 1;
        if (this->streamer_opens > this->configs->size())
            idx = this->configs->size() - 1;
        // try to grab the first error. if not, freighter nil
        auto err = this->streamer_opens > this->open_errors.size()
                     ? xerrors::NIL
                     : this->open_errors.at(this->streamer_opens - 1);
        if (err) return {nullptr, err};
        return {std::make_unique<Streamer>((*this->configs)[idx]), xerrors::NIL};
    }
};

inline std::shared_ptr<pipeline::mock::StreamerFactory> simple_streamer_factory(
    const std::vector<synnax::ChannelKey> &keys,
    const std::shared_ptr<std::vector<synnax::Frame>> &reads
) {
    const auto cfg = synnax::StreamerConfig{.channels = keys};
    const auto factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, {}, xerrors::NIL}}
        )
    );
    return factory;
}

// Mock implementation of pipeline::Writer for testing.
class Writer final : public pipeline::Writer {
public:
    // Stores all frames written through this writer
    std::shared_ptr<std::vector<synnax::Frame>> writes;

    // Error to return when close() is called
    xerrors::Error close_err;

    // Index at which write() should return false to simulate failure
    // -1 means never return false
    int return_false_ok_on;

    explicit Writer(
        std::shared_ptr<std::vector<synnax::Frame>> writes,
        const xerrors::Error &close_err = xerrors::NIL,
        const int return_false_ok_on = -1
    ):
        writes(std::move(writes)),
        close_err(close_err),
        return_false_ok_on(return_false_ok_on) {}

    xerrors::Error write(const synnax::Frame &fr) override {
        if (this->writes->size() == this->return_false_ok_on)
            return xerrors::VALIDATION;
        this->writes->push_back(fr.deep_copy());
        return xerrors::NIL;
    }

    xerrors::Error close() override { return this->close_err; }
};

class WriterFactory final : public pipeline::WriterFactory {
public:
    // Stores all frames written through this factory's writers. Shared across all
    // writers created by this factory to allow test verification of written data.
    std::shared_ptr<std::vector<synnax::Frame>> writes;

    // A queue of errors to return when opening writers. Each call to open_writer
    // will consume and return the next error in this vector. Empty vector means no
    // errors.
    std::vector<xerrors::Error> open_errors;

    // A queue of errors for writers to return when closed. Each new writer created
    // will consume and use the next error in this vector for its close() method.
    std::vector<xerrors::Error> close_errors;

    // A queue of indices at which writers should return false for write operations.
    // Each new writer will consume the next value. When a writer's writes->size()
    // equals this value, its write() method will return false, simulating a write
    // failure. A value of -1 (default) means never return false.
    std::vector<int> return_false_ok_on;

    // Stores the most recent writer configuration passed to open_writer
    synnax::WriterConfig config;

    // Counts how many times open_writer has been called, useful for testing retry
    // behavior
    size_t writer_opens;

    explicit WriterFactory(
        std::shared_ptr<std::vector<synnax::Frame>> writes =
            std::make_shared<std::vector<synnax::Frame>>(),
        std::vector<xerrors::Error> open_errors = {},
        std::vector<xerrors::Error> close_errors = {},
        std::vector<int> return_false_ok_on = {}
    ):
        writes(std::move(writes)),
        open_errors(std::move(open_errors)),
        close_errors(std::move(close_errors)),
        return_false_ok_on(std::move(return_false_ok_on)),
        config(),
        writer_opens(0) {}

    std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error>
    open_writer(const synnax::WriterConfig &config) override {
        this->writer_opens++;
        this->config = config;
        auto err = this->open_errors.empty() ? xerrors::NIL : this->open_errors.front();
        if (!this->open_errors.empty())
            this->open_errors.erase(this->open_errors.begin());
        auto close_err = this->close_errors.empty() ? xerrors::NIL
                                                    : this->close_errors.front();
        if (!this->close_errors.empty())
            this->close_errors.erase(this->close_errors.begin());
        auto return_false_ok_on = this->return_false_ok_on.empty()
                                    ? -1
                                    : this->return_false_ok_on.front();
        if (!this->return_false_ok_on.empty())
            this->return_false_ok_on.erase(this->return_false_ok_on.begin());
        auto writer = std::make_unique<Writer>(
            this->writes,
            close_err,
            return_false_ok_on
        );
        return {std::move(writer), err};
    }
};

// Mock implementation of pipeline::Sink for testing.
class Sink : public pipeline::Sink {
public:
    // Stores all frames written through this sink
    std::shared_ptr<std::vector<synnax::Frame>> writes;

    // Sequence of errors to return for write operations
    // Each write consumes the next error in the sequence
    std::shared_ptr<std::vector<xerrors::Error>> write_errors;

    // Stores the error passed to stopped_with_err
    xerrors::Error stop_err;

    Sink():
        writes(std::make_shared<std::vector<synnax::Frame>>()),
        write_errors(std::make_shared<std::vector<xerrors::Error>>()) {}

    Sink(
        const std::shared_ptr<std::vector<synnax::Frame>> &writes,
        const std::shared_ptr<std::vector<xerrors::Error>> &write_errors
    ):
        writes(writes), write_errors(write_errors) {}

    xerrors::Error write(const synnax::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        this->writes->emplace_back(frame.deep_copy());
        // try to grab and remove the first error. if not, freighter nil
        if (this->write_errors->empty()) return xerrors::NIL;
        auto err = this->write_errors->front();
        this->write_errors->erase(this->write_errors->begin());
        return err;
    }

    void stopped_with_err(const xerrors::Error &err) override { this->stop_err = err; }
};

// Mock implementation of pipeline::Source for testing.
class Source : public pipeline::Source {
public:
    // A sequence of frames that the Source will return on each read() call.
    // When all frames are consumed, the Source will block briefly and return empty
    // frames.
    std::shared_ptr<std::vector<synnax::Frame>> reads;

    // A sequence of errors to return alongside frames during read() calls.
    // If provided, each read will return the corresponding error at the same index.
    // If nullptr or index exceeds size, returns NIL error.
    std::shared_ptr<std::vector<xerrors::Error>> read_errors;

    // Stores the error passed to stopped_with_err
    xerrors::Error stop_err;

    // Tracks the current position in the reads sequence
    size_t current_read = 0;

    // Tracks how many times read() has been called
    size_t read_count = 0;

    explicit Source(
        std::shared_ptr<std::vector<synnax::Frame>> reads =
            std::make_shared<std::vector<synnax::Frame>>(),
        std::shared_ptr<std::vector<xerrors::Error>> read_errors = nullptr
    ):
        reads(std::move(reads)), read_errors(std::move(read_errors)) {}

    xerrors::Error read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        read_count++;
        std::this_thread::sleep_for(std::chrono::milliseconds(1));

        if (current_read >= reads->size()) {
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            return xerrors::NIL;
        }

        fr.clear();
        const auto &curr_read = reads->at(current_read);
        for (auto [k, s]: curr_read)
            fr.emplace(k, std::move(s));
        auto err = xerrors::NIL;
        if (read_errors != nullptr && read_errors->size() > current_read)
            err = read_errors->at(current_read);
        current_read++;
        return err;
    }

    void stopped_with_err(const xerrors::Error &err) override { this->stop_err = err; }
};

// Helper function to create a simple Source with predefined frames
inline std::shared_ptr<pipeline::mock::Source>
simple_source(const std::shared_ptr<std::vector<synnax::Frame>> &reads) {
    return std::make_shared<pipeline::mock::Source>(reads);
}
}
