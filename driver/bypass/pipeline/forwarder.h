// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <deque>
#include <functional>
#include <iterator>
#include <limits>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <variant>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/thread/thread.h"

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

/// @brief shows a warning on a task status, or stops showing it when message is empty.
using Warn = std::function<
    void(const std::string &message, const std::string &description)>;

/// @brief counts the frames that a Forwarder dropped, by cause.
struct Drops {
    /// @brief frames dropped because the queue was full.
    size_t full = 0;
    /// @brief frames still queued when the task stopped.
    size_t stopped = 0;
    /// @brief frames still queued when a Core error stopped the forwarder.
    size_t failed = 0;

    /// @returns the frames dropped for any cause.
    [[nodiscard]] size_t total() const {
        return this->full + this->stopped + this->failed;
    }
};

/// @brief configures a Forwarder.
struct ForwarderConfig {
    /// @brief the most bytes that queued frames and authority changes hold, overhead
    /// included. A write past it drops its frame, the newest one, and counts it.
    size_t max_queue_bytes = size_t{64} * 1024 * 1024;
    /// @brief the shortest time between two reports of a new drop count. While the Core
    /// is unreachable, the forwarder also reports at this interval.
    x::telem::TimeSpan drop_report_interval = 5 * x::telem::SECOND;
    /// @brief the longest time that close() sends queued ops to a reachable Core. It
    /// drops and counts what is left after it.
    x::telem::TimeSpan close_timeout = 5 * x::telem::SECOND;
    /// @brief the frames that earlier forwarders of the same run dropped. The drop
    /// counts start from them.
    Drops dropped;
    /// @brief warns when the queue passes half its cap, until it drains below a
    /// quarter. Once a frame drops, it reports the drop count instead, and leaves the
    /// final count when the forwarder closes. Called from the forwarder thread only.
    Warn warn;
    /// @brief the backoff from an unreachable Core. The forwarder retries with no
    /// limit, so only the intervals apply.
    x::breaker::Config breaker{
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

    struct Entry {
        Op op;
        size_t bytes;
    };

    std::shared_ptr<::driver::pipeline::WriterFactory> server;
    synnax::framer::WriterConfig config;
    const size_t max_queue_bytes;
    const std::chrono::nanoseconds drop_report_interval;
    const std::chrono::nanoseconds close_timeout;
    const Warn warn;
    const x::breaker::Config backoff;

    // Only the forwarder thread touches these.
    size_t failures = 0;
    bool lagging = false;
    std::string shown_message;
    std::string shown_description;
    std::chrono::steady_clock::time_point drops_shown_at;

    /// @brief the entries that the forwarder thread took from the queue and sends.
    std::deque<Entry> batch;

    std::mutex mu;
    std::condition_variable cv;
    std::deque<Entry> queue;
    /// @brief the bytes that queue and batch hold. Changed only under mu, except that
    /// the forwarder thread subtracts each op it sends.
    std::atomic<size_t> queue_bytes = 0;
    /// @brief true while the forwarder thread waits for the queue.
    bool idle = false;
    Drops drops;
    /// @brief drops.total(), which report() reads without mu.
    std::atomic<size_t> dropped_total = 0;
    /// @brief the frames this forwarder dropped at the cap, for its log.
    size_t cap_drops = 0;
    bool closing = false;
    /// @brief the steady clock time at which close() stops sending, in nanoseconds.
    std::atomic<int64_t> close_deadline{std::numeric_limits<int64_t>::max()};
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

    static size_t frame_bytes(const x::telem::Frame &fr) {
        size_t size = FRAME_OVERHEAD;
        for (const auto &s: *fr.series)
            size += SERIES_OVERHEAD + s.byte_size();
        return size;
    }

    static size_t authorities_bytes(const ::driver::pipeline::Authorities &auth) {
        return AUTHORITIES_OVERHEAD + auth.keys.size() * sizeof(synnax::channel::Key) +
               auth.authorities.size() * sizeof(x::control::Authority);
    }

    /// @brief queues entry. Requires mu. Returns true when the forwarder thread waits
    /// for it.
    bool push_locked(Entry entry) {
        this->queue_bytes.fetch_add(entry.bytes, std::memory_order_relaxed);
        this->queue.push_back(std::move(entry));
        return this->idle;
    }

    [[nodiscard]] bool expired() const {
        return std::chrono::steady_clock::now().time_since_epoch().count() >=
               this->close_deadline.load(std::memory_order_relaxed);
    }

    void show(const std::string &message, const std::string &description) {
        if (message == this->shown_message && description == this->shown_description)
            return;
        this->shown_message = message;
        this->shown_description = description;
        this->warn(message, description);
    }

    void show_drops(const Drops &dropped) {
        this->drops_shown_at = std::chrono::steady_clock::now();
        std::string description;
        const auto add = [&description](const size_t count, const std::string &cause) {
            if (count == 0) return;
            if (!description.empty()) description += " ";
            description += "Dropped " + std::to_string(count) +
                           (count == 1 ? " frame" : " frames") + " because " + cause +
                           ".";
        };
        add(dropped.full, "the queue for the Core was full");
        add(dropped.stopped, "the task stopped before delivery to the Core");
        add(dropped.failed, "a Core error stopped delivery");
        this->show("Dropped frames", description);
    }

    /// @brief shows on the task status whether the queue lags or drops frames.
    void report() {
        if (!this->warn) return;
        const auto bytes = this->queue_bytes.load(std::memory_order_relaxed);
        if (bytes * 2 >= this->max_queue_bytes)
            this->lagging = true;
        else if (bytes * 4 < this->max_queue_bytes)
            this->lagging = false;
        if (this->dropped_total.load(std::memory_order_relaxed) > 0) {
            if (this->shown_message != "Dropped frames" ||
                std::chrono::steady_clock::now() - this->drops_shown_at >=
                    this->drop_report_interval)
                this->show_drops(this->dropped());
            return;
        }
        if (!this->lagging) {
            this->show("", "");
            return;
        }
        const auto mb = this->max_queue_bytes / (size_t{1024} * 1024);
        this->show(
            "Falling behind the Core",
            "More than " + std::to_string(mb / 2) +
                " MB of frames wait for the Core. Frames drop when " +
                std::to_string(mb) + " MB wait."
        );
    }

    /// @brief leaves the final drop count on the task status, which shows it in the
    /// stop status, or clears the warning when no frame dropped.
    void report_closed() {
        if (!this->warn) return;
        const auto dropped = this->dropped();
        if (dropped.total() == 0)
            this->show("", "");
        else
            this->show_drops(dropped);
    }

    /// @brief waits for ops and moves all of them into batch, so the forwarder thread
    /// sends them without the lock. Returns false once closing leaves none, or once the
    /// close timeout passes.
    bool take() {
        std::unique_lock lock(this->mu);
        this->idle = true;
        this->cv.wait(lock, [this] { return this->closing || !this->queue.empty(); });
        this->idle = false;
        if (this->queue.empty() || this->expired()) return false;
        this->batch.swap(this->queue);
        return true;
    }

    /// @brief puts the ops of batch that were not sent back at the front of the queue.
    void unsent() {
        if (this->batch.empty()) return;
        std::lock_guard lock(this->mu);
        this->queue.insert(
            this->queue.begin(),
            std::make_move_iterator(this->batch.begin()),
            std::make_move_iterator(this->batch.end())
        );
        this->batch.clear();
    }

    /// @brief drops every waiting frame and counts it under cause.
    void drop_queue(size_t Drops::*cause) {
        std::lock_guard lock(this->mu);
        for (const auto &entry: this->queue)
            if (std::holds_alternative<x::telem::Frame>(entry.op))
                ++(this->drops.*cause);
        this->dropped_total = this->drops.total();
        this->queue.clear();
        this->queue_bytes = 0;
    }

    /// @brief waits out the backoff after the Core proves unreachable, and reports the
    /// queue while it waits. Returns false once the forwarder closes.
    bool back_off(const x::errors::Error &err) {
        this->failures++;
        const auto interval = this->backoff.interval(this->failures);
        LOG(WARNING) << "[bypass.forwarder] " << this->config.subject.name << " retry "
                     << this->failures << " in " << interval.seconds() << " s: " << err;
        const auto until = std::chrono::steady_clock::now() + interval.chrono();
        while (true) {
            {
                std::unique_lock lock(this->mu);
                const auto wake = std::min(
                    until,
                    std::chrono::steady_clock::now() + this->drop_report_interval
                );
                this->cv.wait_until(lock, wake, [this] { return this->closing; });
                if (this->closing) return false;
            }
            if (std::chrono::steady_clock::now() >= until) return true;
            this->report();
        }
    }

    /// @brief backs off after an unreachable Core. Any other Core error, a license
    /// error included, stops the forwarder so its owner retries it. Returns false when
    /// the forwarder must stop.
    bool wait(const x::errors::Error &err) {
        if (synnax::errors::is_unreachable(err)) {
            this->settle();
            if (this->back_off(err)) return true;
            this->drop_queue(&Drops::stopped);
            return false;
        }
        LOG(ERROR) << "[bypass.forwarder] " << this->config.subject.name
                   << " stopped: " << err;
        {
            std::lock_guard lock(this->mu);
            this->err = err;
        }
        this->drop_queue(&Drops::failed);
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
        const auto name = "fwd:" + this->config.subject.name;
        x::thread::set_name(name.substr(0, x::thread::MAX_NAME_LEN - 1).c_str());
        x::defer::defer settle([this] {
            this->settle();
            this->report_closed();
        });
        while (true) {
            this->report();
            auto [writer, open_err] = this->server->open_writer(this->config);
            if (open_err) {
                if (this->wait(open_err)) continue;
                return;
            }
            this->settle();
            this->failures = 0;
            x::errors::Error op_err;
            while (!op_err && this->take()) {
                while (!this->batch.empty() && !this->expired()) {
                    if (op_err = this->apply(*writer, this->batch.front().op); op_err)
                        break;
                    this->queue_bytes.fetch_sub(
                        this->batch.front().bytes,
                        std::memory_order_relaxed
                    );
                    this->batch.pop_front();
                    this->report();
                }
                this->unsent();
            }
            const auto close_err = writer->close();
            const auto err = close_err ? close_err : op_err;
            if (!err) {
                this->drop_queue(&Drops::stopped);
                return;
            }
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
        drop_report_interval(cfg.drop_report_interval.chrono()),
        close_timeout(cfg.close_timeout.chrono()),
        warn(cfg.warn),
        backoff(cfg.breaker),
        drops(cfg.dropped),
        dropped_total(cfg.dropped.total()) {
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

    /// @brief queues fr for the Core. Drops and counts it when the queue is full, and
    /// never waits on the Core. A frame larger than the cap goes through when the queue
    /// is empty.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error write(const x::telem::Frame &fr) {
        Entry entry{.op = fr.shallow_copy(), .bytes = frame_bytes(fr)};
        bool wake = false;
        size_t logged_drops = 0;
        {
            std::lock_guard lock(this->mu);
            if (this->err) return this->err;
            const auto bytes = this->queue_bytes.load(std::memory_order_relaxed);
            if (bytes > 0 && bytes + entry.bytes > this->max_queue_bytes) {
                this->drops.full++;
                this->dropped_total = this->drops.total();
                if (++this->cap_drops % 1000 == 1) logged_drops = this->cap_drops;
            } else
                wake = this->push_locked(std::move(entry));
        }
        if (wake) this->cv.notify_one();
        if (logged_drops > 0)
            LOG(WARNING) << "[bypass.forwarder] " << this->config.subject.name
                         << " dropped " << logged_drops
                         << " frames: the queue for the Core is full";
        return x::errors::NIL;
    }

    /// @brief queues authorities for the Core, in order with the frames. It never drops
    /// them, but they count toward the queue cap.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error set_authority(const ::driver::pipeline::Authorities &authorities) {
        Entry entry{.op = authorities, .bytes = authorities_bytes(authorities)};
        bool wake = false;
        {
            std::lock_guard lock(this->mu);
            if (this->err) return this->err;
            wake = this->push_locked(std::move(entry));
        }
        if (wake) this->cv.notify_one();
        return x::errors::NIL;
    }

    /// @brief sends what the queue holds while the Core is reachable, for at most
    /// ForwarderConfig::close_timeout, then closes the Core writer. Drops and counts
    /// what is left, and all of the queue when the Core is unreachable.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error close() {
        {
            std::lock_guard lock(this->mu);
            if (!this->closing) {
                this->closing = true;
                const auto deadline = std::chrono::steady_clock::now() +
                                      this->close_timeout;
                this->close_deadline.store(
                    deadline.time_since_epoch().count(),
                    std::memory_order_relaxed
                );
            }
        }
        this->cv.notify_one();
        if (this->thread.joinable()) this->thread.join();
        std::lock_guard lock(this->mu);
        return this->err;
    }

    /// @brief the frames dropped, ForwarderConfig::dropped included.
    [[nodiscard]] Drops dropped() {
        std::lock_guard lock(this->mu);
        return this->drops;
    }
};
}
