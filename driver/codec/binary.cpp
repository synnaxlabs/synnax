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
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <iterator>
#include <limits>
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

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binary.h"
#include "driver/codec/bits.h"
#include "driver/codec/errors.h"
#include "driver/codec/field.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;
namespace payload = x::payload;

/// @brief Width is the size of an integer data type.
struct Width {
    unsigned bits = 0;
    bool is_signed = false;
};

/// @brief returns the width of dt, or nothing when dt is not an integer data type.
std::optional<Width> width_of(const x::telem::DataType &dt) {
    if (dt == x::telem::INT8_T) return Width{.bits = 8, .is_signed = true};
    if (dt == x::telem::INT16_T) return Width{.bits = 16, .is_signed = true};
    if (dt == x::telem::INT32_T) return Width{.bits = 32, .is_signed = true};
    if (dt == x::telem::INT64_T) return Width{.bits = 64, .is_signed = true};
    if (dt == x::telem::UINT8_T) return Width{.bits = 8};
    if (dt == x::telem::UINT16_T) return Width{.bits = 16};
    if (dt == x::telem::UINT32_T) return Width{.bits = 32};
    if (dt == x::telem::UINT64_T) return Width{.bits = 64};
    return std::nullopt;
}

/// @brief writes raw to dst as a T. raw holds a two's complement integer when Signed.
template<typename T, bool Signed>
void put(x::telem::Series &dst, const std::uint64_t raw) {
    if constexpr (Signed)
        dst.write(static_cast<T>(static_cast<std::int64_t>(raw)));
    else
        dst.write(static_cast<T>(raw));
}

template<bool Signed>
void (*put_for(const x::telem::DataType &dt))(x::telem::Series &, std::uint64_t) {
    if (dt == x::telem::INT8_T) return put<std::int8_t, Signed>;
    if (dt == x::telem::INT16_T) return put<std::int16_t, Signed>;
    if (dt == x::telem::INT32_T) return put<std::int32_t, Signed>;
    if (dt == x::telem::INT64_T) return put<std::int64_t, Signed>;
    if (dt == x::telem::UINT8_T) return put<std::uint8_t, Signed>;
    if (dt == x::telem::UINT16_T) return put<std::uint16_t, Signed>;
    if (dt == x::telem::UINT32_T) return put<std::uint32_t, Signed>;
    return put<std::uint64_t, Signed>;
}

/// @brief returns the payload kind of the values of a field of encoding e.
std::string_view field_kind(const Encoding e) {
    if (e == Encoding::SIGNED) return payload::KIND_INTEGER;
    if (e == Encoding::UNSIGNED) return payload::KIND_UNSIGNED;
    return payload::KIND_FLOAT;
}

/// @brief returns the layout of the field of identifier id in payload p.
/// @param context comes before the text of an error.
/// @returns CONFIG_ERROR when the field does not exist, is a float field, has an
/// invalid layout, ends past the payload length, or cannot hold the identifier value.
std::pair<BitRange, x::errors::Error> identifier_bits(
    const icd::FieldIdentifier &id,
    const icd::BinaryPayload &p,
    const Multiplexors::Layout &fields,
    const std::string &context
) {
    const auto found = fields.find(id.field);
    if (found == fields.end())
        return {
            BitRange{},
            x::errors::Error(
                CONFIG_ERROR,
                context + ": field " + id.field.to_string() + " does not exist"
            ),
        };
    const auto &f = *found->second;
    const auto &b = icd::base(f);
    if (std::holds_alternative<icd::FloatBinaryField>(f))
        return {
            BitRange{},
            x::errors::Error(
                CONFIG_ERROR,
                context + ": field " + b.name + " is a float field"
            ),
        };
    auto [bits, err] = BitRange::create(b);
    if (err) return {BitRange{}, as_config(err, context + ": field " + b.name)};
    if (p.length.has_value() && bits.end() > *p.length)
        return {
            BitRange{},
            x::errors::Error(
                CONFIG_ERROR,
                context + ": field " + b.name + ": the field ends past the " +
                    std::to_string(*p.length) + "-byte payload"
            ),
        };
    const bool is_signed = std::holds_alternative<icd::IntegerBinaryField>(f);
    const std::int64_t v = id.value;
    const auto n = b.bit_length;
    const bool fits = is_signed ? n >= 64 || (v >= -(std::int64_t{1} << (n - 1)) &&
                                              v < (std::int64_t{1} << (n - 1)))
                                : v >= 0 && (n >= 63 || v < (std::int64_t{1} << n));
    if (!fits)
        return {
            BitRange{},
            x::errors::Error(
                CONFIG_ERROR,
                context + ": " + std::to_string(v) + " is out of range for field " +
                    b.name
            ),
        };
    return {bits, x::errors::NIL};
}

constexpr std::uint32_t MAX_STANDARD_CAN_ID = 0x7FF;
constexpr std::uint32_t MAX_EXTENDED_CAN_ID = 0x1FFFFFFF;
/// @brief EXTENDED_BIT marks an extended identifier in the keys of Binary::exact.
constexpr std::uint32_t EXTENDED_BIT = 1U << 31;

/// @brief returns v as uppercase hexadecimal text with a 0x prefix.
std::string hex(std::uint32_t v) {
    constexpr std::string_view digits = "0123456789ABCDEF";
    std::string out;
    do {
        out.insert(out.begin(), digits[v & 0xF]);
        v >>= 4;
    } while (v != 0);
    return "0x" + out;
}

/// @brief returns the error for a payload of size bytes that is shorter than the end
/// of a field.
x::errors::Error too_short(const std::size_t size, const std::size_t end) {
    return x::errors::Error(
        x::errors::VALIDATION,
        "the payload has " + std::to_string(size) + " bytes and the field needs " +
            std::to_string(end)
    );
}
}

std::pair<Binary::Field, x::errors::Error>
Binary::Field::create(const icd::BinaryField &f, const x::telem::DataType &dt) {
    const auto &b = icd::base(f);
    auto [bits, bits_err] = BitRange::create(b);
    if (bits_err) return {Field{}, bits_err};
    Field field{
        .name = b.name,
        .bits = bits,
        .scaled = b.scale != 1 || b.offset != 0,
        .scale = b.scale,
        .offset = b.offset,
        .narrow = dt == x::telem::FLOAT32_T,
    };
    auto [encoding, encoding_err] = encoding_of(f);
    if (encoding_err) return {Field{}, encoding_err};
    field.encoding = encoding;
    const auto kind = field_kind(encoding);
    if (!converts(kind, dt))
        return {
            Field{},
            x::errors::Error(
                CONFIG_ERROR,
                "cannot write " + std::string(kind) + " values to " + dt.name() +
                    " channels"
            ),
        };
    if (field.scaled && dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T)
        return {
            Field{},
            x::errors::Error(
                CONFIG_ERROR,
                "a scaled field requires a float channel, not " + dt.name()
            ),
        };
    if (const auto w = width_of(dt); w.has_value()) {
        const bool is_signed = field.encoding == Encoding::SIGNED;
        const unsigned needed = b.bit_length + (w->is_signed && !is_signed ? 1U : 0U);
        if (needed > w->bits)
            return {
                Field{},
                x::errors::Error(
                    CONFIG_ERROR,
                    "cannot write " + std::to_string(b.bit_length) + "-bit " +
                        std::string(kind) + " values to " + dt.name() + " channels"
                ),
            };
        field.put = is_signed ? put_for<true>(dt) : put_for<false>(dt);
    }
    auto [raw, default_err] = default_raw(f);
    if (default_err) return {Field{}, as_config(default_err, "default")};
    if (raw.has_value()) {
        auto dst = x::telem::Series(dt, 1);
        if (const auto err = field.append(*raw, dst))
            return {Field{}, as_config(err, "default")};
        field.fallback = std::move(dst);
    }
    return {std::move(field), x::errors::NIL};
}

x::errors::Error
Binary::Field::append(const std::uint64_t raw, x::telem::Series &dst) const {
    if (this->put != nullptr) {
        this->put(dst, raw);
        return x::errors::NIL;
    }
    double v = 0;
    switch (this->encoding) {
        case Encoding::UNSIGNED:
            v = static_cast<double>(raw);
            break;
        case Encoding::SIGNED:
            v = static_cast<double>(static_cast<std::int64_t>(raw));
            break;
        case Encoding::FLOAT32:
            v = std::bit_cast<float>(static_cast<std::uint32_t>(raw));
            break;
        case Encoding::FLOAT64:
            v = std::bit_cast<double>(raw);
            break;
    }
    return this->append_float(v, dst);
}

x::errors::Error Binary::Field::append_float(double v, x::telem::Series &dst) const {
    if (this->scaled) v = v * this->scale + this->offset;
    if (!this->narrow) {
        dst.write(v);
        return x::errors::NIL;
    }
    if (std::isfinite(v) && std::abs(v) > std::numeric_limits<float>::max())
        return x::errors::Error(
            x::errors::VALIDATION,
            x::json::Number(v).to_string() + " overflows float32"
        );
    dst.write(static_cast<float>(v));
    return x::errors::NIL;
}

x::errors::Error Binary::Field::decode(
    const std::span<const std::uint8_t> payload,
    x::telem::Series &dst
) const {
    if (payload.size() < this->bits.end()) {
        if (this->fallback.has_value()) {
            dst.write(*this->fallback);
            return x::errors::NIL;
        }
        return too_short(payload.size(), this->bits.end());
    }
    const auto *p = payload.data();
    if (this->encoding == Encoding::SIGNED)
        return this->append(static_cast<std::uint64_t>(this->bits.read_signed(p)), dst);
    return this->append(this->bits.read(p), dst);
}

x::errors::Error Binary::Message::decode(const std::span<const std::uint8_t> payload) {
    if (payload.size() < this->muxes.end()) {
        const auto &m = *std::ranges::find_if(
            this->muxes.all(),
            [&](const Multiplexors::Entry &mux) {
                return mux.bits.end() > payload.size();
            }
        );
        return wrap(too_short(payload.size(), m.bits.end()), "field " + m.name);
    }
    for (std::size_t i = 0; i < this->muxes.all().size(); i++) {
        const auto &m = this->muxes.all()[i];
        this->raw[i] = m.is_signed
                         ? m.bits.read_signed(payload.data())
                         : static_cast<std::int64_t>(m.bits.read(payload.data()));
    }
    for (std::size_t i = 0; i < this->fields.size(); i++) {
        const auto &f = this->fields[i];
        auto &dst = this->samples[i];
        dst.clear();
        if (!this->muxes.selected(f.condition, this->raw)) {
            if (f.fallback.has_value()) dst.write(*f.fallback);
            continue;
        }
        if (const auto err = f.decode(payload, dst))
            return wrap(err, "field " + f.name);
    }
    return x::errors::NIL;
}

std::pair<Binary, x::errors::Error>
Binary::create(const std::vector<BinaryMessage> &messages) {
    Binary bin;
    bin.messages.reserve(messages.size());
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &m = messages[i];
        Message msg{.name = m.name};
        Multiplexors::Layout by_key;
        for (const auto &f: m.payload.fields)
            by_key[icd::base(f).key] = &f;
        for (const auto &b: m.bindings) {
            const auto found = by_key.find(b.field);
            if (found == by_key.end())
                return {
                    Binary{},
                    x::errors::Error(
                        CONFIG_ERROR,
                        "message " + m.name + ": field " + b.field.to_string() +
                            " does not exist"
                    ),
                };
            const auto &base = icd::base(*found->second);
            const auto context = "message " + m.name + ": field " + base.name;
            auto [field, err] = Field::create(*found->second, b.data_type);
            if (err) return {Binary{}, as_config(err, context)};
            auto [condition, mux_err] = msg.muxes.resolve(base, by_key);
            if (mux_err) return {Binary{}, as_config(mux_err, context)};
            field.condition = std::move(condition);
            if (const auto length = m.payload.length;
                length.has_value() && field.bits.end() > *length)
                return {
                    Binary{},
                    x::errors::Error(
                        CONFIG_ERROR,
                        context + ": the field ends past the " +
                            std::to_string(*length) + "-byte payload"
                    ),
                };
            msg.fields.push_back(std::move(field));
            msg.samples.emplace_back(b.data_type, 1);
        }
        msg.raw.resize(msg.muxes.all().size());
        bin.messages.push_back(std::move(msg));
        if (auto err = bin.route(m, i, by_key)) return {Binary{}, err};
    }
    return {std::move(bin), x::errors::NIL};
}

x::errors::Error Binary::route(
    const BinaryMessage &m,
    const std::size_t i,
    const Multiplexors::Layout &fields
) {
    if (!m.payload.identifier.has_value()) {
        if (this->fallback.has_value())
            return x::errors::Error(
                CONFIG_ERROR,
                "messages " + this->messages[*this->fallback].name + " and " + m.name +
                    " both have no identifier"
            );
        this->fallback = i;
        return x::errors::NIL;
    }
    const auto &identifier = *m.payload.identifier;
    if (const auto *can = std::get_if<icd::CanIdentifier>(&identifier))
        return this->route_can(*can, i);
    const auto &id = std::get<icd::FieldIdentifier>(identifier);
    auto [bits, err] = identifier_bits(
        id,
        m.payload,
        fields,
        "message " + m.name + ": identifier"
    );
    if (err) return err;
    const std::int64_t v = id.value;
    const auto n = bits.width();
    auto h = std::ranges::find(this->headers, bits, &Header::bits);
    if (h == this->headers.end()) {
        this->headers.push_back({.bits = bits});
        h = std::prev(this->headers.end());
    }
    const auto raw = n >= 64
                       ? static_cast<std::uint64_t>(v)
                       : static_cast<std::uint64_t>(v) & ((std::uint64_t{1} << n) - 1);
    const auto [it, inserted] = h->messages.try_emplace(raw, i);
    if (!inserted)
        return x::errors::Error(
            CONFIG_ERROR,
            "messages " + this->messages[it->second].name + " and " + m.name +
                " match the same identifier value " + std::to_string(v)
        );
    return x::errors::NIL;
}

x::errors::Error Binary::route_can(const icd::CanIdentifier &id, const std::size_t i) {
    const std::uint32_t width = id.extended ? MAX_EXTENDED_CAN_ID : MAX_STANDARD_CAN_ID;
    const auto &name = this->messages[i].name;
    if (id.id > width)
        return x::errors::Error(
            CONFIG_ERROR,
            "message " + name + ": identifier: id " + hex(id.id) +
                " exceeds the maximum " + hex(width)
        );
    const std::uint32_t mask = id.mask.value_or(width) & width;
    const Masked rule{
        .id = id.id & mask,
        .mask = mask,
        .extended = id.extended,
        .message = i,
    };
    std::optional<Masked> clash;
    const auto check = [&](const Masked &other) {
        if (other.extended != rule.extended ||
            ((other.id ^ rule.id) & other.mask & rule.mask) != 0)
            return;
        if (!clash.has_value() || other.message < clash->message) clash = other;
    };
    for (const auto &[key, message]: this->exact)
        check({
            .id = key & MAX_EXTENDED_CAN_ID,
            .mask = (key & EXTENDED_BIT) != 0 ? MAX_EXTENDED_CAN_ID
                                              : MAX_STANDARD_CAN_ID,
            .extended = (key & EXTENDED_BIT) != 0,
            .message = message,
        });
    for (const auto &other: this->masked)
        check(other);
    if (clash.has_value())
        return x::errors::Error(
            CONFIG_ERROR,
            "messages " + this->messages[clash->message].name + " and " + name +
                " both match " + (id.extended ? "extended " : "") + "CAN identifier " +
                hex((clash->id & clash->mask) | (rule.id & rule.mask))
        );
    if (mask == width)
        this->exact[rule.id | (id.extended ? EXTENDED_BIT : 0)] = i;
    else
        this->masked.push_back(rule);
    return x::errors::NIL;
}

std::pair<std::optional<std::size_t>, x::errors::Error> Binary::decode_as(
    const std::optional<std::size_t> i,
    const std::span<const std::uint8_t> payload
) {
    if (!i.has_value()) return {std::nullopt, x::errors::NIL};
    auto &m = this->messages[*i];
    if (auto err = m.decode(payload)) {
        if (this->messages.size() > 1) err = wrap(err, "message " + m.name);
        return {std::nullopt, err};
    }
    return {i, x::errors::NIL};
}

std::pair<std::optional<std::size_t>, x::errors::Error>
Binary::decode(const std::span<const std::uint8_t> payload) {
    std::optional<std::size_t> found;
    for (const auto &h: this->headers) {
        if (payload.size() < h.bits.end()) continue;
        const auto it = h.messages.find(h.bits.read(payload.data()));
        if (it == h.messages.end()) continue;
        if (found.has_value())
            return {
                std::nullopt,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "frame matches messages " +
                        this->messages[std::min(*found, it->second)].name + " and " +
                        this->messages[std::max(*found, it->second)].name
                ),
            };
        found = it->second;
    }
    if (!found.has_value()) found = this->fallback;
    return this->decode_as(found, payload);
}

std::pair<std::optional<std::size_t>, x::errors::Error> Binary::decode(
    const std::uint32_t id,
    const bool extended,
    const std::span<const std::uint8_t> payload
) {
    std::optional<std::size_t> found;
    if (const auto it = this->exact.find(id | (extended ? EXTENDED_BIT : 0));
        it != this->exact.end())
        found = it->second;
    else
        for (const auto &m: this->masked)
            if (m.extended == extended && (id & m.mask) == m.id) {
                found = m.message;
                break;
            }
    if (!found.has_value()) found = this->fallback;
    return this->decode_as(found, payload);
}
}
