// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include "x/cpp/xerrors/errors.h"
#include "freighter/cpp/freighter.h"

namespace driver {
const xerrors::Error BASE_ERROR = xerrors::BASE_ERROR.sub("driver");
const xerrors::Error HARDWARE_ERROR = BASE_ERROR.sub("hardware");
const xerrors::Error CRITICAL_HARDWARE_ERROR = HARDWARE_ERROR.sub("critical");
const xerrors::Error TEMPORARY_HARDWARE_ERROR = HARDWARE_ERROR.sub("temporary");
const xerrors::Error CONFIGURATION_ERROR = BASE_ERROR.sub("configuration");
}
