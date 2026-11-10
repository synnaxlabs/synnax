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
#include <type_traits>
#include <unordered_map>
#include <unordered_set>
#include <utility>

#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

x::errors::Error field_error(const std::string &field, const std::string &msg) {
    return x::errors::Error(CONFIG_ERROR, "field " + field + ": " + msg);
}

const icd::BaseField &base(const icd::BinaryField &field) {
    return field;
}

x::errors::Error validate_scaling(const icd::BaseField &field) {
    if (!std::isfinite(field.scale) || field.scale == 0)
        return field_error(field.name, "scale must be finite and non-zero");
    if (!std::isfinite(field.offset))
        return field_error(field.name, "offset must be finite");
    return x::errors::NIL;
}

/// @returns the field of each key, in key order.
/// @returns CONFIG_ERROR when a key is duplicated or names no field, or when a field's
/// scaling is invalid.
template<typename Field>
std::pair<std::vector<const Field *>, x::errors::Error> select(
    const std::unordered_map<icd::FieldKey, const Field *> &by_key,
    const std::vector<icd::FieldKey> &keys
) {
    std::unordered_set<icd::FieldKey> seen;
    std::vector<const Field *> fields;
    fields.reserve(keys.size());
    for (const auto &key: keys) {
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
        if (auto err = validate_scaling(base(*it->second))) return {{}, err};
        fields.push_back(it->second);
    }
    return {std::move(fields), x::errors::NIL};
}

/// @returns each field by its key.
template<typename Field>
std::unordered_map<icd::FieldKey, const Field *>
keyed(const std::vector<Field> &fields) {
    std::unordered_map<icd::FieldKey, const Field *> by_key;
    for (const auto &f: fields)
        by_key[base(f).key] = &f;
    return by_key;
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

std::string format(const double v) {
    if (std::isnan(v)) return "NaN";
    char buf[32];
    const auto [ptr, ec] = std::to_chars(buf, buf + sizeof(buf), v);
    return {buf, ptr};
}

constexpr double max_double = std::numeric_limits<double>::max();

/// @returns the largest double below the exclusive integer bound end.
double below(const double end) {
    return end - 1 < end ? end - 1 : std::nextafter(end, 0.0);
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

Plan::Limit::Limit(
    const icd::BaseField &field,
    const std::optional<Type> type,
    const std::uint8_t bits
):
    integer(type == Type::SIGNED || type == Type::UNSIGNED),
    signed_(type == Type::SIGNED),
    exact(
        field.scale == 1 && field.offset == 0 && (this->integer || !type.has_value())
    ),
    bits(bits),
    scale(field.scale),
    offset(field.offset),
    lo(-max_double),
    hi(max_double),
    min(field.min),
    max(field.max) {
    if (type == Type::FLOAT32) {
        this->hi = std::numeric_limits<float>::max();
        this->lo = -this->hi;
    } else if (this->integer) {
        this->lo = signed_ ? -std::ldexp(1.0, bits - 1) : 0;
        this->hi = std::ldexp(1.0, signed_ ? bits - 1 : bits);
    }
    const double top = this->integer ? below(this->hi) : this->hi;
    double a = this->lo * this->scale + this->offset;
    double b = top * this->scale + this->offset;
    if (a > b) std::swap(a, b);
    std::string low = format(a);
    std::string high = format(b);
    if (this->integer && this->exact) {
        low = this->signed_ ? std::to_string(signed_min(bits)) : "0";
        high = this->signed_ ? std::to_string(signed_max(bits))
                             : std::to_string(unsigned_max(bits));
    }
    if (this->min && *this->min > a) low = format(*this->min);
    if (this->max && *this->max < b) high = format(*this->max);
    this->range = low + " to " + high;
}

bool Plan::Limit::accepts(const double value) const {
    if ((this->min && value < *this->min) || (this->max && value > *this->max))
        return false;
    const double raw = (value - this->offset) / this->scale;
    if (!this->integer) return std::abs(raw) <= this->hi;
    const double r = std::round(raw);
    return r >= this->lo && r < this->hi;
}

bool Plan::Limit::accepts(const std::int64_t value) const {
    if (!this->exact) return this->accepts(static_cast<double>(value));
    const auto v = static_cast<double>(value);
    if ((this->min && v < *this->min) || (this->max && v > *this->max)) return false;
    if (!this->integer) return true;
    if (this->signed_)
        return value >= signed_min(this->bits) && value <= signed_max(this->bits);
    return value >= 0 && static_cast<std::uint64_t>(value) <= unsigned_max(this->bits);
}

bool Plan::Limit::accepts(const std::uint64_t value) const {
    if (!this->exact) return this->accepts(static_cast<double>(value));
    const auto v = static_cast<double>(value);
    if ((this->min && v < *this->min) || (this->max && v > *this->max)) return false;
    if (!this->integer) return true;
    if (this->signed_) return std::cmp_less_equal(value, signed_max(this->bits));
    return value <= unsigned_max(this->bits);
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

void Values::clear() {
    for (auto &s: this->slots)
        s.kind = Kind::ABSENT;
}

namespace {
/// @returns true when T holds the value: a double truncated toward zero for integer
/// types, or an integer.
template<typename T, typename V>
bool holds(const V value) {
    if constexpr (std::is_integral_v<V>) {
        if constexpr (std::is_floating_point_v<T>)
            return true;
        else
            return std::in_range<T>(value);
    } else if constexpr (std::is_floating_point_v<T>)
        return !std::isfinite(value) ||
               std::abs(value) <= static_cast<double>(std::numeric_limits<T>::max());
    else {
        const double t = std::trunc(value);
        const double end = std::ldexp(1.0, std::numeric_limits<T>::digits);
        if constexpr (std::is_signed_v<T>)
            return t >= -end && t < end;
        else
            return t >= 0 && t < end;
    }
}

/// @returns true when a series of the data type holds the value, which is a double or
/// an integer.
template<typename V>
bool holds(const V value, const x::telem::DataType &type) {
    if (type == x::telem::FLOAT64_T) return holds<double>(value);
    if (type == x::telem::FLOAT32_T) return holds<float>(value);
    if (type == x::telem::INT64_T) return holds<std::int64_t>(value);
    if (type == x::telem::INT32_T) return holds<std::int32_t>(value);
    if (type == x::telem::INT16_T) return holds<std::int16_t>(value);
    if (type == x::telem::INT8_T) return holds<std::int8_t>(value);
    if (type == x::telem::UINT64_T) return holds<std::uint64_t>(value);
    if (type == x::telem::UINT32_T) return holds<std::uint32_t>(value);
    if (type == x::telem::UINT16_T) return holds<std::uint16_t>(value);
    if (type == x::telem::UINT8_T) return holds<std::uint8_t>(value);
    return false;
}
}

bool Values::fits(const std::size_t slot, const x::telem::DataType &type) const {
    const auto &s = this->slots[slot];
    switch (s.kind) {
        case Kind::ABSENT:
            return true;
        case Kind::FLOAT:
            return holds(s.value, type);
        case Kind::INT:
            return holds(static_cast<std::int64_t>(s.integer), type);
        case Kind::UINT:
            return holds(s.integer, type);
    }
    return false;
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

x::errors::Error Plan::set(
    Values &values,
    const std::size_t slot,
    const x::telem::SampleValue &value
) const {
    const auto &limit = this->limits[slot];
    const auto refuse = [&limit](const std::string &v) {
        return x::errors::Error(
            RANGE_ERROR,
            v + " is outside the range " + limit.range
        );
    };
    return std::visit(
        [&values, slot, &limit, &refuse]<typename T>(const T &v) -> x::errors::Error {
            if constexpr (std::is_same_v<T, std::string>)
                return x::errors::Error(ENCODE_ERROR, "cannot encode a string sample");
            else if constexpr (std::is_floating_point_v<T>) {
                const auto d = static_cast<double>(v);
                if (!limit.accepts(d)) return refuse(format(d));
                values.set(slot, d);
            } else {
                using I = std::conditional_t<
                    std::is_same_v<T, x::telem::TimeStamp> || std::is_signed_v<T>,
                    std::int64_t,
                    std::uint64_t>;
                I i;
                if constexpr (std::is_same_v<T, x::telem::TimeStamp>)
                    i = v.nanoseconds();
                else
                    // NOLINTNEXTLINE(bugprone-signed-char-misuse): int8_t sign-extends.
                    i = static_cast<I>(v);
                if (!limit.accepts(i)) return refuse(std::to_string(i));
                if (limit.exact)
                    values.set(slot, i);
                else
                    values.set(slot, static_cast<double>(i));
            }
            return x::errors::NIL;
        },
        value
    );
}

void Plan::set_raw(
    Values &values,
    const std::size_t slot,
    const std::int64_t raw
) const {
    const auto &limit = this->limits[slot];
    if (!limit.exact)
        values.set(slot, static_cast<double>(raw) * limit.scale + limit.offset);
    else if (limit.integer && !limit.signed_)
        values.set(slot, static_cast<std::uint64_t>(raw));
    else
        values.set(slot, raw);
}

std::pair<Plan, x::errors::Error> Plan::compile(const icd::Message &message) {
    std::vector<icd::FieldKey> keys;
    std::visit(
        [&keys](const auto &payload) {
            keys.reserve(payload.fields.size());
            for (const auto &f: payload.fields)
                keys.push_back(base(f).key);
        },
        message.payload
    );
    return compile(message, keys);
}

std::pair<Plan, x::errors::Error>
Plan::compile(const icd::Message &message, const std::vector<icd::FieldKey> &keys) {
    Plan plan;
    plan.keys = keys;
    auto err = std::visit(
        [&plan](const auto &payload) { return plan.compile_payload(payload); },
        message.payload
    );
    if (err) return {{}, err};
    return {std::move(plan), x::errors::NIL};
}

x::errors::Error Plan::compile_payload(const icd::TextPayload &payload) {
    if (payload.delimiter.empty())
        return x::errors::Error(CONFIG_ERROR, "text delimiter is empty");
    this->text = true;
    this->delimiter = payload.delimiter;
    const auto [fields, err] = select(keyed(payload.fields), this->keys);
    if (err) return err;
    this->limits.resize(fields.size());
    for (std::size_t slot = 0; slot < fields.size(); slot++) {
        const auto &field = *fields[slot];
        const auto &fb = base(field);
        if (fb.multiplexor.has_value())
            return field_error(fb.name, "text fields cannot be multiplexed");
        this->limits[slot] = Limit(fb, std::nullopt, 0);
        Text t{.slot = slot, .scale = fb.scale, .offset = fb.offset};
        if (const auto *d = std::get_if<icd::DelimitedTextField>(&field)) {
            t.position = d->position;
            this->positional.push_back(std::move(t));
            continue;
        }
        const auto &tagged = std::get<icd::TaggedTextField>(field);
        if (tagged.tag.empty()) return field_error(fb.name, "tag is empty");
        t.tag = tagged.tag;
        this->tagged.push_back(std::move(t));
    }
    std::ranges::stable_sort(this->positional, [](const Text &a, const Text &b) {
        return a.position < b.position;
    });
    return x::errors::NIL;
}

x::errors::Error Plan::compile_payload(const icd::BinaryPayload &payload) {
    const auto by_key = keyed(payload.fields);
    const auto [fields, err] = select(by_key, this->keys);
    if (err) return err;
    this->limits.resize(fields.size());
    std::unordered_map<icd::FieldKey, int> mux_index;
    std::vector<std::pair<std::size_t, Binary>> ordered;
    for (std::size_t slot = 0; slot < fields.size(); slot++) {
        const auto &bf = *fields[slot];
        auto [bits, bits_err] = BitRange::compile(
            bf.start_bit,
            bf.bit_length,
            bf.byte_order
        );
        if (bits_err) return field_error(bf.name, bits_err.data);
        Binary b{
            .name = bf.name,
            .slot = slot,
            .bits = bits,
            .scale = bf.scale,
            .offset = bf.offset,
        };
        if (bf.float_) {
            if (bf.bit_length != 32 && bf.bit_length != 64)
                return field_error(bf.name, "float fields must be 32 or 64 bits");
            b.type = bf.bit_length == 32 ? Type::FLOAT32 : Type::FLOAT64;
        } else {
            b.type = bf.signed_ ? Type::SIGNED : Type::UNSIGNED;
            b.exact = bf.scale == 1 && bf.offset == 0;
        }
        this->limits[slot] = Limit(bf, b.type, bf.bit_length);

        // Resolves the chain of multiplexors above the field. fill is the mux whose
        // condition the next link sets, or -1 for the field's own condition.
        const auto target = [this, &b](const int fill) -> Condition & {
            return fill < 0 ? b.condition : this->muxes[fill].condition;
        };
        const icd::BinaryField *cur = &bf;
        int fill = -1;
        std::size_t depth = 0;
        const auto cycle = field_error(bf.name, "multiplexors form a cycle");
        while (cur->multiplexor.has_value()) {
            if (cur->multiplex_values.empty())
                return field_error(cur->name, "multiplex values are empty");
            target(fill).values.assign(
                cur->multiplex_values.begin(),
                cur->multiplex_values.end()
            );
            if (++depth > payload.fields.size()) return cycle;
            const auto &mux_key = *cur->multiplexor;
            const auto mit = by_key.find(mux_key);
            if (mit == by_key.end())
                return field_error(cur->name, "no multiplexor " + mux_key.to_string());
            const auto *mf = mit->second;
            if (mf->float_)
                return field_error(mf->name, "multiplexors must be integers");
            if (const auto existing = mux_index.find(mux_key);
                existing != mux_index.end()) {
                target(fill).multiplexor = existing->second;
                for (auto m = this->muxes[existing->second].condition.multiplexor;
                     m >= 0;
                     m = this->muxes[m].condition.multiplexor)
                    if (++depth > payload.fields.size()) return cycle;
                break;
            }
            auto [mux_bits, mux_err] = BitRange::compile(
                mf->start_bit,
                mf->bit_length,
                mf->byte_order
            );
            if (mux_err) return field_error(mf->name, mux_err.data);
            const auto index = static_cast<int>(this->muxes.size());
            this->muxes.push_back({.bits = mux_bits, .signed_ = mf->signed_});
            mux_index[mux_key] = index;
            target(fill).multiplexor = index;
            fill = index;
            cur = mf;
        }
        ordered.emplace_back(depth, std::move(b));
    }

    // Encode reads multiplexors from the payload it builds, so each one must be written
    // before the fields it selects.
    std::ranges::stable_sort(ordered, [](const auto &a, const auto &b) {
        return a.first < b.first;
    });
    for (auto &[depth, b]: ordered) {
        this->length_ = std::max(this->length_, b.bits.end());
        this->binary.push_back(std::move(b));
    }
    for (const auto &m: this->muxes)
        this->mux_end = std::max(this->mux_end, m.bits.end());
    this->length_ = std::max(this->length_, this->mux_end);
    if (!payload.length.has_value()) return x::errors::NIL;
    if (this->length_ > *payload.length)
        return x::errors::Error(
            CONFIG_ERROR,
            "fields span " + std::to_string(this->length_) +
                " bytes, but the message length is " + std::to_string(*payload.length)
        );
    this->length_ = *payload.length;
    return x::errors::NIL;
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

void Plan::encode(const Values &values, std::vector<std::uint8_t> &payload) const {
    if (this->text)
        this->encode_text(values, payload);
    else
        this->encode_binary(values, payload);
}

void Plan::encode_binary(
    const Values &values,
    std::vector<std::uint8_t> &payload
) const {
    if (payload.size() < this->length_) payload.resize(this->length_, 0);
    auto *p = payload.data();
    // Multiplexors come before the fields they select, so each read sees the value this
    // encode wrote.
    const auto raw = [this, p](const int mux) { return read(this->muxes[mux], p); };
    for (const auto &f: this->binary) {
        const auto &s = values.slots[f.slot];
        if (s.kind == Values::Kind::ABSENT || !this->selected(f.condition, raw))
            continue;
        if (s.kind != Values::Kind::FLOAT) {
            f.bits.write(p, s.integer);
            continue;
        }
        const double v = (s.value - f.offset) / f.scale;
        switch (f.type) {
            case Type::FLOAT32:
                f.bits.write(p, std::bit_cast<std::uint32_t>(static_cast<float>(v)));
                break;
            case Type::FLOAT64:
                f.bits.write(p, std::bit_cast<std::uint64_t>(v));
                break;
            case Type::SIGNED:
                f.bits.write(
                    p,
                    static_cast<std::uint64_t>(static_cast<std::int64_t>(std::round(v)))
                );
                break;
            case Type::UNSIGNED:
                f.bits.write(p, static_cast<std::uint64_t>(std::round(v)));
                break;
        }
    }
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
