// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/args/args.h"
#include "x/cpp/log/log.h"

#include "absl/log/log.h"
#include "core/pkg/version/version.h"
#include "driver/cmd/cmd.h"

namespace driver::cmd::sub {
std::string version() {
    return "v" + std::string(SYNNAX_DRIVER_VERSION) + " (" +
           std::string(SYNNAX_BUILD_TIMESTAMP) + ")";
}

int version(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "Synnax Driver " << version() << x::log::RESET();
    return 0;
}
}
