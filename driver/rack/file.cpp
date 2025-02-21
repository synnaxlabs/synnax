// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// internal
#include "driver/rack/rack.h"

/// module
#include "x/cpp/xos/xos.h"

xerrors::Error rack::Config::load_config_file(xargs::Parser &args) {
    std::string config_path = args.optional("--config", "");
    if (config_path.empty()) return xerrors::NIL;
    auto p = xjson::Parser::from_file_path(config_path);
    auto conn = p.optional_child("connection");
    this->connection.override(conn);

    auto remote_info = p.optional_child("remote_info");
    this->remote.override(remote_info);

    this->integrations = p.optional("integrations", this->integrations);
    return p.error();
}
