// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstdint>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_set>
#include <utility>
#include <variant>
#include <vector>

#include "x/cpp/base64/base64.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/field.h"
#include "x/cpp/json/literal.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/text.h"
#include "x/cpp/json/timestamp.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

namespace x::json {
namespace {
errors::Error invalid(const std::string &message) {
    return errors::Error(errors::VALIDATION, message);
}

/// @brief returns true when dt holds numbers: an integer, float, or boolean type.
bool numeric(const telem::DataType &dt) {
    return dt == telem::FLOAT64_T || dt == telem::FLOAT32_T || dt == telem::BOOLEAN_T ||
           dt == telem::INT64_T || dt == telem::INT32_T || dt == telem::INT16_T ||
           dt == telem::INT8_T || dt == telem::UINT64_T || dt == telem::UINT32_T ||
           dt == telem::UINT16_T || dt == telem::UINT8_T;
}

bool integer(const telem::DataType &dt) {
    return numeric(dt) && dt != telem::FLOAT64_T && dt != telem::FLOAT32_T &&
           dt != telem::BOOLEAN_T;
}

/// @brief returns the kind of raw, one valid JSON value: '"', '0' for a number, 't',
/// 'f', 'n', '{', or '['.
char kind_of(const std::string_view raw) {
    const char c = raw.empty() ? '\0' : raw[0];
    return c == '-' || (c >= '0' && c <= '9') ? '0' : c;
}

std::string kind_name(const std::string_view raw) {
    switch (kind_of(raw)) {
        case '"':
            return "a string";
        case '0':
            return "a number";
        case 't':
        case 'f':
            return "a boolean";
        case 'n':
            return "null";
        case '{':
            return "an object";
        default:
            return "an array";
    }
}

/// @brief returns true when JSON values convert to and from samples of dt.
bool convertible(const telem::DataType &dt) {
    return dt == telem::TIMESTAMP_T || dt == telem::STRING_T || dt == telem::JSON_T ||
           dt == telem::UUID_T || dt == telem::BYTES_T || numeric(dt);
}

/// @brief returns v as a Number, or nothing when it is not numeric.
std::optional<Number> to_number(const telem::SampleValue &v) {
    return std::visit(
        [](const auto &s) -> std::optional<Number> {
            using T = std::decay_t<decltype(s)>;
            if constexpr (std::is_floating_point_v<T>)
                return Number(static_cast<double>(s));
            else if constexpr (std::is_integral_v<T> && std::is_signed_v<T>)
                return Number(static_cast<std::int64_t>(s));
            else if constexpr (std::is_integral_v<T>)
                return Number(static_cast<std::uint64_t>(s));
            else
                return std::nullopt;
        },
        v
    );
}

/// @brief checks that the value of e converts to a sample of dt.
/// @returns VALIDATION, naming the label of e, when it does not.
errors::Error check_enum(const Enum &e, const telem::DataType &dt) {
    const auto wrap = [&e](const std::string &message) {
        return invalid("enum label " + quote(e.label) + ": " + message);
    };
    if (dt == telem::STRING_T && std::holds_alternative<std::string>(e.value))
        return errors::NIL;
    const auto n = to_number(e.value);
    if (!numeric(dt) || !n.has_value())
        return wrap(
            "a " + telem::DataType::infer(e.value).name() +
            " value does not convert to " + dt.name()
        );
    if (const auto err = n->check_fits(dt)) return wrap(err.data);
    if (const auto f = n->to_float64(); dt == telem::BOOLEAN_T && f != 0 && f != 1)
        return wrap(n->to_string() + " is not a boolean");
    return errors::NIL;
}

bool is_nan(const telem::SampleValue &v) {
    const auto n = to_number(v);
    return n.has_value() && std::isnan(n->to_float64());
}

/// @brief marks an open array on the stack that compact keeps.
constexpr std::size_t ARRAY = static_cast<std::size_t>(-1);

/// @brief appends raw, one valid JSON object or array, to dst without whitespace. open
/// and names are scratch space: open holds ARRAY for each open array, and for each open
/// object the index of its set of member names in names.
/// @returns VALIDATION when an object repeats a member name.
errors::Error compact(
    const std::string_view raw,
    std::string &dst,
    std::vector<std::size_t> &open,
    std::vector<std::unordered_set<std::string>> &names
) {
    open.clear();
    std::size_t objects = 0;
    std::string name;
    char last = '\0';
    for (std::size_t i = 0; i < raw.size();) {
        const char c = raw[i];
        if (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
            i++;
            continue;
        }
        if (c != '"') {
            if (c == '{') {
                if (objects == names.size()) names.emplace_back();
                names[objects].clear();
                open.push_back(objects++);
            }
            if (c == '[') open.push_back(ARRAY);
            if (c == '}') objects--;
            if (c == '}' || c == ']') open.pop_back();
            dst += c;
            last = c;
            i++;
            continue;
        }
        const bool member = !open.empty() && open.back() != ARRAY &&
                            (last == '{' || last == ',');
        name.clear();
        const auto scan = scan_literal(raw.substr(i), member ? &name : nullptr);
        if (member && !names[open.back()].insert(name).second)
            return invalid("duplicate object member name " + quote(name));
        dst += raw.substr(i, scan.end);
        last = c;
        i += scan.end;
    }
    return errors::NIL;
}
}

std::pair<Decoder, errors::Error> Decoder::create(Field field) {
    const auto &dt = field.data_type;
    if (!convertible(dt))
        return {Decoder(), invalid("cannot convert a JSON value to " + dt.name())};
    if (field.range != telem::UNKNOWN_T && !integer(field.range))
        return {
            Decoder(),
            invalid("range " + field.range.name() + " is not an integer data type"),
        };
    Decoder d;
    for (const auto &e: field.enums) {
        if (const auto err = check_enum(e, dt)) return {Decoder(), err};
        if (const auto n = to_number(e.value))
            d.labels.push_back({e.label, *n, ""});
        else
            d.labels.push_back({e.label, Number(), std::get<std::string>(e.value)});
    }
    d.field = std::move(field);
    return {std::move(d), errors::NIL};
}

errors::Error Decoder::append(const std::string_view raw, telem::Series &dst) {
    const auto &dt = this->field.data_type;
    if (dt == telem::TIMESTAMP_T) return this->append_timestamp(raw, dst);
    if (dt == telem::STRING_T) return this->append_string(raw, dst);
    if (dt == telem::JSON_T) return this->append_json(raw, dst);
    if (dt == telem::UUID_T) return this->append_uuid(raw, dst);
    if (dt == telem::BYTES_T) return this->append_bytes(raw, dst);
    return this->append_numeric(raw, dst);
}

errors::Error Decoder::append_float(const double v, telem::Series &dst) const {
    const Number n(v);
    if (std::isnan(v) || std::isinf(v))
        return invalid(n.to_string() + " is not finite");
    if (const auto &dt = this->field.data_type; !numeric(dt))
        return invalid("cannot convert a float to " + dt.name());
    return this->append_number(n, dst);
}

errors::Error Decoder::append_string(const std::string_view raw, telem::Series &dst) {
    if (kind_of(raw) != '"') return invalid("expected a string, got " + kind_name(raw));
    if (const auto err = unquote(raw, this->text)) return err;
    if (const auto *l = this->find_label(this->text); l != nullptr) {
        dst.write(l->text);
        return errors::NIL;
    }
    dst.write(this->text);
    return errors::NIL;
}

errors::Error Decoder::append_json(const std::string_view raw, telem::Series &dst) {
    if (const auto kind = kind_of(raw); kind != '{' && kind != '[') {
        dst.write(std::string(raw));
        return errors::NIL;
    }
    this->text.clear();
    if (const auto err = compact(raw, this->text, this->open, this->names)) return err;
    dst.write(this->text);
    return errors::NIL;
}

errors::Error Decoder::append_uuid(const std::string_view raw, telem::Series &dst) {
    if (kind_of(raw) != '"')
        return invalid("expected a UUID string, got " + kind_name(raw));
    if (const auto err = unquote(raw, this->text)) return err;
    const auto [id, err] = uuid::UUID::parse(this->text);
    if (err) return invalid("invalid UUID " + quote(this->text));
    dst.write(id.data(), 1);
    return errors::NIL;
}

errors::Error Decoder::append_bytes(const std::string_view raw, telem::Series &dst) {
    if (kind_of(raw) != '"')
        return invalid("expected a base64 string, got " + kind_name(raw));
    if (const auto err = unquote(raw, this->text)) return err;
    if (this->text.find_first_of("\r\n") != std::string::npos)
        return invalid("base64 must not hold a line break");
    auto [data, err] = base64::decode_strict(this->text);
    if (err) return invalid(err.data);
    dst.write(data);
    return errors::NIL;
}

errors::Error Decoder::append_numeric(const std::string_view raw, telem::Series &dst) {
    switch (kind_of(raw)) {
        case '"': {
            if (const auto err = unquote(raw, this->text)) return err;
            if (const auto *l = this->find_label(this->text); l != nullptr) {
                this->write_number(l->number, dst);
                return errors::NIL;
            }
            const auto [n, err] = Number::parse_quoted(this->text);
            if (err && !this->labels.empty())
                return invalid("unknown enum label " + quote(this->text));
            if (err) return err;
            return this->append_number(n, dst);
        }
        case 't':
            return this->append_number(Number(std::int64_t{1}), dst);
        case 'f':
            return this->append_number(Number(std::int64_t{0}), dst);
        case '0': {
            const auto [n, err] = Number::parse(raw);
            if (err) return err;
            return this->append_number(n, dst);
        }
        default:
            return invalid("expected a number or boolean, got " + kind_name(raw));
    }
}

bool Decoder::labeled(const std::string_view raw) {
    if (this->labels.empty() || kind_of(raw) != '"' || unquote(raw, this->text))
        return false;
    return this->find_label(this->text) != nullptr;
}

errors::Error
Decoder::append_timestamp(const std::string_view raw, telem::Series &dst) {
    const auto format = this->field.time_format;
    const auto kind = kind_of(raw);
    std::pair<telem::TimeStamp, errors::Error> parsed;
    if (format == TimeFormat::ISO8601) {
        if (kind != '"')
            return invalid("expected an ISO 8601 string, got " + kind_name(raw));
        if (const auto err = unquote(raw, this->text)) return err;
        parsed = parse_iso8601(this->text);
    } else if (kind == '0')
        parsed = parse_unix(raw, format);
    else if (kind == '"') {
        if (const auto err = unquote(raw, this->text)) return err;
        parsed = parse_unix(this->text, format);
    } else
        return invalid("expected a number, got " + kind_name(raw));
    if (parsed.second) return parsed.second;
    dst.write(parsed.first);
    return errors::NIL;
}

errors::Error Decoder::append_number(const Number &n, telem::Series &dst) const {
    if (this->field.range != telem::UNKNOWN_T)
        if (const auto err = n.check_fits(this->field.range)) return err;
    if (const auto err = n.check_fits(this->field.data_type)) return err;
    this->write_number(n, dst);
    return errors::NIL;
}

void Decoder::write_number(const Number &n, telem::Series &dst) const {
    const auto &dt = this->field.data_type;
    if (dt == telem::FLOAT64_T)
        dst.write(n.to_float64());
    else if (dt == telem::FLOAT32_T)
        dst.write(static_cast<float>(n.to_float64()));
    else if (dt == telem::BOOLEAN_T)
        dst.write(static_cast<std::uint8_t>(n.to_float64() != 0));
    else if (dt == telem::INT64_T)
        dst.write(n.to_int64().value());
    else if (dt == telem::INT32_T)
        dst.write(static_cast<std::int32_t>(n.to_int64().value()));
    else if (dt == telem::INT16_T)
        dst.write(static_cast<std::int16_t>(n.to_int64().value()));
    else if (dt == telem::INT8_T)
        dst.write(static_cast<std::int8_t>(n.to_int64().value()));
    else if (dt == telem::UINT64_T)
        dst.write(n.to_uint64().value());
    else if (dt == telem::UINT32_T)
        dst.write(static_cast<std::uint32_t>(n.to_uint64().value()));
    else if (dt == telem::UINT16_T)
        dst.write(static_cast<std::uint16_t>(n.to_uint64().value()));
    else
        dst.write(static_cast<std::uint8_t>(n.to_uint64().value()));
}

const Decoder::Label *Decoder::find_label(const std::string_view text) const {
    for (const auto &l: this->labels)
        if (l.label == text) return &l;
    return nullptr;
}

std::pair<Encoder, errors::Error> Encoder::create(Field field) {
    const auto &dt = field.data_type;
    if (!convertible(dt))
        return {Encoder(), invalid("cannot convert " + dt.name() + " to a JSON value")};
    if (dt != telem::TIMESTAMP_T && !numeric(dt) && field.type != Type::String)
        return {
            Encoder(),
            invalid("data type " + dt.name() + " only serializes as a string"),
        };
    Encoder e;
    for (const auto &en: field.enums) {
        if (const auto err = check_enum(en, dt)) return {Encoder(), err};
        std::string text;
        if (const auto err = append_quoted(en.label, text))
            return {
                Encoder(),
                invalid("enum label " + quote(en.label) + ": " + err.data),
            };
        e.labels.push_back({std::move(text), dt.cast(en.value)});
    }
    e.field = std::move(field);
    return {std::move(e), errors::NIL};
}

errors::Error
Encoder::append(const telem::Series &s, const std::size_t i, std::string &out) const {
    const auto &dt = this->field.data_type;
    const auto index = static_cast<int>(i);
    if (dt == telem::TIMESTAMP_T) {
        append_timestamp(s.at<telem::TimeStamp>(index), this->field.time_format, out);
        return errors::NIL;
    }
    if (dt == telem::UUID_T) {
        out += '"';
        out += s.at<uuid::UUID>(index).to_string();
        out += '"';
        return errors::NIL;
    }
    if (dt == telem::BYTES_T) {
        out += '"';
        out += base64::encode(s.at<std::string>(index));
        out += '"';
        return errors::NIL;
    }
    auto sample = s.at(index);
    // A boolean matches and writes as 0 or 1.
    if (dt == telem::BOOLEAN_T) sample = dt.cast(sample);
    for (const auto &l: this->labels)
        if (l.sample == sample || (is_nan(l.sample) && is_nan(sample))) {
            out += l.text;
            return errors::NIL;
        }
    if (!numeric(dt)) return append_quoted(std::get<std::string>(sample), out);
    const auto n = to_number(sample).value();
    if (this->field.type == Type::Boolean) {
        if (std::isnan(n.to_float64())) return invalid("NaN is not a JSON boolean");
        out += n.to_float64() != 0 ? "true" : "false";
        return errors::NIL;
    }
    if (this->field.type == Type::String) {
        out += '"';
        out += n.to_string();
        out += '"';
        return errors::NIL;
    }
    if (dt == telem::FLOAT64_T || dt == telem::FLOAT32_T)
        return append_float(n.to_float64(), out);
    out += n.to_string();
    return errors::NIL;
}
}
