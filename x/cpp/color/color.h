// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>

#include "x/cpp/color/json.gen.h"
#include "x/cpp/color/proto.gen.h"
#include "x/cpp/color/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace x::color {

/// @brief from_hex parses a 6- or 8-character hex color, with or without a leading
/// '#'. Mirrors the Go color.FromHex.
std::pair<Color, x::errors::Error> from_hex(const std::string &input);

/// @brief from_css parses a CSS color: a '#'-prefixed hex or an rgb()/rgba() value.
/// Mirrors the Go color.FromCSS.
std::pair<Color, x::errors::Error> from_css(const std::string &input);
}
