// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <span>
#include <string>
#include <utility>

#include "driver/mil1553/read.h"

namespace driver::mil1553 {
namespace {
/// @returns the transfer as it moved on the bus: the command word, the status word
/// when answered, and the data words, each big-endian.
std::vector<std::uint8_t> raw_frame(const Transfer &t) {
    std::vector<std::uint16_t> words{t.command.encode()};
    if (t.answered) words.push_back(t.status.encode());
    words.insert(words.end(), t.words.begin(), t.words.begin() + t.count());
    std::vector<std::uint8_t> out;
    codec::mil1553::to_payload(words, out);
    return out;
}
}

Source::Source(bus::ReadConfig cfg, synnax::mil1553::Properties props, Acquire acquire):
    cfg(std::move(cfg)),
    decoder(this->cfg),
    props(std::move(props)),
    acquire(std::move(acquire)),
    buf(READ_BATCH) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return this->decoder.writer_config();
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->decoder.channels();
}

x::errors::Error Source::start() {
    this->decoder.reset();
    std::vector<std::optional<x::telem::TimeSpan>> periods;
    for (const auto i: this->cfg.streamed)
        periods.push_back(this->cfg.messages[i].entry.period);
    this->schedule = bus::Schedule(periods, x::telem::TimeStamp::now());
    auto [conn, err] = this->acquire();
    if (err) return err;
    this->conn = std::move(conn);
    auto guard = this->conn->lock();
    return guard.transport().second;
}

x::errors::Error Source::stop() {
    this->conn.reset();
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &breaker, x::telem::Frame &fr) {
    common::ReadResult res;
    res.error = this->props.role == synnax::mil1553::ROLE_BUS_CONTROLLER
                  ? this->poll(breaker)
                  : this->listen();
    if (res.error) return res;
    this->decoder.flush(fr);
    res.warning = this->decoder.warning(x::telem::TimeStamp::now());
    return res;
}

x::errors::Error Source::poll(x::breaker::Breaker &breaker) {
    const auto now = x::telem::TimeStamp::now();
    this->due.clear();
    this->schedule.due(now, this->due);
    if (this->due.empty()) {
        auto wait = bus::READ_TIMEOUT;
        if (const auto next = this->schedule.next()) wait = std::min(wait, *next - now);
        breaker.wait_for(wait);
        return x::errors::NIL;
    }
    auto guard = this->conn->lock();
    auto [ch, err] = guard.transport();
    if (err) return err;
    for (const auto j: this->due) {
        const auto &id = std::get<synnax::library::Mil1553Identifier>(
            *this->cfg.messages[this->cfg.streamed[j]].entry.identifier
        );
        const auto cmd = codec::mil1553::Command::from(id);
        auto [t, t_err] = ch->transact(cmd, {});
        if (t_err) {
            guard.close();
            return t_err;
        }
        this->add(this->cfg.streamed[j], t);
    }
    return x::errors::NIL;
}

x::errors::Error Source::listen() {
    bus::Batch batch;
    {
        auto guard = this->conn->lock();
        auto [ch, err] = guard.transport();
        if (err) return err;
        auto [b, read_err] = ch->read(this->buf, bus::READ_TIMEOUT);
        if (read_err) {
            guard.close();
            return read_err;
        }
        batch = b;
    }
    for (std::size_t i = 0; i < batch.count; i++) {
        const auto &t = this->buf[i];
        if (const auto m = this->cfg.matcher.match(t.command))
            this->add(this->cfg.streamed[*m], t);
        else
            this->decoder.raw(raw_frame(t));
    }
    if (batch.dropped > 0)
        this->decoder.warn(
            "the card dropped " + std::to_string(batch.dropped) +
            " transfers because the Driver fell behind"
        );
    return x::errors::NIL;
}

void Source::add(const std::size_t message, const Transfer &t) {
    this->decoder.raw(raw_frame(t));
    if (!t.answered) {
        this->decoder.warn(address(t.command) + " did not answer");
        return;
    }
    if (t.status.message_error)
        this->decoder.warn(address(t.command) + " reported a message error");
    else if (t.status.busy)
        this->decoder.warn(address(t.command) + " was busy");
    if (t.count() == 0 || t.status.message_error) return;
    this->payload.clear();
    codec::mil1553::to_payload(std::span(t.words.data(), t.count()), this->payload);
    this->decoder.decode(message, this->payload, t.time);
}
}
