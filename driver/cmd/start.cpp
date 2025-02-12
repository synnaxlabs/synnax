// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

int cmd::sub::start(const int argc, char *argv[]) {
    bool sigint_enabled = true;
    bool stdin_enabled = true;
    for (int i = 1; i < argc; i++) {
        const std::string arg = argv[i];
        if (arg == "--block-stdin-stop")
            stdin_enabled = false;
        else if (arg == "--block-sigint-stop")
            sigint_enabled = false;
    }
    LOG(INFO) << xlog::BLUE << "[driver] starting synnax driver " << cmd::version() << xlog::RESET;
    rack::Rack r;
    r.start(argc, argv);
    xshutdown::listen(sigint_enabled, stdin_enabled);
    LOG(INFO) << xlog::BLUE << "[driver] received shutdown signal. stopping driver" << xlog::RESET;
    if (const auto err = r.stop())
        LOG(ERROR) << "[driver] stopped with error: " << err;
    else LOG(INFO) << xlog::BLUE << "[driver] stopped" << xlog::RESET;
    return 0;
}
