// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>

/// module
#include "synnax/pkg/version/version.h"

/// external
#include "glog/logging.h"

/// internal
#include "driver/cmd/cmd.h"
#include "x/cpp/xlog/xlog.h"

std::string cmd::version() {
    return "v" +
        std::string(SYNNAX_DRIVER_VERSION)+
            " (" +
            std::string(SYNNAX_BUILD_TIMESTAMP)
    + ")";
}

int cmd::sub::version(int argc, char **argv) {
    LOG(INFO) << xlog::BLUE << "Synnax Driver " << cmd::version() << xlog::RESET;
    return 0;
}
