// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"
#include "driver/codec/errors.h"

namespace driver::codec {
/// @brief returns the payload kind of samples of dt, or nothing when no field kind
/// writes to dt.
std::optional<std::string_view> kind_of(const x::telem::DataType &dt);

/// @brief reports whether a field of kind k converts to samples of dt. A number or
/// boolean also converts to a float, and an unsigned number or boolean to any integer.
bool converts(std::string_view k, const x::telem::DataType &dt);

/// @brief Number is how an encoder reads the samples of a field's channel.
enum class Number : std::uint8_t { FLOAT32, FLOAT64, SIGNED, UNSIGNED };

/// @brief returns how an encoder reads the samples of a dt channel. kind is the kind of
/// the field that the channel writes to, and scaled is whether the field scales.
/// @returns CONFIG_ERROR when dt does not hold numbers, or when the field scales and dt
/// does not hold floats.
std::pair<Number, x::errors::Error>
number_of(const x::telem::DataType &dt, std::string_view kind, bool scaled);

/// @brief writes v to dst, which holds float32 when narrow and float64 otherwise.
/// @returns VALIDATION when narrow and v is finite but overflows float32.
x::errors::Error write_float(double v, bool narrow, x::telem::Series &dst);

/// @brief returns CONFIG_ERROR when scale is zero, which turns every value into the
/// offset.
x::errors::Error check_scale(double scale);

/// @brief returns CONFIG_ERROR when scaled, true for a field whose scale or offset
/// changes its value, and dt does not hold floats.
x::errors::Error check_scaled_channel(bool scaled, const x::telem::DataType &dt);

/// @brief returns the index in bindings of the binding of each bound field.
/// @param fields maps the key of each field of the message to the field.
/// @returns CONFIG_ERROR when a binding names a field that is not in fields, or when
/// two bindings name one field.
template<typename F>
std::pair<std::map<synnax::icd::FieldKey, std::size_t>, x::errors::Error>
index_bindings(
    const std::vector<Binding> &bindings,
    const std::map<synnax::icd::FieldKey, const F *> &fields
) {
    std::map<synnax::icd::FieldKey, std::size_t> indices;
    for (std::size_t i = 0; i < bindings.size(); i++) {
        const auto &key = bindings[i].field;
        const auto found = fields.find(key);
        if (found == fields.end())
            return {
                {},
                x::errors::Error(
                    CONFIG_ERROR,
                    "field " + key.to_string() + " does not exist"
                ),
            };
        if (!indices.try_emplace(key, i).second)
            return {
                {},
                x::errors::Error(
                    CONFIG_ERROR,
                    "field " + synnax::icd::base(*found->second).name +
                        " has more than one binding"
                ),
            };
    }
    return {std::move(indices), x::errors::NIL};
}

/// @brief returns err as a CONFIG_ERROR with context before its text.
x::errors::Error as_config(const x::errors::Error &err, const std::string &context);

/// @brief returns VALIDATION when v, a value after the scale and offset of b, is
/// outside the min and max of b, or is NaN and b has a min or a max.
x::errors::Error check_limits(const synnax::icd::BaseField &b, double v);

/// @brief returns VALIDATION when v, a float32 sample, is outside the min and max of b.
/// A sample that equals a limit at float32 precision is at that limit.
x::errors::Error check_limits(const synnax::icd::BaseField &b, float v);

/// @brief returns VALIDATION when raw, a value before the scale and offset of b,
/// decodes outside the min and max of b, or decodes to NaN and b has a min or a max. It
/// allows the float64 rounding of the decode, so a value that decodes to a limit
/// passes. When narrow, a value that decodes to a limit at float32 precision is at that
/// limit.
x::errors::Error
check_raw_limits(const synnax::icd::BaseField &b, double raw, bool narrow = false);

/// @brief reports whether inner is outer or a pointer into the value at outer.
bool within(const x::json::Pointer &inner, const x::json::Pointer &outer);
}
