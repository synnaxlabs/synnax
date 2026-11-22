// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <bit>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <limits>
#include <map>
#include <optional>
#include <span>
#include <string>
#include <string_view>
#include <type_traits>
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

x::errors::Error config_error(const std::string &text) {
    return x::errors::Error(CONFIG_ERROR, text);
}
}

std::pair<Binary::Field, x::errors::Error>
Binary::Field::create(const icd::BinaryField &f, const x::telem::DataType &dt) {
    const auto &b = icd::base(f);
    if (b.multiplexor.has_value())
        return {Field{}, config_error("multiplexed fields are not supported yet")};
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
    std::string_view kind;
    if (std::holds_alternative<icd::FloatBinaryField>(f)) {
        kind = payload::KIND_FLOAT;
        if (b.bit_length != 32 && b.bit_length != 64)
            return {
                Field{},
                config_error(
                    "a float field must be 32 or 64 bits, not " +
                    std::to_string(b.bit_length)
                ),
            };
        field.type = b.bit_length == 32 ? Type::FLOAT32 : Type::FLOAT64;
    } else if (std::holds_alternative<icd::IntegerBinaryField>(f)) {
        kind = payload::KIND_INTEGER;
        field.type = Type::SIGNED;
    } else {
        kind = payload::KIND_UNSIGNED;
        field.type = Type::UNSIGNED;
    }
    if (!converts(kind, dt))
        return {
            Field{},
            config_error(
                "cannot write " + std::string(kind) + " values to " + dt.name() +
                " channels"
            ),
        };
    if (field.scaled && dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T)
        return {
            Field{},
            config_error("a scaled field requires a float channel, not " + dt.name()),
        };
    if (const auto w = width_of(dt); w.has_value()) {
        const bool is_signed = field.type == Type::SIGNED;
        const unsigned needed = b.bit_length + (w->is_signed && !is_signed ? 1U : 0U);
        if (needed > w->bits)
            return {
                Field{},
                config_error(
                    "cannot write " + std::to_string(b.bit_length) + "-bit " +
                    std::string(kind) + " values to " + dt.name() + " channels"
                ),
            };
        field.put = is_signed ? put_for<true>(dt) : put_for<false>(dt);
    }
    auto dst = x::telem::Series(dt, 1);
    const auto n = b.bit_length;
    const auto err = std::visit(
        [&]<typename F>(const F &v) -> x::errors::Error {
            if (!v.default_.has_value()) return x::errors::NIL;
            if constexpr (std::is_same_v<F, icd::FloatBinaryField>) {
                double d = v.default_->value;
                if (!std::isfinite(d))
                    return config_error(
                        x::json::Number(d).to_string() + " is not finite"
                    );
                if (n == 32) {
                    if (std::abs(d) > std::numeric_limits<float>::max())
                        return config_error(
                            x::json::Number(d).to_string() + " overflows float32"
                        );
                    d = static_cast<float>(d);
                }
                return field.append_float(d, dst);
            } else if constexpr (std::is_same_v<F, icd::IntegerBinaryField>) {
                const auto i = v.default_->value;
                const auto shift = 64 - n;
                if ((i << shift) >> shift != i)
                    return config_error(
                        std::to_string(i) + " is out of range for " +
                        std::to_string(n) + " bits"
                    );
                return field.append(static_cast<std::uint64_t>(i), dst);
            } else {
                const auto u = v.default_->value;
                if (n < 64 && u >> n != 0)
                    return config_error(
                        std::to_string(u) + " is out of range for " +
                        std::to_string(n) + " bits"
                    );
                return field.append(u, dst);
            }
        },
        f
    );
    if (err) return {Field{}, as_config(err, "default")};
    if (!dst.empty()) field.fallback = std::move(dst);
    return {std::move(field), x::errors::NIL};
}

x::errors::Error
Binary::Field::append(const std::uint64_t raw, x::telem::Series &dst) const {
    if (this->put != nullptr) {
        this->put(dst, raw);
        return x::errors::NIL;
    }
    double v = 0;
    switch (this->type) {
        case Type::UNSIGNED:
            v = static_cast<double>(raw);
            break;
        case Type::SIGNED:
            v = static_cast<double>(static_cast<std::int64_t>(raw));
            break;
        case Type::FLOAT32:
            v = std::bit_cast<float>(static_cast<std::uint32_t>(raw));
            break;
        case Type::FLOAT64:
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
        return x::errors::Error(
            x::errors::VALIDATION,
            "the payload has " + std::to_string(payload.size()) +
                " bytes and the field needs " + std::to_string(this->bits.end())
        );
    }
    const auto *p = payload.data();
    if (this->type == Type::SIGNED)
        return this->append(static_cast<std::uint64_t>(this->bits.read_signed(p)), dst);
    return this->append(this->bits.read(p), dst);
}

x::errors::Error Binary::Message::decode(const std::span<const std::uint8_t> payload) {
    for (std::size_t i = 0; i < this->fields.size(); i++) {
        auto &dst = this->samples[i];
        dst.clear();
        if (const auto err = this->fields[i].decode(payload, dst))
            return wrap(err, "field " + this->fields[i].name);
    }
    return x::errors::NIL;
}

std::pair<Binary, x::errors::Error>
Binary::create(const std::vector<BinaryMessage> &messages) {
    Binary bin;
    bin.messages.reserve(messages.size());
    for (std::size_t i = 0; i < messages.size(); i++) {
        const auto &m = messages[i];
        if (m.payload.identifier.has_value())
            return {
                Binary{},
                config_error(
                    "message " + m.name + ": identifiers are not supported yet"
                ),
            };
        if (bin.fallback.has_value())
            return {
                Binary{},
                config_error(
                    "messages " + messages[*bin.fallback].name + " and " + m.name +
                    " both have no identifier"
                ),
            };
        bin.fallback = i;
        Message msg{.name = m.name};
        std::map<icd::FieldKey, const icd::BinaryField *> by_key;
        for (const auto &f: m.payload.fields)
            by_key[icd::base(f).key] = &f;
        for (const auto &b: m.bindings) {
            const auto found = by_key.find(b.field);
            if (found == by_key.end())
                return {
                    Binary{},
                    config_error(
                        "message " + m.name + ": field " + b.field.to_string() +
                        " does not exist"
                    ),
                };
            const auto &base = icd::base(*found->second);
            const auto context = "message " + m.name + ": field " + base.name;
            auto [field, err] = Field::create(*found->second, b.data_type);
            if (err) return {Binary{}, as_config(err, context)};
            if (const auto length = m.payload.length;
                length.has_value() && field.bits.end() > *length)
                return {
                    Binary{},
                    config_error(
                        context + ": the field ends past the " +
                        std::to_string(*length) + "-byte payload"
                    ),
                };
            msg.fields.push_back(std::move(field));
            msg.samples.emplace_back(b.data_type, 1);
        }
        bin.messages.push_back(std::move(msg));
    }
    return {std::move(bin), x::errors::NIL};
}

std::pair<std::optional<std::size_t>, x::errors::Error>
Binary::decode(const std::span<const std::uint8_t> payload) {
    if (!this->fallback.has_value()) return {std::nullopt, x::errors::NIL};
    auto &m = this->messages[*this->fallback];
    if (auto err = m.decode(payload)) {
        if (this->messages.size() > 1) err = wrap(err, "message " + m.name);
        return {std::nullopt, err};
    }
    return {this->fallback, x::errors::NIL};
}
}
