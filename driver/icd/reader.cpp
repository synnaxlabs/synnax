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
#include <cstdint>
#include <map>
#include <optional>
#include <span>
#include <string>
#include <string_view>
#include <type_traits>
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

#include "driver/codec/binary.h"
#include "driver/codec/json.h"
#include "driver/codec/text.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"

namespace driver::icd {
namespace {
/// @brief maps each channel that a read writes to the message that writes it.
using Writers = std::unordered_map<synnax::channel::Key, std::string>;

/// @brief records that the message name writes to key, which no other binding may.
x::errors::Error
claim(Writers &writers, const synnax::channel::Key key, const std::string &name) {
    const auto [other, claimed] = writers.try_emplace(key, name);
    if (claimed) return x::errors::NIL;
    const auto ch = std::to_string(key);
    if (other->second == name)
        return x::errors::Error(
            driver::errors::CONFIGURATION_ERROR,
            "message " + name + " writes to channel " + ch + " twice"
        );
    return x::errors::Error(
        driver::errors::CONFIGURATION_ERROR,
        "messages " + other->second + " and " + name + " write to the same channel " +
            ch
    );
}

/// @brief reports whether the field f, a variant of field kinds, takes a default.
template<typename Field>
bool has_default(const Field &f) {
    return std::visit([](const auto &v) { return v.default_.has_value(); }, f);
}

/// @brief returns the name of the field of payload with key, or key when payload has no
/// such field.
template<typename Payload>
std::string field_name(const Payload &payload, const synnax::icd::FieldKey &key) {
    for (const auto &f: payload.fields)
        if (const auto &b = synnax::icd::base(f); b.key == key) return b.name;
    return key.to_string();
}

/// @brief rejects an index field that takes a default, since each payload that leaves
/// it out would then write the same timestamp.
template<typename Payload>
x::errors::Error
check_index(const synnax::icd::ReadMessage &rm, const Payload &payload) {
    if (!rm.index.has_value()) return x::errors::NIL;
    for (const auto &f: payload.fields) {
        const auto &b = synnax::icd::base(f);
        if (b.key != *rm.index) continue;
        if (has_default(f))
            return x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "the index field " + b.name + " cannot take a default"
            );
    }
    return x::errors::NIL;
}

/// @brief Selection is the multiplexor and its values that select a binary field, or
/// nothing when every frame of its message holds a sample for the field.
using Selection = std::optional<
    std::pair<synnax::icd::FieldKey, std::vector<std::int32_t>>>;

Selection selection(const synnax::icd::BinaryField &f) {
    const auto &b = synnax::icd::base(f);
    if (!b.multiplexor.has_value() || has_default(f)) return std::nullopt;
    auto values = b.multiplex_values;
    std::ranges::sort(values);
    return std::pair{*b.multiplexor, std::move(values)};
}

/// @brief rejects two bound fields on one index channel that different multiplexor
/// values select, since a frame would then write one without the other. groups holds
/// the index group of each binding of rm, and indexes the key of each index group.
x::errors::Error check_selections(
    const synnax::icd::ReadMessage &rm,
    const synnax::icd::BinaryPayload &payload,
    const std::vector<std::optional<std::size_t>> &groups,
    const std::vector<synnax::channel::Key> &indexes
) {
    std::map<std::size_t, std::pair<const synnax::icd::BinaryField *, Selection>> first;
    for (std::size_t k = 0; k < rm.bindings.size(); k++) {
        const auto group = groups[k];
        if (!group.has_value()) continue;
        const auto f = std::ranges::find(
            payload.fields,
            rm.bindings[k].field,
            [](const synnax::icd::BinaryField &v) { return synnax::icd::base(v).key; }
        );
        if (f == payload.fields.end()) continue;
        auto sel = selection(*f);
        const auto [seen, added] = first.try_emplace(*group, &*f, sel);
        if (added || seen->second.second == sel) continue;
        return x::errors::Error(
            driver::errors::CONFIGURATION_ERROR,
            "fields " + synnax::icd::base(*seen->second.first).name + " and " +
                synnax::icd::base(*f).name + " share index channel " +
                std::to_string(indexes[*group]) +
                ", but different multiplexor values select them"
        );
    }
    return x::errors::NIL;
}

/// @brief returns the series of fr for key, after adding an empty series of dt that
/// holds cap samples when fr has none.
x::telem::Series &series(
    x::telem::Frame &fr,
    const synnax::channel::Key key,
    const x::telem::DataType &dt,
    const std::size_t cap
) {
    for (std::size_t i = 0; i < fr.size(); i++)
        if ((*fr.channels)[i] == key) return (*fr.series)[i];
    fr.emplace(key, x::telem::Series(dt, cap));
    return fr.series->back();
}

/// @brief Decoded holds the codec messages of each format of a reader.
struct Decoded {
    std::vector<codec::BinaryMessage> binary;
    std::vector<codec::TextMessage> text;
    std::vector<codec::JSONMessage> json;
};
}

std::pair<std::vector<codec::Binding>, x::errors::Error> Reader::bind(
    const synnax::icd::ReadMessage &rm,
    const synnax::icd::Message &msg,
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
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "channel " + std::to_string(b.channel) + " does not exist"
                ),
            };
        const auto &ch = found->second;
        bindings.push_back({.field = b.field, .data_type = ch.data_type});
        m.channels.push_back(ch.key);
        const bool index_field = rm.index == b.field;
        if (index_field && !ch.is_index)
            return {
                {},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "the index field must write to an index channel"
                ),
            };
        if (!index_field && ch.is_index) {
            const auto name = std::visit(
                [&](const auto &p) { return field_name(p, b.field); },
                msg.payload
            );
            return {
                {},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "field " + name + " cannot write to index channel " + ch.name +
                        ", because only the index field can. Bind field " + name +
                        " to a data channel"
                ),
            };
        }
        if (index_field) m.index_field = i;
        const auto key = index_field ? ch.key : ch.index;
        if (key == 0) {
            m.groups.emplace_back();
            continue;
        }
        auto idx = std::ranges::find(m.indexes, key, &Index::key);
        if (idx == m.indexes.end()) idx = m.indexes.insert(idx, Index{.key = key});
        if (index_field) idx->field = i;
        m.groups.emplace_back(idx - m.indexes.begin());
    }
    if (rm.index.has_value() && !m.index_field.has_value())
        return {
            {},
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "the index field has no channel"
            ),
        };
    return {std::move(bindings), x::errors::NIL};
}

std::pair<Reader, x::errors::Error> Reader::create(
    const std::vector<synnax::icd::ReadMessage> &reads,
    const synnax::icd::ICD &ic,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
) {
    Reader r;
    Decoded decoded;
    Writers writers;
    const synnax::icd::Message *first = nullptr;
    for (const auto &rm: reads) {
        const auto msg = std::ranges::find(
            ic.messages,
            rm.message,
            &synnax::icd::Message::key
        );
        if (msg == ic.messages.end())
            return {
                Reader{},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "message " + rm.message.to_string() + " does not exist in ICD " +
                        ic.name
                ),
            };
        const auto &name = msg->name;
        if (first == nullptr) first = &*msg;
        if (first->payload.index() != msg->payload.index())
            return {
                Reader{},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "messages " + first->name + " and " + name +
                        " have different formats"
                ),
            };
        if (rm.bindings.empty())
            return {
                Reader{},
                x::errors::Error(
                    driver::errors::CONFIGURATION_ERROR,
                    "message " + name + " binds no fields"
                ),
            };
        Message m;
        auto [bindings, err] = bind(rm, *msg, channels, m);
        if (!err)
            err = std::visit(
                [&](const auto &p) { return check_index(rm, p); },
                msg->payload
            );
        if (const auto *p = std::get_if<synnax::icd::BinaryPayload>(&msg->payload);
            p != nullptr && !err) {
            std::vector<synnax::channel::Key> keys;
            keys.reserve(m.indexes.size());
            for (const auto &idx: m.indexes)
                keys.push_back(idx.key);
            err = check_selections(rm, *p, m.groups, keys);
        }
        if (err) return {Reader{}, err.wrap("message " + name)};
        for (const auto &idx: m.indexes)
            if (auto claim_err = claim(writers, idx.key, name))
                return {Reader{}, claim_err};
        for (std::size_t k = 0; k < m.channels.size(); k++) {
            if (k == m.index_field) continue;
            if (auto claim_err = claim(writers, m.channels[k], name))
                return {Reader{}, claim_err};
        }
        std::visit(
            [&]<typename P>(const P &p) {
                if constexpr (std::is_same_v<P, synnax::icd::BinaryPayload>)
                    decoded.binary.push_back(
                        {.name = name, .payload = p, .bindings = std::move(bindings)}
                    );
                else if constexpr (std::is_same_v<P, synnax::icd::TextPayload>)
                    decoded.text.push_back(
                        {.name = name, .payload = p, .bindings = std::move(bindings)}
                    );
                else
                    decoded.json.push_back(
                        {.name = name, .payload = p, .bindings = std::move(bindings)}
                    );
            },
            msg->payload
        );
        r.messages.push_back(std::move(m));
    }
    x::errors::Error err;
    if (!decoded.binary.empty()) {
        auto [d, d_err] = codec::Binary::create(decoded.binary);
        r.decoder = std::move(d);
        err = d_err;
    } else if (!decoded.text.empty()) {
        auto [d, d_err] = codec::Text::create(decoded.text);
        r.decoder = std::move(d);
        err = d_err;
    } else {
        auto [d, d_err] = codec::JSON::create(decoded.json);
        r.decoder = std::move(d);
        err = d_err;
    }
    if (err) return {Reader{}, err};
    return {std::move(r), x::errors::NIL};
}

std::vector<synnax::channel::Key> Reader::channels() const {
    std::vector<synnax::channel::Key> keys;
    for (const auto &m: this->messages)
        for (std::size_t k = 0; k < m.channels.size(); k++)
            if (k != m.index_field) keys.push_back(m.channels[k]);
    for (const auto &m: this->messages)
        for (const auto &idx: m.indexes)
            keys.push_back(idx.key);
    return keys;
}

std::pair<std::optional<std::size_t>, x::errors::Error> Reader::read(
    const std::span<const std::uint8_t> payload,
    const x::telem::TimeStamp received,
    x::telem::Frame &fr,
    const std::size_t cap,
    const std::optional<double> fill
) {
    const auto [i, err] = std::visit(
        [&]<typename D>(D &d) {
            if constexpr (std::is_same_v<D, codec::Binary>)
                return d.decode(payload);
            else
                return d.decode(
                    std::string_view(
                        reinterpret_cast<const char *>(payload.data()),
                        payload.size()
                    )
                );
        },
        this->decoder
    );
    this->failed.clear();
    if (err || !i.has_value()) return {std::nullopt, err};
    const auto msg = *i;
    const auto &m = this->messages[msg];
    const auto &samples = std::visit(
        [msg](const auto &d) -> const std::vector<x::telem::Series> & {
            return d.samples(msg);
        },
        this->decoder
    );
    this->filled.assign(samples.size(), false);
    if (const auto *json = std::get_if<codec::JSON>(&this->decoder); json != nullptr)
        for (const auto &f: json->failures(msg)) {
            this->failed.push_back(
                {.channel = m.channels[f.binding],
                 .missing = f.missing,
                 .error = f.error}
            );
            if (const auto dt = samples[f.binding].data_type();
                fill.has_value() &&
                (dt == x::telem::FLOAT32_T || dt == x::telem::FLOAT64_T))
                this->filled[f.binding] = true;
        }
    this->present.assign(m.indexes.size(), true);
    for (std::size_t k = 0; k < samples.size(); k++)
        if (const auto group = m.groups[k];
            samples[k].empty() && !this->filled[k] && group.has_value())
            this->present[*group] = false;
    for (std::size_t g = 0; g < m.indexes.size(); g++) {
        if (!this->present[g]) continue;
        const auto &[key, field] = m.indexes[g];
        series(fr, key, x::telem::TIMESTAMP_T, cap)
            .write(
                field.has_value() ? samples[*field].at<x::telem::TimeStamp>(0)
                                  : received
            );
    }
    for (std::size_t k = 0; k < samples.size(); k++) {
        if (k == m.index_field) continue;
        if (const auto group = m.groups[k]; group.has_value() && !this->present[*group])
            continue;
        const auto &dt = samples[k].data_type();
        if (fill.has_value() && this->filled[k])
            series(fr, m.channels[k], dt, cap).write(dt.cast(*fill));
        else if (!samples[k].empty())
            series(fr, m.channels[k], dt, cap).write(samples[k]);
    }
    return {msg, x::errors::NIL};
}
}
