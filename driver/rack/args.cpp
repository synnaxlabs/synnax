// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <vector>

#include "driver/rack/rack.h"

namespace driver::rack {
x::errors::Error Config::load_args(x::args::Parser &args) {
    this->connection.override(args);
    this->timing.override(args);
    this->manager.override(args);
    this->remote_info.override(args);
    const auto enable = args.field("enable-integrations", std::vector<std::string>{});
    const auto disable = args.field("disable-integrations", std::vector<std::string>{});
    if (!enable.empty() && !disable.empty())
        args.field_err(
            "enable-integrations",
            "cannot be used with --disable-integrations"
        );
    else if (!enable.empty()) {
        this->integrations.clear();
        for (const auto &i: enable)
            if (std::ranges::find(this->integrations, i) == this->integrations.end())
                this->integrations.push_back(i);
    } else
        std::erase_if(this->integrations, [&](const std::string &i) {
            return std::ranges::find(disable, i) != disable.end();
        });
    return args.error();
}
}
