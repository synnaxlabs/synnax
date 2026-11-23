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

#include "driver/bypass/outage.h"
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

/// @returns bytes as a whole number of the largest unit that holds it, such as
/// "128 KB".
inline std::string format_size(const size_t bytes) {
    constexpr size_t KB = 1024;
    if (bytes >= KB * KB * KB) return std::to_string(bytes / (KB * KB * KB)) + " GB";
    if (bytes >= KB * KB) return std::to_string(bytes / (KB * KB)) + " MB";
    if (bytes >= KB) return std::to_string(bytes / KB) + " KB";
    return std::to_string(bytes) + " B";
}

/// @brief shows a warning on a task status, or stops showing it when message is empty.
using Warn = std::function<
    void(const std::string &message, const std::string &description)>;

/// @brief counts the frames that a Forwarder dropped, by cause.
struct Drops {
    /// @brief frames dropped because the queue budget was full.
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
    /// @brief the bytes that queued frames and authority changes hold, overhead
    /// included. Forwarders that share it share one cap. A write past it drops its
    /// frame, the newest one, and counts it.
    std::shared_ptr<Budget> budget = std::make_shared<Budget>(
        OutageConfig{}.queue_size
    );
    /// @brief the shortest time between two reports of a new drop count. While the Core
    /// is unreachable, the forwarder also reports at this interval.
    x::telem::TimeSpan drop_report_interval = OutageConfig{}.drop_report_interval;
    /// @brief the longest time that close() sends queued ops to the Core. It drops and
    /// counts what is left after it, and ends a write or close that blocks on the Core.
    x::telem::TimeSpan drain_timeout = OutageConfig{}.drain_timeout;
    /// @brief the longest time that a frame waits in the queue for others to go to the
    /// Core with it.
    x::telem::TimeSpan forward_interval = OutageConfig{}.forward_interval;
    /// @brief the bytes of queued frames that go to the Core before forward_interval
    /// passes.
    size_t forward_size = OutageConfig{}.forward_size;
    /// @brief the frames that earlier forwarders of the same run dropped. The drop
    /// counts start from them.
    Drops dropped;
    /// @brief warns when the budget passes half its size, until it drains below a
    /// quarter. Once a frame drops, it reports the drop count instead, and leaves the
    /// final count when the forwarder closes. Called from the forwarder thread only.
    Warn warn;
    /// @brief called from the forwarder thread each time the Core writer opens.
    std::function<void()> on_open;
    /// @brief called from the forwarder thread each time an open Core writer closes,
    /// fails, or loses its stream.
    std::function<void()> on_close;
    /// @brief the backoff from an unreachable Core. The forwarder retries with no
    /// limit, so only the intervals apply.
    x::breaker::Config breaker{
        .base_interval = OutageConfig{}.reopen_interval,
        .scale = 1.1f,
        .max_interval = OutageConfig{}.max_reopen_interval,
    };
};

/// @brief delivers a task's frames and authority changes to the Core in order, on its
/// own thread, so a task never waits on the Core. It opens and reopens the Core writer
/// while the Core is unreachable, also when the stream of an idle writer fails. Any
/// other Core error stops it, drops and counts what it holds, and its next write
/// returns that error. All methods are safe to call from one task thread.
class Forwarder {
    using Op = std::variant<x::telem::Frame, ::driver::pipeline::Authorities>;

    struct Entry {
        Op op;
        size_t bytes;
    };

    std::shared_ptr<::driver::pipeline::WriterFactory> server;
    synnax::framer::WriterConfig config;
    const std::shared_ptr<Budget> budget;
    const std::chrono::nanoseconds drop_report_interval;
    const std::chrono::nanoseconds drain_timeout;
    const std::chrono::nanoseconds forward_interval;
    const size_t forward_size;
    const Warn warn;
    /// @brief the description of the warning that the queue lags.
    const std::string lag_description;
    const std::function<void()> on_open;
    const std::function<void()> on_close;
    const x::breaker::Config backoff;

    // Only the forwarder thread touches these.
    size_t failures = 0;
    bool lagging = false;
    std::string shown_message;
    std::string shown_description;
    std::chrono::steady_clock::time_point drops_shown_at;
    /// @brief the time until which the forwarder waits for more ops after it takes a
    /// batch, so a steady stream of writes need not wake it.
    std::chrono::steady_clock::time_point linger_until;
    /// @brief true once the forwarder tried the Core again after close() began.
    bool retried_on_close = false;

    /// @brief the entries that the forwarder thread took from the queue and sends.
    std::deque<Entry> batch;

    std::mutex mu;
    std::condition_variable cv;
    std::deque<Entry> queue;
    /// @brief the bytes that queue and batch hold, which budget also counts. Changed
    /// only under mu, except that the forwarder thread subtracts each op it sends.
    std::atomic<size_t> queue_bytes = 0;
    /// @brief true while the forwarder thread waits for an op with no time limit.
    bool idle = false;
    /// @brief true while the forwarder thread waits for more ops until a time.
    bool waiting = false;
    /// @brief the bytes that queue holds, without batch.
    size_t pending_bytes = 0;
    /// @brief when the oldest op in queue came.
    std::chrono::steady_clock::time_point first_queued;
    /// @brief true while queue holds an authority change, which goes at once.
    bool urgent = false;
    /// @brief the open Core writer, which close() cancels at the drain deadline.
    ::driver::pipeline::Writer *current = nullptr;
    /// @brief true once the forwarder thread ends.
    bool done = false;
    std::condition_variable done_cv;
    /// @brief true from a drop until the forwarder thread reports it while it waits
    /// for the queue.
    bool unreported = false;
    Drops drops;
    /// @brief drops.total(), which report() reads without mu.
    std::atomic<size_t> dropped_total = 0;
    /// @brief the frames this forwarder dropped at the cap, for its log.
    size_t cap_drops = 0;
    bool closing = false;
    /// @brief the steady clock time at which close() stops sending, in nanoseconds.
    std::atomic<int64_t> close_deadline{std::numeric_limits<int64_t>::max()};
    /// @brief set once the stream of the open Core writer ends.
    bool lost = false;
    /// @brief set by retry() to end a backoff early.
    bool retrying = false;
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

    /// @brief queues entry, whose bytes the budget holds. Requires mu. Returns true
    /// when the forwarder thread must wake for it: it waits with no time limit, or the
    /// queue just passed forward_size.
    bool push_locked(Entry entry) {
        if (this->queue.empty()) this->first_queued = std::chrono::steady_clock::now();
        const bool below = this->pending_bytes < this->forward_size;
        this->pending_bytes += entry.bytes;
        this->queue_bytes.fetch_add(entry.bytes, std::memory_order_relaxed);
        this->queue.push_back(std::move(entry));
        return this->idle ||
               (this->waiting && below && this->pending_bytes >= this->forward_size);
    }

    [[nodiscard]] std::chrono::steady_clock::time_point deadline() const {
        return std::chrono::steady_clock::time_point(
            std::chrono::duration_cast<std::chrono::steady_clock::duration>(
                std::chrono::nanoseconds(
                    this->close_deadline.load(std::memory_order_relaxed)
                )
            )
        );
    }

    [[nodiscard]] bool expired() const {
        return std::chrono::steady_clock::now() >= this->deadline();
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
            if (!description.empty()) description += ' ';
            description += "Dropped " + std::to_string(count) +
                           (count == 1 ? " frame" : " frames") + " because " + cause +
                           ".";
        };
        add(dropped.full, "the Driver's queue for the Core was full");
        add(dropped.stopped, "the task stopped before delivery to the Core");
        add(dropped.failed, "a Core error stopped delivery");
        this->show("Dropped frames", description);
    }

    /// @brief shows on the task status whether the queue lags or drops frames. The
    /// queue lags while it holds frames and the budget is more than half full, until it
    /// empties or the budget drains below a quarter.
    void report() {
        if (!this->warn) return;
        const auto bytes = this->budget->used();
        if (this->queue_bytes.load(std::memory_order_relaxed) == 0 ||
            bytes * 4 < this->budget->size())
            this->lagging = false;
        else if (bytes * 2 >= this->budget->size())
            this->lagging = true;
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
        this->show("Falling behind the Core", this->lag_description);
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
    /// sends them without the lock. It waits up to forward_interval for more ops, but
    /// takes an authority change, forward_size bytes, or a closing queue at once. While
    /// it waits, it reports new drops at most once per drop report interval. Returns
    /// false once the Core writer's stream ends, once closing leaves no op, or once the
    /// drain timeout passes.
    bool take() {
        std::unique_lock lock(this->mu);
        const auto ready = [this] {
            return this->closing || this->lost || this->urgent ||
                   this->pending_bytes >= this->forward_size;
        };
        bool lingered = false;
        while (!ready()) {
            const auto now = std::chrono::steady_clock::now();
            if (!this->queue.empty()) {
                const auto due = this->first_queued + this->forward_interval;
                if (lingered || now >= due) break;
                this->waiting = true;
                this->cv.wait_until(lock, due, ready);
                this->waiting = false;
                continue;
            }
            if (now < this->linger_until) {
                this->waiting = true;
                this->cv.wait_until(lock, this->linger_until, ready);
                this->waiting = false;
                lingered = true;
                continue;
            }
            lingered = false;
            const auto any = [&] { return ready() || !this->queue.empty(); };
            this->idle = true;
            if (this->unreported)
                this->cv.wait_until(
                    lock,
                    this->drops_shown_at + this->drop_report_interval,
                    any
                );
            else
                this->cv.wait(lock, [&] { return any() || this->unreported; });
            this->idle = false;
            if (any()) continue;
            if (!this->unreported ||
                std::chrono::steady_clock::now() <
                    this->drops_shown_at + this->drop_report_interval)
                continue;
            this->unreported = false;
            lock.unlock();
            this->report();
            lock.lock();
        }
        this->linger_until = std::chrono::steady_clock::now() + this->forward_interval;
        if (this->lost || this->queue.empty() || this->expired()) return false;
        this->batch.swap(this->queue);
        this->pending_bytes = 0;
        this->urgent = false;
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
        this->pending_bytes = 0;
        this->urgent = false;
        this->budget->release(this->queue_bytes.exchange(0));
    }

    /// @brief waits out the backoff after the Core proves unreachable, or until
    /// retry(), and reports the queue while it waits. Once close() begins, it stops at
    /// once when the queue is empty. Otherwise it tries the Core again at once, and
    /// then within the drain timeout only. Returns false once no try is left.
    bool back_off(const x::errors::Error &err) {
        this->failures++;
        const auto interval = this->backoff.interval(this->failures);
        LOG(WARNING) << "[bypass.forwarder] " << this->config.subject.name << " retry "
                     << this->failures << " in " << interval.seconds() << " s: " << err;
        auto until = std::chrono::steady_clock::now() + interval.chrono();
        const auto retry_now = [this] {
            return this->closing && !this->retried_on_close;
        };
        while (true) {
            {
                std::unique_lock lock(this->mu);
                if (this->closing) until = std::min(until, this->deadline());
                const auto wake = std::min(
                    until,
                    std::chrono::steady_clock::now() + this->drop_report_interval
                );
                this->cv.wait_until(lock, wake, [&] {
                    return retry_now() || this->retrying;
                });
                if (this->closing && this->queue.empty()) return false;
                if (retry_now()) {
                    this->retried_on_close = true;
                    return true;
                }
                if (this->retrying) {
                    this->retrying = false;
                    return true;
                }
                if (this->closing) until = std::min(until, this->deadline());
            }
            if (std::chrono::steady_clock::now() >= until) return !this->expired();
            this->report();
        }
    }

    /// @brief backs off after an unreachable Core. Any other Core error, a license
    /// error included, stops the forwarder so its owner retries it. Returns false when
    /// the forwarder must stop.
    bool wait(const x::errors::Error &err) {
        if (this->expired()) {
            this->drop_queue(&Drops::stopped);
            this->settle();
            return false;
        }
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

    /// @brief sends batch to writer, and gives back the bytes of each op it sends.
    x::errors::Error send(::driver::pipeline::Writer &writer) {
        while (!this->batch.empty() && !this->expired()) {
            if (auto err = this->apply(writer, this->batch.front().op)) return err;
            const auto sent = this->batch.front().bytes;
            this->queue_bytes.fetch_sub(sent, std::memory_order_relaxed);
            this->budget->release(sent);
            this->batch.pop_front();
            this->report();
        }
        return x::errors::NIL;
    }

    void run() {
        const auto name = "fwd:" + this->config.subject.name;
        x::thread::set_name(name.substr(0, x::thread::MAX_NAME_LEN - 1).c_str());
        x::defer::defer settle([this] {
            this->settle();
            this->report_closed();
            {
                std::lock_guard lock(this->mu);
                this->done = true;
            }
            this->done_cv.notify_all();
        });
        while (true) {
            this->report();
            auto [writer, open_err] = this->server->open_writer(this->config);
            if (open_err) {
                if (this->wait(open_err)) continue;
                return;
            }
            {
                std::lock_guard lock(this->mu);
                this->current = writer.get();
            }
            if (this->on_open) this->on_open();
            this->settle();
            this->failures = 0;
            std::thread watcher([this, w = writer.get()] {
                if (!w->wait()) return;
                {
                    std::lock_guard lock(this->mu);
                    this->lost = true;
                }
                this->cv.notify_one();
            });
            x::errors::Error op_err;
            while (!op_err && this->take()) {
                op_err = this->send(*writer);
                this->unsent();
            }
            bool lost = false;
            {
                std::lock_guard lock(this->mu);
                lost = this->lost;
            }
            const auto close_err = writer->close();
            {
                std::lock_guard lock(this->mu);
                this->current = nullptr;
            }
            watcher.join();
            if (this->on_close) this->on_close();
            {
                std::lock_guard lock(this->mu);
                this->lost = false;
            }
            auto err = close_err ? close_err : op_err;
            // The Core ends a writer's stream with an error unless the writer closes.
            if (!err && lost) err = synnax::framer::WRITER_CLOSED;
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
        budget(cfg.budget),
        drop_report_interval(cfg.drop_report_interval.chrono()),
        drain_timeout(cfg.drain_timeout.chrono()),
        forward_interval(cfg.forward_interval.chrono()),
        forward_size(cfg.forward_size),
        warn(cfg.warn),
        lag_description(
            "More than " + format_size(this->budget->size() / 2) +
            " of frames from this Driver wait for the Core. Frames drop when they "
            "fill the " +
            format_size(this->budget->size()) + " queue that all tasks share."
        ),
        on_open(cfg.on_open),
        on_close(cfg.on_close),
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

    /// @brief queues fr for the Core, and takes its series when it keeps it. Drops and
    /// counts it when the budget is full, and never waits on the Core. A frame larger
    /// than the whole budget goes through when the queue of this forwarder is empty.
    /// @returns the Core error that stopped the forwarder, if any. fr is unchanged
    /// then.
    x::errors::Error write(x::telem::Frame &fr) {
        const auto bytes = frame_bytes(fr);
        bool wake = false;
        size_t logged_drops = 0;
        {
            std::lock_guard lock(this->mu);
            if (this->err) return this->err;
            const bool alone = bytes > this->budget->size() &&
                               this->queue_bytes.load(std::memory_order_relaxed) == 0;
            if (this->budget->reserve(bytes, alone))
                wake = this->push_locked({.op = std::move(fr), .bytes = bytes});
            else {
                this->drops.full++;
                this->dropped_total = this->drops.total();
                if (++this->cap_drops % 1000 == 1) logged_drops = this->cap_drops;
                if (this->warn && !this->unreported) {
                    this->unreported = true;
                    wake = this->idle;
                }
            }
        }
        if (wake) this->cv.notify_one();
        if (logged_drops > 0)
            LOG(WARNING) << "[bypass.forwarder] " << this->config.subject.name
                         << " dropped " << logged_drops
                         << " frames: the Driver's queue for the Core is full";
        return x::errors::NIL;
    }

    /// @brief queues fr for the Core, as write(x::telem::Frame &) does.
    x::errors::Error write(x::telem::Frame &&fr) {
        auto owned = std::move(fr);
        return this->write(owned);
    }

    /// @brief queues authorities for the Core, in order with the frames. It never drops
    /// them, but they count toward the budget.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error set_authority(const ::driver::pipeline::Authorities &authorities) {
        Entry entry{.op = authorities, .bytes = authorities_bytes(authorities)};
        bool wake = false;
        {
            std::lock_guard lock(this->mu);
            if (this->err) return this->err;
            this->budget->reserve(entry.bytes, true);
            this->urgent = true;
            wake = this->push_locked(std::move(entry)) || this->waiting;
        }
        if (wake) this->cv.notify_one();
        return x::errors::NIL;
    }

    /// @brief ends the current backoff from an unreachable Core, or the next one, so
    /// the forwarder opens the Core writer again at once.
    void retry() {
        {
            std::lock_guard lock(this->mu);
            this->retrying = true;
        }
        this->cv.notify_one();
    }

    /// @brief sends what the queue holds while the Core is reachable, for at most
    /// ForwarderConfig::drain_timeout, then closes the Core writer. Drops and counts
    /// what is left, and all of the queue when the Core is unreachable.
    /// @returns the Core error that stopped the forwarder, if any.
    x::errors::Error close() {
        {
            std::unique_lock lock(this->mu);
            if (!this->closing) {
                this->closing = true;
                const auto deadline = std::chrono::steady_clock::now() +
                                      this->drain_timeout;
                this->close_deadline.store(
                    std::chrono::duration_cast<std::chrono::nanoseconds>(
                        deadline.time_since_epoch()
                    )
                        .count(),
                    std::memory_order_relaxed
                );
            }
            this->cv.notify_one();
            if (!this->done_cv.wait_until(
                    lock,
                    this->deadline(),
                    [this] { return this->done; }
                ) &&
                this->current != nullptr)
                this->current->cancel();
        }
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
