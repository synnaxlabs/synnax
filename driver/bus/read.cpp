// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <tuple>
#include <utility>

#include "driver/bus/read.h"
#include "driver/transport/errors.h"

namespace driver::bus {
Source::Source(ReadConfig cfg, Acquire acquire, x::telem::NowFunc now):
    cfg(std::move(cfg)),
    now(std::move(now)),
    decoder(this->cfg, this->now),
    acquire(std::move(acquire)),
    next_query(this->cfg.polled.size()) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return this->decoder.writer_config();
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->decoder.channels();
}

x::errors::Error Source::start() {
    this->next_poll = this->now();
    this->restart_polls();
    this->decoder.reset();
    auto [conn, err] = this->acquire();
    if (err) return err;
    this->conn = std::move(conn);
    auto guard = this->conn->lock();
    return guard.transport().second.skip(transport::UNREACHABLE_ERROR);
}

x::errors::Error Source::stop() {
    this->conn.reset();
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &breaker, x::telem::Frame &fr) {
    common::ReadResult res;
    x::errors::Error err;
    {
        auto guard = this->conn->lock();
        Transport *t = nullptr;
        std::tie(t, err) = guard.transport();
        if (!err) {
            if (guard.opens() != this->opens) {
                this->opens = guard.opens();
                if (this->cfg.framer != nullptr) this->cfg.framer->reset();
                this->restart_polls();
                this->decoder.restart();
            }
            err = this->exchange(*t, breaker);
            if (err) guard.fail(err);
        }
    }
    if (err) {
        if (!err.matches(transport::UNREACHABLE_ERROR)) {
            res.error = err;
            return res;
        }
        this->restart_polls();
        this->decoder.restart();
        this->decoder.warn("reconnecting to " + this->cfg.device + ": " + err.data);
        const auto wait = std::min(this->conn->retry_at() - this->now(), READ_TIMEOUT);
        if (wait > x::telem::TimeSpan::ZERO()) breaker.wait_for(wait);
    }
    this->decoder.flush(fr);
    res.warning = this->decoder.warning();
    return res;
}

void Source::restart_polls() {
    this->next_query = this->cfg.polled.size();
    this->awaiting.reset();
    this->missed.reset();
    this->idle = x::telem::TimeSpan::ZERO();
}

x::errors::Error Source::exchange(Transport &t, const x::breaker::Breaker &breaker) {
    const auto now = this->now();
    if (this->missed.has_value() && now >= this->quiet_until) this->missed.reset();
    const bool quiet = !this->missed.has_value();
    if (quiet && this->next_query >= this->cfg.polled.size() &&
        !this->cfg.polled.empty() && now >= this->next_poll) {
        const auto period = this->cfg.poll_rate.period();
        this->next_query = 0;
        this->next_poll = this->next_poll + period;
        if (this->next_poll <= now) this->next_poll = now + period;
    }
    if (quiet && this->next_query < this->cfg.polled.size())
        return this->query(t, breaker);
    auto timeout = READ_TIMEOUT;
    if (!quiet)
        timeout = std::min(timeout, this->quiet_until - now);
    else if (!this->cfg.polled.empty())
        timeout = std::min(timeout, this->next_poll - now);
    return this->receive(t, std::max(timeout, x::telem::MILLISECOND)).second;
}

x::errors::Error Source::query(Transport &t, const x::breaker::Breaker &breaker) {
    const auto message = this->cfg.polled[this->next_query++];
    if (const auto err = this->drain(t)) return err;
    if (const auto err = t.write(this->cfg.messages[message].query, WRITE_TIMEOUT))
        return err;
    const auto deadline = this->now() + this->cfg.poll_timeout;
    this->awaiting = message;
    x::errors::Error err;
    while (!err && this->awaiting.has_value() && breaker.running()) {
        const auto now = this->now();
        if (now >= deadline) {
            this->missed = message;
            this->quiet_until = now + this->cfg.poll_timeout;
            break;
        }
        err = this->receive(t, std::min(deadline - now, READ_TIMEOUT)).second;
    }
    this->awaiting.reset();
    return err;
}

x::errors::Error Source::drain(Transport &t) {
    const auto deadline = this->now() + READ_TIMEOUT;
    while (this->now() < deadline) {
        const auto [n, err] = this->receive(t, x::telem::MILLISECOND);
        if (err) return err;
        if (n == 0) break;
    }
    if (this->cfg.framer != nullptr) this->cfg.framer->reset();
    return x::errors::NIL;
}

std::pair<std::size_t, x::errors::Error>
Source::receive(Transport &t, const x::telem::TimeSpan timeout) {
    auto [chunk, err] = t.read(timeout);
    if (err) return {0, err};
    if (chunk.errors > 0)
        this->decoder.warn("dropped bytes received with parity or framing errors");
    if (chunk.overruns > 0)
        this->decoder.warn("lost received bytes to receive overruns");
    if (chunk.data.empty()) {
        this->idle += timeout;
        return {0, x::errors::NIL};
    }
    if (this->cfg.framer == nullptr) {
        this->handle(chunk.data, chunk.time);
        return {chunk.data.size(), x::errors::NIL};
    }
    const auto before = this->cfg.framer->errors();
    if (this->idle >= this->cfg.gap) this->cfg.framer->gap();
    this->idle = x::telem::TimeSpan::ZERO();
    const auto framing_err = this->cfg.framer->write(
        chunk.data,
        [&](const std::span<const std::uint8_t> frame) {
            this->handle(frame, chunk.time);
        }
    );
    if (framing_err) this->decoder.warn(framing_err.data);
    if (this->cfg.framer->errors() > before)
        this->decoder.warn("dropped corrupt or unframed segments of the byte stream");
    return {chunk.data.size(), x::errors::NIL};
}

void Source::handle(
    const std::span<const std::uint8_t> frame,
    const x::telem::TimeStamp time
) {
    this->decoder.raw(frame);
    if (this->awaiting.has_value() &&
        this->cfg.messages[*this->awaiting].reply.match(frame)) {
        const auto message = *this->awaiting;
        this->awaiting.reset();
        this->decoder.decode(message, frame, time);
        return;
    }
    if (this->missed.has_value() &&
        this->cfg.messages[*this->missed].reply.match(frame)) {
        this->decoder.late(*this->missed);
        this->quiet_until = this->now() + this->cfg.poll_timeout;
        return;
    }
    if (const auto i = this->cfg.matcher.match(frame))
        this->decoder.decode(this->cfg.streamed[*i], frame, time);
}
}
