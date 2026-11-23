// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cmath>
#include <cstddef>
#include <map>
#include <optional>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"
#include "driver/codec/text.h"
#include "driver/errors/errors.h"
#include "driver/icd/writer.h"

namespace driver::icd {
namespace {
namespace icd = synnax::icd;

std::string text(const double v) {
    return x::json::Number(v).to_string();
}

/// @returns VALIDATION when v, a decoded value, is outside min and max.
x::errors::Error check_limits(
    const double v,
    const std::optional<double> &min,
    const std::optional<double> &max
) {
    if (std::isnan(v) && (min.has_value() || max.has_value()))
        return x::errors::Error(
            x::errors::VALIDATION,
            "NaN is not within the min and max"
        );
    if (min.has_value() && v < *min)
        return x::errors::Error(
            x::errors::VALIDATION,
            text(v) + " is below the min of " + text(*min)
        );
    if (max.has_value() && v > *max)
        return x::errors::Error(
            x::errors::VALIDATION,
            text(v) + " is above the max of " + text(*max)
        );
    return x::errors::NIL;
}

x::errors::Error config_error(const std::string &message) {
    return x::errors::Error(driver::errors::CONFIGURATION_ERROR, message);
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
            config_error(
                "message " + wm.message.to_string() + " does not exist in ICD " +
                ic.name
            ),
        };
    const auto refuse = [&msg](const std::string &message) {
        return std::pair{
            Writer{},
            config_error("message " + msg->name + ": " + message)
        };
    };
    const auto *text_payload = std::get_if<icd::TextPayload>(&msg->payload);
    if (text_payload == nullptr) return refuse("the Driver writes only text messages");
    if (std::ranges::none_of(wm.bindings, [](const icd::WriteBinding &b) {
            return std::holds_alternative<icd::ChannelWriteBinding>(b);
        }))
        return refuse("binds no channels");
    auto payload = *text_payload;
    std::map<icd::FieldKey, icd::TextField *> fields;
    for (auto &f: payload.fields)
        fields[icd::base(f).key] = &f;
    std::map<icd::FieldKey, int> counts;
    for (const auto &b: wm.bindings)
        counts[icd::base(b).field]++;
    Writer w;
    w.name = msg->name;
    codec::TextMessage encoded{.name = msg->name};
    for (const auto &b: wm.bindings) {
        const auto found = fields.find(icd::base(b).field);
        if (const auto *cb = std::get_if<icd::ChannelWriteBinding>(&b)) {
            const auto ch = channels.find(cb->channel);
            if (ch == channels.end())
                return refuse(
                    "channel " + std::to_string(cb->channel) + " does not exist"
                );
            Binding binding{.channel = cb->channel};
            if (found != fields.end()) {
                const auto &base = icd::base(*found->second);
                binding.field = base.name;
                binding.min = base.min;
                binding.max = base.max;
            }
            w.bindings.push_back(binding);
            w.last.emplace_back(ch->second.data_type, 1);
            encoded.bindings.push_back(
                {.field = cb->field, .data_type = ch->second.data_type}
            );
            continue;
        }
        const auto &vb = std::get<icd::ValueWriteBinding>(b);
        if (found == fields.end())
            return refuse("field " + vb.field.to_string() + " does not exist");
        auto &f = *found->second;
        const auto &field = icd::base(f).name;
        if (counts[vb.field] > 1)
            return refuse("field " + field + " has more than one binding");
        const auto *value = std::get_if<icd::FloatWriteValue>(&vb.value);
        if (value == nullptr)
            return refuse(
                "field " + field + " of kind float needs a value of its kind"
            );
        std::visit([&](auto &v) { v.default_ = value->value; }, f);
    }
    for (const auto &f: payload.fields) {
        const auto &b = icd::base(f);
        if (!b.default_.has_value()) continue;
        const double decoded = b.default_->value * b.scale + b.offset;
        if (const auto err = check_limits(decoded, b.min, b.max))
            return refuse("field " + b.name + ": default: " + err.data);
    }
    encoded.payload = std::move(payload);
    auto [encoder, err] = codec::TextEncoder::create(encoded);
    if (err) return {Writer{}, err};
    w.encoder = std::move(encoder);
    w.rows.resize(w.bindings.size());
    return {std::move(w), x::errors::NIL};
}

x::errors::Error
Writer::check(const std::size_t binding, const x::telem::SampleValue &v) {
    const auto &b = this->bindings[binding];
    if (auto err = check_limits(x::telem::cast<double>(v), b.min, b.max))
        return x::errors::Error(err, "field " + b.field + ": " + err.data);
    return this->encoder.check(binding, v);
}

bool Writer::ready() const {
    for (std::size_t b = 0; b < this->last.size(); b++)
        if (this->last[b].empty() && !this->encoder.has_default(b)) return false;
    return true;
}

Writer::Result Writer::write(const x::telem::Frame &fr, const Send &send) {
    std::ranges::fill(this->rows, nullptr);
    std::size_t n = 0;
    for (const auto &[key, s]: fr)
        for (std::size_t b = 0; b < this->bindings.size(); b++)
            if (this->bindings[b].channel == key) {
                this->rows[b] = &s;
                n = std::max(n, s.size());
            }
    Result result;
    for (std::size_t row = 0; row < n; row++) {
        bool good = false;
        for (std::size_t b = 0; b < this->rows.size(); b++) {
            const auto *s = this->rows[b];
            if (s == nullptr || row >= s->size()) continue;
            const auto v = s->at(static_cast<int>(row));
            if (auto err = this->check(b, v)) {
                if (!result.refused)
                    result.refused = x::errors::Error(
                        err,
                        "message " + this->name + ": " + err.data
                    );
                continue;
            }
            this->last[b].clear();
            this->last[b].write(v);
            good = true;
        }
        if (!good || !this->ready()) continue;
        if (auto err = this->encoder.encode(this->last, this->payload)) {
            result.error = err;
            return result;
        }
        if (auto err = send(this->payload)) {
            result.error = err;
            return result;
        }
    }
    return result;
}
}
