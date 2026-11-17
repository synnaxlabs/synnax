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
#include <condition_variable>
#include <deque>
#include <memory>
#include <mutex>
#include <thread>

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/bypass/bypass.h"
#include "driver/pipeline/control.h"

namespace driver::bypass::pipeline {
/// @brief a pipeline Streamer that merges local bus frames with Core frames. The Core
/// half reads on a background thread, so bus frames never wait for the Core. No
/// authority filtering is applied here; the bypass Writer filters unauthorized channels
/// before it publishes to the bus.
///
/// While the Core is unreachable, the Core half reopens with backoff and no retry
/// limit, and read() keeps returning bus frames. Any other Core error ends read() with
/// that error.
class Streamer final : public ::driver::pipeline::Streamer {
    std::shared_ptr<::driver::pipeline::StreamerFactory> factory;
    synnax::framer::StreamerConfig config;
    std::shared_ptr<Subscription> subscription;
    std::shared_ptr<Bus> bus;
    x::breaker::Breaker breaker;
    std::thread core_thread;

    std::mutex core_mu;
    /// @brief the open Core streamer, if any. Guarded by core_mu, and changed only on
    /// core_thread after construction.
    std::unique_ptr<::driver::pipeline::Streamer> core;

    std::mutex server_mu;
    std::deque<x::telem::Frame> server_frames;
    bool server_done = false;
    std::atomic<bool> closed{false};
    x::errors::Error server_err{x::errors::NIL};

    std::mutex notify_mu;
    std::condition_variable notify_cv;

public:
    /// @param core the Core streamer, or nullptr if its first open failed because the
    /// Core is unreachable.
    /// @param open_err the error of the first open, or NIL.
    Streamer(
        std::shared_ptr<::driver::pipeline::StreamerFactory> factory,
        synnax::framer::StreamerConfig config,
        std::unique_ptr<::driver::pipeline::Streamer> core,
        const x::errors::Error &open_err,
        std::shared_ptr<Subscription> subscription,
        std::shared_ptr<Bus> bus,
        const x::breaker::Config &breaker_config
    ):
        factory(std::move(factory)),
        config(std::move(config)),
        subscription(std::move(subscription)),
        bus(std::move(bus)),
        breaker(breaker_config),
        core(std::move(core)) {
        this->breaker.start();
        this->subscription->set_on_push([this] { this->notify_cv.notify_one(); });
        this->core_thread = std::thread([this, open_err] { this->run_core(open_err); });
    }

    ~Streamer() override { this->close(); }

    std::pair<x::telem::Frame, x::errors::Error> read() override {
        while (true) {
            x::telem::Frame local;
            while (this->subscription->try_pop(local)) {
                if (!local.empty()) {
                    VLOG(1) << "[bus.streamer] delivering local frame with "
                            << local.size() << " channels (bypassed Core)";
                    return {std::move(local), x::errors::NIL};
                }
            }
            {
                std::lock_guard lock(this->server_mu);
                if (!this->server_frames.empty()) {
                    auto frame = std::move(this->server_frames.front());
                    this->server_frames.pop_front();
                    VLOG(1) << "[bus.streamer] delivering Core frame with "
                            << frame.size() << " channels";
                    return {std::move(frame), x::errors::NIL};
                }
                if (this->server_done) return {x::telem::Frame{}, this->server_err};
            }
            std::unique_lock lock(this->notify_mu);
            this->notify_cv.wait_for(lock, std::chrono::milliseconds(5), [this] {
                if (!this->subscription->empty()) return true;
                std::lock_guard sg(this->server_mu);
                return !this->server_frames.empty() || this->server_done;
            });
        }
    }

    x::errors::Error close() override {
        bool expected = false;
        if (!this->closed.compare_exchange_strong(expected, true))
            return x::errors::NIL;
        this->subscription->set_on_push(nullptr);
        this->close_send();
        if (this->core_thread.joinable()) this->core_thread.join();
        return this->server_err.skip({freighter::ERR_EOF, freighter::STREAM_CLOSED});
    }

    void close_send() override {
        std::lock_guard lock(this->core_mu);
        this->breaker.stop();
        if (this->core != nullptr) this->core->close_send();
    }

private:
    void run_core(x::errors::Error err) {
        if (this->core != nullptr) err = this->read_core();
        while (synnax::errors::is_unreachable(err) && this->breaker.running() &&
               this->breaker.wait_unlimited(err))
            err = this->stream_core();
        // A close_send() that ends the wait for the Core is not a Core failure.
        if (!this->breaker.running() && synnax::errors::is_unreachable(err))
            err = freighter::STREAM_CLOSED;
        {
            std::lock_guard lock(this->server_mu);
            this->server_err = err;
            this->server_done = true;
        }
        this->notify_cv.notify_one();
    }

    /// @returns the error that ended this session with the Core.
    x::errors::Error stream_core() {
        auto [opened, open_err] = this->factory->open_streamer(this->config);
        if (open_err) return open_err;
        {
            std::lock_guard lock(this->core_mu);
            this->core = std::move(opened);
            if (!this->breaker.running()) this->core->close_send();
        }
        return this->read_core();
    }

    /// @returns the error that ended the read of the open Core streamer.
    x::errors::Error read_core() {
        this->breaker.reset_unlimited();
        auto err = x::errors::NIL;
        while (!err) {
            auto [frame, read_err] = this->core->read();
            err = read_err;
            if (err || frame.empty()) continue;
            this->bus->advance_alignments(frame);
            {
                std::lock_guard lock(this->server_mu);
                this->server_frames.push_back(std::move(frame));
            }
            this->notify_cv.notify_one();
        }
        std::unique_ptr<::driver::pipeline::Streamer> done;
        {
            std::lock_guard lock(this->core_mu);
            done = std::move(this->core);
        }
        if (const auto close_err = done->close(); close_err) return close_err;
        return err;
    }
};

/// @brief a StreamerFactory that wraps streamers with bus subscription capability.
/// Injects the subject's group into exclude_groups for Core-side deduplication.
class StreamerFactory final : public ::driver::pipeline::StreamerFactory {
    std::shared_ptr<::driver::pipeline::StreamerFactory> server;
    std::shared_ptr<Bus> bus;
    x::control::Subject subject;
    x::breaker::Config breaker_config;

public:
    /// @param breaker_config the backoff of each streamer's Core half between reopens.
    StreamerFactory(
        std::shared_ptr<::driver::pipeline::StreamerFactory> server,
        const std::shared_ptr<Bus> &bus,
        x::control::Subject subject,
        x::breaker::Config breaker_config
    ):
        server(std::move(server)),
        bus(bus),
        subject(std::move(subject)),
        breaker_config(std::move(breaker_config)) {}

    /// @brief opens the Core streamer and subscribes to the bus. If the Core is
    /// unreachable, returns the streamer anyway, and the streamer reopens its Core half
    /// in the background. Sink::opened() then means only that bus frames reach the
    /// sink. Core frames reach it once the Core half opens.
    /// @returns any Core open error other than an unreachable Core.
    std::pair<std::unique_ptr<::driver::pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        if (this->subject.group != 0)
            config.exclude_groups.push_back(this->subject.group);
        auto [core, open_err] = this->server->open_streamer(config);
        if (open_err && !synnax::errors::is_unreachable(open_err))
            return {nullptr, open_err};
        auto subscription = this->bus->subscribe(config.channels);
        VLOG(1) << "[bus.streamer_factory] opened streamer for "
                << config.channels.size() << " channels, subject=" << this->subject.name
                << ", exclude_groups=" << this->subject.group;
        return {
            std::make_unique<Streamer>(
                this->server,
                std::move(config),
                std::move(core),
                open_err,
                std::move(subscription),
                this->bus,
                this->breaker_config
            ),
            x::errors::NIL,
        };
    }
};
}
