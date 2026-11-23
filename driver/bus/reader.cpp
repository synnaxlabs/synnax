// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <cstddef>
#include <memory>
#include <optional>
#include <sstream>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/reader.h"
#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"
#include "driver/transport/transport.h"

namespace driver::bus {
namespace {
common::Warning dropped(const x::errors::Error &err) {
    return {.message = "Dropped invalid data", .description = err.data};
}
}

Reader::Reader(
    std::unique_ptr<codec::framing::Framer> framer,
    icd::Reader messages,
    Clock now
):
    framer(std::move(framer)), messages(std::move(messages)), now(std::move(now)) {}

std::pair<Reader, x::errors::Error> Reader::create(
    std::unique_ptr<codec::framing::Framer> framer,
    icd::Reader messages,
    std::optional<Poll> poll,
    Clock now
) {
    Reader r(std::move(framer), std::move(messages), std::move(now));
    if (!poll.has_value()) return {std::move(r), x::errors::NIL};
    if (const auto hz = poll->rate.hz(); !(hz > 0) || hz > 1e9) {
        std::ostringstream text;
        text << "the poll rate must be above 0 Hz and at most 1 GHz, not "
             << poll->rate;
        return {
            Reader{},
            x::errors::Error(driver::errors::CONFIGURATION_ERROR, text.str()),
        };
    }
    if (const auto err = r.framer->encode(poll->query, r.query))
        return {
            Reader{},
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "the framing cannot hold the poll query: " + err.data
            ),
        };
    r.period = poll->rate.period();
    return {std::move(r), x::errors::NIL};
}

std::size_t Reader::decode(x::telem::Frame &fr, common::Warning &warning) {
    std::size_t n = 0;
    while (n < BATCH) {
        const auto [frame, err] = this->framer->next();
        if (err) {
            if (warning.empty()) warning = dropped(err);
            continue;
        }
        if (!frame.has_value()) break;
        n++;
        this->stamp = std::max(this->received, this->stamp + x::telem::NANOSECOND);
        const auto read_err = this->messages.read(*frame, this->stamp, fr, BATCH)
                                  .second;
        if (read_err && warning.empty())
            warning = {
                .message = "Failed to decode message",
                .description = read_err.data,
            };
    }
    return n;
}

std::pair<x::telem::TimeSpan, x::errors::Error> Reader::send(transport::Transport &t) {
    if (this->query.empty()) return {READ_TIMEOUT, x::errors::NIL};
    const auto now = this->now();
    if (now >= this->next) {
        if (auto err = t.write(this->query, WRITE_TIMEOUT)) return {READ_TIMEOUT, err};
        this->next += this->period.chrono();
        if (this->next <= now) this->next = now + this->period.chrono();
    }
    return {
        std::min(READ_TIMEOUT, x::telem::TimeSpan(this->next - now)),
        x::errors::NIL
    };
}

common::ReadResult Reader::fail(const x::errors::Error &err) {
    this->framer->clear();
    return {.error = err};
}

common::ReadResult Reader::read(transport::Transport &t, x::telem::Frame &fr) {
    common::ReadResult res;
    const auto [timeout, send_err] = this->send(t);
    if (send_err) return this->fail(send_err);
    if (this->decode(fr, res.warning) > 0) return res;
    const auto [chunk, err] = t.read(timeout);
    if (err) return this->fail(err);
    if (!chunk.data.empty()) {
        this->quiet = x::telem::TimeSpan::ZERO();
        this->framer->write(chunk.data);
        this->received = chunk.time;
    } else {
        this->quiet += timeout;
        if (this->quiet >= READ_TIMEOUT) {
            this->quiet = x::telem::TimeSpan::ZERO();
            if (const auto idle_err = this->framer->idle();
                idle_err && res.warning.empty())
                res.warning = dropped(idle_err);
        }
    }
    this->decode(fr, res.warning);
    return res;
}
}
