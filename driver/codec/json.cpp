// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/field.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/text.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/errors.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;
namespace payload = x::payload;

/// @brief returns the payload kind of samples of dt, or nothing when no field kind
/// writes to dt.
std::optional<std::string_view> kind_of(const x::telem::DataType &dt) {
    if (dt == x::telem::FLOAT32_T || dt == x::telem::FLOAT64_T)
        return payload::KIND_FLOAT;
    if (dt == x::telem::INT8_T || dt == x::telem::INT16_T || dt == x::telem::INT32_T ||
        dt == x::telem::INT64_T)
        return payload::KIND_INTEGER;
    if (dt == x::telem::UINT8_T || dt == x::telem::UINT16_T ||
        dt == x::telem::UINT32_T || dt == x::telem::UINT64_T)
        return payload::KIND_UNSIGNED;
    if (dt == x::telem::BOOLEAN_T) return payload::KIND_BOOLEAN;
    if (dt == x::telem::STRING_T) return payload::KIND_STRING;
    if (dt == x::telem::TIMESTAMP_T) return payload::KIND_TIMESTAMP;
    if (dt == x::telem::JSON_T) return payload::KIND_JSON;
    if (dt == x::telem::UUID_T) return payload::KIND_UUID;
    if (dt == x::telem::BYTES_T) return payload::KIND_BYTES;
    return std::nullopt;
}

/// @brief reports whether a field of kind k converts to samples of dt. A number or
/// boolean also converts to a float, and an unsigned number or boolean to any integer.
bool converts(const std::string_view k, const x::telem::DataType &dt) {
    const auto dk = kind_of(dt);
    if (!dk.has_value()) return false;
    if (*dk == k) return true;
    if (*dk == payload::KIND_FLOAT)
        return k == payload::KIND_INTEGER || k == payload::KIND_UNSIGNED ||
               k == payload::KIND_BOOLEAN;
    if (*dk == payload::KIND_INTEGER)
        return k == payload::KIND_UNSIGNED || k == payload::KIND_BOOLEAN;
    if (*dk == payload::KIND_UNSIGNED) return k == payload::KIND_BOOLEAN;
    return false;
}

/// @brief returns the error err with context before its text.
x::errors::Error wrap(const x::errors::Error &err, const std::string &context) {
    return {err, context + ": " + err.data};
}

/// @brief returns err as a CONFIG_ERROR with context before its text.
x::errors::Error as_config(const x::errors::Error &err, const std::string &context) {
    return {CONFIG_ERROR, context + ": " + err.data};
}

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
};

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
                out.timestamp_fallback = v.default_;
            } else if constexpr (std::is_same_v<F, icd::JSONJSONField>)
                out.kind = payload::KIND_JSON;
            else if constexpr (std::is_same_v<F, icd::UuidJSONField>)
                out.kind = payload::KIND_UUID;
            else {
                static_assert(std::is_same_v<F, icd::BytesJSONField>);
                out.kind = payload::KIND_BYTES;
            }
            return {std::move(out), x::errors::NIL};
        },
        jf
    );
}

const icd::JSONBaseField &base_of(const icd::JSONField &f) {
    return std::visit([](const auto &v) -> const icd::JSONBaseField & { return v; }, f);
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
    const auto &b = base_of(f);
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
        auto [number, err] = x::json::Decoder::create(
            {.data_type = x::telem::FLOAT64_T, .range = v.sample.range}
        );
        if (err) return {Field{}, err};
        field.number = std::move(number);
        // A scaled value need not be an integer.
        v.sample.range = x::telem::UNKNOWN_T;
    }
    auto [sample, sample_err] = x::json::Decoder::create(std::move(v.sample));
    if (sample_err) return {Field{}, sample_err};
    field.sample = std::move(sample);
    if (v.timestamp_fallback.has_value()) {
        field.fallback = *v.timestamp_fallback;
        return {std::move(field), x::errors::NIL};
    }
    auto dst = one_sample(dt);
    x::errors::Error err;
    if (v.float_fallback.has_value())
        err = field.append_float(*v.float_fallback, dst);
    else if (v.fallback.has_value())
        err = field.decode(*v.fallback, dst);
    else
        return {std::move(field), x::errors::NIL};
    if (err) return {Field{}, wrap(err, "default")};
    field.fallback = dst.at(0);
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
    // An enum label maps straight to a channel value, which takes no scale.
    if (!this->scaled || raw.front() == '"') return this->sample.append(raw, dst);
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
    std::vector<x::json::Pointer> pointers;
    if (j.router.ruled) pointers.push_back(j.router.pointer);
    j.messages.reserve(messages.size());
    for (const auto &m: messages) {
        Message msg{.name = m.name, .first_value = pointers.size()};
        std::map<icd::FieldKey, const icd::JSONField *> by_key;
        for (const auto &f: m.payload.fields)
            by_key[base_of(f).key] = &f;
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
                        "message " + m.name + ": field " + base_of(*found->second).name
                    ),
                };
            pointers.push_back(field.pointer);
            msg.fields.push_back(std::move(field));
            msg.samples.push_back(one_sample(b.data_type));
        }
        j.messages.push_back(std::move(msg));
    }
    j.extractor = x::json::Extractor(std::move(pointers));
    return {std::move(j), x::errors::NIL};
}

std::pair<std::optional<std::size_t>, x::errors::Error>
JSON::decode(const std::string_view doc) {
    if (const auto err = this->extractor.extract(doc, this->values))
        return {std::nullopt, err};
    std::optional<std::string_view> match;
    if (this->router.ruled) match = this->values[0];
    auto [message, err] = this->router.find(match);
    if (err || !message.has_value()) return {std::nullopt, err};
    auto &m = this->messages[*message];
    if (auto decode_err = m.decode(this->values)) {
        if (this->messages.size() > 1)
            decode_err = wrap(decode_err, "message " + m.name);
        return {std::nullopt, decode_err};
    }
    return {message, x::errors::NIL};
}

x::errors::Error
JSON::Message::decode(const std::vector<std::optional<std::string_view>> &values) {
    for (std::size_t i = 0; i < this->fields.size(); i++) {
        auto &f = this->fields[i];
        auto &dst = this->samples[i];
        dst.clear();
        const auto &raw = values[this->first_value + i];
        x::errors::Error err;
        if (raw.has_value() && !absent(*raw, f.data_type))
            err = f.decode(*raw, dst);
        else if (f.fallback.has_value())
            dst.write(*f.fallback);
        else
            err = x::errors::Error(
                x::errors::VALIDATION,
                "no value at " + f.pointer.to_string()
            );
        if (err) return wrap(err, "field " + f.name);
    }
    return x::errors::NIL;
}
}
