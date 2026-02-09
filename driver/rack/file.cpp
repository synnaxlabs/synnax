// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/os/os.h"
#include "x/cpp/path/path.h"

#include "driver/rack/rack.h"

namespace driver::rack {
x::errors::Error
Config::load_config_file(x::args::Parser &args, x::breaker::Breaker &breaker) {
    std::string config_path = args.field("--config", "");
    if (config_path.empty()) {
        if (breaker.retry_count() == 0) LOG(INFO) << "no config file specified";
        return x::errors::NIL;
    }
    if (breaker.retry_count() == 0)
        LOG(INFO) << "loading config file from "
                  << x::path::resolve_relative(config_path);
    auto p = x::json::Parser::from_file_path(config_path);
    auto conn = p.optional_child("connection");
    this->connection.override(conn);
    auto remote_info = p.optional_child("remote_info");
    this->remote_info.override(remote_info);
    auto timing_config = p.optional_child("timing");
    this->timing.override(timing_config);
    this->integrations = p.field("integrations", this->integrations);
    return p.error();
}
}
