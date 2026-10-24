// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <deque>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <span>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/bus/types.gen.h"
#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/sync/list.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/bus/transport.h"

/// @brief fixtures for bus task tests: ICD messages and an in-memory transport.
namespace driver::bus::testutil {
/// @returns bytes holding the characters of s.
inline std::vector<std::uint8_t> bytes(const std::string &s) {
    return {s.begin(), s.end()};
}

/// @returns a string holding data.
inline std::string text(const std::span<const std::uint8_t> data) {
    return {reinterpret_cast<const char *>(data.data()), data.size()};
}

/// @returns the samples of a bytes series.
inline std::vector<std::string> samples(const x::telem::Series &series) {
    std::vector<std::string> out;
    const auto *p = reinterpret_cast<const std::uint8_t *>(series.data());
    const auto *end = p + series.byte_size();
    while (p + 4 <= end) {
        const std::uint32_t n = p[0] | p[1] << 8 | p[2] << 16 |
                                static_cast<std::uint32_t>(p[3]) << 24;
        out.emplace_back(reinterpret_cast<const char *>(p + 4), n);
        p += 4 + n;
    }
    return out;
}

/// @returns a little-endian unsigned binary field with a new key.
inline synnax::icd::BinaryField binary_field(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length = 8
) {
    synnax::icd::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    return f;
}

/// @returns a text field read from item position of a delimited line.
inline synnax::icd::DelimitedTextField
delimited_field(const std::string &name, const std::uint32_t position) {
    synnax::icd::DelimitedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = position;
    return f;
}

/// @returns a text field read from the text after tag.
inline synnax::icd::TaggedTextField
tagged_field(const std::string &name, const std::string &tag) {
    synnax::icd::TaggedTextField f;
    f.key = x::uuid::create();
    f.name = name;
    f.tag = tag;
    return f;
}

/// @returns a binary message with a new key.
inline synnax::icd::Message binary_message(
    const std::string &name,
    std::vector<synnax::icd::BinaryField> fields,
    std::optional<synnax::icd::Identifier> identifier = std::nullopt
) {
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = synnax::icd::BinaryPayload{
        .identifier = std::move(identifier),
        .fields = std::move(fields),
    };
    return m;
}

/// @returns a text message with a new key that matches lines starting with prefix,
/// split on delimiter.
inline synnax::icd::Message text_message(
    const std::string &name,
    std::vector<synnax::icd::TextField> fields,
    const std::string &prefix = "",
    const std::string &delimiter = ","
) {
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = name;
    m.payload = synnax::icd::TextPayload{
        .delimiter = delimiter,
        .prefix = prefix,
        .fields = std::move(fields),
    };
    return m;
}

/// @returns the binary payload of the message.
inline synnax::icd::BinaryPayload &binary(synnax::icd::Message &message) {
    return std::get<synnax::icd::BinaryPayload>(message.payload);
}

/// @returns the parts every field of the message shares, in order.
inline std::vector<const synnax::icd::BaseField *>
field_bases(const synnax::icd::Message &message) {
    std::vector<const synnax::icd::BaseField *> bases;
    if (const auto *b = std::get_if<synnax::icd::BinaryPayload>(&message.payload))
        for (const auto &f: b->fields)
            bases.push_back(&f);
    else
        for (const auto &f: std::get<synnax::icd::TextPayload>(message.payload).fields)
            bases.push_back(
                std::visit(
                    [](const auto &v) -> const synnax::icd::BaseField * { return &v; },
                    f
                )
            );
    return bases;
}

/// @returns the key of field i of the message.
inline synnax::icd::FieldKey
field_key(const synnax::icd::Message &message, const std::size_t i) {
    return field_bases(message).at(i)->key;
}

/// @returns an ICD holding messages.
inline synnax::icd::ICD create_icd(std::vector<synnax::icd::Message> messages) {
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "test";
    ic.messages = std::move(messages);
    return ic;
}

/// @returns an index channel with the given key.
inline synnax::channel::Channel index_channel(const synnax::channel::Key key) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.name = "index_" + std::to_string(key);
    ch.data_type = x::telem::TIMESTAMP_T;
    ch.is_index = true;
    return ch;
}

/// @returns a data channel with the given key, stamped on index.
inline synnax::channel::Channel data_channel(
    const synnax::channel::Key key,
    const synnax::channel::Key index,
    const x::telem::DataType &data_type = x::telem::FLOAT64_T
) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.name = "data_" + std::to_string(key);
    ch.data_type = data_type;
    ch.index = index;
    ch.is_virtual = index == 0;
    return ch;
}

/// @brief one read an in-memory Transport returns.
struct Read {
    /// @brief the bytes the read returns.
    std::vector<std::uint8_t> data;
    /// @brief the bytes the line dropped for parity or framing errors.
    std::size_t errors = 0;
    /// @brief the receive overruns the line reported.
    std::size_t overruns = 0;
    /// @brief when the bytes arrive. Later reads wait behind this one.
    std::chrono::steady_clock::time_point at;
};

/// @brief the state an in-memory Transport shares with the test that scripts it.
struct Wire {
    std::mutex mu;
    std::condition_variable cv;
    /// @brief what the transport returns from read, in order.
    std::deque<Read> reads;
    /// @brief the error the next read returns instead of a chunk.
    x::errors::Error read_err;
    /// @brief errors the opener returns, in order, before it succeeds.
    std::deque<x::errors::Error> open_errs;
    /// @brief errors the next writes return, in order, before writes succeed. A failed
    /// write sends nothing.
    std::deque<x::errors::Error> write_errs;
    /// @brief how long each write blocks before its bytes go out, as on a slow link.
    x::telem::TimeSpan write_delay = x::telem::TimeSpan::ZERO();
    /// @brief the number of writes that have begun, including those still blocked by
    /// write_delay.
    int started = 0;
    /// @brief every write, in order.
    std::vector<std::vector<std::uint8_t>> writes;
    /// @brief called with each write while mu is held. It can queue a reply.
    std::function<void(Wire &, std::span<const std::uint8_t>)> on_write;
    /// @brief the number of successful opens.
    int opens = 0;
    /// @brief the number of open attempts, failed or not.
    int attempts = 0;
    /// @brief the number of transports closed.
    int closes = 0;

    /// @brief queues a chunk for read.
    void push(
        const std::vector<std::uint8_t> &chunk,
        const std::size_t errors = 0,
        const std::size_t overruns = 0
    ) {
        {
            std::lock_guard lock(this->mu);
            this->reads.push_back({chunk, errors, overruns});
        }
        this->cv.notify_all();
    }

    /// @brief makes the next read fail with err.
    void fail(const x::errors::Error &err) {
        {
            std::lock_guard lock(this->mu);
            this->read_err = err;
        }
        this->cv.notify_all();
    }

    /// @returns a copy of the writes so far.
    std::vector<std::vector<std::uint8_t>> written() {
        std::lock_guard lock(this->mu);
        return this->writes;
    }
};

/// @brief an in-memory Transport that reads the chunks a test queues on its Wire and
/// records what it writes.
class Transport final : public bus::Transport {
    std::shared_ptr<Wire> wire;
    std::vector<std::uint8_t> last;

public:
    explicit Transport(std::shared_ptr<Wire> wire): wire(std::move(wire)) {}

    ~Transport() override {
        std::lock_guard lock(this->wire->mu);
        this->wire->closes++;
    }

    std::pair<transport::Chunk, x::errors::Error>
    read(const x::telem::TimeSpan timeout) override {
        std::unique_lock lock(this->wire->mu);
        const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
        const auto ready = [this] {
            return this->wire->read_err ||
                   (!this->wire->reads.empty() &&
                    this->wire->reads.front().at <= std::chrono::steady_clock::now());
        };
        while (!ready()) {
            auto until = deadline;
            if (!this->wire->reads.empty())
                until = std::min(until, this->wire->reads.front().at);
            if (this->wire->cv.wait_until(lock, until) == std::cv_status::timeout &&
                until == deadline)
                break;
        }
        if (this->wire->read_err) {
            auto err = this->wire->read_err;
            this->wire->read_err = x::errors::NIL;
            return {{}, err};
        }
        if (!ready()) return {{}, x::errors::NIL};
        auto read = std::move(this->wire->reads.front());
        this->wire->reads.pop_front();
        this->last = std::move(read.data);
        return {
            transport::Chunk{
                .data = this->last,
                .time = x::telem::TimeStamp::now(),
                .errors = read.errors,
                .overruns = read.overruns,
            },
            x::errors::NIL,
        };
    }

    x::errors::Error
    write(const std::span<const std::uint8_t> data, const x::telem::TimeSpan) override {
        std::unique_lock lock(this->wire->mu);
        if (!this->wire->write_errs.empty()) {
            auto err = this->wire->write_errs.front();
            this->wire->write_errs.pop_front();
            return err;
        }
        this->wire->started++;
        if (const auto delay = this->wire->write_delay;
            delay > x::telem::TimeSpan::ZERO()) {
            lock.unlock();
            std::this_thread::sleep_for(delay.chrono());
            lock.lock();
        }
        this->wire->writes.emplace_back(data.begin(), data.end());
        if (this->wire->on_write) this->wire->on_write(*this->wire, data);
        lock.unlock();
        this->wire->cv.notify_all();
        return x::errors::NIL;
    }
};

/// @returns an Opener of Transports on wire. It fails with each of wire->open_errs
/// before it succeeds.
inline Opener opener(const std::shared_ptr<Wire> &wire) {
    return [wire]() -> std::pair<std::unique_ptr<bus::Transport>, x::errors::Error> {
        std::lock_guard lock(wire->mu);
        wire->attempts++;
        if (!wire->open_errs.empty()) {
            auto err = wire->open_errs.front();
            wire->open_errs.pop_front();
            return {nullptr, err};
        }
        wire->opens++;
        return {std::make_unique<Transport>(wire), x::errors::NIL};
    };
}

/// @returns an Acquire of one connection on wire from connections. Tasks built from
/// Acquires that share connections share the connection.
inline Acquire acquire(
    const std::shared_ptr<Wire> &wire,
    const std::shared_ptr<Connections> &connections = std::make_shared<Connections>()
) {
    return acquirer(connections, "dev", x::json::json::object(), opener(wire));
}

/// @brief a steady clock that moves only when a test sets or advances it. Safe for
/// concurrent use.
class Clock {
    mutable std::mutex mu;
    x::telem::TimeStamp time = x::telem::TimeStamp(x::telem::SECOND);

public:
    /// @returns the current time.
    x::telem::TimeStamp now() const {
        std::lock_guard lock(this->mu);
        return this->time;
    }

    /// @brief moves the clock forward by span.
    void advance(const x::telem::TimeSpan span) {
        std::lock_guard lock(this->mu);
        this->time = this->time + span;
    }

    /// @returns a NowFunc that reads this clock, which must outlive it.
    x::telem::NowFunc func() const {
        return [this] { return this->now(); };
    }
};

/// @brief resolves a read config over messages. The fields of message i are stamped on
/// index channel 100 + i and map in order to FLOAT64 channels numbered from 1.
/// @param edit changes the config before it resolves.
/// @param channels channels to add to the generated ones, such as a raw channel. A
/// channel with the key of a generated one replaces it.
inline ReadConfig read_config(
    const std::vector<synnax::icd::Message> &messages,
    const std::optional<::synnax::bus::Framing> &framing,
    const std::function<void(::synnax::bus::ReadConfig &)> &edit = nullptr,
    const ::synnax::bus::PollConfig &poll = {},
    std::vector<synnax::channel::Channel> channels = {},
    const Medium medium = Medium::BYTES
) {
    ::synnax::bus::ReadConfig cfg;
    cfg.device = "dev";
    std::vector<synnax::channel::Channel> generated;
    synnax::channel::Key next = 1;
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto index = static_cast<synnax::channel::Key>(100 + i);
        generated.push_back(index_channel(index));
        ::synnax::bus::ReadMessage rm{.message = messages[i].key};
        for (const auto *f: field_bases(messages[i])) {
            generated.push_back(data_channel(next, index));
            rm.fields.push_back({.field = f->key, .channel = next++, .index = index});
        }
        cfg.messages.push_back(rm);
    }
    channels.insert(channels.begin(), generated.begin(), generated.end());
    if (edit) edit(cfg);
    x::json::Parser parser(x::json::json::object());
    std::optional<Stream> stream;
    if (framing.has_value()) stream = Stream{.framing = *framing};
    auto [out, err] = ReadConfig::resolve(
        parser,
        cfg,
        poll,
        stream,
        create_icd(messages),
        channels,
        medium
    );
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_FALSE(err) << err;
    out.device = "dev";
    return std::move(out);
}

/// @brief resolves a write config over one message.
/// @param bindings maps each command channel to the index of the field it drives.
inline WriteConfig write_config(
    const synnax::icd::Message &message,
    const std::vector<std::pair<synnax::channel::Key, std::size_t>> &bindings,
    const std::optional<::synnax::bus::Framing> &framing,
    const Medium medium = Medium::BYTES
) {
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    ::synnax::bus::WriteMessage wm{.message = message.key};
    std::vector<synnax::channel::Channel> channels;
    for (const auto &[ch, field]: bindings) {
        wm.fields.push_back({.field = field_key(message, field), .channel = ch});
        channels.push_back(data_channel(ch, 0));
    }
    cfg.messages = {wm};
    x::json::Parser parser(x::json::json::object());
    auto [out, err] = WriteConfig::resolve(
        parser,
        cfg,
        framing,
        create_icd({message}),
        channels,
        medium
    );
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_FALSE(err) << err;
    out.device = "dev";
    return std::move(out);
}

/// @returns every value of the channel across frames, in order.
inline std::vector<double>
values(const x::sync::List<x::telem::Frame> &frames, const synnax::channel::Key key) {
    std::vector<double> out;
    for (const auto &fr: frames.snapshot())
        for (const auto &[k, series]: fr) {
            if (k != key) continue;
            for (std::size_t i = 0; i < series.size(); i++)
                out.push_back(series.at<double>(static_cast<int>(i)));
        }
    return out;
}
}
