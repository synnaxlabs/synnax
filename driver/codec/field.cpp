// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <string>
#include <string_view>

#include "x/cpp/errors/errors.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/errors.h"
#include "driver/codec/field.h"

namespace driver::codec {
namespace payload = x::payload;

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

x::errors::Error wrap(const x::errors::Error &err, const std::string &context) {
    return {err, context + ": " + err.data};
}

x::errors::Error as_config(const x::errors::Error &err, const std::string &context) {
    return {CONFIG_ERROR, context + ": " + err.data};
}
}
