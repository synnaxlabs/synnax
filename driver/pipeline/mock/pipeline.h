// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "client/cpp/synnax.h"
#include "driver/pipeline/control.h"
#include "driver/pipeline/acquisition.h"
#include "freighter/cpp/freighter.h"

namespace pipeline::mock {
struct StreamerConfig {
    std::shared_ptr<std::vector<synnax::Frame> > reads;
    std::shared_ptr<std::vector<xerrors::Error> > read_errors;
    xerrors::Error close_err;
};

class Streamer final : public pipeline::Streamer {
public:
    StreamerConfig config;
    size_t current_read = 0;

    explicit Streamer(
        StreamerConfig config
    ) : config(std::move(config)) {
    }

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

    void close_send() override {
    };
};

class StreamerFactory final : public pipeline::StreamerFactory {
public:
    std::vector<xerrors::Error> open_errors;
    std::shared_ptr<std::vector<StreamerConfig> > configs;
    synnax::StreamerConfig config;
    size_t streamer_opens = 0;

    StreamerFactory(
        std::vector<xerrors::Error> open_errors,
        std::shared_ptr<std::vector<StreamerConfig> > configs
    ) : open_errors(std::move(open_errors)), configs(std::move(configs)) {
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, xerrors::Error> open_streamer(
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
                       ? xerrors::NIL
                       : this->open_errors.at(this->streamer_opens - 1);
        if (err) return {nullptr, err};
        return {
            std::make_unique<Streamer>((*this->configs)[idx]),
            xerrors::NIL
        };
    }
};

inline std::shared_ptr<pipeline::StreamerFactory> simple_streamer_factory(
    const std::vector<synnax::ChannelKey> &keys,
    const std::shared_ptr<std::vector<synnax::Frame> > &reads
) {
    const auto cfg = synnax::StreamerConfig{.channels = keys};
    const auto factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<xerrors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig> >(
            std::vector{pipeline::mock::StreamerConfig{reads, {}, xerrors::NIL}})
    );
    return factory;
}


class Writer final : public pipeline::Writer {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    xerrors::Error close_err;
    int return_false_ok_on;

    explicit Writer(
        std::shared_ptr<std::vector<synnax::Frame> > writes,
        const xerrors::Error &close_err = xerrors::NIL,
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

    xerrors::Error close() override {
        return this->close_err;
    }
};

class WriterFactory final : public pipeline::WriterFactory {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    std::vector<xerrors::Error> open_errors;
    std::vector<xerrors::Error> close_errors;
    std::vector<int> return_false_ok_on;
    WriterConfig config;
    size_t writer_opens;

    explicit WriterFactory(
        std::shared_ptr<std::vector<synnax::Frame> > writes,
        std::vector<xerrors::Error> open_errors = {},
        std::vector<xerrors::Error> close_errors = {},
        std::vector<int> return_false_ok_on = {}
    ) : writes(
            std::move(writes)), open_errors(std::move(open_errors)),
        close_errors(std::move(close_errors)),
        return_false_ok_on(std::move(return_false_ok_on)),
        config(),
        writer_opens(0) {
    }

    std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error> open_writer(
        const WriterConfig &config) override {
        this->writer_opens++;
        this->config = config;
        auto err = this->open_errors.empty()
                       ? xerrors::NIL
                       : this->open_errors.front();
        if (!this->open_errors.empty())
            this->open_errors.erase(
                this->open_errors.begin());
        auto close_err = this->close_errors.empty()
                             ? xerrors::NIL
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
        auto writer = std::make_unique<Writer>(
            this->writes, close_err, return_false_ok_on);
        return {std::move(writer), err};
    }
};

class Sink : public pipeline::Sink {
public:
    std::shared_ptr<std::vector<synnax::Frame> > writes;
    std::shared_ptr<std::vector<xerrors::Error> > write_errors;
    xerrors::Error stop_err;

    Sink() : writes(std::make_shared<std::vector<synnax::Frame> >()),
             write_errors(std::make_shared<std::vector<xerrors::Error> >()) {
    }

    xerrors::Error write(const synnax::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        this->writes->emplace_back(frame.deep_copy());
        // try to grab and remove the first error. if not, freighter nil
        if (this->write_errors->empty()) return xerrors::NIL;
        auto err = this->write_errors->front();
        this->write_errors->erase(this->write_errors->begin());
        return err;
    }

    void stopped_with_err(const xerrors::Error &err) override {
        this->stop_err = err;
    }
};
}
