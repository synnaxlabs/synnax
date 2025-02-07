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
#include "driver/pipeline/control.h"
#include "driver/pipeline/acquisition.h"
#include "freighter/cpp/freighter.h"

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

    void closeSend() override {};
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
        const synnax::StreamerConfig config
    ) override {
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


class MockWriter final : public pipeline::Writer {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    freighter::Error close_err;
    int return_false_ok_on;

    explicit MockWriter(
        std::shared_ptr<std::vector<synnax::Frame> > writes,
        const freighter::Error &close_err = freighter::NIL,
        const int return_false_ok_on = -1
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

