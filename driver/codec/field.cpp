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
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <utility>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/errors.h"
#include "driver/codec/field.h"

namespace driver::codec {
namespace payload = x::payload;
namespace icd = synnax::icd;

namespace {
/// @brief DECODE_ROUNDING bounds the float64 rounding of raw × scale + offset, relative
/// to |raw × scale| + |offset|.
constexpr double DECODE_ROUNDING = 4 * 0x1p-52;

std::string format(const double v) {
    return x::json::Number(v).to_string();
}

/// @returns VALIDATION when decoded, the decoded value of raw, is more than slack
/// outside the min and max of b.
x::errors::Error out_of_range(
    const icd::BaseField &b,
    const double raw,
    const double decoded,
    const double slack
) {
    std::string side;
    double limit = 0;
    if (b.min.has_value() && decoded < *b.min - slack) {
        side = "below the min";
        limit = *b.min;
    } else if (b.max.has_value() && decoded > *b.max + slack) {
        side = "above the max";
        limit = *b.max;
    } else
        return x::errors::NIL;
    if (raw == decoded)
        return {
            x::errors::VALIDATION,
            format(decoded) + " is " + side + " " + format(limit)
        };
    return {
        x::errors::VALIDATION,
        format(raw) + " decodes to " + format(decoded) + ", which is " + side + " " +
            format(limit)
    };
}
}

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

std::pair<Number, x::errors::Error> number_of(
    const x::telem::DataType &dt,
    const std::string_view kind,
    const bool scaled
) {
    const auto k = kind_of(dt);
    auto n = Number::FLOAT64;
    if (dt == x::telem::FLOAT32_T)
        n = Number::FLOAT32;
    else if (k == payload::KIND_INTEGER)
        n = Number::SIGNED;
    else if (k == payload::KIND_UNSIGNED || k == payload::KIND_BOOLEAN)
        n = Number::UNSIGNED;
    else if (k != payload::KIND_FLOAT)
        return {
            n,
            x::errors::Error(
                CONFIG_ERROR,
                "cannot write " + dt.name() + " channels to " + std::string(kind) +
                    " fields"
            ),
        };
    return {n, check_scaled_channel(scaled, dt)};
}

x::errors::Error write_float(const double v, const bool narrow, x::telem::Series &dst) {
    if (!narrow) {
        dst.write(v);
        return x::errors::NIL;
    }
    if (std::isfinite(v) && std::abs(v) > std::numeric_limits<float>::max())
        return {x::errors::VALIDATION, format(v) + " overflows float32"};
    dst.write(static_cast<float>(v));
    return x::errors::NIL;
}

x::errors::Error check_scale(const double scale) {
    if (scale == 0) return {CONFIG_ERROR, "scale must be non-zero"};
    return x::errors::NIL;
}

x::errors::Error check_scaled_channel(const bool scaled, const x::telem::DataType &dt) {
    if (!scaled || kind_of(dt) == payload::KIND_FLOAT) return x::errors::NIL;
    return {CONFIG_ERROR, "a scaled field requires a float channel, not " + dt.name()};
}

x::errors::Error as_config(const x::errors::Error &err, const std::string &context) {
    return {CONFIG_ERROR, err.wrap(context).data};
}

x::errors::Error check_limits(const icd::BaseField &b, const double v) {
    return out_of_range(b, v, v, 0);
}

x::errors::Error check_limits(const icd::BaseField &b, const float v) {
    for (const auto &limit: {b.min, b.max})
        if (limit.has_value() && static_cast<float>(*limit) == v) return x::errors::NIL;
    return check_limits(b, static_cast<double>(v));
}

x::errors::Error check_raw_limits(const icd::BaseField &b, const double raw) {
    const double scaled = raw * b.scale;
    return out_of_range(
        b,
        raw,
        scaled + b.offset,
        DECODE_ROUNDING * (std::abs(scaled) + std::abs(b.offset))
    );
}

bool within(const x::json::Pointer &inner, const x::json::Pointer &outer) {
    return outer.tokens.size() <= inner.tokens.size() &&
           std::equal(outer.tokens.begin(), outer.tokens.end(), inner.tokens.begin());
}
}
