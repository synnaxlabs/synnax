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

/// @brief returns the error err with context before its text.
x::errors::Error wrap(const x::errors::Error &err, const std::string &context);

/// @brief returns err as a CONFIG_ERROR with context before its text.
x::errors::Error as_config(const x::errors::Error &err, const std::string &context);

/// @brief reports whether inner is outer or a pointer into the value at outer.
bool within(const x::json::Pointer &inner, const x::json::Pointer &outer);
}
