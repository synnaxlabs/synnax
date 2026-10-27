// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "client/cpp/errors/errors.h"

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
    sink(std::move(sink)) {}

x::errors::Error Control::start() {
    if (this->running()) return x::errors::NIL;
    this->open_result.emplace();
    auto opened = this->open_result->get_future();
    Base::start();
    const auto err = opened.get();
    if (err) Base::stop();
    return err;
}

bool Control::resolve_open(const x::errors::Error &err) {
    if (!this->open_result.has_value()) return false;
    this->open_result->set_value(err);
    this->open_result.reset();
    return true;
}

bool Control::stop() {
    if (this->streamer != nullptr) this->streamer->close_send();
    const bool was_running = Base::stop();
    return was_running;
}

void Control::run() {
    auto [s, open_err] = this->factory->open_streamer(this->config);
    this->streamer = std::move(s);
    if (open_err) {
        if (synnax::errors::is_temporarily_unavailable(open_err)) {
            this->resolve_open(x::errors::NIL);
            if (breaker.wait(open_err.message())) {
                this->run();
                return;
            }
        }
        if (!this->resolve_open(open_err)) this->sink->stopped_with_err(open_err);
        return;
    }
    this->sink->opened();
    this->resolve_open(x::errors::NIL);

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
    const auto close_err = this->streamer->close();
    if (synnax::errors::is_temporarily_unavailable(close_err) && breaker.wait()) {
        this->run();
        return;
    }
    if (sink_err)
        this->sink->stopped_with_err(sink_err);
    else if (close_err)
        this->sink->stopped_with_err(close_err);
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
