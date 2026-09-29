// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "driver/bus/connection.h"

namespace driver::bus {
Acquire acquirer(
    std::shared_ptr<Connections> connections,
    std::string key,
    x::json::json settings,
    Opener open
) {
    return
        [connections = std::move(connections),
         key = std::move(key),
         settings = std::move(settings),
         open = std::move(open)] { return connections->acquire(key, settings, open); };
}
}
