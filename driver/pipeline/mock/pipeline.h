// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>

#include "client/cpp/synnax.h"

#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"

namespace driver::pipeline::mock {
// Configuration for a mock Streamer that allows controlling its behavior in tests.
struct StreamerConfig {
    // A sequence of frames that the Streamer will return on each read() call.
    // When all frames are consumed, the Streamer will block briefly and return
    // empty frames.
    std::shared_ptr<std::vector<x::telem::Frame>> reads;

    // A sequence of errors to return alongside frames during read() calls.
    // If provided, each read will return the corresponding error at the same index.
    // If nullptr or index exceeds size, returns NIL error.
    std::shared_ptr<std::vector<x::errors::Error>> read_errors;

    // Error to return when close() is called on the Streamer.
    x::errors::Error close_err;
};

// Mock implementation of pipeline::Streamer for testing.
class Streamer final : public pipeline::Streamer {
public:
    // Configuration controlling this Streamer's behavior
    StreamerConfig config;

    // Tracks the current position in the reads sequence
    size_t current_read = 0;

    explicit Streamer(StreamerConfig config): config(std::move(config)) {}

    std::pair<x::telem::Frame, x::errors::Error> read() override {
        if (current_read >= config.reads->size()) {
            // block "indefinitely"
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            if (this->config.read_errors != nullptr &&
                !this->config.read_errors->empty())
                return {x::telem::Frame{}, this->config.read_errors->at(0)};
            return {x::telem::Frame(0), x::errors::NIL};
        }
        auto fr = std::move(config.reads->at(current_read));
        auto err = x::errors::NIL;
        if (config.read_errors != nullptr && config.read_errors->size() > current_read)
            err = config.read_errors->at(current_read);
        current_read++;
        return {std::move(fr), err};
    }

    x::errors::Error close() override { return config.close_err; }

    void close_send() override {
        if (this->config.read_errors == nullptr)
            this->config
                .read_errors = std::make_shared<std::vector<x::errors::Error>>();
        this->config.read_errors->push_back(freighter::STREAM_CLOSED);
    }
};

// Factory for creating mock Streamers with configurable behavior.
class StreamerFactory final : public pipeline::StreamerFactory {
public:
    // Sequence of errors to return when opening new Streamers.
    // Each call to open_streamer consumes the next error.
    std::vector<x::errors::Error> open_errors;

    // Sequence of configurations for created Streamers.
    // Each new Streamer takes the next config, or the last config if exhausted.
    std::shared_ptr<std::vector<StreamerConfig>> configs;

    // Stores the most recent streamer configuration passed to open_streamer
    synnax::framer::StreamerConfig config;

    // Counts how many times open_streamer has been called.
    // Atomic with release semantics so that readers who observe the incremented
    // value are guaranteed to see the config assignment that precedes it.
    std::atomic<size_t> streamer_opens{0};

    StreamerFactory(
        std::vector<x::errors::Error> open_errors,
        std::shared_ptr<std::vector<StreamerConfig>> configs
    ):
        open_errors(std::move(open_errors)), configs(std::move(configs)) {}

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(const synnax::framer::StreamerConfig config) override {
        this->config = config;
        this->streamer_opens.fetch_add(1, std::memory_order_release);
        const auto opens = this->streamer_opens.load(std::memory_order_relaxed);
        /// try to grab the next config
        size_t idx = opens - 1;
        if (opens > this->configs->size()) idx = this->configs->size() - 1;
        // try to grab the first error. if not, freighter nil
        auto err = opens > this->open_errors.size() ? x::errors::NIL
                                                    : this->open_errors.at(opens - 1);
        if (err) return {nullptr, err};
        return {std::make_unique<Streamer>((*this->configs)[idx]), x::errors::NIL};
    }
};

inline std::shared_ptr<StreamerFactory> simple_streamer_factory(
    const std::vector<synnax::channel::Key> &keys,
    const std::shared_ptr<std::vector<x::telem::Frame>> &reads
) {
    const auto cfg = synnax::framer::StreamerConfig{.channels = keys};
    const auto factory = std::make_shared<StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<StreamerConfig>>(
            std::vector{StreamerConfig{reads, {}, x::errors::NIL}}
        )
    );
    return factory;
}

enum class OpType { Write, SetAuthority };

// Mock implementation of pipeline::Writer for testing.
class Writer final : public pipeline::Writer {
public:
    /// Stores all frames written through this writer.
    std::shared_ptr<std::vector<x::telem::Frame>> writes;

    /// Stores all authority changes forwarded to this writer.
    std::shared_ptr<std::vector<pipeline::Authorities>> authority_changes;

    /// Tracks the order of write() and set_authority() calls.
    std::shared_ptr<std::vector<OpType>> ops;

    /// Error to return when close() is called.
    x::errors::Error close_err;

    /// Index at which write() should return false to simulate failure.
    /// -1 means never return false.
    int return_false_ok_on;

    explicit Writer(
        std::shared_ptr<std::vector<x::telem::Frame>> writes,
        const x::errors::Error &close_err = x::errors::NIL,
        const int return_false_ok_on = -1,
        std::shared_ptr<std::vector<pipeline::Authorities>> authority_changes =
            std::make_shared<std::vector<pipeline::Authorities>>(),
        std::shared_ptr<std::vector<OpType>> ops =
            std::make_shared<std::vector<OpType>>()
    ):
        writes(std::move(writes)),
        authority_changes(std::move(authority_changes)),
        ops(std::move(ops)),
        close_err(close_err),
        return_false_ok_on(return_false_ok_on) {}

    x::errors::Error write(const x::telem::Frame &fr) override {
        this->ops->push_back(OpType::Write);
        if (this->return_false_ok_on != -1 &&
            this->writes->size() == static_cast<size_t>(this->return_false_ok_on))
            return x::errors::VALIDATION;
        this->writes->push_back(fr.deep_copy());
        return x::errors::NIL;
    }

    x::errors::Error set_authority(const pipeline::Authorities &authorities) override {
        this->ops->push_back(OpType::SetAuthority);
        this->authority_changes->push_back(authorities);
        return x::errors::NIL;
    }

    x::errors::Error close() override { return this->close_err; }
};

class WriterFactory final : public pipeline::WriterFactory {
public:
    /// Stores all frames written through this factory's writers.
    std::shared_ptr<std::vector<x::telem::Frame>> writes;

    /// Stores all authority changes forwarded through this factory's writers.
    std::shared_ptr<std::vector<pipeline::Authorities>> authority_changes;

    /// Tracks the order of write() and set_authority() calls across all writers.
    std::shared_ptr<std::vector<OpType>> ops;

    /// A queue of errors to return when opening writers.
    std::vector<x::errors::Error> open_errors;

    /// A queue of errors for writers to return when closed.
    std::vector<x::errors::Error> close_errors;

    /// A queue of indices at which writers should return false for write operations.
    std::vector<int> return_false_ok_on;

    /// Stores the most recent writer configuration passed to open_writer.
    synnax::framer::WriterConfig config;

    /// Counts how many times open_writer has been called.
    /// Atomic with release semantics so that readers who observe the incremented
    /// value are guaranteed to see the config assignment that precedes it.
    std::atomic<size_t> writer_opens;

    explicit WriterFactory(
        std::shared_ptr<std::vector<x::telem::Frame>> writes =
            std::make_shared<std::vector<x::telem::Frame>>(),
        std::vector<x::errors::Error> open_errors = {},
        std::vector<x::errors::Error> close_errors = {},
        std::vector<int> return_false_ok_on = {}
    ):
        writes(std::move(writes)),
        authority_changes(std::make_shared<std::vector<pipeline::Authorities>>()),
        ops(std::make_shared<std::vector<OpType>>()),
        open_errors(std::move(open_errors)),
        close_errors(std::move(close_errors)),
        return_false_ok_on(std::move(return_false_ok_on)),
        config(),
        writer_opens(0) {}

    std::pair<std::unique_ptr<pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &config) override {
        this->config = config;
        this->writer_opens.fetch_add(1, std::memory_order_release);
        auto err = this->open_errors.empty() ? x::errors::NIL
                                             : this->open_errors.front();
        if (!this->open_errors.empty())
            this->open_errors.erase(this->open_errors.begin());
        auto close_err = this->close_errors.empty() ? x::errors::NIL
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
            return_false_ok_on,
            this->authority_changes,
            this->ops
        );
        return {std::move(writer), err};
    }
};

// Mock implementation of pipeline::Sink for testing.
class Sink : public pipeline::Sink {
public:
    // Stores all frames written through this sink
    std::shared_ptr<std::vector<x::telem::Frame>> writes;

    // Sequence of errors to return for write operations
    // Each write consumes the next error in the sequence
    std::shared_ptr<std::vector<x::errors::Error>> write_errors;

    // Stores the error passed to stopped_with_err
    x::errors::Error stop_err;

    Sink():
        writes(std::make_shared<std::vector<x::telem::Frame>>()),
        write_errors(std::make_shared<std::vector<x::errors::Error>>()) {}

    Sink(
        const std::shared_ptr<std::vector<x::telem::Frame>> &writes,
        const std::shared_ptr<std::vector<x::errors::Error>> &write_errors
    ):
        writes(writes), write_errors(write_errors) {}

    x::errors::Error write(x::telem::Frame &frame) override {
        if (frame.empty()) return x::errors::NIL;
        this->writes->emplace_back(frame.deep_copy());
        // try to grab and remove the first error. if not, freighter nil
        if (this->write_errors->empty()) return x::errors::NIL;
        auto err = this->write_errors->front();
        this->write_errors->erase(this->write_errors->begin());
        return err;
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->stop_err = err;
    }
};

// Mock implementation of pipeline::Source for testing.
class Source : public pipeline::Source {
public:
    // A sequence of frames that the Source will return on each read() call.
    // When all frames are consumed, the Source will block briefly and return empty
    // frames.
    std::shared_ptr<std::vector<x::telem::Frame>> reads;

    // A sequence of errors to return alongside frames during read() calls.
    // If provided, each read will return the corresponding error at the same index.
    // If nullptr or index exceeds size, returns NIL error.
    std::shared_ptr<std::vector<x::errors::Error>> read_errors;

    // Stores the error passed to stopped_with_err
    x::errors::Error stop_err;

    // Tracks the current position in the reads sequence
    size_t current_read = 0;

    // Tracks how many times read() has been called
    size_t read_count = 0;

    explicit Source(
        std::shared_ptr<std::vector<x::telem::Frame>> reads =
            std::make_shared<std::vector<x::telem::Frame>>(),
        std::shared_ptr<std::vector<x::errors::Error>> read_errors = nullptr
    ):
        reads(std::move(reads)), read_errors(std::move(read_errors)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        read_count++;
        std::this_thread::sleep_for(std::chrono::milliseconds(1));

        if (current_read >= reads->size()) {
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            return x::errors::NIL;
        }

        const auto &curr_read = reads->at(current_read);
        for (auto [k, s]: curr_read)
            fr.emplace(k, std::move(s));
        auto err = x::errors::NIL;
        if (read_errors != nullptr && read_errors->size() > current_read)
            err = read_errors->at(current_read);
        current_read++;
        return err;
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->stop_err = err;
    }
};

// Helper function to create a simple Source with predefined frames
inline std::shared_ptr<Source>
simple_source(const std::shared_ptr<std::vector<x::telem::Frame>> &reads) {
    return std::make_shared<Source>(reads);
}
}
