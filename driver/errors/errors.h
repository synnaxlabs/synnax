// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"

namespace driver {
const freighter::Error BASE_ERROR = synnax::BASE_ERROR.sub("driver");
const freighter::Error HARDWARE_ERROR = BASE_ERROR.sub("hardware");
const freighter::Error CRITICAL_HARDWARE_ERROR = HARDWARE_ERROR.sub("critical");
const freighter::Error TEMPORARY_HARDWARE_ERROR = HARDWARE_ERROR.sub("temporary");
const freighter::Error CONFIGURATION_ERROR = BASE_ERROR.sub("configuration");
}
