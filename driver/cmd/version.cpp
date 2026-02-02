// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

std::string driver::cmd::version() {
    return "v" + std::string(SYNNAX_DRIVER_VERSION) + " (" +
           std::string(SYNNAX_BUILD_TIMESTAMP) + ")";
}

int driver::cmd::sub::version(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "Synnax Driver " << driver::cmd::version()
              << x::log::RESET();
    return 0;
}
