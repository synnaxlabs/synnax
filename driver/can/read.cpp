// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>

#include "driver/can/read.h"

namespace driver::can {
Source::Source(bus::ReadConfig cfg, Acquire acquire):
    cfg(std::move(cfg)), decoder(this->cfg), acquire(std::move(acquire)) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return this->decoder.writer_config();
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->decoder.channels();
}

x::errors::Error Source::start() {
    this->decoder.clear_warning();
    auto [link, err] = this->acquire();
    if (err) return err;
    this->link = std::move(link);
    return this->link->bus()
        .second.skip(TEMPORARY_HARDWARE_ERROR)
        .skip(CRITICAL_HARDWARE_ERROR);
}

x::errors::Error Source::stop() {
    this->link.reset();
    this->last.reset();
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &, x::telem::Frame &fr) {
    common::ReadResult res;
    auto [bus, err] = this->link->bus();
    if (err) {
        res.error = err;
        return res;
    }
    if (bus != this->last) {
        this->last = bus;
        this->aligner = {};
    }
    const auto deadline = x::telem::TimeStamp::now() + bus::READ_TIMEOUT;
    auto timeout = bus::READ_TIMEOUT;
    Frame frame;
    for (std::size_t n = 0; n < MAX_BATCH; n++) {
        auto [got, recv_err] = bus->receive(frame, timeout);
        if (recv_err) {
            this->link->close(bus);
            res.error = recv_err;
            break;
        }
        if (!got) break;
        this->handle(frame);
        timeout = x::telem::TimeSpan::ZERO();
        if (x::telem::TimeStamp::now() >= deadline) break;
    }
    this->decoder.flush(fr);
    res.warning = this->decoder.warning(x::telem::TimeStamp::now());
    return res;
}

void Source::handle(const Frame &frame) {
    if (frame.type == Type::BUS_ERROR) {
        this->decoder.warn("the adapter reported a bus error");
        return;
    }
    const std::uint32_t id = frame.id | (frame.extended ? 1u << 31 : 0u);
    this->raw.assign(
        {static_cast<std::uint8_t>(id),
         static_cast<std::uint8_t>(id >> 8),
         static_cast<std::uint8_t>(id >> 16),
         static_cast<std::uint8_t>(id >> 24)}
    );
    this->raw.insert(this->raw.end(), frame.payload().begin(), frame.payload().end());
    this->decoder.raw(this->raw);
    if (frame.type != Type::DATA) return;
    if (const auto i = this->cfg.matcher.match(frame.id, frame.extended))
        this->decoder
            .decode(this->cfg.streamed[*i], frame.payload(), this->stamp(frame));
}

x::telem::TimeStamp Source::stamp(const Frame &frame) {
    if (frame.clock == Clock::HOST) return frame.time;
    return this->aligner.align(frame.time, x::telem::TimeStamp::now());
}
}
