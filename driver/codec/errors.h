// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::codec {
/// @brief base error for failures while decoding or encoding a payload.
const x::errors::Error BASE_ERROR = driver::errors::BASE_ERROR.sub("codec");
/// @brief a message layout the codec cannot compile.
const x::errors::Error LAYOUT_ERROR = driver::errors::CONFIGURATION_ERROR.sub("codec");
/// @brief a payload too short to hold a field the plan decodes.
const x::errors::Error SHORT_PAYLOAD_ERROR = BASE_ERROR.sub("short_payload");
/// @brief a value the plan cannot encode.
const x::errors::Error ENCODE_ERROR = BASE_ERROR.sub("encode");
}
