// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/xenv/xenv.h"

#include "driver/rack/rack.h"

const std::string ENV_PREFIX = "SYNNAX_DRIVER_";

xerrors::Error rack::Config::load_env() {
    xenv::Parser p(ENV_PREFIX);
    this->connection.override(p);
    this->timing.override(p);
    this->manager.override(p);
    this->remote_info.override(p);
    return xerrors::NIL;
}
