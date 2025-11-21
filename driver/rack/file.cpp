// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/xos/xos.h"
#include "x/cpp/xpath/xpath.h"

#include "driver/rack/rack.h"

xerrors::Error
rack::Config::load_config_file(xargs::Parser &args, breaker::Breaker &breaker) {
    std::string config_path = args.field("--config", "");
    if (config_path.empty()) {
        if (breaker.retry_count() == 0) LOG(INFO) << "no config file specified";
        return xerrors::NIL;
    }
    if (breaker.retry_count() == 0)
        LOG(INFO) << "loading config file from "
                  << xpath::resolve_relative(config_path);
    auto p = xjson::Parser::from_file_path(config_path);
    auto conn = p.optional_child("connection");
    this->connection.override(conn);
    auto remote_info = p.optional_child("remote_info");
    this->remote_info.override(remote_info);
    auto timing_config = p.optional_child("timing");
    this->timing.override(timing_config);
    this->integrations = p.field("integrations", this->integrations);
    return p.error();
}
