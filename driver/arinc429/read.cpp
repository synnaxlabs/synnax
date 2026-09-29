// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "driver/arinc429/read.h"

namespace driver::arinc429 {
Source::Source(
    bus::ReadConfig cfg,
    std::shared_ptr<Backend> backend,
    synnax::arinc429::Properties props
):
    cfg(std::move(cfg)),
    decoder(this->cfg),
    backend(std::move(backend)),
    props(std::move(props)),
    buf(READ_BATCH) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return this->decoder.writer_config();
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->decoder.channels();
}

x::errors::Error Source::start() {
    this->decoder.clear_warning();
    auto [ch, err] = this->backend->open(this->props, Direction::RECEIVE);
    if (err) return err;
    this->channel = std::move(ch);
    return x::errors::NIL;
}

x::errors::Error Source::stop() {
    this->channel.reset();
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &, x::telem::Frame &fr) {
    common::ReadResult res;
    if (this->channel == nullptr) {
        auto [ch, err] = this->backend->open(this->props, Direction::RECEIVE);
        if (err) {
            res.error = err;
            return res;
        }
        this->channel = std::move(ch);
    }
    const auto [batch, err] = this->channel->read(this->buf, bus::READ_TIMEOUT);
    if (err) {
        this->channel.reset();
        res.error = err;
        return res;
    }
    std::size_t parity_errors = 0;
    for (std::size_t i = 0; i < batch.count; i++) {
        const auto &[word, time] = this->buf[i];
        const auto bytes = word.bytes();
        this->decoder.raw(bytes);
        if (!word.parity_valid()) {
            parity_errors++;
            continue;
        }
        if (const auto m = this->cfg.matcher.match(word))
            this->decoder.decode(this->cfg.streamed[*m], bytes, time);
    }
    if (parity_errors > 0)
        this->decoder.warn(
            "dropped " + std::to_string(parity_errors) + " words with even parity"
        );
    if (batch.dropped > 0)
        this->decoder.warn(
            "the receiver dropped " + std::to_string(batch.dropped) +
            " words because the Driver fell behind"
        );
    this->decoder.flush(fr);
    res.warning = this->decoder.warning(x::telem::TimeStamp::now());
    return res;
}
}
