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
/// @brief a message layout the codec cannot compile.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub("codec");
}
