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
#include "driver/codec/text.h"
#include "driver/errors/errors.h"
#include "driver/icd/writer.h"

namespace driver::icd {
std::pair<Writer, x::errors::Error> Writer::create(
    const synnax::icd::WriteMessage &wm,
    const synnax::icd::ICD &ic,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
) {
    const auto msg = std::ranges::find(
        ic.messages,
        wm.message,
        &synnax::icd::Message::key
    );
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
    const auto *text_payload = std::get_if<synnax::icd::TextPayload>(&msg->payload);
    if (text_payload == nullptr) return refuse("the Driver writes only text messages");
    if (std::ranges::none_of(wm.bindings, [](const synnax::icd::WriteBinding &b) {
            return std::holds_alternative<synnax::icd::ChannelWriteBinding>(b);
        }))
        return refuse("binds no channels");
    auto payload = *text_payload;
    std::map<synnax::icd::FieldKey, synnax::icd::TextField *> fields;
    for (auto &f: payload.fields)
        fields[synnax::icd::base(f).key] = &f;
    std::map<synnax::icd::FieldKey, int> counts;
    for (const auto &b: wm.bindings)
        counts[synnax::icd::base(b).field]++;
    Writer w;
    w.name = msg->name;
    codec::TextMessage encoded{.name = msg->name};
    for (const auto &b: wm.bindings) {
        if (const auto *cb = std::get_if<synnax::icd::ChannelWriteBinding>(&b)) {
            const auto ch = channels.find(cb->channel);
            if (ch == channels.end())
                return refuse(
                    "channel " + std::to_string(cb->channel) + " does not exist"
                );
            w.channels.push_back(cb->channel);
            w.last.emplace_back(ch->second.data_type, 1);
            encoded.bindings.push_back(
                {.field = cb->field, .data_type = ch->second.data_type}
            );
            continue;
        }
        const auto &vb = std::get<synnax::icd::ValueWriteBinding>(b);
        const auto found = fields.find(vb.field);
        if (found == fields.end())
            return refuse("field " + vb.field.to_string() + " does not exist");
        auto &f = *found->second;
        const auto &field = synnax::icd::base(f).name;
        if (counts[vb.field] > 1)
            return refuse("field " + field + " has more than one binding");
        const auto *value = std::get_if<synnax::icd::FloatWriteValue>(&vb.value);
        if (value == nullptr)
            return refuse(
                "field " + field + " of kind float needs a value of its kind"
            );
        std::visit([&](auto &v) { v.default_ = value->value; }, f);
        if (const auto err = codec::check_raw_limits(
                synnax::icd::base(f),
                value->value.value
            ))
            return refuse("field " + field + ": " + err.data);
    }
    encoded.payload = std::move(payload);
    auto [encoder, err] = codec::TextEncoder::create(encoded);
    if (err) return {Writer{}, err};
    w.encoder = std::move(encoder);
    w.commands.resize(w.channels.size());
    return {std::move(w), x::errors::NIL};
}

bool Writer::ready() const {
    for (std::size_t b = 0; b < this->last.size(); b++)
        if (this->last[b].empty() && !this->encoder.has_default(b)) return false;
    return true;
}

std::optional<x::telem::SampleValue>
Writer::command(const std::size_t binding, std::size_t i) const {
    for (const auto *s: this->commands[binding]) {
        if (i < s->size()) return s->at(static_cast<int>(i));
        i -= s->size();
    }
    return std::nullopt;
}

Writer::Result Writer::write(const x::telem::Frame &fr, const Send &send) {
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
    Result result;
    for (std::size_t row = 0; row < n; row++) {
        bool good = false;
        for (std::size_t b = 0; b < this->channels.size(); b++) {
            const auto v = this->command(b, row);
            if (!v.has_value()) continue;
            if (auto err = this->encoder.check(b, *v)) {
                if (!result.refused)
                    result.refused = x::errors::Error(
                        err,
                        "message " + this->name + ": " + err.data
                    );
                continue;
            }
            this->last[b].clear();
            this->last[b].write(*v);
            good = true;
        }
        if (result.error || !good || !this->ready()) continue;
        if (auto err = this->encoder.encode(this->last, this->payload))
            result.error = err;
        else if (auto send_err = send(this->payload))
            result.error = send_err;
    }
    return result;
}
}
