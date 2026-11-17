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
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"

#include "driver/pipeline/acquisition.h"

namespace driver::bypass::pipeline {
/// @brief an estimate of the bookkeeping bytes that one heap allocation costs.
constexpr size_t ALLOCATION_OVERHEAD = 32;
/// @brief the bytes a queued frame holds in addition to its series.
constexpr size_t FRAME_OVERHEAD = sizeof(x::telem::Frame) +
                                  sizeof(std::vector<synnax::channel::Key>) +
                                  sizeof(std::vector<x::telem::Series>) +
                                  4 * ALLOCATION_OVERHEAD;
/// @brief the bytes a queued series holds in addition to its samples.
constexpr size_t SERIES_OVERHEAD = sizeof(x::telem::Series) +
                                   sizeof(synnax::channel::Key) +
                                   2 * ALLOCATION_OVERHEAD;
/// @brief the bytes a queued authority change holds in addition to its keys and
/// authorities.
constexpr size_t AUTHORITIES_OVERHEAD = sizeof(::driver::pipeline::Authorities) +
                                        2 * ALLOCATION_OVERHEAD;

/// @brief configures a Forwarder.
struct ForwarderConfig {
    /// @brief the most bytes that queued frames and authority changes hold, overhead
    /// included. A write past it drops its frame and counts it.
    size_t max_queue_bytes = size_t{64} * 1024 * 1024;
    /// @brief backs off from an unreachable Core, with no retry limit.
    x::breaker::Config breaker{
        .name = "forwarder",
        .base_interval = x::telem::SECOND,
        .scale = 1.1f,
        .max_interval = x::telem::MINUTE,
    };
};

/// @brief delivers a task's frames and authority changes to the Core in order, on its
/// own thread, so a task never waits on the Core. It opens and reopens the Core writer
/// while the Core is unreachable. Any other Core error stops it, drops and counts what
/// it holds, and its next write returns that error. All methods are safe to call from
/// one task thread.
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
    bool settled = false;
    std::condition_variable settled_cv;
    std::thread thread;

    /// @brief ends the wait in opened() with the result of the first open.
    void settle() {
        {
            std::lock_guard lock(this->mu);
            if (this->settled) return;
            this->settled = true;
        }
        this->settled_cv.notify_all();
    }

    static size_t byte_size(const Op &op) {
        const auto *fr = std::get_if<x::telem::Frame>(&op);
        if (fr == nullptr) {
            const auto &auth = std::get<::driver::pipeline::Authorities>(op);
            return AUTHORITIES_OVERHEAD +
                   auth.keys.size() * sizeof(synnax::channel::Key) +
                   auth.authorities.size() * sizeof(x::control::Authority);
        }
        size_t size = FRAME_OVERHEAD;
        for (const auto &s: *fr->series)
            size += SERIES_OVERHEAD + s.byte_size();
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

    /// @brief backs off after an unreachable Core. Any other Core error, a license
    /// error included, stops the forwarder so its owner retries it. Returns false when
    /// the forwarder must stop.
    bool wait(const x::errors::Error &err) {
        if (synnax::errors::is_unreachable(err)) {
            this->settle();
            if (!this->is_closing() && this->breaker.wait_unlimited(err)) return true;
            this->drop_queue();
            return false;
        }
        LOG(ERROR) << "[bypass.forwarder] " << this->config.subject.name
                   << " stopped: " << err;
        {
            std::lock_guard lock(this->mu);
            this->err = err;
        }
        this->drop_queue();
        this->settle();
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
        x::defer::defer settle([this] { this->settle(); });
        while (true) {
            auto [writer, open_err] = this->server->open_writer(this->config);
            if (open_err) {
                if (this->wait(open_err)) continue;
                return;
            }
            this->settle();
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

    /// @brief waits for the first open of the Core writer.
    /// @returns the error that stopped the forwarder before its first open, or NIL once
    /// a writer opens or the Core proves unreachable. It never waits through retries.
    x::errors::Error opened() {
        std::unique_lock lock(this->mu);
        this->settled_cv.wait(lock, [this] { return this->settled; });
        return this->err;
    }

    /// @brief queues fr for the Core. Drops and counts it when the queue is full. A
    /// frame larger than the cap goes through when the queue is empty.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error write(const x::telem::Frame &fr) {
        std::lock_guard lock(this->mu);
        if (this->err) return this->err;
        Op op = fr.shallow_copy();
        const auto size = byte_size(op);
        if (this->queue_bytes > 0 && this->queue_bytes + size > this->max_queue_bytes) {
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

    /// @brief queues authorities for the Core, in order with the frames. It never drops
    /// them, but they count toward the queue cap.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error set_authority(const ::driver::pipeline::Authorities &authorities) {
        std::lock_guard lock(this->mu);
        if (this->err) return this->err;
        Op op = authorities;
        this->queue_bytes += byte_size(op);
        this->queue.push_back(std::move(op));
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
