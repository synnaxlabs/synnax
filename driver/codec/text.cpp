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
#include <limits>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/text.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/errors.h"
#include "driver/codec/field.h"
#include "driver/codec/text.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;
}

std::pair<Text::Field, x::errors::Error> Text::Field::create(
    const icd::TextField &f,
    const x::telem::DataType &dt,
    const std::string_view delimiter
) {
    const auto &b = icd::base(f);
    Field field{
        .name = b.name,
        .scaled = b.scale != 1 || b.offset != 0,
        .scale = b.scale,
        .offset = b.offset,
        .narrow = dt == x::telem::FLOAT32_T,
    };
    if (const auto *tagged = std::get_if<icd::TaggedTextField>(&f)) {
        if (tagged->tag.empty())
            return {Field{}, x::errors::Error(CONFIG_ERROR, "the tag is empty")};
        if (tagged->tag.contains(delimiter))
            return {
                Field{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "the tag contains the delimiter " + x::json::quote(delimiter)
                ),
            };
        if (x::strings::TRIMMED.contains(tagged->tag.front()))
            return {
                Field{},
                x::errors::Error(CONFIG_ERROR, "the tag starts with whitespace")
            };
        field.tagged = true;
        field.tag = tagged->tag;
    } else
        field.position = std::get<icd::DelimitedTextField>(f).position;
    if (!converts(x::payload::KIND_FLOAT, dt))
        return {
            Field{},
            x::errors::Error(
                CONFIG_ERROR,
                "cannot write float values to " + dt.name() + " channels"
            ),
        };
    if (!b.default_.has_value()) return {std::move(field), x::errors::NIL};
    const double d = b.default_->value;
    auto dst = x::telem::Series(dt, 1);
    if (const auto err = field.append_float(d, dst))
        return {Field{}, as_config(err, "default")};
    field.fallback = std::move(dst);
    return {std::move(field), x::errors::NIL};
}

x::errors::Error Text::Field::append_float(double v, x::telem::Series &dst) const {
    if (this->scaled) v = v * this->scale + this->offset;
    if (!std::isfinite(v))
        return x::errors::Error(
            x::errors::VALIDATION,
            x::json::Number(v).to_string() + " is not finite"
        );
    if (!this->narrow) {
        dst.write(v);
        return x::errors::NIL;
    }
    if (std::abs(v) > std::numeric_limits<float>::max())
        return x::errors::Error(
            x::errors::VALIDATION,
            x::json::Number(v).to_string() + " overflows float32"
        );
    dst.write(static_cast<float>(v));
    return x::errors::NIL;
}

x::errors::Error Text::Field::decode(
    const std::vector<std::string_view> &items,
    x::telem::Series &dst
) const {
    std::string_view value;
    if (this->tagged) {
        for (const auto item: items) {
            const auto t = x::strings::trim(item);
            const auto claimed = [&t](const std::string &l) {
                return t.starts_with(l);
            };
            if (!t.starts_with(this->tag) || std::ranges::any_of(this->longer, claimed))
                continue;
            value = x::strings::trim(t.substr(this->tag.size()));
            break;
        }
    } else if (this->position < items.size())
        value = x::strings::trim(items[this->position]);
    if (value.empty()) {
        if (this->fallback.has_value()) {
            dst.write(*this->fallback);
            return x::errors::NIL;
        }
        return x::errors::Error(
            x::errors::VALIDATION,
            this->tagged ? "no value after tag " + x::json::quote(this->tag)
                         : "no value at position " + std::to_string(this->position)
        );
    }
    const auto [v, err] = x::json::parse_float64(value);
    if (err) return err;
    return this->append_float(v, dst);
}

std::pair<Text, x::errors::Error>
Text::create(const std::vector<TextMessage> &messages) {
    Text t;
    t.messages.reserve(messages.size());
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &m = messages[i];
        if (m.payload.delimiter.empty())
            return {
                Text{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "message " + m.name + ": the delimiter is empty"
                )
            };
        Message msg{.name = m.name, .delimiter = m.payload.delimiter};
        std::map<icd::FieldKey, const icd::TextField *> by_key;
        for (const auto &f: m.payload.fields)
            by_key[icd::base(f).key] = &f;
        for (const auto &b: m.bindings) {
            const auto found = by_key.find(b.field);
            if (found == by_key.end())
                return {
                    Text{},
                    x::errors::Error(
                        CONFIG_ERROR,
                        "message " + m.name + ": field " + b.field.to_string() +
                            " does not exist"
                    ),
                };
            const auto &f = *found->second;
            auto [field, err] = Field::create(f, b.data_type, m.payload.delimiter);
            if (err)
                return {
                    Text{},
                    as_config(
                        err,
                        "message " + m.name + ": field " + icd::base(f).name
                    ),
                };
            if (field.tagged)
                for (const auto &other: m.payload.fields)
                    if (const auto *o = std::get_if<icd::TaggedTextField>(&other);
                        o != nullptr && o->tag.size() > field.tag.size() &&
                        o->tag.starts_with(field.tag))
                        field.longer.push_back(o->tag);
            msg.fields.push_back(std::move(field));
            msg.samples.emplace_back(b.data_type, 1);
        }
        t.messages.push_back(std::move(msg));
        if (auto err = t.route(m.name, m.payload.prefix, i)) return {Text{}, err};
    }
    std::ranges::stable_sort(t.prefixed, [](const Prefixed &a, const Prefixed &b) {
        return a.prefix.size() > b.prefix.size();
    });
    return {std::move(t), x::errors::NIL};
}

x::errors::Error
Text::route(const std::string &name, const std::string &prefix, const std::size_t i) {
    if (prefix.empty()) {
        if (this->fallback.has_value())
            return x::errors::Error(
                CONFIG_ERROR,
                "messages " + this->messages[*this->fallback].name + " and " + name +
                    " both have no prefix"
            );
        this->fallback = i;
        return x::errors::NIL;
    }
    const auto same = std::ranges::find(this->prefixed, prefix, &Prefixed::prefix);
    if (same != this->prefixed.end())
        return x::errors::Error(
            CONFIG_ERROR,
            "messages " + this->messages[same->message].name + " and " + name +
                " have the same prefix " + x::json::quote(prefix)
        );
    this->prefixed.push_back({.prefix = prefix, .message = i});
    return x::errors::NIL;
}

std::pair<std::optional<std::size_t>, x::errors::Error>
Text::decode(const std::string_view line) {
    if (x::strings::trim(line).empty()) return {std::nullopt, x::errors::NIL};
    std::optional<std::size_t> found = this->fallback;
    for (const auto &p: this->prefixed)
        if (line.starts_with(p.prefix)) {
            found = p.message;
            break;
        }
    if (!found.has_value()) return {std::nullopt, x::errors::NIL};
    auto &m = this->messages[*found];
    this->items.clear();
    for (std::size_t start = 0;;) {
        const auto end = line.find(m.delimiter, start);
        this->items.push_back(line.substr(start, end - start));
        if (end == std::string_view::npos) break;
        start = end + m.delimiter.size();
    }
    for (std::size_t i = 0; i < m.fields.size(); i++) {
        auto &dst = m.samples[i];
        dst.clear();
        if (auto err = m.fields[i].decode(this->items, dst)) {
            err = wrap(err, "field " + m.fields[i].name);
            if (this->messages.size() > 1) err = wrap(err, "message " + m.name);
            return {std::nullopt, err};
        }
    }
    return {found, x::errors::NIL};
}
}
