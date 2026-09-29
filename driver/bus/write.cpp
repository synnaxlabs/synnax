// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <set>
#include <string>
#include <utility>

#include "x/cpp/thread/thread.h"

#include "driver/bus/write.h"

namespace driver::bus {
Sink::Sink(
    WriteConfig cfg,
    Acquire acquire,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
):
    common::Sink(cfg.commands),
    cfg(std::move(cfg)),
    acquire(std::move(acquire)),
    states(this->cfg.messages.size()),
    status(ctx, task) {
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++)
        for (const auto &b: this->cfg.messages[i].bindings)
            this->targets[b.channel].emplace_back(i, b.slot);
    this->reset();
}

Sink::~Sink() {
    this->stop();
}

void Sink::reset() {
    const auto now = x::telem::TimeStamp::now();
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        const auto &m = this->cfg.messages[i];
        auto &s = this->states[i];
        s.values = m.initial;
        s.seen.assign(m.plan.size(), false);
        s.unseen = m.bindings.size();
        s.due = now;
    }
    this->queue.clear();
}

x::errors::Error Sink::start() {
    std::lock_guard lock(this->mu);
    if (this->running) return x::errors::NIL;
    auto [conn, err] = this->acquire();
    if (err) return err;
    this->conn = std::move(conn);
    this->reset();
    this->running = true;
    {
        std::lock_guard status_lock(this->status_mu);
        this->status.reset();
        this->status.status.details.running = true;
    }
    this->thread = std::thread([this] { this->run(); });
    return x::errors::NIL;
}

x::errors::Error Sink::stop() {
    {
        std::lock_guard lock(this->mu);
        this->running = false;
    }
    this->cv.notify_all();
    if (this->thread.joinable()) this->thread.join();
    std::lock_guard lock(this->mu);
    this->reset();
    this->conn.reset();
    return x::errors::NIL;
}

x::errors::Error Sink::write(x::telem::Frame &frame) {
    std::set<std::size_t> changed;
    {
        std::lock_guard lock(this->mu);
        for (const auto &[key, series]: frame) {
            if (series.size() == 0) continue;
            const auto it = this->targets.find(key);
            if (it == this->targets.end()) continue;
            const auto value = series.at(-1);
            for (const auto &[message, slot]: it->second) {
                auto &s = this->states[message];
                if (const auto err = s.values.set(slot, value)) return err;
                if (!s.seen[slot]) {
                    s.seen[slot] = true;
                    s.unseen--;
                }
                changed.insert(message);
            }
        }
        for (const auto message: changed)
            if (!this->cfg.messages[message].entry.period.has_value())
                this->enqueue(message);
    }
    this->cv.notify_all();
    return x::errors::NIL;
}

void Sink::enqueue(const std::size_t message) {
    const auto &m = this->cfg.messages[message];
    const auto &s = this->states[message];
    if (s.unseen > 0) return;
    std::vector<std::uint8_t> payload;
    if (const auto err = m.plan.encode(s.values, payload)) {
        this->warn(m.entry.name + ": " + err.data);
        return;
    }
    if (!m.prefix.empty()) {
        // The token of a text line is its item 0, which the plan leaves empty.
        std::string_view prefix = m.prefix;
        const std::string_view body(
            reinterpret_cast<const char *>(payload.data()),
            payload.size()
        );
        const auto &delimiter = m.entry.delimiter;
        if (m.entry.format == synnax::library::FORMAT_TEXT && !delimiter.empty() &&
            prefix.ends_with(delimiter) && body.starts_with(delimiter))
            prefix.remove_suffix(delimiter.size());
        payload.insert(payload.begin(), prefix.begin(), prefix.end());
    }
    if (this->cfg.framer == nullptr) {
        this->queue.push_back(std::move(payload));
        return;
    }
    std::vector<std::uint8_t> wire;
    if (const auto err = this->cfg.framer->encode(payload, wire)) {
        this->warn(m.entry.name + ": " + err.data);
        return;
    }
    this->queue.push_back(std::move(wire));
}

void Sink::run() {
    x::thread::set_name("bus:write");
    std::unique_lock lock(this->mu);
    while (this->running) {
        const auto now = x::telem::TimeStamp::now();
        auto next = x::telem::TimeStamp::max();
        for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
            const auto &period = this->cfg.messages[i].entry.period;
            if (!period.has_value()) continue;
            auto &s = this->states[i];
            if (now >= s.due) {
                this->enqueue(i);
                s.due = s.due + *period;
                if (s.due <= now) s.due = now + *period;
            }
            next = std::min(next, s.due);
        }
        if (this->queue.empty()) {
            if (next == x::telem::TimeStamp::max())
                this->cv.wait(lock);
            else
                this->cv.wait_for(lock, (next - now).chrono());
            continue;
        }
        const auto payload = std::move(this->queue.front());
        this->queue.pop_front();
        lock.unlock();
        this->send(payload);
        lock.lock();
    }
}

void Sink::send(const std::vector<std::uint8_t> &payload) {
    x::errors::Error err;
    {
        auto guard = this->conn->lock();
        auto [t, open_err] = guard.transport();
        err = open_err;
        if (!err) err = t->write(payload, WRITE_TIMEOUT);
        if (err) guard.close();
    }
    if (err) return this->warn(err.data);
    std::lock_guard lock(this->status_mu);
    this->status.clear_warning();
}

void Sink::warn(const std::string &message) {
    std::lock_guard lock(this->status_mu);
    this->status.send_warning(message);
}
}
