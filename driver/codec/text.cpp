// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <charconv>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <map>
#include <optional>
#include <span>
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

/// @brief MAX_POSITION is the largest position of a delimited field. A line with more
/// items does not fit in a 64 KiB frame.
constexpr std::uint32_t MAX_POSITION = 65535;

/// @returns CONFIG_ERROR when tag is empty, contains delimiter, or starts with
/// whitespace.
x::errors::Error check_tag(const std::string &tag, const std::string_view delimiter) {
    if (tag.empty()) return x::errors::Error(CONFIG_ERROR, "the tag is empty");
    if (tag.contains(delimiter))
        return x::errors::Error(
            CONFIG_ERROR,
            "the tag contains the delimiter " + x::json::quote(delimiter)
        );
    if (x::strings::TRIMMED.contains(tag.front()))
        return x::errors::Error(CONFIG_ERROR, "the tag starts with whitespace");
    return x::errors::NIL;
}

/// @brief returns the double of the shortest decimal text of v, so that a float32
/// sample gains no float64 digits. A value that is not finite stays as it is.
double widen(const float v) {
    if (!std::isfinite(v)) return v;
    std::array<char, 32> buf{};
    const auto res = std::to_chars(buf.data(), buf.data() + buf.size(), v);
    return x::json::parse_float64({buf.data(), res.ptr}).first;
}

/// @brief appends the shortest decimal text that reads back as v to out.
template<typename T>
void append_number(const T v, std::string &out) {
    std::array<char, 32> buf{};
    const auto res = std::to_chars(buf.data(), buf.data() + buf.size(), v);
    out.append(buf.data(), res.ptr);
}
}

std::pair<Text::Field, x::errors::Error> Text::Field::create(
    const icd::TextField &f,
    const x::telem::DataType &dt,
    const std::string_view delimiter
) {
    const auto &b = icd::base(f);
    if (auto err = check_scale(b.scale)) return {Field{}, err};
    Field field{
        .name = b.name,
        .scaled = b.scale != 1 || b.offset != 0,
        .scale = b.scale,
        .offset = b.offset,
        .narrow = dt == x::telem::FLOAT32_T,
    };
    if (const auto *tagged = std::get_if<icd::TaggedTextField>(&f)) {
        if (auto err = check_tag(tagged->tag, delimiter)) return {Field{}, err};
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
    return write_float(v, this->narrow, dst);
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
            err = err.wrap("field " + m.fields[i].name);
            if (this->messages.size() > 1) err = err.wrap("message " + m.name);
            return {std::nullopt, err};
        }
    }
    return {found, x::errors::NIL};
}

x::errors::Error TextEncoder::Field::append(
    const std::span<const x::telem::Series> samples,
    std::string &out
) const {
    if (this->binding.has_value()) {
        const auto &s = samples[*this->binding];
        if (!s.empty()) return this->append(s.at(0), out);
    }
    if (!this->fallback.has_value())
        return x::errors::Error(x::errors::VALIDATION, "no value and no default");
    append_number(*this->fallback, out);
    return x::errors::NIL;
}

x::errors::Error
TextEncoder::Field::append(const x::telem::SampleValue &v, std::string &out) const {
    if (this->number == Number::SIGNED) {
        append_number(x::telem::cast<std::int64_t>(v), out);
        return x::errors::NIL;
    }
    if (this->number == Number::UNSIGNED) {
        append_number(x::telem::cast<std::uint64_t>(v), out);
        return x::errors::NIL;
    }
    auto d = this->number == Number::FLOAT32 ? widen(x::telem::cast<float>(v))
                                             : x::telem::cast<double>(v);
    if (this->scaled) d = (d - this->base.offset) / this->base.scale;
    if (!std::isfinite(d))
        return x::errors::Error(
            x::errors::VALIDATION,
            x::json::Number(d).to_string() + " is not finite"
        );
    append_number(d, out);
    return x::errors::NIL;
}

std::pair<TextEncoder::Field, x::errors::Error> TextEncoder::Field::create(
    const icd::TextField &f,
    const std::optional<x::telem::DataType> &dt,
    const std::string_view delimiter
) {
    const auto &b = icd::base(f);
    Field field{
        .base = b,
        .scaled = b.scale != 1 || b.offset != 0,
    };
    if (const auto *t = std::get_if<icd::TaggedTextField>(&f)) {
        if (auto err = check_tag(t->tag, delimiter)) return {Field{}, err};
        field.tag = t->tag;
    }
    if (b.default_.has_value()) {
        const double d = b.default_->value;
        if (!std::isfinite(d))
            return {
                Field{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "default: " + x::json::Number(d).to_string() + " is not finite"
                ),
            };
        field.fallback = d;
    }
    if (!dt.has_value()) {
        if (field.fallback.has_value()) return {std::move(field), x::errors::NIL};
        return {
            Field{},
            x::errors::Error(CONFIG_ERROR, "needs a channel or a default")
        };
    }
    if (auto err = check_scale(b.scale)) return {Field{}, err};
    const auto [number, err] = number_of(*dt, x::payload::KIND_FLOAT, field.scaled);
    if (err) return {Field{}, err};
    field.number = number;
    return {std::move(field), x::errors::NIL};
}

std::pair<TextEncoder, x::errors::Error> TextEncoder::create(const TextMessage &m) {
    const auto refuse = [&m](const std::string &text) {
        return std::pair{
            TextEncoder{},
            x::errors::Error(CONFIG_ERROR, "message " + m.name + ": " + text),
        };
    };
    const auto &delimiter = m.payload.delimiter;
    if (delimiter.empty()) return refuse("the delimiter is empty");
    std::map<icd::FieldKey, const icd::TextField *> by_key;
    for (const auto &f: m.payload.fields)
        by_key[icd::base(f).key] = &f;
    auto [bound, bind_err] = index_bindings(m.bindings, by_key);
    if (bind_err) return {TextEncoder{}, as_config(bind_err, "message " + m.name)};
    TextEncoder enc;
    enc.name = m.name;
    enc.delimiter = delimiter;
    enc.bound.resize(m.bindings.size());
    std::size_t fixed = 0;
    if (const auto &prefix = m.payload.prefix; !prefix.empty()) {
        for (std::size_t start = 0;;) {
            const auto end = prefix.find(delimiter, start);
            enc.items.push_back({.text = prefix.substr(start, end - start)});
            if (end == std::string::npos) break;
            start = end + delimiter.size();
        }
        fixed = enc.items.size() - (enc.items.back().text.empty() ? 1 : 0);
    }
    for (const auto &f: m.payload.fields) {
        const auto &b = icd::base(f);
        const auto context = "field " + b.name;
        const auto found = bound.find(b.key);
        std::optional<x::telem::DataType> dt;
        if (found != bound.end()) dt = m.bindings[found->second].data_type;
        auto [field, err] = Field::create(f, dt, delimiter);
        if (err)
            return {
                TextEncoder{},
                as_config(err, "message " + m.name + ": " + context)
            };
        const auto index = enc.fields.size();
        if (found != bound.end()) {
            field.binding = found->second;
            enc.bound[found->second] = index;
        }
        if (std::holds_alternative<icd::TaggedTextField>(f)) {
            enc.tagged.push_back(index);
        } else {
            const auto position = std::get<icd::DelimitedTextField>(f).position;
            if (position > MAX_POSITION)
                return refuse(
                    context + ": position " + std::to_string(position) +
                    " is past the last position " + std::to_string(MAX_POSITION)
                );
            if (position < fixed)
                return refuse(
                    context + ": the prefix fills position " + std::to_string(position)
                );
            if (position >= enc.items.size()) enc.items.resize(position + 1);
            auto &item = enc.items[position];
            if (item.field >= 0)
                return refuse(
                    "fields " +
                    enc.fields[static_cast<std::size_t>(item.field)].base.name +
                    " and " + b.name + " both have position " + std::to_string(position)
                );
            item.field = static_cast<int>(index);
        }
        enc.fields.push_back(std::move(field));
    }
    if (!enc.tagged.empty() && !enc.items.empty() && enc.items.back().field < 0 &&
        enc.items.back().text.empty())
        enc.items.pop_back();
    return {std::move(enc), x::errors::NIL};
}

x::errors::Error TextEncoder::encode(
    const std::span<const x::telem::Series> samples,
    std::vector<std::uint8_t> &payload
) {
    this->line.clear();
    bool first = true;
    const auto next = [&] {
        if (!first) this->line += this->delimiter;
        first = false;
    };
    for (const auto &item: this->items) {
        next();
        if (item.field < 0) {
            this->line += item.text;
            continue;
        }
        const auto &f = this->fields[static_cast<std::size_t>(item.field)];
        if (auto err = f.append(samples, this->line))
            return err.wrap("message " + this->name + ": field " + f.base.name);
    }
    for (const auto i: this->tagged) {
        next();
        const auto &f = this->fields[i];
        this->line += f.tag;
        if (auto err = f.append(samples, this->line))
            return err.wrap("message " + this->name + ": field " + f.base.name);
    }
    payload.assign(this->line.begin(), this->line.end());
    return x::errors::NIL;
}

x::errors::Error TextEncoder::check(
    const std::size_t binding,
    const x::telem::Series &s,
    const std::size_t i
) {
    const auto &f = this->fields[this->bound[binding]];
    const auto v = s.at(static_cast<int>(i));
    this->line.clear();
    auto err = f.append(v, this->line);
    if (!err)
        err = f.number == Number::FLOAT32
                ? check_limits(f.base, x::telem::cast<float>(v))
                : check_limits(f.base, x::telem::cast<double>(v));
    if (err) return err.wrap("message " + this->name + ": field " + f.base.name);
    return x::errors::NIL;
}
}
