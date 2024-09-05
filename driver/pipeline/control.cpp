// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "driver/pipeline/control.h"

#include <utility>
#include "driver/errors/errors.h"
#include <exception>
#include <stdexcept>

using namespace pipeline;

class SynnaxStreamer final : public pipeline::Streamer {
    std::unique_ptr<synnax::Streamer> internal;

public:
    explicit SynnaxStreamer(
        std::unique_ptr<synnax::Streamer> internal
    ) : internal(std::move(internal)) {
    }

    std::pair<synnax::Frame, freighter::Error> read() override {
        return this->internal->read();
    }

    freighter::Error close() override {
        return this->internal->close();
    }

    void closeSend() override {
        this->internal->closeSend();
    }
};

class SynnaxStreamerFactory final : public StreamerFactory {
    std::shared_ptr<synnax::Synnax> client;

public:
    explicit SynnaxStreamerFactory(
        std::shared_ptr<synnax::Synnax> client
    ) : client(std::move(client)) {
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, freighter::Error> openStreamer(
        synnax::StreamerConfig config) override {
        auto [ss, err] = client->telem.openStreamer(config);
        if (err) return {nullptr, err};
        return {
            std::make_unique<SynnaxStreamer>(
                std::make_unique<synnax::Streamer>(std::move(ss))),
            freighter::NIL
        };
    }
};

Control::Control(
    std::shared_ptr<synnax::Synnax> client,
    synnax::StreamerConfig streamer_config,
    std::shared_ptr<pipeline::Sink> sink,
    const breaker::Config &breaker_config
) : thread(nullptr),
    factory(std::make_shared<SynnaxStreamerFactory>(std::move(client))),
    config(std::move(streamer_config)),
    sink(std::move(sink)),
    breaker(breaker::Breaker(breaker_config)) {
}

Control::Control(
    std::shared_ptr<StreamerFactory> streamer_factory,
    synnax::StreamerConfig streamer_config,
    std::shared_ptr<Sink> sink,
    const breaker::Config &breaker_config
) : thread(nullptr),
    factory(std::move(streamer_factory)),
    config(std::move(streamer_config)),
    sink(std::move(sink)),
    breaker(breaker::Breaker(breaker_config)) {
}


void Control::ensureThreadJoined() const {
    if (
        this->thread == nullptr ||
        !this->thread->joinable() ||
        std::this_thread::get_id() == this->thread->get_id()
    )
        return;
    this->thread->join();
}


void Control::start() {
    if (this->breaker.running()) return;
    this->ensureThreadJoined();
    this->breaker.start();
    this->thread = std::make_unique<std::thread>(&Control::run, this);
    LOG(INFO) << "[control] started";
}

void Control::stop() {
    const auto was_running = this->breaker.running();
    // Stop the breaker and join the thread regardless of whether it was running.
    // This ensures that the thread gets joined even in the case of an internal error.
    if (this->streamer) this->streamer->closeSend();
    this->breaker.stop();
    this->ensureThreadJoined();
    if (was_running) LOG(INFO) << "[control] stopped";
}

void Control::run() {
    try {
        this->runInternal();
    } catch (const std::exception &e) {
        LOG(ERROR) << "[control] Unhandled standard exception: " << e.what();
    } catch (...) {
        LOG(ERROR) << "[control] Unhandled unknown exception";
    }
    this->stop();
}

void Control::runInternal() {
    auto [s, open_err] = this->factory->openStreamer(this->config);
    this->streamer = std::move(s);
    if (open_err) {
        if (
            open_err.matches(freighter::UNREACHABLE)
            && breaker.wait(open_err.message())
        )
            return runInternal();
        return this->sink->stopped_with_err(open_err);
    }

    while (breaker.running()) {
        auto [cmd_frame, cmd_err] = this->streamer->read();
        if (cmd_err) break;
        const auto sink_err = this->sink->write(std::move(cmd_frame));
        if (sink_err) {
            if (
                sink_err.matches(driver::TEMPORARY_HARDWARE_ERROR)
                && breaker.wait(sink_err.message())
            )
                continue;
            break;
        }
        this->breaker.reset();
    }
    const auto close_err = this->streamer->close();
    if (
        close_err.matches(freighter::UNREACHABLE)
        && breaker.wait()
    )
        return runInternal();
    if (close_err) this->sink->stopped_with_err(close_err);
}
