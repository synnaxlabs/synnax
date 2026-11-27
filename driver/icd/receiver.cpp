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
#include <map>
#include <memory>
#include <optional>
#include <span>
#include <sstream>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/json.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"
#include "driver/icd/receiver.h"
#include "driver/transport/transport.h"

namespace driver::icd {
namespace {
common::Warning dropped(const x::errors::Error &err) {
    return {.message = "Dropped invalid data", .description = err.data};
}

/// @returns the warning about the line errors and overruns of chunk. It names only the
/// kinds that occurred, so its text stays the same from read to read.
common::Warning line_errors(const transport::Chunk &chunk) {
    std::string kinds;
    if (chunk.errors > 0) kinds = "parity errors, framing errors, or breaks";
    if (chunk.overruns > 0)
        kinds += kinds.empty() ? "receive overruns" : ", and receive overruns";
    return {
        .message = "Received serial line errors",
        .description = "The port reported " + kinds +
                       ". Check that the baud rate, parity, and data bits match the "
                       "device.",
    };
}

x::errors::Error refuse(const std::string &text) {
    return {driver::errors::CONFIGURATION_ERROR, text};
}

/// @brief returns the framing of ic as JSON, or null when it has none.
x::json::json framing_of(const synnax::icd::ICD &ic) {
    if (!ic.framing.has_value()) return nullptr;
    return synnax::icd::to_json(*ic.framing);
}
}

std::pair<Receiver, x::errors::Error> Receiver::create(
    const std::vector<synnax::icd::ReadEntry> &entries,
    const std::vector<synnax::icd::ICD> &icds,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
    const bool datagram,
    transport::Clock now
) {
    Receiver r;
    r.datagram = datagram;
    r.now = std::move(now);
    std::vector<Reader::Source> sources;
    sources.reserve(entries.size());
    for (std::size_t i = 0; i < entries.size(); i++) {
        const auto ic = std::ranges::find(icds, entries[i].icd, &synnax::icd::ICD::key);
        if (ic == icds.end())
            return {
                Receiver{},
                refuse(
                    "entry " + std::to_string(i) + ": ICD " +
                    entries[i].icd.to_string() + " does not exist"
                ),
            };
        if (!datagram && !ic->framing.has_value())
            return {
                Receiver{},
                refuse("ICD " + ic->name + " has no framing to split the stream with"),
            };
        if (!sources.empty() && framing_of(sources.front().ic) != framing_of(*ic))
            return {
                Receiver{},
                refuse(
                    "ICDs " + sources.front().ic.name + " and " + ic->name +
                    " frame the stream differently"
                ),
            };
        sources.push_back({.ic = *ic, .reads = entries[i].messages});
        r.owners.insert(r.owners.end(), entries[i].messages.size(), i);
    }
    auto [reader, reader_err] = Reader::create(sources, channels);
    if (reader_err) return {Receiver{}, reader_err};
    r.reader = std::move(reader);
    if (!sources.empty())
        if (const auto &framing = sources.front().ic.framing; framing.has_value()) {
            auto [framer, err] = codec::framing::create(*framing);
            if (err) return {Receiver{}, err};
            r.framer = std::move(framer);
        }
    for (std::size_t i = 0; i < entries.size(); i++) {
        const auto &entry_poll = entries[i].poll;
        if (!entry_poll.has_value()) continue;
        const auto &poll = *entry_poll;
        const auto context = "entry " + std::to_string(i) + ": ";
        if (const auto hz = poll.rate.hz(); !(hz > 0) || hz > 1e9) {
            std::ostringstream text;
            text << context
                 << "the poll rate must be above 0 Hz and at most 1 GHz, not "
                 << poll.rate;
            return {Receiver{}, refuse(text.str())};
        }
        Poll p{.entry = i, .period = poll.rate.period()};
        if (r.framer == nullptr)
            p.query = poll.query;
        else if (const auto err = r.framer->encode(poll.query, p.query))
            return {
                Receiver{},
                refuse(context + "the framing cannot hold the poll query: " + err.data),
            };
        r.polls.push_back(std::move(p));
    }
    return {std::move(r), x::errors::NIL};
}

void Receiver::parse(
    const std::span<const std::uint8_t> payload,
    x::telem::Frame &fr,
    common::Warning &warning
) {
    this->stamp = std::max(this->received, this->stamp + x::telem::NANOSECOND);
    auto [message, err] = this->reader.read(payload, this->stamp, fr, BATCH);
    if (message.has_value() && this->reply.has_value() &&
        this->reply->entry == this->owners[*message])
        this->reply.reset();
    if (!err && !this->reader.failures().empty())
        err = this->reader.failures().front().error;
    if (err && warning.empty())
        warning = {.message = "Failed to decode message", .description = err.data};
}

std::size_t
Receiver::drain(std::size_t n, x::telem::Frame &fr, common::Warning &warning) {
    while (n < BATCH) {
        const auto [frame, err] = this->framer->next();
        if (err) {
            if (warning.empty()) warning = dropped(err);
            continue;
        }
        if (!frame.has_value()) break;
        n++;
        this->parse(*frame, fr, warning);
    }
    return n;
}

std::size_t Receiver::decode(x::telem::Frame &fr, common::Warning &warning) {
    auto n = this->drain(0, fr, warning);
    if (!this->datagram) return n;
    while (n < BATCH && this->framer->pending() > 0) {
        const auto err = this->framer->idle();
        if (!err) break;
        if (warning.empty()) warning = dropped(err);
        n = this->drain(n, fr, warning);
    }
    if (n == BATCH) return n;
    if (const auto left = this->framer->pending(); left > 0 && warning.empty())
        warning = dropped(
            x::errors::Error(
                x::errors::VALIDATION,
                "dropped " + std::to_string(left) +
                    " bytes of a partial frame at the end of a datagram"
            )
        );
    this->framer->clear();
    return n;
}

std::pair<x::telem::TimeSpan, x::errors::Error>
Receiver::send(transport::Transport &t) {
    if (this->polls.empty()) return {READ_TIMEOUT, x::errors::NIL};
    const auto now = this->now();
    if (this->reply.has_value() && now >= this->reply->deadline) this->reply.reset();
    if (const auto p = std::ranges::min_element(this->polls, {}, &Poll::next);
        !this->reply.has_value() && now >= p->next) {
        if (auto err = t.write(p->query, transport::WRITE_TIMEOUT))
            return {READ_TIMEOUT, err};
        this->reply = Reply{
            .entry = p->entry,
            .deadline = now + REPLY_TIMEOUT.chrono()
        };
        p->next += p->period.chrono();
        if (p->next <= now) p->next = now + p->period.chrono();
    }
    auto ready = std::ranges::min_element(this->polls, {}, &Poll::next)->next;
    if (this->reply.has_value()) ready = std::max(ready, this->reply->deadline);
    return {std::min(READ_TIMEOUT, x::telem::TimeSpan(ready - now)), x::errors::NIL};
}

std::vector<synnax::channel::Key> Receiver::channels() const {
    return this->reader.channels();
}

void Receiver::clear() {
    if (this->framer != nullptr) this->framer->clear();
}

common::ReadResult Receiver::fail(const x::errors::Error &err) {
    this->clear();
    return {.error = err};
}

common::ReadResult Receiver::read(transport::Transport &t, x::telem::Frame &fr) {
    common::ReadResult res;
    const auto [timeout, send_err] = this->send(t);
    if (send_err) return this->fail(send_err);
    if (this->framer != nullptr && this->decode(fr, res.warning) > 0) return res;
    const auto [chunk, err] = t.read(timeout);
    if (err) return this->fail(err);
    if ((chunk.errors > 0 || chunk.overruns > 0) && res.warning.empty())
        res.warning = line_errors(chunk);
    if (this->framer == nullptr) {
        this->received = chunk.time;
        if (!chunk.data.empty()) this->parse(chunk.data, fr, res.warning);
        return res;
    }
    if (!chunk.data.empty()) {
        this->quiet = x::telem::TimeSpan::ZERO();
        this->framer->write(chunk.data);
        this->received = chunk.time;
    } else if (!this->datagram) {
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
