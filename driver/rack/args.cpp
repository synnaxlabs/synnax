// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/rack/rack.h"

xerrors::Error rack::Config::load_args(xargs::Parser &args) {
    this->connection.override(args);
    this->timing.override(args);
    this->remote_info.override(args);
    this->override_integrations(
        args.field("enable-integrations", std::vector<std::string>{}),
        args.field("disable-integrations", std::vector<std::string>{})
    );
    return xerrors::NIL;
}
