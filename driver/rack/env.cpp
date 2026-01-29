// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/env/env.h"

#include "driver/rack/rack.h"

namespace driver::rack {
const std::string ENV_PREFIX = "SYNNAX_DRIVER_";

x::errors::Error Config::load_env() {
    x::env::Parser p(ENV_PREFIX);
    this->connection.override(p);
    this->timing.override(p);
    this->manager.override(p);
    this->remote_info.override(p);
    return x::errors::NIL;
}
}
