// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/can/can.h"

namespace driver::can::backends {
/// @returns every backend, keyed by its synnax::can::BACKEND_* name. A backend whose
/// library does not load, or that the platform lacks, is an Unavailable that reports
/// why.
[[nodiscard]] Backends load();
}
