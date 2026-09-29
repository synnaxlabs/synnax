// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "driver/bus/read.h"

namespace driver::bus {
Source::Source(ReadConfig cfg, Acquire acquire):
    cfg(std::move(cfg)),
    decoder(this->cfg),
    acquire(std::move(acquire)),
    next_query(this->cfg.polled.size()) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return this->decoder.writer_config();
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->decoder.channels();
}

x::errors::Error Source::start() {
    this->next_poll = x::telem::TimeStamp::now();
    this->next_query = this->cfg.polled.size();
    this->decoder.clear_warning();
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
    {
        auto guard = this->conn->lock();
        auto [t, err] = guard.transport();
        if (!err) {
            if (guard.opens() != this->opens) {
                this->opens = guard.opens();
                if (this->cfg.framer != nullptr) this->cfg.framer->reset();
            }
            err = this->exchange(*t, breaker);
        }
        if (err) {
            guard.close();
            this->next_query = this->cfg.polled.size();
            res.error = err;
            return res;
        }
    }
    this->decoder.flush(fr);
    res.warning = this->decoder.warning(x::telem::TimeStamp::now());
    return res;
}

x::errors::Error Source::exchange(Transport &t, const x::breaker::Breaker &breaker) {
    const auto now = x::telem::TimeStamp::now();
    if (this->next_query >= this->cfg.polled.size() && !this->cfg.polled.empty() &&
        now >= this->next_poll) {
        this->next_query = 0;
        this->next_poll = std::max(this->next_poll + this->cfg.poll_rate.period(), now);
    }
    if (this->next_query < this->cfg.polled.size()) return this->query(t, breaker, now);
    auto timeout = READ_TIMEOUT;
    if (!this->cfg.polled.empty()) timeout = std::min(timeout, this->next_poll - now);
    return this->receive(t, std::max(timeout, x::telem::MILLISECOND));
}

x::errors::Error Source::query(
    Transport &t,
    const x::breaker::Breaker &breaker,
    const x::telem::TimeStamp now
) {
    const auto message = this->cfg.polled[this->next_query++];
    if (const auto err = this->drain(t)) return err;
    if (const auto err = t.write(this->cfg.messages[message].query, WRITE_TIMEOUT))
        return err;
    const auto deadline = now + this->cfg.poll_timeout;
    this->awaiting = message;
    x::errors::Error err;
    while (!err && this->awaiting.has_value() && breaker.running()) {
        const auto left = deadline - x::telem::TimeStamp::now();
        if (left <= x::telem::TimeSpan::ZERO()) {
            this->decoder.warn(
                "no reply to the query of " + this->cfg.messages[message].entry.name +
                " within " + this->cfg.poll_timeout.to_string()
            );
            break;
        }
        err = this->receive(t, std::min(left, READ_TIMEOUT));
    }
    this->awaiting.reset();
    return err;
}

x::errors::Error Source::drain(Transport &t) {
    const auto deadline = x::telem::TimeStamp::now() + READ_TIMEOUT;
    while (x::telem::TimeStamp::now() < deadline) {
        auto [chunk, err] = t.read(x::telem::MILLISECOND);
        if (err) return err;
        if (chunk.data.empty()) break;
        this->consume(chunk);
    }
    if (this->cfg.framer != nullptr) this->cfg.framer->reset();
    return x::errors::NIL;
}

x::errors::Error Source::receive(Transport &t, const x::telem::TimeSpan timeout) {
    auto [chunk, err] = t.read(timeout);
    if (!err) this->consume(chunk);
    return err;
}

void Source::consume(const transport::Chunk &chunk) {
    if (chunk.data.empty()) return;
    if (this->cfg.framer == nullptr) return this->handle(chunk.data, chunk.time);
    const auto before = this->cfg.framer->errors();
    const auto framing_err = this->cfg.framer->write(
        chunk.data,
        [&](const std::span<const std::uint8_t> frame) {
            this->handle(frame, chunk.time);
        }
    );
    if (framing_err) this->decoder.warn(framing_err.data);
    if (const auto n = this->cfg.framer->errors() - before; n > 0)
        this->decoder.warn(
            "dropped " + std::to_string(n) +
            " corrupt or unframed segments of the byte stream"
        );
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
    if (const auto i = this->cfg.matcher.match(frame))
        this->decoder.decode(this->cfg.streamed[*i], frame, time);
}
}
