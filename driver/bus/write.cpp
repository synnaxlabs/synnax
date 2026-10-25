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

#include "x/cpp/strings/strings.h"
#include "x/cpp/thread/thread.h"

#include "driver/bus/write.h"
#include "driver/errors/errors.h"
#include "driver/transport/errors.h"

namespace driver::bus {
Sink::Sink(
    WriteConfig cfg,
    std::unique_ptr<Transmitter> transmitter,
    x::telem::NowFunc now
):
    common::Sink(cfg.commands),
    cfg(std::move(cfg)),
    transmitter(std::move(transmitter)),
    now(std::move(now)),
    states(this->cfg.messages.size()),
    warnings(this->now) {
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        const auto &bindings = this->cfg.messages[i].bindings;
        for (std::size_t b = 0; b < bindings.size(); b++)
            this->targets[bindings[b].channel].emplace_back(i, b);
    }
    this->reset();
}

Sink::~Sink() {
    this->stop();
}

void Sink::reset() {
    const auto now = this->now();
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        const auto &m = this->cfg.messages[i];
        auto &s = this->states[i];
        s.values = m.initial;
        s.seen.assign(m.plan.size(), false);
        s.unseen = m.bindings.size();
        s.due = now;
        s.pending.reset();
    }
    this->queue.clear();
    this->warnings.clear();
    this->failure = x::errors::NIL;
    this->fatal = x::errors::NIL;
    this->resend_at = now;
    this->shown.clear();
}

x::errors::Error Sink::start() {
    std::lock_guard lock(this->mu);
    if (this->running) return x::errors::NIL;
    if (const auto err = this->transmitter->acquire()) return err;
    this->reset();
    this->schedule(this->now());
    this->running = true;
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
    this->transmitter->release();
    return x::errors::NIL;
}

x::errors::Error Sink::write(x::telem::Frame &frame) {
    {
        std::lock_guard lock(this->mu);
        if (this->fatal) return this->fatal;
        std::set<std::size_t> changed;
        for (const auto &[key, series]: frame) {
            if (series.empty()) continue;
            const auto it = this->targets.find(key);
            if (it == this->targets.end()) continue;
            const auto value = series.at(-1);
            for (const auto &[message, binding]: it->second) {
                const auto &m = this->cfg.messages[message];
                const auto &b = m.bindings[binding];
                auto &s = this->states[message];
                if (const auto err = m.plan.set(s.values, b.slot, value)) {
                    this->warnings.add(
                        m.definition.name + "." + b.field + ": " + err.data
                    );
                    continue;
                }
                if (!s.seen[b.slot]) {
                    s.seen[b.slot] = true;
                    s.unseen--;
                }
                changed.insert(message);
            }
        }
        for (const auto message: changed)
            if (!this->cfg.messages[message].definition.period.has_value())
                this->enqueue(message);
        this->report();
    }
    this->cv.notify_all();
    return x::errors::NIL;
}

void Sink::enqueue(const std::size_t message) {
    const auto &m = this->cfg.messages[message];
    const auto &s = this->states[message];
    if (s.unseen > 0) return;
    std::vector<std::uint8_t> payload;
    m.plan.encode(s.values, payload);
    if (!m.prefix.empty()) {
        // The prefix of a text line is its item 0, which the plan leaves empty.
        std::string_view prefix = m.prefix;
        const std::string_view body(
            reinterpret_cast<const char *>(payload.data()),
            payload.size()
        );
        const auto &delimiter = std::get<synnax::icd::TextPayload>(m.definition.payload)
                                    .delimiter;
        if (prefix.ends_with(delimiter) && body.starts_with(delimiter))
            prefix.remove_suffix(delimiter.size());
        payload.insert(payload.begin(), prefix.begin(), prefix.end());
    }
    if (this->cfg.framer == nullptr) {
        this->push(message, std::move(payload));
        return;
    }
    std::vector<std::uint8_t> wire;
    if (const auto err = this->cfg.framer->encode(payload, wire)) {
        this->warnings.add(m.definition.name + ": " + err.data);
        return;
    }
    this->push(message, std::move(wire));
}

void Sink::push(const std::size_t message, std::vector<std::uint8_t> payload) {
    auto &pending = this->states[message].pending;
    if (!pending.has_value()) this->queue.push_back(message);
    pending = std::move(payload);
}

void Sink::report() {
    std::vector<std::string> out;
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        const auto &m = this->cfg.messages[i];
        const auto &s = this->states[i];
        if (s.unseen == 0 || s.unseen == m.bindings.size()) continue;
        std::vector<std::string> fields;
        for (const auto &b: m.bindings)
            if (!s.seen[b.slot]) fields.push_back(b.field);
        out.push_back(
            m.definition.name + " waits for " + x::strings::join(fields, ", ")
        );
    }
    if (this->fatal) out.push_back(this->fatal.data);
    if (this->failure)
        out.push_back(
            "reconnecting to " + this->cfg.device + ": " + this->failure.data
        );
    this->warnings.collect(out);
    auto warning = x::strings::join(out, "; ");
    if (warning == this->shown) return;
    this->shown = warning;
    if (warning.empty())
        this->set_warning(x::errors::NIL);
    else
        this->set_warning(
            x::errors::Error(driver::errors::TEMPORARY_HARDWARE_ERROR, warning)
        );
}

x::telem::TimeStamp Sink::schedule(const x::telem::TimeStamp now) {
    auto next = x::telem::TimeStamp::max();
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        const auto &period = this->cfg.messages[i].definition.period;
        if (!period.has_value()) continue;
        auto &s = this->states[i];
        if (now >= s.due) {
            this->enqueue(i);
            s.due = s.due + *period;
            if (s.due <= now) s.due = now + *period;
        }
        next = std::min(next, s.due);
    }
    return next;
}

void Sink::run() {
    x::thread::set_name("bus:write");
    std::unique_lock lock(this->mu);
    while (this->running) {
        const auto now = this->now();
        auto next = this->schedule(now);
        this->report();
        next = std::min(next, this->warnings.expiry());
        const bool sending = !this->queue.empty() && !this->fatal;
        if (!sending || now < this->resend_at) {
            if (sending) next = std::min(next, this->resend_at);
            if (next == x::telem::TimeStamp::max())
                this->cv.wait(lock);
            else
                this->cv.wait_for(lock, (next - now).chrono());
            continue;
        }
        const auto message = this->queue.front();
        this->queue.pop_front();
        auto payload = std::move(this->states[message].pending.value());
        this->states[message].pending.reset();
        lock.unlock();
        const auto err = this->transmitter->send(message, payload);
        lock.lock();
        if (!err) {
            this->failure = x::errors::NIL;
            continue;
        }
        if (!err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR)) {
            this->fatal = err;
            continue;
        }
        this->failure = err;
        this->resend_at = this->now() + RESEND_INTERVAL;
        if (!this->states[message].pending.has_value())
            this->push(message, std::move(payload));
    }
}

x::errors::Error ConnectionTransmitter::acquire() {
    auto [conn, err] = this->acquirer();
    if (err) return err;
    this->conn = std::move(conn);
    auto guard = this->conn->lock();
    return guard.transport().second.skip(transport::UNREACHABLE_ERROR);
}

void ConnectionTransmitter::release() {
    this->conn.reset();
}

x::errors::Error
ConnectionTransmitter::send(std::size_t, const std::span<const std::uint8_t> payload) {
    auto guard = this->conn->lock();
    auto [t, err] = guard.transport();
    if (err) return err;
    err = t->write(payload, WRITE_TIMEOUT);
    if (err) guard.fail(err);
    return err;
}
}
