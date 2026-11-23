// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <string_view>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/telem/telem.h"

namespace driver::codec {
/// @brief returns the payload kind of samples of dt, or nothing when no field kind
/// writes to dt.
std::optional<std::string_view> kind_of(const x::telem::DataType &dt);

/// @brief reports whether a field of kind k converts to samples of dt. A number or
/// boolean also converts to a float, and an unsigned number or boolean to any integer.
bool converts(std::string_view k, const x::telem::DataType &dt);

/// @brief returns the error err with context before its text.
x::errors::Error wrap(const x::errors::Error &err, const std::string &context);

/// @brief returns err as a CONFIG_ERROR with context before its text.
x::errors::Error as_config(const x::errors::Error &err, const std::string &context);

/// @brief reports whether inner is outer or a pointer into the value at outer.
bool within(const x::json::Pointer &inner, const x::json::Pointer &outer);

/// @brief returns VALIDATION when v, a value after the scale and offset of b, is
/// outside the min and max of b.
x::errors::Error check_limits(const synnax::icd::BaseField &b, double v);

/// @brief returns VALIDATION when v, a float32 sample, is outside the min and max of b.
/// A sample that equals a limit at float32 precision is at that limit.
x::errors::Error check_limits(const synnax::icd::BaseField &b, float v);
}
