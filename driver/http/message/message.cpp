// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/json.h"
#include "driver/errors/errors.h"
#include "driver/http/message/message.h"

namespace driver::http::message {
namespace {
x::errors::Error invalid(const std::string &data) {
    return {driver::errors::CONFIGURATION_ERROR, data};
}

/// @brief maps each channel that a read writes to the message that writes it.
using Writers = std::unordered_map<synnax::channel::Key, std::string>;

/// @brief records that the message name writes to key, which no other binding may.
x::errors::Error
claim(Writers &writers, const synnax::channel::Key key, const std::string &name) {
    const auto [other, claimed] = writers.try_emplace(key, name);
    if (claimed) return x::errors::NIL;
    const auto ch = std::to_string(key);
    if (other->second == name)
        return invalid("message " + name + " writes to channel " + ch + " twice");
    return invalid(
        "messages " + other->second + " and " + name + " write to the same channel " +
        ch
    );
}

/// @brief rejects an index field that takes a default, since each response that leaves
/// it out would then write the same timestamp.
x::errors::Error check_index(
    const synnax::icd::ReadMessage &rm,
    const synnax::icd::JSONPayload &payload
) {
    if (!rm.index.has_value()) return x::errors::NIL;
    for (const auto &f: payload.fields) {
        const auto *t = std::get_if<synnax::icd::TimestampJSONField>(&f);
        if (t != nullptr && t->key == *rm.index && t->default_.has_value())
            return invalid("the index field " + t->name + " cannot take a default");
    }
    return x::errors::NIL;
}
}

std::pair<std::vector<codec::Binding>, x::errors::Error> Reader::bind(
    const synnax::icd::ReadMessage &rm,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
    Message &m
) {
    std::vector<codec::Binding> bindings;
    bindings.reserve(rm.bindings.size());
    for (std::size_t i = 0; i < rm.bindings.size(); i++) {
        const auto &b = rm.bindings[i];
        const auto found = channels.find(b.channel);
        if (found == channels.end())
            return {
                {},
                invalid("channel " + std::to_string(b.channel) + " does not exist"),
            };
        const auto &ch = found->second;
        bindings.push_back({.field = b.field, .data_type = ch.data_type});
        m.channels.push_back(ch.key);
        if (i == 0)
            m.index = ch.index;
        else if (ch.index != m.index)
            return {{}, invalid("every channel must share one index channel")};
        const bool index_field = rm.index == b.field;
        if (index_field && !ch.is_index)
            return {{}, invalid("the index field must write to an index channel")};
        if (!index_field && ch.is_index)
            return {{}, invalid("only the index field can write to an index channel")};
        if (index_field) m.index_field = i;
    }
    if (rm.index.has_value() && !m.index_field.has_value())
        return {{}, invalid("the index field has no channel")};
    return {std::move(bindings), x::errors::NIL};
}

std::pair<Reader, x::errors::Error> Reader::create(
    const std::vector<synnax::icd::ReadMessage> &reads,
    const synnax::icd::ICD &ic,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
) {
    if (reads.empty()) return {Reader{}, invalid("the endpoint reads no messages")};
    Reader r;
    std::vector<codec::JSONMessage> decoded;
    decoded.reserve(reads.size());
    Writers writers;
    for (const auto &rm: reads) {
        const auto msg = std::ranges::find_if(
            ic.messages,
            [&](const synnax::icd::Message &m) { return m.key == rm.message; }
        );
        if (msg == ic.messages.end())
            return {
                Reader{},
                invalid(
                    "message " + rm.message.to_string() + " does not exist in ICD " +
                    ic.name
                ),
            };
        const auto &name = msg->name;
        const auto *payload = std::get_if<synnax::icd::JSONPayload>(&msg->payload);
        if (payload == nullptr)
            return {Reader{}, invalid("message " + name + " is not a JSON message")};
        if (rm.bindings.empty())
            return {Reader{}, invalid("message " + name + " binds no fields")};
        Message m;
        auto [bindings, err] = bind(rm, channels, m);
        if (!err) err = check_index(rm, *payload);
        if (err)
            return {
                Reader{},
                x::errors::Error(err, "message " + name + ": " + err.data)
            };
        if (m.index != 0)
            if (auto claim_err = claim(writers, m.index, name))
                return {Reader{}, claim_err};
        for (std::size_t k = 0; k < m.channels.size(); k++) {
            if (k == m.index_field) continue;
            if (auto claim_err = claim(writers, m.channels[k], name))
                return {Reader{}, claim_err};
        }
        decoded.push_back({
            .name = name,
            .payload = *payload,
            .bindings = std::move(bindings),
        });
        r.messages.push_back(std::move(m));
    }
    auto [decoder, err] = codec::JSON::create(decoded);
    if (err) return {Reader{}, err};
    r.decoder = std::move(decoder);
    return {std::move(r), x::errors::NIL};
}

x::errors::Error Reader::read(
    const std::string_view body,
    const x::telem::TimeStamp received,
    x::telem::Frame &fr
) {
    const auto [i, err] = this->decoder.decode(body);
    if (err) return err;
    if (!i.has_value())
        return {x::errors::VALIDATION, "the response matches no message"};
    const auto &m = this->messages[*i];
    const auto &samples = this->decoder.samples(*i);
    auto stamp = received;
    if (m.index_field.has_value())
        stamp = samples[*m.index_field].at<x::telem::TimeStamp>(0);
    if (m.index != 0) fr.emplace(m.index, x::telem::Series(stamp));
    for (std::size_t k = 0; k < samples.size(); k++)
        if (k != m.index_field) fr.emplace(m.channels[k], samples[k].deep_copy());
    return x::errors::NIL;
}
}
