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
#include <variant>
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

x::errors::Error refuse(const std::string &text) {
    return {driver::errors::CONFIGURATION_ERROR, text};
}

/// @brief returns the framing of ic as JSON, or null when it has none.
x::json::json framing_of(const synnax::icd::ICD &ic) {
    if (!ic.framing.has_value()) return nullptr;
    return synnax::icd::to_json(*ic.framing);
}

/// @brief returns the name of the message with key in ic.
std::string
message_name(const synnax::icd::ICD &ic, const synnax::icd::MessageKey &key) {
    const auto m = std::ranges::find(ic.messages, key, &synnax::icd::Message::key);
    return m == ic.messages.end() ? key.to_string() : m->name;
}

/// @brief returns the error when entries a and b, both on ic, read one message.
x::errors::Error check_shared(
    const std::size_t a,
    const std::size_t b,
    const std::vector<synnax::icd::ReadEntry> &entries,
    const synnax::icd::ICD &ic
) {
    for (const auto &first: entries[a].messages)
        for (const auto &second: entries[b].messages)
            if (first.message == second.message)
                return refuse(
                    "entries " + std::to_string(a) + " and " + std::to_string(b) +
                    " both read message " + message_name(ic, first.message) +
                    " of ICD " + ic.name
                );
    return x::errors::NIL;
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
    std::vector<const synnax::icd::ICD *> found;
    for (std::size_t i = 0; i < entries.size(); i++) {
        const auto context = "entry " + std::to_string(i) + ": ";
        const auto ic = std::ranges::find(icds, entries[i].icd, &synnax::icd::ICD::key);
        if (ic == icds.end())
            return {
                Receiver{},
                refuse(
                    context + "ICD " + entries[i].icd.to_string() + " does not exist"
                ),
            };
        if (!datagram && !ic->framing.has_value())
            return {
                Receiver{},
                refuse("ICD " + ic->name + " has no framing to split the stream with"),
            };
        if (!found.empty() && framing_of(*found.front()) != framing_of(*ic))
            return {
                Receiver{},
                refuse(
                    "ICDs " + found.front()->name + " and " + ic->name +
                    " frame the stream differently"
                ),
            };
        for (std::size_t j = 0; j < i; j++)
            if (found[j]->key == ic->key)
                if (auto err = check_shared(j, i, entries, *ic))
                    return {Receiver{}, err};
        auto [reader, err] = Reader::create(entries[i].messages, *ic, channels);
        if (err) return {Receiver{}, x::errors::Error(err, context + err.data)};
        r.readers.push_back(std::move(reader));
        found.push_back(&*ic);
    }
    if (!found.empty())
        if (const auto &framing = found.front()->framing; framing.has_value()) {
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
        Poll p{.period = poll.rate.period()};
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

void Receiver::dispatch(
    const std::span<const std::uint8_t> payload,
    x::telem::Frame &fr,
    common::Warning &warning
) {
    this->stamp = std::max(this->received, this->stamp + x::telem::NANOSECOND);
    x::errors::Error read_err;
    for (auto &reader: this->readers) {
        const auto [message, err] = reader.read(payload, this->stamp, fr, BATCH);
        if (!message.has_value()) {
            if (!read_err) read_err = err;
            continue;
        }
        read_err = err;
        if (!err && !reader.failures().empty())
            read_err = reader.failures().front().error;
        break;
    }
    if (read_err && warning.empty())
        warning = {.message = "Failed to decode message", .description = read_err.data};
}

std::size_t Receiver::decode(x::telem::Frame &fr, common::Warning &warning) {
    std::size_t n = 0;
    while (n < BATCH) {
        const auto [frame, err] = this->framer->next();
        if (err) {
            if (warning.empty()) warning = dropped(err);
            continue;
        }
        if (!frame.has_value()) break;
        n++;
        this->dispatch(*frame, fr, warning);
    }
    if (this->datagram && n < BATCH) {
        if (const auto err = this->framer->idle(); err && warning.empty())
            warning = dropped(err);
        this->framer->clear();
    }
    return n;
}

std::pair<x::telem::TimeSpan, x::errors::Error>
Receiver::send(transport::Transport &t) {
    auto wait = READ_TIMEOUT;
    if (this->polls.empty()) return {wait, x::errors::NIL};
    const auto now = this->now();
    for (auto &p: this->polls) {
        if (now >= p.next) {
            if (auto err = t.write(p.query, WRITE_TIMEOUT)) return {wait, err};
            p.next += p.period.chrono();
            if (p.next <= now) p.next = now + p.period.chrono();
        }
        wait = std::min(wait, x::telem::TimeSpan(p.next - now));
    }
    return {wait, x::errors::NIL};
}

common::ReadResult Receiver::fail(const x::errors::Error &err) {
    if (this->framer != nullptr) this->framer->clear();
    return {.error = err};
}

common::ReadResult Receiver::read(transport::Transport &t, x::telem::Frame &fr) {
    common::ReadResult res;
    const auto [timeout, send_err] = this->send(t);
    if (send_err) return this->fail(send_err);
    if (this->framer != nullptr && this->decode(fr, res.warning) > 0) return res;
    const auto [chunk, err] = t.read(timeout);
    if (err) return this->fail(err);
    if (this->framer == nullptr) {
        this->received = chunk.time;
        if (!chunk.data.empty()) this->dispatch(chunk.data, fr, res.warning);
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
