// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <deque>
#include <memory>
#include <mutex>
#include <thread>
#include <variant>

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/pipeline/acquisition.h"

namespace driver::bypass::pipeline {
/// @brief configures a Forwarder.
struct ForwarderConfig {
    /// @brief the most bytes of frames that wait for the Core. A write past it drops
    /// its frame and counts it.
    size_t max_queue_bytes = 64 * 1024 * 1024;
    /// @brief backs off from Core errors. An unreachable Core has no retry limit.
    x::breaker::Config breaker{
        .name = "forwarder",
        .base_interval = x::telem::SECOND,
        .max_retries = 50,
        .scale = 1.1f,
        .max_interval = x::telem::MINUTE,
    };
};

/// @brief delivers a task's frames and authority changes to the Core in order, on its
/// own thread, so a task never waits on the Core. It opens and reopens the Core writer
/// while the Core is unreachable. Any other Core error stops it, and its next write
/// returns that error. All methods are safe to call from one task thread.
class Forwarder {
    using Op = std::variant<x::telem::Frame, ::driver::pipeline::Authorities>;

    std::shared_ptr<::driver::pipeline::WriterFactory> server;
    synnax::framer::WriterConfig config;
    const size_t max_queue_bytes;
    x::breaker::Breaker breaker;

    std::mutex mu;
    std::condition_variable cv;
    std::deque<Op> queue;
    size_t queue_bytes = 0;
    size_t dropped_frames = 0;
    bool closing = false;
    x::errors::Error err = x::errors::NIL;
    std::thread thread;

    static size_t byte_size(const Op &op) {
        const auto *fr = std::get_if<x::telem::Frame>(&op);
        if (fr == nullptr) return 0;
        size_t size = 0;
        for (const auto &s: *fr->series)
            size += s.byte_size();
        return size;
    }

    /// @brief waits for an op, or returns false once closing leaves none.
    bool next(Op &op) {
        std::unique_lock lock(this->mu);
        this->cv.wait(lock, [this] { return this->closing || !this->queue.empty(); });
        if (this->queue.empty()) return false;
        const auto &front = this->queue.front();
        if (const auto *fr = std::get_if<x::telem::Frame>(&front))
            op = fr->shallow_copy();
        else
            op = std::get<::driver::pipeline::Authorities>(front);
        return true;
    }

    void pop() {
        std::lock_guard lock(this->mu);
        this->queue_bytes -= byte_size(this->queue.front());
        this->queue.pop_front();
    }

    [[nodiscard]] bool is_closing() {
        std::lock_guard lock(this->mu);
        return this->closing;
    }

    /// @brief drops every waiting frame and counts it.
    void drop_queue() {
        std::lock_guard lock(this->mu);
        for (const auto &op: this->queue)
            if (std::holds_alternative<x::telem::Frame>(op)) this->dropped_frames++;
        this->queue.clear();
        this->queue_bytes = 0;
    }

    /// @brief backs off after a Core error. Returns false when the forwarder must
    /// stop, and records the error unless the Core was only unreachable.
    bool wait(const x::errors::Error &err) {
        if (synnax::errors::is_unreachable(err)) {
            if (!this->is_closing() && this->breaker.wait_unlimited(err)) return true;
            this->drop_queue();
            return false;
        }
        if (synnax::errors::is_temporarily_unavailable(err) && !this->is_closing() &&
            this->breaker.wait(err))
            return true;
        LOG(ERROR) << "[bypass.forwarder] " << this->config.subject.name
                   << " stopped: " << err;
        std::lock_guard lock(this->mu);
        this->err = err;
        return false;
    }

    /// @brief applies op to writer, and keeps the authorities that a reopened writer
    /// needs.
    x::errors::Error apply(::driver::pipeline::Writer &writer, const Op &op) {
        if (const auto *fr = std::get_if<x::telem::Frame>(&op))
            return writer.write(*fr);
        const auto &auth = std::get<::driver::pipeline::Authorities>(op);
        if (const auto err = writer.set_authority(auth)) return err;
        auth.apply(this->config);
        return x::errors::NIL;
    }

    void run() {
        while (true) {
            auto [writer, open_err] = this->server->open_writer(this->config);
            if (open_err) {
                if (this->wait(open_err)) continue;
                return;
            }
            this->breaker.reset();
            Op op;
            x::errors::Error op_err;
            while (this->next(op)) {
                if (op_err = this->apply(*writer, op); op_err) break;
                this->pop();
            }
            const auto close_err = writer->close();
            const auto err = close_err ? close_err : op_err;
            if (!err) return;
            if (!this->wait(err)) return;
        }
    }

public:
    Forwarder(
        std::shared_ptr<::driver::pipeline::WriterFactory> server,
        synnax::framer::WriterConfig config,
        const ForwarderConfig &cfg = {}
    ):
        server(std::move(server)),
        config(std::move(config)),
        max_queue_bytes(cfg.max_queue_bytes),
        breaker(cfg.breaker) {
        this->breaker.start();
        this->thread = std::thread([this] { this->run(); });
    }

    Forwarder(const Forwarder &) = delete;
    Forwarder &operator=(const Forwarder &) = delete;

    ~Forwarder() { static_cast<void>(this->close()); }

    /// @brief queues fr for the Core. Drops and counts it when the queue is full.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error write(const x::telem::Frame &fr) {
        std::lock_guard lock(this->mu);
        if (this->err) return this->err;
        Op op = fr.shallow_copy();
        const auto size = byte_size(op);
        if (this->queue_bytes + size > this->max_queue_bytes) {
            if (this->dropped_frames++ % 1000 == 0)
                LOG(WARNING) << "[bypass.forwarder] " << this->config.subject.name
                             << " dropped " << this->dropped_frames
                             << " frames: the Core is not taking them fast enough";
            return x::errors::NIL;
        }
        this->queue_bytes += size;
        this->queue.push_back(std::move(op));
        this->cv.notify_one();
        return x::errors::NIL;
    }

    /// @brief queues authorities for the Core, in order with the frames.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error set_authority(const ::driver::pipeline::Authorities &authorities) {
        std::lock_guard lock(this->mu);
        if (this->err) return this->err;
        this->queue.emplace_back(authorities);
        this->cv.notify_one();
        return x::errors::NIL;
    }

    /// @brief sends what the queue holds while the Core is reachable, then closes the
    /// Core writer. Drops and counts what is left when the Core is unreachable.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error close() {
        {
            std::lock_guard lock(this->mu);
            this->closing = true;
        }
        this->cv.notify_one();
        this->breaker.stop();
        if (this->thread.joinable()) this->thread.join();
        std::lock_guard lock(this->mu);
        return this->err;
    }

    /// @brief the number of frames dropped since the forwarder opened.
    [[nodiscard]] size_t dropped() {
        std::lock_guard lock(this->mu);
        return this->dropped_frames;
    }
};
}
