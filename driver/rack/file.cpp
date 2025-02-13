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

xerrors::Error rack::Config::load_config_file(const int argc, char **argv) {
    std::string config_path;
    for (int i = 2; i < argc; i++) {
        const std::string arg = argv[i];
        if (arg == "--config") {
            config_path = argv[++i];
            break;
        }
    }
    if (config_path.empty()) return xerrors::NIL;
    auto p = config::Parser::from_file_path(config_path);
    auto conn = p.optional_child("connection");
    this->connection.override(conn);

    auto remote_info = p.optional_child("remote_info");
    this->remote.override(remote_info);

    this->integrations = p.optional("integrations", this->integrations);
    return p.error();
}
