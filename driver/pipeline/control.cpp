// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/control.h"

namespace driver::pipeline {
Control::Control(
    const std::shared_ptr<synnax::Synnax> &client,
    const synnax::framer::StreamerConfig &streamer_config,
    const std::shared_ptr<Sink> &sink,
    const x::breaker::Config &breaker_config,
    std::string thread_name
):
    Control(
        std::make_shared<SynnaxStreamerFactory>(client),
        streamer_config,
        sink,
        breaker_config,
        std::move(thread_name)
    ) {}

Control::Control(
    std::shared_ptr<StreamerFactory> streamer_factory,
    synnax::framer::StreamerConfig streamer_config,
    std::shared_ptr<Sink> sink,
    const x::breaker::Config &breaker_config,
    std::string thread_name
):
    Base(breaker_config, std::move(thread_name)),
    factory(std::move(streamer_factory)),
    config(std::move(streamer_config)),
    sink(std::move(sink)) {
    this->config.on_missed = [this] {
        LOG(WARNING) << "[control] the Core dropped commands because the pipeline "
                        "fell behind";
        this->sink->missed();
    };
}

void Control::interrupt() {
    std::lock_guard lock{this->streamer_mu};
    if (this->streamer != nullptr) this->streamer->close_send();
}

void Control::run() {
    while (this->run_once()) {}
}

void Control::stopped_with_err(const x::errors::Error &err) {
    this->sink->stopped_with_err(err);
}

x::errors::Error
Control::set_channels(const std::vector<synnax::channel::Key> &channels) {
    std::lock_guard lock{this->streamer_mu};
    if (this->streamer != nullptr)
        if (const auto err = this->streamer->set_channels(channels)) return err;
    this->config.channels = channels;
    return x::errors::NIL;
}

bool Control::run_once() {
    synnax::framer::StreamerConfig config;
    {
        std::lock_guard lock{this->streamer_mu};
        config = this->config;
    }
    auto [s, open_err] = this->factory->open_streamer(config);
    {
        std::lock_guard lock{this->streamer_mu};
        this->streamer = std::move(s);
        // A set_channels that ran during the open missed this streamer. A streamer
        // that rejects the channels fails its next read, which ends this run.
        if (this->streamer != nullptr && this->config.channels != config.channels)
            static_cast<void>(this->streamer->set_channels(this->config.channels));
    }
    if (open_err) {
        if (this->wait_for_core(open_err)) return true;
        this->sink->stopped_with_err(open_err);
        return false;
    }
    this->breaker.reset_unlimited();
    this->sink->opened();

    x::errors::Error sink_err = x::errors::NIL;
    while (breaker.running()) {
        auto [cmd_frame, cmd_err] = this->streamer->read();
        if (cmd_err) break;
        if (sink_err = this->sink->write(cmd_frame); sink_err) {
            if (sink_err.matches(errors::TEMPORARY_HARDWARE_ERROR) &&
                breaker.wait(sink_err.message()))
                continue;
            break;
        }
        this->breaker.reset();
    }
    x::errors::Error close_err;
    {
        std::lock_guard lock{this->streamer_mu};
        close_err = this->streamer->close();
    }
    if (this->wait_for_core(close_err)) return true;
    if (sink_err)
        this->sink->stopped_with_err(sink_err);
    else if (close_err)
        this->sink->stopped_with_err(close_err);
    return false;
}

SynnaxStreamer::SynnaxStreamer(synnax::framer::Streamer internal):
    internal(std::move(internal)) {}

std::pair<x::telem::Frame, x::errors::Error> SynnaxStreamer::read() {
    return this->internal.read();
}

x::errors::Error SynnaxStreamer::close() {
    return this->internal.close();
}

void SynnaxStreamer::close_send() {
    this->internal.close_send();
}

x::errors::Error
SynnaxStreamer::set_channels(const std::vector<synnax::channel::Key> &channels) {
    return this->internal.set_channels(channels);
}

SynnaxStreamerFactory::SynnaxStreamerFactory(
    const std::shared_ptr<synnax::Synnax> &client
):
    client(client) {}

std::pair<std::unique_ptr<Streamer>, x::errors::Error>
SynnaxStreamerFactory::open_streamer(synnax::framer::StreamerConfig config) {
    auto [ss, err] = client->telem.open_streamer(config);
    if (err) return {nullptr, err};
    return {std::make_unique<SynnaxStreamer>(std::move(ss)), x::errors::NIL};
}
}
