// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/bus/types.gen.h"
#include "client/cpp/channel/types.gen.h"
#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/bus/transport.h"

/// @brief fixtures for bus task tests: library messages and an in-memory transport.
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
inline synnax::library::BinaryField binary_field(
    const std::string &name,
    const std::uint16_t start_bit,
    const std::uint8_t bit_length = 8
) {
    synnax::library::BinaryField f;
    f.key = x::uuid::create();
    f.name = name;
    f.start_bit = start_bit;
    f.bit_length = bit_length;
    return f;
}

/// @returns a text field read from item position of a delimited line.
inline synnax::library::DelimitedField
delimited_field(const std::string &name, const std::uint32_t position) {
    synnax::library::DelimitedField f;
    f.key = x::uuid::create();
    f.name = name;
    f.position = position;
    return f;
}

/// @returns a text field read from the text after tag.
inline synnax::library::TaggedField
tagged_field(const std::string &name, const std::string &tag) {
    synnax::library::TaggedField f;
    f.key = x::uuid::create();
    f.name = name;
    f.tag = tag;
    return f;
}

/// @returns a binary message with a new key.
inline synnax::library::MessageEntry binary_message(
    const std::string &name,
    std::vector<synnax::library::Field> fields,
    std::optional<synnax::library::Identifier> identifier = std::nullopt
) {
    synnax::library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = name;
    m.fields = std::move(fields);
    m.identifier = std::move(identifier);
    return m;
}

/// @returns a text message with a new key, split on delimiter.
inline synnax::library::MessageEntry text_message(
    const std::string &name,
    std::vector<synnax::library::Field> fields,
    std::optional<synnax::library::Identifier> identifier = std::nullopt,
    const std::string &delimiter = ","
) {
    auto m = binary_message(name, std::move(fields), std::move(identifier));
    m.format = synnax::library::FORMAT_TEXT;
    m.delimiter = delimiter;
    return m;
}

/// @returns the key of the field.
inline synnax::library::FieldKey key(const synnax::library::Field &field) {
    return std::visit([](const auto &f) { return f.key; }, field);
}

/// @returns a library holding messages.
inline synnax::library::Library
library(std::vector<synnax::library::MessageEntry> messages) {
    synnax::library::Library lib;
    lib.key = x::uuid::create();
    lib.name = "test";
    for (auto &m: messages)
        lib.entries.emplace_back(std::move(m));
    return lib;
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

/// @brief the state an in-memory Transport shares with the test that scripts it.
struct Wire {
    std::mutex mu;
    std::condition_variable cv;
    /// @brief chunks the transport returns from read, in order.
    std::deque<std::vector<std::uint8_t>> reads;
    /// @brief the error the next read returns instead of a chunk.
    x::errors::Error read_err;
    /// @brief errors the opener returns, in order, before it succeeds.
    std::deque<x::errors::Error> open_errs;
    /// @brief every write, in order.
    std::vector<std::vector<std::uint8_t>> writes;
    /// @brief called with each write while mu is held. It can queue a reply.
    std::function<void(Wire &, std::span<const std::uint8_t>)> on_write;
    /// @brief the number of successful opens.
    int opens = 0;
    /// @brief the number of transports closed.
    int closes = 0;

    /// @brief queues a chunk for read.
    void push(const std::vector<std::uint8_t> &chunk) {
        {
            std::lock_guard lock(this->mu);
            this->reads.push_back(chunk);
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
        this->wire->cv.wait_for(lock, timeout.chrono(), [this] {
            return !this->wire->reads.empty() || this->wire->read_err;
        });
        if (this->wire->read_err) {
            auto err = this->wire->read_err;
            this->wire->read_err = x::errors::NIL;
            return {{}, err};
        }
        if (this->wire->reads.empty()) return {{}, x::errors::NIL};
        this->last = std::move(this->wire->reads.front());
        this->wire->reads.pop_front();
        return {
            transport::Chunk{.data = this->last, .time = x::telem::TimeStamp::now()},
            x::errors::NIL,
        };
    }

    x::errors::Error
    write(const std::span<const std::uint8_t> data, const x::telem::TimeSpan) override {
        {
            std::lock_guard lock(this->wire->mu);
            this->wire->writes.emplace_back(data.begin(), data.end());
            if (this->wire->on_write) this->wire->on_write(*this->wire, data);
        }
        this->wire->cv.notify_all();
        return x::errors::NIL;
    }
};

/// @returns an Opener of Transports on wire. It fails with each of wire->open_errs
/// before it succeeds.
inline Opener opener(const std::shared_ptr<Wire> &wire) {
    return [wire]() -> std::pair<std::unique_ptr<bus::Transport>, x::errors::Error> {
        std::lock_guard lock(wire->mu);
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

/// @brief resolves a read config over messages. Message i is stamped on index channel
/// 100 + i, and its fields map in order to FLOAT64 channels numbered from 1.
/// @param edit changes the config before it resolves.
/// @param channels channels to add to the generated ones, such as a raw channel.
inline ReadConfig read_config(
    const std::vector<synnax::library::MessageEntry> &messages,
    const std::optional<::synnax::bus::Framing> &framing,
    const std::function<void(::synnax::bus::ReadConfig &)> &edit = nullptr,
    const ::synnax::bus::PollConfig &poll = {},
    std::vector<synnax::channel::Channel> channels = {}
) {
    ::synnax::bus::ReadConfig cfg;
    cfg.device = "dev";
    synnax::channel::Key next = 1;
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto index = static_cast<synnax::channel::Key>(100 + i);
        channels.push_back(index_channel(index));
        ::synnax::bus::ReadMessage rm{.message = messages[i].key, .index = index};
        for (const auto &f: messages[i].fields) {
            channels.push_back(data_channel(next, index));
            rm.fields.push_back({.field = key(f), .channel = next++});
        }
        cfg.messages.push_back(rm);
    }
    if (edit) edit(cfg);
    x::json::Parser parser(x::json::json::object());
    auto out = ReadConfig::resolve(
        parser,
        cfg,
        poll,
        framing,
        library(messages),
        channels
    );
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    return out;
}

/// @brief resolves a write config over one message.
/// @param bindings maps each command channel to the index of the field it drives.
inline WriteConfig write_config(
    const synnax::library::MessageEntry &message,
    const std::vector<std::pair<synnax::channel::Key, std::size_t>> &bindings,
    const std::optional<::synnax::bus::Framing> &framing
) {
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    ::synnax::bus::WriteMessage wm{.message = message.key};
    std::vector<synnax::channel::Channel> channels;
    for (const auto &[ch, field]: bindings) {
        wm.fields.push_back({.field = key(message.fields[field]), .channel = ch});
        channels.push_back(data_channel(ch, 0));
    }
    cfg.messages = {wm};
    x::json::Parser parser(x::json::json::object());
    auto out = WriteConfig::resolve(parser, cfg, framing, library({message}), channels);
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    return out;
}

/// @returns every value of the channel across frames, in order.
inline std::vector<double>
values(const std::vector<x::telem::Frame> &frames, const synnax::channel::Key key) {
    std::vector<double> out;
    for (const auto &fr: frames)
        for (const auto &[k, series]: fr) {
            if (k != key) continue;
            for (std::size_t i = 0; i < series.size(); i++)
                out.push_back(series.at<double>(static_cast<int>(i)));
        }
    return out;
}
}
