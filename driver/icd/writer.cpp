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
#include <string>
#include <tuple>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"
#include "driver/codec/field.h"
#include "driver/codec/json.h"
#include "driver/codec/text.h"
#include "driver/errors/errors.h"
#include "driver/icd/writer.h"

namespace driver::icd {
namespace {
namespace icd = synnax::icd;

/// @brief sets the default of text field f to v. It returns the problem with v, to
/// follow the name of f, or nothing.
std::optional<std::string> set_value(icd::TextField &f, const icd::WriteValue &v) {
    const auto *value = std::get_if<icd::FloatWriteValue>(&v);
    if (value == nullptr) return " of kind float needs a value of its kind";
    std::visit([&](auto &t) { t.default_ = value->value; }, f);
    if (const auto err = codec::check_raw_limits(icd::base(f), value->value.value))
        return ": " + err.data;
    return std::nullopt;
}

/// @brief sets the default of binary field f to v. It returns the problem with v, to
/// follow the name of f, or nothing.
std::optional<std::string> set_value(icd::BinaryField &f, const icd::WriteValue &v) {
    return std::visit(
        [](auto &field, const auto &value) -> std::optional<std::string> {
            using F = std::decay_t<decltype(field)>;
            using V = std::decay_t<decltype(value)>;
            if constexpr (
                (std::is_same_v<F, icd::FloatBinaryField> &&
                 std::is_same_v<V, icd::FloatWriteValue>) ||
                (std::is_same_v<F, icd::IntegerBinaryField> &&
                 std::is_same_v<V, icd::IntegerWriteValue>) ||
                (std::is_same_v<F, icd::UnsignedBinaryField> &&
                 std::is_same_v<V, icd::UnsignedWriteValue>)
            ) {
                field.default_ = value.value;
                const auto raw = static_cast<double>(value.value.value);
                if (const auto err = codec::check_raw_limits(field, raw))
                    return ": " + err.data;
                return std::nullopt;
            } else
                return " of kind " + field.kind + " needs a value of its kind";
        },
        f,
        v
    );
}

/// @brief sets the default of field to value, and returns false when value is not of
/// the kind of field.
template<typename Field, typename Value>
bool set_default(Field &field, const Value &value) {
    if constexpr (
        std::is_same_v<Field, icd::TimestampJSONField> &&
        std::is_same_v<Value, icd::TimestampWriteValue>
    )
        field.default_ = icd::ConstantTimestampDefault{.value = value.value};
    else if constexpr (
        std::is_same_v<Field, icd::UuidJSONField> &&
        std::is_same_v<Value, icd::UuidWriteValue>
    )
        field.default_ = icd::ConstantUUIDDefault{.value = value.value};
    else if constexpr (requires { field.default_ = value.value; }) {
        // Kinds that share a C++ type, such as string and JSON, differ only by name.
        if (field.type != value.type) return false;
        field.default_ = value.value;
    } else
        return false;
    return true;
}

/// @brief sets the default of JSON field f to v. It returns the problem with v, to
/// follow the name of f, or nothing.
std::optional<std::string> set_value(icd::JSONField &f, const icd::WriteValue &v) {
    return std::visit(
        [](auto &field, const auto &value) -> std::optional<std::string> {
            if (!set_default(field, value))
                return " of kind " + field.type + " needs a value of its kind";
            using F = std::decay_t<decltype(field)>;
            if constexpr (
                std::is_same_v<F, icd::FloatJSONField> ||
                std::is_same_v<F, icd::IntegerJSONField> ||
                std::is_same_v<F, icd::UnsignedJSONField>
            ) {
                const auto raw = static_cast<double>(field.default_.value().value);
                if (const auto err = codec::check_raw_limits(field, raw))
                    return ": " + err.data;
            }
            return std::nullopt;
        },
        f,
        v
    );
}

/// @brief sets the value of each value binding of wm as the default of its field in
/// fields. It returns the problem, or nothing.
template<typename Field>
std::optional<std::string>
set_values(std::vector<Field> &fields, const icd::WriteMessage &wm) {
    std::map<icd::FieldKey, int> counts;
    for (const auto &b: wm.bindings)
        counts[icd::base(b).field]++;
    for (const auto &b: wm.bindings) {
        const auto *vb = std::get_if<icd::ValueWriteBinding>(&b);
        if (vb == nullptr) continue;
        const auto f = std::ranges::find(fields, vb->field, [](const Field &f) {
            return icd::base(f).key;
        });
        if (f == fields.end())
            return "field " + vb->field.to_string() + " does not exist";
        const auto &name = icd::base(*f).name;
        if (counts[vb->field] > 1)
            return "field " + name + " has more than one binding";
        if (auto problem = set_value(*f, vb->value)) return "field " + name + *problem;
    }
    return std::nullopt;
}

/// @brief replaces the sample of last with sample i of s, a series of the data type of
/// last.
void set_last(x::telem::Series &last, const x::telem::Series &s, const std::size_t i) {
    last.clear();
    if (s.data_type().is_variable()) {
        last.write(s.at<std::string>(static_cast<int>(i)));
        return;
    }
    const auto *bytes = reinterpret_cast<const std::uint8_t *>(s.data());
    last.write(bytes + (i * s.data_type().density()), 1);
}
}

std::pair<Writer, x::errors::Error> Writer::create(
    const icd::WriteMessage &wm,
    const icd::ICD &ic,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
) {
    const auto msg = std::ranges::find(ic.messages, wm.message, &icd::Message::key);
    if (msg == ic.messages.end())
        return {
            Writer{},
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "message " + wm.message.to_string() + " does not exist in ICD " +
                    ic.name
            ),
        };
    const auto refuse = [&msg](const std::string &message) {
        return std::pair{
            Writer{},
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "message " + msg->name + ": " + message
            ),
        };
    };
    if (std::ranges::none_of(wm.bindings, [](const icd::WriteBinding &b) {
            return std::holds_alternative<icd::ChannelWriteBinding>(b);
        }))
        return refuse("binds no channels");
    Writer w;
    std::vector<codec::Binding> bindings;
    for (const auto &b: wm.bindings) {
        const auto *cb = std::get_if<icd::ChannelWriteBinding>(&b);
        if (cb == nullptr) continue;
        const auto ch = channels.find(cb->channel);
        if (ch == channels.end())
            return refuse("channel " + std::to_string(cb->channel) + " does not exist");
        w.channels.push_back(cb->channel);
        w.last.emplace_back(ch->second.data_type, 1);
        bindings.push_back({.field = cb->field, .data_type = ch->second.data_type});
    }
    std::optional<std::string> problem;
    x::errors::Error err;
    const auto build = [&](auto m, const auto create) {
        problem = set_values(m.payload.fields, wm);
        if (!problem) std::tie(w.encoder, err) = create(m);
    };
    std::visit(
        [&]<typename Payload>(const Payload &payload) {
            if constexpr (std::is_same_v<Payload, icd::BinaryPayload>)
                build(
                    codec::BinaryMessage{
                        .name = msg->name,
                        .payload = payload,
                        .bindings = std::move(bindings),
                    },
                    codec::BinaryEncoder::create
                );
            else if constexpr (std::is_same_v<Payload, icd::TextPayload>)
                build(
                    codec::TextMessage{
                        .name = msg->name,
                        .payload = payload,
                        .bindings = std::move(bindings),
                    },
                    codec::TextEncoder::create
                );
            else {
                static_assert(std::is_same_v<Payload, icd::JSONPayload>);
                build(
                    codec::JSONMessage{
                        .name = msg->name,
                        .payload = payload,
                        .bindings = std::move(bindings),
                    },
                    codec::JSONEncoder::create
                );
            }
        },
        msg->payload
    );
    if (problem) return refuse(*problem);
    if (err) return {Writer{}, err};
    w.commands.resize(w.channels.size());
    return {std::move(w), x::errors::NIL};
}

std::optional<Writer::Command>
Writer::command(const std::size_t binding, std::size_t i) const {
    for (const auto *s: this->commands[binding]) {
        if (i < s->size()) return Command{.series = s, .index = i};
        i -= s->size();
    }
    return std::nullopt;
}

std::size_t Writer::gather(const x::telem::Frame &fr) {
    for (auto &series: this->commands)
        series.clear();
    for (const auto &[key, s]: fr)
        for (std::size_t b = 0; b < this->channels.size(); b++)
            if (this->channels[b] == key) this->commands[b].push_back(&s);
    std::size_t n = 0;
    for (const auto &series: this->commands) {
        std::size_t size = 0;
        for (const auto *s: series)
            size += s->size();
        n = std::max(n, size);
    }
    return n;
}

bool Writer::apply(const std::size_t row, x::errors::Error &refused) {
    bool good = false;
    for (std::size_t b = 0; b < this->channels.size(); b++) {
        const auto cmd = this->command(b, row);
        if (!cmd.has_value()) continue;
        if (auto err = std::visit(
                [&](auto &e) { return e.check(b, *cmd->series, cmd->index); },
                this->encoder
            )) {
            if (!refused) refused = err;
            continue;
        }
        set_last(this->last[b], *cmd->series, cmd->index);
        good = true;
    }
    return good;
}

Writer::Result Writer::write(const x::telem::Frame &fr, const Send &send) {
    const auto rows = this->gather(fr);
    Result result;
    for (std::size_t row = 0; row < rows; row++)
        if (this->apply(row, result.refused) && !result.error)
            result.error = this->encode(send);
    return result;
}

x::errors::Error Writer::keep(const x::telem::Frame &fr) {
    const auto rows = this->gather(fr);
    x::errors::Error refused;
    for (std::size_t row = 0; row < rows; row++)
        this->apply(row, refused);
    return refused;
}

x::errors::Error Writer::encode(const Send &send) {
    if (!std::visit([this](auto &e) { return e.ready(this->last); }, this->encoder))
        return x::errors::NIL;
    if (auto err = std::visit(
            [&](auto &e) { return e.encode(this->last, this->payload); },
            this->encoder
        ))
        return err;
    return send(this->payload);
}
}
