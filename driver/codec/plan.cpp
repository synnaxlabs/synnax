// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <bit>
#include <charconv>
#include <cmath>
#include <limits>
#include <string_view>
#include <unordered_map>
#include <unordered_set>

#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;

x::errors::Error field_error(const std::string &field, const std::string &msg) {
    return x::errors::Error(CONFIG_ERROR, "field " + field + ": " + msg);
}

const library::BaseField &base(const library::Field &field) {
    return std::visit(
        [](const auto &f) -> const library::BaseField & { return f; },
        field
    );
}

x::errors::Error validate_scaling(const library::BaseField &field) {
    if (!std::isfinite(field.scale) || field.scale == 0)
        return field_error(field.name, "scale must be finite and non-zero");
    if (!std::isfinite(field.offset))
        return field_error(field.name, "offset must be finite");
    return x::errors::NIL;
}

bool is_space(const char c) {
    return c == ' ' || c == '\t' || c == '\r' || c == '\n';
}

std::string_view trim_left(std::string_view s) {
    while (!s.empty() && is_space(s.front()))
        s.remove_prefix(1);
    return s;
}

std::string_view trim(std::string_view s) {
    s = trim_left(s);
    while (!s.empty() && is_space(s.back()))
        s.remove_suffix(1);
    return s;
}

bool parse(std::string_view s, double &out) {
    s = trim(s);
    if (s.size() > 1 && s.front() == '+') s.remove_prefix(1);
    if (s.empty()) return false;
    const auto [ptr, ec] = std::from_chars(s.data(), s.data() + s.size(), out);
    return ec == std::errc() && ptr == s.data() + s.size();
}

std::uint64_t unsigned_max(const std::uint8_t bits) {
    return bits == 64 ? std::numeric_limits<std::uint64_t>::max()
                      : (std::uint64_t{1} << bits) - 1;
}

std::int64_t signed_max(const std::uint8_t bits) {
    return static_cast<std::int64_t>(unsigned_max(bits - 1));
}

std::int64_t signed_min(const std::uint8_t bits) {
    return -signed_max(bits) - 1;
}

void append(std::vector<std::uint8_t> &out, const std::string_view s) {
    out.insert(out.end(), s.begin(), s.end());
}

template<typename T>
void append_number(std::vector<std::uint8_t> &out, const T value) {
    char buf[32];
    const auto [ptr, ec] = std::to_chars(buf, buf + sizeof(buf), value);
    append(out, std::string_view(buf, ptr - buf));
}
}

void Values::set(const std::size_t slot, const double value) {
    this->slots[slot] = {.value = value, .integer = 0, .kind = Kind::FLOAT};
}

void Values::set(const std::size_t slot, const std::int64_t value) {
    this->slots[slot] = {
        .value = static_cast<double>(value),
        .integer = static_cast<std::uint64_t>(value),
        .kind = Kind::INT,
    };
}

void Values::set(const std::size_t slot, const std::uint64_t value) {
    this->slots[slot] = {
        .value = static_cast<double>(value),
        .integer = value,
        .kind = Kind::UINT,
    };
}

x::errors::Error
Values::set(const std::size_t slot, const x::telem::SampleValue &value) {
    return std::visit(
        [this, slot]<typename T>(const T &v) -> x::errors::Error {
            if constexpr (std::is_same_v<T, std::string>)
                return x::errors::Error(ENCODE_ERROR, "cannot encode a string sample");
            else if constexpr (std::is_same_v<T, x::telem::TimeStamp>)
                this->set(slot, static_cast<std::int64_t>(v.nanoseconds()));
            else if constexpr (std::is_floating_point_v<T>)
                this->set(slot, static_cast<double>(v));
            else if constexpr (std::is_signed_v<T>)
                this->set(slot, static_cast<std::int64_t>(v));
            else
                this->set(slot, static_cast<std::uint64_t>(v));
            return x::errors::NIL;
        },
        value
    );
}

void Values::clear() {
    for (auto &s: this->slots)
        s.kind = Kind::ABSENT;
}

std::size_t Values::write(const std::size_t slot, x::telem::Series &series) const {
    const auto &s = this->slots[slot];
    switch (s.kind) {
        case Kind::ABSENT:
            return 0;
        case Kind::FLOAT:
            return series.write_casted(&s.value, 1);
        case Kind::INT: {
            const auto v = static_cast<std::int64_t>(s.integer);
            return series.write_casted(&v, 1);
        }
        case Kind::UINT:
            return series.write_casted(&s.integer, 1);
    }
    return 0;
}

std::pair<Plan, x::errors::Error> Plan::compile(const library::MessageEntry &message) {
    std::vector<library::FieldKey> keys;
    keys.reserve(message.fields.size());
    for (const auto &f: message.fields)
        keys.push_back(base(f).key);
    return compile(message, keys);
}

std::pair<Plan, x::errors::Error> Plan::compile(
    const library::MessageEntry &message,
    const std::vector<library::FieldKey> &keys
) {
    std::unordered_map<library::FieldKey, const library::Field *> by_key;
    for (const auto &f: message.fields)
        by_key[base(f).key] = &f;

    Plan plan;
    plan.keys = keys;
    plan.text = message.format == library::FORMAT_TEXT;
    if (!plan.text && message.format != library::FORMAT_BINARY)
        return {
            {},
            x::errors::Error(CONFIG_ERROR, "unknown format: " + message.format)
        };
    if (plan.text && message.delimiter.empty())
        return {{}, x::errors::Error(CONFIG_ERROR, "text delimiter is empty")};
    plan.delimiter = message.delimiter;

    std::unordered_set<library::FieldKey> seen;
    std::unordered_map<library::FieldKey, int> mux_index;
    std::vector<std::pair<std::size_t, Binary>> ordered;
    for (std::size_t slot = 0; slot < keys.size(); slot++) {
        const auto &key = keys[slot];
        if (!seen.insert(key).second)
            return {
                {},
                x::errors::Error(CONFIG_ERROR, "duplicate field " + key.to_string()),
            };
        const auto it = by_key.find(key);
        if (it == by_key.end())
            return {
                {},
                x::errors::Error(CONFIG_ERROR, "no field with key " + key.to_string()),
            };
        const auto &field = *it->second;
        const auto &fb = base(field);
        if (auto err = validate_scaling(fb)) return {{}, err};

        if (plan.text) {
            if (fb.multiplexor.has_value())
                return {{}, field_error(fb.name, "text fields cannot be multiplexed")};
            Text t{.slot = slot, .scale = fb.scale, .offset = fb.offset};
            if (const auto *d = std::get_if<library::DelimitedField>(&field)) {
                t.position = d->position;
                plan.positional.push_back(std::move(t));
            } else if (const auto *tg = std::get_if<library::TaggedField>(&field)) {
                if (tg->tag.empty()) return {{}, field_error(fb.name, "tag is empty")};
                t.tag = tg->tag;
                plan.tagged.push_back(std::move(t));
            } else
                return {{}, field_error(fb.name, "text messages need text fields")};
            continue;
        }

        const auto *bf = std::get_if<library::BinaryField>(&field);
        if (bf == nullptr)
            return {{}, field_error(fb.name, "binary messages need binary fields")};
        auto [bits, err] = BitRange::compile(
            bf->start_bit,
            bf->bit_length,
            bf->byte_order
        );
        if (err) return {{}, field_error(fb.name, err.data)};
        Binary b{
            .name = fb.name,
            .slot = slot,
            .bits = bits,
            .scale = fb.scale,
            .offset = fb.offset,
        };
        if (bf->float_) {
            if (bf->bit_length != 32 && bf->bit_length != 64)
                return {{}, field_error(fb.name, "float fields must be 32 or 64 bits")};
            b.type = bf->bit_length == 32 ? Type::FLOAT32 : Type::FLOAT64;
        } else {
            b.type = bf->signed_ ? Type::SIGNED : Type::UNSIGNED;
            b.exact = fb.scale == 1 && fb.offset == 0;
        }

        // Resolves the chain of multiplexors above the field. fill is the mux whose
        // condition the next link sets, or -1 for the field's own condition.
        const auto target = [&plan, &b](const int fill) -> Condition & {
            return fill < 0 ? b.condition : plan.muxes[fill].condition;
        };
        const library::BaseField *cur = &fb;
        int fill = -1;
        std::size_t depth = 0;
        const auto cycle = field_error(fb.name, "multiplexors form a cycle");
        while (cur->multiplexor.has_value()) {
            if (cur->multiplex_values.empty())
                return {{}, field_error(cur->name, "multiplex values are empty")};
            target(fill).values.assign(
                cur->multiplex_values.begin(),
                cur->multiplex_values.end()
            );
            if (++depth > message.fields.size()) return {{}, cycle};
            const auto &mux_key = *cur->multiplexor;
            const auto mit = by_key.find(mux_key);
            if (mit == by_key.end())
                return {
                    {},
                    field_error(cur->name, "no multiplexor " + mux_key.to_string()),
                };
            const auto *mf = std::get_if<library::BinaryField>(mit->second);
            if (mf == nullptr || mf->float_)
                return {
                    {},
                    field_error(
                        base(*mit->second).name,
                        "multiplexors must be integers"
                    ),
                };
            if (const auto existing = mux_index.find(mux_key);
                existing != mux_index.end()) {
                target(fill).multiplexor = existing->second;
                for (auto m = plan.muxes[existing->second].condition.multiplexor;
                     m >= 0;
                     m = plan.muxes[m].condition.multiplexor)
                    if (++depth > message.fields.size()) return {{}, cycle};
                break;
            }
            auto [mux_bits, mux_err] = BitRange::compile(
                mf->start_bit,
                mf->bit_length,
                mf->byte_order
            );
            if (mux_err) return {{}, field_error(mf->name, mux_err.data)};
            const auto index = static_cast<int>(plan.muxes.size());
            plan.muxes.push_back({.bits = mux_bits, .signed_ = mf->signed_});
            mux_index[mux_key] = index;
            target(fill).multiplexor = index;
            fill = index;
            cur = mf;
        }
        ordered.emplace_back(depth, std::move(b));
    }

    // Encode reads multiplexors from the payload it builds, so each one must be written
    // before the fields it selects.
    std::stable_sort(ordered.begin(), ordered.end(), [](const auto &a, const auto &b) {
        return a.first < b.first;
    });
    for (auto &[depth, b]: ordered) {
        plan.length_ = std::max(plan.length_, b.bits.end());
        plan.binary.push_back(std::move(b));
    }
    for (const auto &m: plan.muxes)
        plan.mux_end = std::max(plan.mux_end, m.bits.end());
    plan.length_ = std::max(plan.length_, plan.mux_end);
    if (!plan.text && message.length.has_value()) {
        if (plan.length_ > *message.length)
            return {
                {},
                x::errors::Error(
                    CONFIG_ERROR,
                    "fields span " + std::to_string(plan.length_) +
                        " bytes, but the message length is " +
                        std::to_string(*message.length)
                ),
            };
        plan.length_ = *message.length;
    }
    std::stable_sort(
        plan.positional.begin(),
        plan.positional.end(),
        [](const Text &a, const Text &b) { return a.position < b.position; }
    );
    return {std::move(plan), x::errors::NIL};
}

std::int64_t Plan::read(const Mux &mux, const std::uint8_t *payload) {
    return mux.signed_ ? mux.bits.read_signed(payload)
                       : static_cast<std::int64_t>(mux.bits.read(payload));
}

x::errors::Error
Plan::decode(const std::span<const std::uint8_t> payload, Values &values) const {
    values.invalid_ = 0;
    if (this->text) {
        this->decode_text(payload, values);
        return x::errors::NIL;
    }
    return this->decode_binary(payload, values);
}

x::errors::Error
Plan::decode_binary(const std::span<const std::uint8_t> payload, Values &values) const {
    if (payload.size() < this->mux_end)
        return x::errors::Error(
            SHORT_PAYLOAD_ERROR,
            "payload of " + std::to_string(payload.size()) +
                " bytes is too short for its multiplexors"
        );
    const auto *p = payload.data();
    for (std::size_t i = 0; i < this->muxes.size(); i++)
        values.muxes[i] = read(this->muxes[i], p);
    const auto raw = [&values](const int mux) { return values.muxes[mux]; };
    for (const auto &f: this->binary) {
        auto &s = values.slots[f.slot];
        if (!this->selected(f.condition, raw)) {
            s.kind = Values::Kind::ABSENT;
            continue;
        }
        if (payload.size() < f.bits.end())
            return x::errors::Error(
                SHORT_PAYLOAD_ERROR,
                "payload of " + std::to_string(payload.size()) +
                    " bytes is too short for field " + f.name
            );
        switch (f.type) {
            case Type::UNSIGNED: {
                const auto v = f.bits.read(p);
                s.integer = v;
                s.value = static_cast<double>(v) * f.scale + f.offset;
                s.kind = f.exact ? Values::Kind::UINT : Values::Kind::FLOAT;
                break;
            }
            case Type::SIGNED: {
                const auto v = f.bits.read_signed(p);
                s.integer = static_cast<std::uint64_t>(v);
                s.value = static_cast<double>(v) * f.scale + f.offset;
                s.kind = f.exact ? Values::Kind::INT : Values::Kind::FLOAT;
                break;
            }
            case Type::FLOAT32: {
                const auto bits = static_cast<std::uint32_t>(f.bits.read(p));
                s.value = static_cast<double>(std::bit_cast<float>(bits)) * f.scale +
                          f.offset;
                s.kind = Values::Kind::FLOAT;
                break;
            }
            case Type::FLOAT64:
                s.value = std::bit_cast<double>(f.bits.read(p)) * f.scale + f.offset;
                s.kind = Values::Kind::FLOAT;
                break;
        }
    }
    return x::errors::NIL;
}

void Plan::decode_text(
    const std::span<const std::uint8_t> payload,
    Values &values
) const {
    const std::string_view line(
        reinterpret_cast<const char *>(payload.data()),
        payload.size()
    );
    const auto read = [&values](const Text &f, const std::string_view item) {
        double v = 0;
        if (!parse(item, v)) return;
        values.slots[f.slot] = {
            .value = v * f.scale + f.offset,
            .kind = Values::Kind::FLOAT,
        };
    };
    for (const auto &f: this->positional)
        values.slots[f.slot].kind = Values::Kind::ABSENT;
    for (const auto &f: this->tagged)
        values.slots[f.slot].kind = Values::Kind::ABSENT;

    std::size_t next = 0;
    std::size_t start = 0;
    for (std::uint32_t index = 0;; index++) {
        const auto end = line.find(this->delimiter, start);
        const auto item = line.substr(
            start,
            end == std::string_view::npos ? std::string_view::npos : end - start
        );
        for (;
             next < this->positional.size() && this->positional[next].position == index;
             next++)
            read(this->positional[next], item);
        const auto trimmed = trim_left(item);
        for (const auto &f: this->tagged)
            if (!values.present(f.slot) && trimmed.starts_with(f.tag))
                read(f, trimmed.substr(f.tag.size()));
        if (end == std::string_view::npos) break;
        if (next == this->positional.size() && this->tagged.empty()) break;
        start = end + this->delimiter.size();
    }
    for (const auto &f: this->positional)
        if (!values.present(f.slot)) values.invalid_++;
    for (const auto &f: this->tagged)
        if (!values.present(f.slot)) values.invalid_++;
}

x::errors::Error
Plan::encode(const Values &values, std::vector<std::uint8_t> &payload) const {
    if (this->text) {
        this->encode_text(values, payload);
        return x::errors::NIL;
    }
    return this->encode_binary(values, payload);
}

x::errors::Error
Plan::encode_binary(const Values &values, std::vector<std::uint8_t> &payload) const {
    if (payload.size() < this->length_) payload.resize(this->length_, 0);
    auto *p = payload.data();
    // Multiplexors come before the fields they select, so each read sees the value this
    // encode wrote.
    const auto raw = [this, p](const int mux) { return read(this->muxes[mux], p); };
    for (const auto &f: this->binary) {
        const auto &s = values.slots[f.slot];
        if (s.kind == Values::Kind::ABSENT || !this->selected(f.condition, raw))
            continue;
        const auto bits = f.bits.length();
        if (f.exact && s.kind == Values::Kind::INT) {
            const auto v = static_cast<std::int64_t>(s.integer);
            if (f.type == Type::SIGNED)
                f.bits.write(
                    p,
                    static_cast<std::uint64_t>(
                        std::clamp(v, signed_min(bits), signed_max(bits))
                    )
                );
            else
                f.bits.write(p, v < 0 ? 0 : std::min(s.integer, unsigned_max(bits)));
            continue;
        }
        if (f.exact && s.kind == Values::Kind::UINT) {
            const auto max = f.type == Type::SIGNED
                               ? static_cast<std::uint64_t>(signed_max(bits))
                               : unsigned_max(bits);
            f.bits.write(p, std::min(s.integer, max));
            continue;
        }
        const double v = (s.value - f.offset) / f.scale;
        switch (f.type) {
            case Type::FLOAT32:
                f.bits.write(p, std::bit_cast<std::uint32_t>(static_cast<float>(v)));
                continue;
            case Type::FLOAT64:
                f.bits.write(p, std::bit_cast<std::uint64_t>(v));
                continue;
            default:
                break;
        }
        if (std::isnan(v))
            return x::errors::Error(ENCODE_ERROR, "field " + f.name + " is NaN");
        const double r = std::round(v);
        if (f.type == Type::SIGNED) {
            const auto min = signed_min(bits);
            const auto max = signed_max(bits);
            std::int64_t raw;
            if (r <= static_cast<double>(min))
                raw = min;
            else if (r >= static_cast<double>(max))
                raw = max;
            else
                raw = static_cast<std::int64_t>(r);
            f.bits.write(p, static_cast<std::uint64_t>(raw));
        } else {
            const auto max = unsigned_max(bits);
            std::uint64_t raw;
            if (r <= 0)
                raw = 0;
            else if (r >= static_cast<double>(max))
                raw = max;
            else
                raw = static_cast<std::uint64_t>(r);
            f.bits.write(p, raw);
        }
    }
    return x::errors::NIL;
}

void Plan::encode_text(const Values &values, std::vector<std::uint8_t> &payload) const {
    payload.clear();
    const auto write = [&values, &payload](const Text &f) {
        const auto &s = values.slots[f.slot];
        if (f.scale == 1 && f.offset == 0 && s.kind == Values::Kind::INT)
            append_number(payload, static_cast<std::int64_t>(s.integer));
        else if (f.scale == 1 && f.offset == 0 && s.kind == Values::Kind::UINT)
            append_number(payload, s.integer);
        else
            append_number(payload, (s.value - f.offset) / f.scale);
    };
    bool first = true;
    std::size_t next = 0;
    if (!this->positional.empty()) {
        const auto last = this->positional.back().position;
        for (std::uint32_t index = 0; index <= last; index++) {
            if (!first) append(payload, this->delimiter);
            first = false;
            bool written = false;
            for (; next < this->positional.size() &&
                   this->positional[next].position == index;
                 next++) {
                const auto &f = this->positional[next];
                if (written || !values.present(f.slot)) continue;
                write(f);
                written = true;
            }
        }
    }
    for (const auto &f: this->tagged) {
        if (!values.present(f.slot)) continue;
        if (!first) append(payload, this->delimiter);
        first = false;
        append(payload, f.tag);
        write(f);
    }
}
}
