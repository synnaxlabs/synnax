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
#include <cstdint>
#include <map>
#include <optional>
#include <span>
#include <stdexcept>
#include <string>
#include <string_view>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/base64/base64.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/field.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/template.h"
#include "x/cpp/json/text.h"
#include "x/cpp/json/timestamp.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/field.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;
namespace payload = x::payload;

/// @brief Variant is what the variant of a JSON field sets: the options of its samples
/// and its default.
struct Variant {
    std::string_view kind;
    x::json::Field sample;
    /// @brief fallback is the default of a field that is not a float or timestamp, as a
    /// JSON value.
    std::optional<std::string> fallback;
    std::optional<double> float_fallback;
    std::optional<x::telem::TimeStamp> timestamp_fallback;
    void (*generate)(x::telem::Series &dst) = nullptr;
};

void append_now(x::telem::Series &dst) {
    dst.write(x::telem::TimeStamp::now());
}

void append_random_uuid(x::telem::Series &dst) {
    const auto id = x::uuid::create();
    dst.write(id.data(), 1);
}

template<typename Entries, typename Convert>
std::vector<x::json::Enum> enums_of(const Entries &entries, const Convert &convert) {
    std::vector<x::json::Enum> out;
    out.reserve(entries.size());
    for (const auto &e: entries)
        out.push_back({.label = e.label, .value = convert(e.value)});
    return out;
}

std::pair<Variant, x::errors::Error>
read_variant(const icd::JSONField &jf, const x::telem::DataType &dt) {
    Variant out{.sample = {.data_type = dt}};
    return std::visit(
        [&]<typename F>(const F &v) -> std::pair<Variant, x::errors::Error> {
            if constexpr (std::is_same_v<F, icd::FloatJSONField>) {
                out.kind = payload::KIND_FLOAT;
                out.sample.enums = enums_of(v.enum_values, [](const auto &n) {
                    return x::telem::SampleValue{n.value};
                });
                if (v.default_.has_value()) out.float_fallback = v.default_->value;
            } else if constexpr (std::is_same_v<F, icd::IntegerJSONField>) {
                out.kind = payload::KIND_INTEGER;
                out.sample.range = x::telem::INT64_T;
                out.sample.enums = enums_of(v.enum_values, [](const auto &n) {
                    return x::telem::SampleValue{n.value};
                });
                if (v.default_.has_value())
                    out.fallback = std::to_string(v.default_->value);
            } else if constexpr (std::is_same_v<F, icd::UnsignedJSONField>) {
                out.kind = payload::KIND_UNSIGNED;
                out.sample.range = x::telem::UINT64_T;
                out.sample.enums = enums_of(v.enum_values, [](const auto &n) {
                    return x::telem::SampleValue{n.value};
                });
                if (v.default_.has_value())
                    out.fallback = std::to_string(v.default_->value);
            } else if constexpr (std::is_same_v<F, icd::StringJSONField>) {
                out.kind = payload::KIND_STRING;
                out.sample.enums = enums_of(v.enum_values, [](const std::string &s) {
                    return x::telem::SampleValue{s};
                });
                if (v.default_.has_value())
                    out.fallback = x::json::json(*v.default_).dump();
            } else if constexpr (std::is_same_v<F, icd::BooleanJSONField>) {
                out.kind = payload::KIND_BOOLEAN;
                out.sample.enums = enums_of(v.enum_values, [](const bool b) {
                    return x::telem::SampleValue{static_cast<std::uint8_t>(b)};
                });
                if (v.default_.has_value())
                    out.fallback = *v.default_ ? "true" : "false";
            } else if constexpr (std::is_same_v<F, icd::TimestampJSONField>) {
                out.kind = payload::KIND_TIMESTAMP;
                auto [format, err] = x::json::parse_time_format(v.time_format);
                if (err) return {{}, err};
                out.sample.time_format = format;
                if (v.default_.has_value())
                    std::visit(
                        [&]<typename D>(const D &d) {
                            if constexpr (
                                std::is_same_v<D, icd::ConstantTimestampDefault>
                            )
                                out.timestamp_fallback = d.value;
                            else {
                                static_assert(
                                    std::is_same_v<D, icd::NowTimestampDefault>
                                );
                                out.generate = append_now;
                            }
                        },
                        *v.default_
                    );
            } else if constexpr (std::is_same_v<F, icd::JSONJSONField>) {
                out.kind = payload::KIND_JSON;
                if (v.default_.has_value()) {
                    if (!x::json::valid(*v.default_))
                        return {
                            {},
                            x::errors::Error(
                                x::errors::VALIDATION,
                                "default: not valid JSON"
                            ),
                        };
                    out.fallback = *v.default_;
                }
            } else if constexpr (std::is_same_v<F, icd::UuidJSONField>) {
                out.kind = payload::KIND_UUID;
                if (v.default_.has_value())
                    std::visit(
                        [&]<typename D>(const D &d) {
                            if constexpr (std::is_same_v<D, icd::ConstantUUIDDefault>)
                                out.fallback = x::json::json(d.value.to_string())
                                                   .dump();
                            else {
                                static_assert(
                                    std::is_same_v<D, icd::RandomUUIDDefault>
                                );
                                out.generate = append_random_uuid;
                            }
                        },
                        *v.default_
                    );
            } else {
                static_assert(std::is_same_v<F, icd::BytesJSONField>);
                out.kind = payload::KIND_BYTES;
                if (v.default_.has_value())
                    out.fallback = x::json::json(*v.default_).dump();
            }
            return {std::move(out), x::errors::NIL};
        },
        jf
    );
}

/// @brief returns a series of dt that has room for one sample.
x::telem::Series one_sample(const x::telem::DataType &dt) {
    return {dt, dt.is_variable() ? std::size_t{0} : std::size_t{1}};
}

/// @brief reports whether raw, the value at the pointer of a field, is no value for a
/// channel of data type dt: null except on a JSON channel, or an empty string on a
/// channel that cannot hold one. A string, bytes, or JSON channel holds one.
bool absent(const std::string_view raw, const x::telem::DataType &dt) {
    if (dt == x::telem::JSON_T) return false;
    if (raw.front() == 'n') return true;
    if (dt == x::telem::STRING_T || dt == x::telem::BYTES_T) return false;
    return raw == R"("")";
}
}

std::pair<JSON::Field, x::errors::Error>
JSON::Field::create(const icd::JSONField &f, const x::telem::DataType &dt) {
    const auto &b = icd::base(f);
    auto [pointer, ptr_err] = x::json::Pointer::parse(b.pointer);
    if (ptr_err) return {Field{}, ptr_err};
    Field field;
    field.name = b.name;
    field.pointer = std::move(pointer);
    field.data_type = dt;
    field.scale = b.scale;
    field.offset = b.offset;
    auto [v, variant_err] = read_variant(f, dt);
    if (variant_err) return {Field{}, variant_err};
    field.boolean = v.kind == payload::KIND_BOOLEAN;
    if (!converts(v.kind, dt))
        return {
            Field{},
            x::errors::Error(
                x::errors::VALIDATION,
                "cannot write " + std::string(v.kind) + " values to " + dt.name() +
                    " channels"
            ),
        };
    field.scaled = field.scale != 1 || field.offset != 0;
    if (field.scaled) {
        if (dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T)
            return {
                Field{},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "a scaled field requires a float channel, not " + dt.name()
                ),
            };
        // The enums of number fail a string that is no label as an unknown label. A
        // label itself goes to sample.
        auto [number, err] = x::json::Decoder::create({
            .data_type = x::telem::FLOAT64_T,
            .enums = v.sample.enums,
            .range = v.sample.range,
        });
        if (err) return {Field{}, err};
        field.number = std::move(number);
        // A scaled value need not be an integer.
        v.sample.range = x::telem::UNKNOWN_T;
    }
    auto [sample, sample_err] = x::json::Decoder::create(std::move(v.sample));
    if (sample_err) return {Field{}, sample_err};
    field.sample = std::move(sample);
    field.generate = v.generate;
    auto dst = one_sample(dt);
    x::errors::Error err;
    if (v.timestamp_fallback.has_value())
        dst.write(*v.timestamp_fallback);
    else if (v.float_fallback.has_value())
        err = field.append_float(*v.float_fallback, dst);
    else if (v.fallback.has_value())
        err = field.decode(*v.fallback, dst);
    else
        return {std::move(field), x::errors::NIL};
    if (err) return {Field{}, wrap(err, "default")};
    field.fallback = std::move(dst);
    return {std::move(field), x::errors::NIL};
}

x::errors::Error
JSON::Field::decode(const std::string_view raw, x::telem::Series &dst) {
    if (this->boolean &&
        (raw.front() == '-' || (raw.front() >= '0' && raw.front() <= '9')))
        return x::errors::Error(
            x::errors::VALIDATION,
            "expected a boolean, got number " + std::string(raw)
        );
    if (raw.front() == '"' && (this->boolean || this->scaled)) {
        // An enum label maps straight to a channel value, which takes no scale.
        if (this->sample.labeled(raw)) return this->sample.append(raw, dst);
        if (this->boolean)
            return x::errors::Error(
                x::errors::VALIDATION,
                "expected a boolean, got string " + std::string(raw)
            );
    }
    if (!this->scaled) return this->sample.append(raw, dst);
    this->scratch.clear();
    if (const auto err = this->number.append(raw, this->scratch)) return err;
    return this->append_float(this->scratch.at<double>(0), dst);
}

x::errors::Error JSON::Field::append_float(double v, x::telem::Series &dst) const {
    if (this->scaled) v = v * this->scale + this->offset;
    return this->sample.append_float(v, dst);
}

std::pair<JSON, x::errors::Error>
JSON::create(const std::vector<JSONMessage> &messages) {
    auto [router, router_err] = Router::create(messages);
    if (router_err) return {JSON{}, router_err};
    JSON j;
    j.router = std::move(router);
    auto pointers = j.router.pointers();
    j.messages.reserve(messages.size());
    for (const auto &m: messages) {
        Message msg{.name = m.name, .first_value = pointers.size()};
        std::map<icd::FieldKey, const icd::JSONField *> by_key;
        for (const auto &f: m.payload.fields)
            by_key[icd::base(f).key] = &f;
        for (const auto &b: m.bindings) {
            const auto found = by_key.find(b.field);
            if (found == by_key.end())
                return {
                    JSON{},
                    x::errors::Error(
                        CONFIG_ERROR,
                        "message " + m.name + ": field " + b.field.to_string() +
                            " does not exist"
                    ),
                };
            auto [field, err] = Field::create(*found->second, b.data_type);
            if (err)
                return {
                    JSON{},
                    as_config(
                        err,
                        "message " + m.name + ": field " +
                            icd::base(*found->second).name
                    ),
                };
            pointers.push_back(field.pointer);
            msg.fields.push_back(std::move(field));
            msg.samples.push_back(one_sample(b.data_type));
        }
        msg.failures.reserve(msg.fields.size());
        j.messages.push_back(std::move(msg));
    }
    j.extractor = x::json::Extractor(std::move(pointers));
    return {std::move(j), x::errors::NIL};
}

std::pair<std::optional<std::size_t>, x::errors::Error>
JSON::decode(const std::string_view doc) {
    if (const auto err = this->extractor.extract(doc, this->values))
        return {std::nullopt, err};
    auto [message, err] = this->router.find(this->values);
    if (err || !message.has_value()) return {std::nullopt, err};
    auto &m = this->messages[*message];
    m.decode(this->values);
    if (this->messages.size() > 1)
        for (auto &f: m.failures)
            f.error = wrap(f.error, "message " + m.name);
    return {message, x::errors::NIL};
}

void JSON::Message::decode(const std::vector<std::optional<std::string_view>> &values) {
    this->failures.clear();
    for (std::size_t i = 0; i < this->fields.size(); i++) {
        auto &f = this->fields[i];
        auto &dst = this->samples[i];
        dst.clear();
        const auto &raw = values[this->first_value + i];
        x::errors::Error err;
        const bool missing = !raw.has_value() || absent(*raw, f.data_type);
        if (!missing)
            err = f.decode(*raw, dst);
        else if (f.generate != nullptr)
            f.generate(dst);
        else if (f.fallback.has_value())
            dst.write(*f.fallback);
        else
            err = x::errors::Error(
                x::errors::VALIDATION,
                "no value at " + f.pointer.to_string()
            );
        if (!err) continue;
        dst.clear();
        this->failures.push_back({
            .binding = i,
            .missing = missing,
            .error = wrap(err, "field " + f.name),
        });
    }
}

namespace {
x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(x::errors::VALIDATION, message);
}

/// @brief reports whether k holds numbers or booleans.
bool numeric(const std::string_view k) {
    return k == payload::KIND_FLOAT || k == payload::KIND_INTEGER ||
           k == payload::KIND_UNSIGNED || k == payload::KIND_BOOLEAN;
}

/// @brief reports whether samples of dt write to a field of kind k.
bool writes(const std::string_view k, const x::telem::DataType &dt) {
    const auto dk = kind_of(dt);
    return dk.has_value() && (*dk == k || (numeric(*dk) && numeric(k)));
}

/// @brief reports whether a field of kind k changes or checks the number that an
/// encoder writes for a sample of dt.
bool needs_conversion(
    const std::string_view k,
    const x::telem::DataType &dt,
    const bool scaled,
    const bool bounded
) {
    if (k == payload::KIND_FLOAT) return scaled || bounded;
    if (k == payload::KIND_INTEGER || k == payload::KIND_UNSIGNED)
        return scaled || bounded || kind_of(dt) != k;
    if (k == payload::KIND_BOOLEAN) return kind_of(dt) != k;
    return false;
}

/// @brief reports whether token is "0" or digits that do not start with "0".
bool array_index(const std::string &token) {
    if (token.empty() || (token.size() > 1 && token[0] == '0')) return false;
    return std::ranges::all_of(token, [](const char c) {
        return c >= '0' && c <= '9';
    });
}

/// @brief parses text as a pointer that names object members.
/// @returns VALIDATION for an array index, since a decoder reads one as an element.
std::pair<x::json::Pointer, x::errors::Error> member_pointer(const std::string &text) {
    auto [pointer, err] = x::json::Pointer::parse(text);
    if (err) return {{}, err};
    if (std::ranges::any_of(pointer.tokens, array_index))
        return {
            {},
            invalid(
                "pointer " + x::json::quote(text) +
                " holds an array index, which a document cannot place"
            ),
        };
    return {std::move(pointer), x::errors::NIL};
}

/// @brief returns the template field that writes the value of m, a value rule.
std::pair<x::json::Template::Field, x::errors::Error>
match_literal(const icd::JSONMatch &m) {
    auto [pointer, ptr_err] = member_pointer(icd::base(m).pointer);
    if (ptr_err) return {{}, ptr_err};
    std::string text;
    const auto err = std::visit(
        [&text]<typename M>(const M &v) -> x::errors::Error {
            if constexpr (std::is_same_v<M, icd::FloatJSONMatch>)
                return x::json::append_float(v.value.value, text);
            else if constexpr (
                std::is_same_v<M, icd::IntegerJSONMatch> ||
                std::is_same_v<M, icd::UnsignedJSONMatch>
            )
                text = std::to_string(v.value.value);
            else if constexpr (std::is_same_v<M, icd::StringJSONMatch>)
                return x::json::append_quoted(v.value, text);
            else if constexpr (std::is_same_v<M, icd::BooleanJSONMatch>)
                text = v.value ? "true" : "false";
            else {
                static_assert(std::is_same_v<M, icd::PresentJSONMatch>);
                throw std::logic_error("a present rule has no value to write");
            }
            return x::errors::NIL;
        },
        m
    );
    if (err) return {{}, err};
    return {
        {.label = "match", .pointer = std::move(pointer), .literal = std::move(text)},
        x::errors::NIL,
    };
}
}

std::pair<JSONEncoder::Default, x::errors::Error>
JSONEncoder::Default::create(const icd::JSONField &f) {
    Default d;
    const auto err = std::visit(
        [&d]<typename F>(const F &v) -> x::errors::Error {
            if (!v.default_.has_value()) return x::errors::NIL;
            std::string text;
            if constexpr (std::is_same_v<F, icd::FloatJSONField>) {
                if (!std::isfinite(v.default_->value)) return invalid("must be finite");
                if (auto err = x::json::append_float(v.default_->value, text))
                    return err;
            } else if constexpr (
                std::is_same_v<F, icd::IntegerJSONField> ||
                std::is_same_v<F, icd::UnsignedJSONField>
            )
                text = std::to_string(v.default_->value);
            else if constexpr (std::is_same_v<F, icd::StringJSONField>) {
                if (auto err = x::json::append_quoted(v.default_.value(), text))
                    return err;
            } else if constexpr (std::is_same_v<F, icd::BooleanJSONField>)
                text = v.default_.value() ? "true" : "false";
            else if constexpr (std::is_same_v<F, icd::TimestampJSONField>) {
                auto [format, err] = x::json::parse_time_format(v.time_format);
                if (err) return err;
                if (const auto *c = std::get_if<icd::ConstantTimestampDefault>(
                        &v.default_.value()
                    ))
                    x::json::append_timestamp(c->value, format, text);
                else {
                    d.generator = Generator::NOW;
                    d.time_format = format;
                    return x::errors::NIL;
                }
            } else if constexpr (std::is_same_v<F, icd::JSONJSONField>) {
                if (!x::json::valid(v.default_.value()))
                    return invalid("must be valid JSON");
                text = v.default_.value();
            } else if constexpr (std::is_same_v<F, icd::UuidJSONField>) {
                const auto *c = std::get_if<icd::ConstantUUIDDefault>(
                    &v.default_.value()
                );
                if (c == nullptr) {
                    d.generator = Generator::UUID;
                    return x::errors::NIL;
                }
                text = '"' + c->value.to_string() + '"';
            } else {
                static_assert(std::is_same_v<F, icd::BytesJSONField>);
                if (x::base64::decode_strict(v.default_.value()).second)
                    return invalid("must be base64");
                // Canonical base64 text needs no escapes.
                text = '"' + v.default_.value() + '"';
            }
            d.text = std::move(text);
            return x::errors::NIL;
        },
        f
    );
    if (err) return {Default{}, err};
    return {std::move(d), x::errors::NIL};
}

x::errors::Error JSONEncoder::Default::append(std::string &out) const {
    switch (this->generator) {
        case Generator::NOW:
            x::json::append_timestamp(
                x::telem::TimeStamp::now(),
                this->time_format,
                out
            );
            return x::errors::NIL;
        case Generator::UUID:
            out += '"';
            out += x::uuid::create().to_string();
            out += '"';
            return x::errors::NIL;
        case Generator::NONE:
            break;
    }
    if (!this->text.has_value()) return invalid("no sample and no default");
    out += this->text.value();
    return x::errors::NIL;
}

std::pair<JSONEncoder::Field, x::errors::Error>
JSONEncoder::Field::create(const icd::JSONField &f, const x::telem::DataType &dt) {
    const auto &b = icd::base(f);
    Field field;
    field.name = b.name;
    field.scaled = b.scale != 1 || b.offset != 0;
    field.scale = b.scale;
    field.offset = b.offset;
    field.base = static_cast<const icd::BaseField &>(b);
    auto [v, variant_err] = read_variant(f, dt);
    if (variant_err) return {Field{}, variant_err};
    field.kind = v.kind;
    if (!writes(v.kind, dt))
        return {
            Field{},
            invalid(
                "cannot write " + dt.name() + " channels to " + std::string(v.kind) +
                " fields"
            ),
        };
    if (field.scaled && dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T)
        return {
            Field{},
            invalid("a scaled field requires a float channel, not " + dt.name()),
        };
    field.converted = needs_conversion(
        v.kind,
        dt,
        field.scaled,
        b.min.has_value() || b.max.has_value()
    );
    if (v.kind == payload::KIND_JSON) return {std::move(field), x::errors::NIL};
    auto type = x::json::Type::Number;
    if (v.kind == payload::KIND_STRING || v.kind == payload::KIND_UUID ||
        v.kind == payload::KIND_BYTES)
        type = x::json::Type::String;
    else if (v.kind == payload::KIND_BOOLEAN)
        type = x::json::Type::Boolean;
    auto [encoder, err] = x::json::Encoder::create({
        .data_type = dt,
        .time_format = v.sample.time_format,
        .type = type,
        .enums = std::move(v.sample.enums),
    });
    if (err) return {Field{}, err};
    field.encoder = std::move(encoder);
    return {std::move(field), x::errors::NIL};
}

x::errors::Error JSONEncoder::Field::append(
    const x::telem::Series &s,
    const std::size_t i,
    std::string &out
) const {
    if (this->kind == payload::KIND_JSON) {
        const auto text = s.at<std::string>(static_cast<int>(i));
        if (!x::json::valid(text)) return invalid("invalid JSON");
        out += text;
        return x::errors::NIL;
    }
    if (!this->converted) return this->encoder.append(s, i, out);
    auto sample = s.at(static_cast<int>(i));
    // A boolean matches and converts as 0 or 1.
    if (s.data_type() == x::telem::BOOLEAN_T) sample = x::telem::BOOLEAN_T.cast(sample);
    // An enum label writes a value that the ICD names, with no conversion or check.
    if (const auto *label = this->encoder.label(sample)) {
        out += *label;
        return x::errors::NIL;
    }
    const auto n = x::json::to_number(sample).value();
    const auto v = n.to_float64();
    if (!std::isfinite(v)) return invalid(n.to_string() + " is not a JSON number");
    const auto *narrow = std::get_if<float>(&sample);
    if (auto err = narrow != nullptr ? check_limits(this->base, *narrow)
                                     : check_limits(this->base, v))
        return err;
    if (this->kind == payload::KIND_FLOAT)
        return x::json::append_float((v - this->offset) / this->scale, out);
    if (this->kind != payload::KIND_BOOLEAN) return this->append_integer(n, out);
    if (v != 0 && v != 1)
        return invalid(x::json::Number(v).to_string() + " is not 0 or 1");
    out += v == 1 ? "true" : "false";
    return x::errors::NIL;
}

x::errors::Error
JSONEncoder::Field::append_integer(const x::json::Number &n, std::string &out) const {
    const bool integer = this->kind == payload::KIND_INTEGER;
    if (!this->scaled) {
        if (const auto v = n.to_int64(); integer && v.has_value()) {
            out += std::to_string(*v);
            return x::errors::NIL;
        }
        if (const auto v = n.to_uint64(); !integer && v.has_value()) {
            out += std::to_string(*v);
            return x::errors::NIL;
        }
    }
    auto v = n.to_float64();
    if (this->scaled) v = (v - this->offset) / this->scale;
    const auto r = std::round(v);
    if (integer && r >= -0x1p63 && r < 0x1p63) {
        out += std::to_string(static_cast<std::int64_t>(r));
        return x::errors::NIL;
    }
    if (!integer && r >= 0 && r < 0x1p64) {
        out += std::to_string(static_cast<std::uint64_t>(r));
        return x::errors::NIL;
    }
    return invalid(
        x::json::Number(v).to_string() + " is out of range for " +
        std::string(this->kind) + " values"
    );
}

std::pair<JSONEncoder, x::errors::Error>
JSONEncoder::create(const JSONMessage &message) {
    const auto fail = [&message](const x::errors::Error &err) {
        return std::pair{JSONEncoder{}, as_config(err, "message " + message.name)};
    };
    std::map<icd::FieldKey, const icd::JSONField *> by_key;
    for (const auto &f: message.payload.fields)
        by_key[icd::base(f).key] = &f;
    std::map<icd::FieldKey, std::size_t> bound;
    for (std::size_t i = 0; i < message.bindings.size(); i++) {
        const auto &key = message.bindings[i].field;
        const auto found = by_key.find(key);
        if (found == by_key.end())
            return fail(invalid("field " + key.to_string() + " does not exist"));
        if (!bound.try_emplace(key, i).second)
            return fail(invalid(
                "field " + icd::base(*found->second).name + " has more than one binding"
            ));
    }
    JSONEncoder e;
    e.name = message.name;
    e.fields.resize(message.bindings.size());
    std::vector<x::json::Template::Field> fields;
    const icd::PresentJSONMatch *present = nullptr;
    if (const auto &match = message.payload.match; match.has_value()) {
        present = std::get_if<icd::PresentJSONMatch>(&match.value());
        if (present == nullptr) {
            auto [f, err] = match_literal(match.value());
            if (err) return fail(wrap(err, "match"));
            fields.push_back(std::move(f));
            e.slots.emplace_back();
        }
    }
    for (const auto &f: message.payload.fields) {
        const auto &b = icd::base(f);
        auto [pointer, ptr_err] = member_pointer(b.pointer);
        if (ptr_err) return fail(wrap(ptr_err, "field " + b.name));
        auto [fallback, def_err] = Default::create(f);
        if (def_err) return fail(wrap(def_err, "field " + b.name + ": default"));
        x::json::Template::Field tf{.label = "field " + b.name, .pointer = pointer};
        const auto binding = bound.find(b.key);
        if (binding == bound.end()) {
            if (!fallback.defined())
                return fail(
                    invalid("field " + b.name + ": needs a channel or a default")
                );
            tf.literal = fallback.text;
            fields.push_back(std::move(tf));
            e.slots.push_back({.fallback = std::move(fallback)});
            continue;
        }
        const auto i = binding->second;
        auto [field, err] = Field::create(f, message.bindings[i].data_type);
        if (err) return fail(wrap(err, "field " + b.name));
        field.fallback = std::move(fallback);
        e.fields[i] = std::move(field);
        fields.push_back(std::move(tf));
        e.slots.push_back({.binding = i});
    }
    if (present != nullptr) {
        auto [at, err] = x::json::Pointer::parse(present->pointer);
        if (err) return fail(wrap(err, "match"));
        if (std::ranges::none_of(fields, [&at](const auto &f) {
                return within(f.pointer, at);
            }))
            return fail(invalid(
                "match: no field is at or inside " + x::json::quote(present->pointer)
            ));
    }
    auto [document, err] = x::json::Template::create(fields);
    if (err) return fail(err);
    e.document = std::move(document);
    return {std::move(e), x::errors::NIL};
}

x::errors::Error JSONEncoder::encode(
    const std::span<const x::telem::Series> samples,
    std::vector<std::uint8_t> &payload
) {
    this->text.clear();
    const auto err = this->document.append(
        this->text,
        [this, &samples](const std::size_t slot, std::string &out) -> x::errors::Error {
            const auto &s = this->slots[slot];
            if (!s.binding.has_value()) return s.fallback.append(out);
            const auto &f = this->fields[*s.binding];
            const auto &sample = samples[*s.binding];
            const auto err = sample.empty() ? f.fallback.append(out)
                                            : f.append(sample, 0, out);
            if (err) return wrap(err, "field " + f.name);
            return x::errors::NIL;
        }
    );
    if (err) return wrap(err, "message " + this->name);
    payload.assign(this->text.begin(), this->text.end());
    return x::errors::NIL;
}

x::errors::Error JSONEncoder::check(
    const std::size_t binding,
    const x::telem::Series &s,
    const std::size_t i
) {
    const auto &f = this->fields[binding];
    this->text.clear();
    if (auto err = f.append(s, i, this->text))
        return wrap(err, "message " + this->name + ": field " + f.name);
    return x::errors::NIL;
}
}
