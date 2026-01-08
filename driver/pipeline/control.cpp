// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <stdexcept>
#include <utility>

#include "driver/errors/errors.h"
#include "driver/pipeline/control.h"

namespace driver::pipeline {
Control::Control(
    std::shared_ptr<synnax::Synnax> client,
    synnax::StreamerConfig streamer_config,
    std::shared_ptr<driver::pipeline::Sink> sink,
    const x::breaker::Config &breaker_config,
    std::string thread_name
):
    Control(
        std::make_shared<SynnaxStreamerFactory>(std::move(client)),
        streamer_config,
        sink,
        breaker_config,
        std::move(thread_name)
    ) {}

Control::Control(
    std::shared_ptr<StreamerFactory> streamer_factory,
    synnax::StreamerConfig streamer_config,
    std::shared_ptr<Sink> sink,
    const x::breaker::Config &breaker_config,
    std::string thread_name
):
    Base(breaker_config, std::move(thread_name)),
    factory(std::move(streamer_factory)),
    config(std::move(streamer_config)),
    sink(std::move(sink)) {}

bool Control::stop() {
    if (this->streamer != nullptr) this->streamer->close_send();
    const bool was_running = driver::pipeline::Base::stop();
    return was_running;
}

void Control::run() {
    auto [s, open_err] = this->factory->open_streamer(this->config);
    this->streamer = std::move(s);
    if (open_err) {
        if (open_err.matches(freighter::ERR_UNREACHABLE) &&
            breaker.wait(open_err.message()))
            return this->run();
        return this->sink->stopped_with_err(open_err);
    }

    x::errors::Error sink_err = x::errors::NIL;
    while (breaker.running()) {
        auto [cmd_frame, cmd_err] = this->streamer->read();
        if (cmd_err) break;
        if (sink_err = this->sink->write(cmd_frame); sink_err) {
            if (sink_err.matches(driver::TEMPORARY_HARDWARE_ERROR) &&
                breaker.wait(sink_err.message()))
                continue;
            break;
        }
        this->breaker.reset();
    }
    const auto close_err = this->streamer->close();
    if (close_err.matches(freighter::ERR_UNREACHABLE) && breaker.wait()) return this->run();
    if (sink_err)
        this->sink->stopped_with_err(sink_err);
    else if (close_err)
        this->sink->stopped_with_err(close_err);
}

SynnaxStreamer::SynnaxStreamer(synnax::Streamer internal):
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

SynnaxStreamerFactory::SynnaxStreamerFactory(
    const std::shared_ptr<synnax::Synnax> &client
):
    client(std::move(client)) {}

std::pair<std::unique_ptr<driver::pipeline::Streamer>, x::errors::Error>
SynnaxStreamerFactory::open_streamer(synnax::StreamerConfig config) {
    auto [ss, err] = client->telem.open_streamer(config);
    if (err) return {nullptr, err};
    return {std::make_unique<SynnaxStreamer>(std::move(ss)), x::errors::NIL};
}
}
