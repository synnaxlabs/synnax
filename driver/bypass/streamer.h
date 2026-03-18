// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <thread>

#include "driver/bypass/bypass.h"
#include "driver/pipeline/control.h"

namespace driver::bypass {
/// @brief a pipeline Streamer that merges local bus frames with server frames.
/// Runs the server read on a background thread so that both local bus frames
/// and server frames are delivered without blocking each other. No authority
/// filtering is applied here; the bypass Writer is responsible for filtering
/// unauthorized channels before publishing to the bus (matching Cesium's
/// behavior of stripping unauthorized channels before relaying).
class Streamer final : public pipeline::Streamer {
    std::unique_ptr<pipeline::Streamer> server;
    std::shared_ptr<Subscription> subscription;
    std::thread server_thread;

    std::mutex server_mu;
    std::deque<x::telem::Frame> server_frames;
    bool server_done = false;
    std::atomic<bool> closed{false};
    x::errors::Error server_err{x::errors::NIL};

    std::mutex notify_mu;
    std::condition_variable notify_cv;

public:
    Streamer(
        std::unique_ptr<pipeline::Streamer> server,
        std::shared_ptr<Subscription> subscription
    ):
        server(std::move(server)), subscription(std::move(subscription)) {
        this->subscription->set_on_push([this] { this->notify_cv.notify_one(); });
        this->server_thread = std::thread([this] { this->read_server(); });
    }

    ~Streamer() override { this->close(); }

    std::pair<x::telem::Frame, x::errors::Error> read() override {
        while (true) {
            x::telem::Frame local;
            while (this->subscription->try_pop(local)) {
                if (!local.empty()) {
                    VLOG(1) << "[bus.streamer] delivering local frame with "
                            << local.size() << " channels (bypassed server)";
                    return {std::move(local), x::errors::NIL};
                }
            }
            {
                std::lock_guard lock(this->server_mu);
                if (!this->server_frames.empty()) {
                    auto frame = std::move(this->server_frames.front());
                    this->server_frames.pop_front();
                    VLOG(1) << "[bus.streamer] delivering server frame with "
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
        auto err = this->server->close();
        if (this->server_thread.joinable()) this->server_thread.join();
        return err;
    }

    void close_send() override { this->server->close_send(); }

private:
    void read_server() {
        while (true) {
            auto [frame, err] = this->server->read();
            if (err) {
                {
                    std::lock_guard lock(this->server_mu);
                    this->server_err = err;
                    this->server_done = true;
                }
                this->notify_cv.notify_one();
                return;
            }
            if (!frame.empty()) {
                {
                    std::lock_guard lock(this->server_mu);
                    this->server_frames.push_back(std::move(frame));
                }
                this->notify_cv.notify_one();
            }
        }
    }
};

/// @brief a StreamerFactory that wraps streamers with bus subscription capability.
/// Injects the subject's group into exclude_groups for server-side deduplication.
class StreamerFactory final : public pipeline::StreamerFactory {
    std::shared_ptr<pipeline::StreamerFactory> server;
    Bus &bus;
    x::control::Subject subject;

public:
    StreamerFactory(
        std::shared_ptr<pipeline::StreamerFactory> server,
        Bus &bus,
        x::control::Subject subject
    ):
        server(std::move(server)), bus(bus), subject(std::move(subject)) {}

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        if (this->subject.group != 0)
            config.exclude_groups.push_back(this->subject.group);
        auto [streamer, err] = this->server->open_streamer(config);
        if (err) return {nullptr, err};
        auto subscription = this->bus.subscribe(config.channels);
        VLOG(1) << "[bus.streamer_factory] opened streamer for "
                << config.channels.size() << " channels, subject=" << this->subject.name
                << ", exclude_groups=" << this->subject.group;
        return {
            std::make_unique<Streamer>(std::move(streamer), std::move(subscription)),
            x::errors::NIL,
        };
    }
};
}
