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

#include "x/cpp/strings/strings.h"

#include "driver/bus/read.h"

namespace driver::bus {
Source::Source(ReadConfig cfg, Acquire acquire):
    cfg(std::move(cfg)),
    acquire(std::move(acquire)),
    pending(this->cfg.messages.size()),
    next_query(this->cfg.polled.size()) {}

synnax::framer::WriterConfig Source::writer_config() const {
    std::vector<synnax::channel::Key> keys;
    for (const auto &m: this->cfg.messages) {
        if (m.index != 0) keys.push_back(m.index);
        for (const auto &ch: m.channels)
            keys.push_back(ch.key);
    }
    if (this->cfg.raw != 0) keys.push_back(this->cfg.raw);
    return {
        .channels = keys,
        .mode = common::data_saving_writer_mode(this->cfg.data_saving_disabled),
    };
}

std::vector<synnax::channel::Channel> Source::channels() const {
    std::vector<synnax::channel::Channel> out;
    for (const auto &m: this->cfg.messages)
        out.insert(out.end(), m.channels.begin(), m.channels.end());
    return out;
}

x::errors::Error Source::start() {
    this->next_poll = x::telem::TimeStamp::now();
    this->next_query = this->cfg.polled.size();
    this->held_warning.clear();
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
    this->flush(fr);
    res.warning = this->warning(x::telem::TimeStamp::now());
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
    if (const auto err = t.write(this->cfg.messages[message].query, WRITE_TIMEOUT))
        return err;
    const auto deadline = now + this->cfg.poll_timeout;
    this->awaiting = message;
    x::errors::Error err;
    while (!err && this->awaiting.has_value() && breaker.running()) {
        const auto left = deadline - x::telem::TimeStamp::now();
        if (left <= x::telem::TimeSpan::ZERO()) {
            this->warnings.push_back(
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

x::errors::Error Source::receive(Transport &t, const x::telem::TimeSpan timeout) {
    auto [chunk, err] = t.read(timeout);
    if (err) return err;
    if (chunk.data.empty()) return x::errors::NIL;
    if (this->cfg.framer == nullptr) {
        this->handle(chunk.data, chunk.time);
        return x::errors::NIL;
    }
    const auto before = this->cfg.framer->errors();
    const auto framing_err = this->cfg.framer->write(
        chunk.data,
        [&](const std::span<const std::uint8_t> frame) {
            this->handle(frame, chunk.time);
        }
    );
    if (framing_err) this->warnings.push_back(framing_err.data);
    if (const auto n = this->cfg.framer->errors() - before; n > 0)
        this->warnings.push_back(
            "dropped " + std::to_string(n) +
            " corrupt or unframed segments of the byte stream"
        );
    return x::errors::NIL;
}

void Source::handle(
    const std::span<const std::uint8_t> frame,
    const x::telem::TimeStamp time
) {
    if (this->cfg.raw != 0)
        this->raw.emplace_back(
            reinterpret_cast<const char *>(frame.data()),
            frame.size()
        );
    if (this->awaiting.has_value() &&
        this->cfg.messages[*this->awaiting].reply.match(frame)) {
        const auto message = *this->awaiting;
        this->awaiting.reset();
        this->decode(message, frame, time);
        return;
    }
    if (const auto i = this->cfg.matcher.match(frame))
        this->decode(this->cfg.streamed[*i], frame, time);
}

void Source::decode(
    const std::size_t message,
    const std::span<const std::uint8_t> frame,
    const x::telem::TimeStamp time
) {
    const auto &m = this->cfg.messages[message];
    auto &p = this->pending[message];
    const auto n = p.times.size();
    if (n == p.values.size()) p.values.push_back(m.plan.values());
    auto &values = p.values[n];
    if (const auto err = m.plan.decode(frame, values)) {
        this->warnings.push_back(m.entry.name + ": " + err.data);
        return;
    }
    if (values.invalid() > 0) {
        this->warnings.push_back(
            m.entry.name + ": " + std::to_string(values.invalid()) +
            " fields did not parse as numbers"
        );
        return;
    }
    // A multiplexor that leaves a field out drops the sample, so every channel on the
    // index keeps the same length.
    for (std::size_t s = 0; s < values.size(); s++)
        if (!values.present(s)) return;
    const auto stamp = std::max(time, p.last + x::telem::TimeSpan(1));
    p.last = stamp;
    p.times.push_back(stamp);
}

void Source::flush(x::telem::Frame &fr) {
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        auto &p = this->pending[i];
        const auto n = p.times.size();
        if (n == 0) continue;
        const auto &m = this->cfg.messages[i];
        for (std::size_t s = 0; s < m.channels.size(); s++) {
            x::telem::Series series(m.channels[s].data_type, n);
            for (std::size_t k = 0; k < n; k++)
                p.values[k].write(s, series);
            fr.emplace(m.channels[s].key, std::move(series));
        }
        if (m.index != 0) fr.emplace(m.index, x::telem::Series(p.times));
        p.times.clear();
    }
    if (!this->raw.empty()) {
        fr.emplace(this->cfg.raw, x::telem::Series(this->raw, x::telem::BYTES_T));
        this->raw.clear();
    }
}

std::string Source::warning(const x::telem::TimeStamp now) {
    if (!this->warnings.empty()) {
        this->held_warning = x::strings::join(this->warnings, "; ");
        this->held_at = now;
        this->warnings.clear();
    } else if (now - this->held_at >= WARNING_HOLD)
        this->held_warning.clear();
    return this->held_warning;
}
}
