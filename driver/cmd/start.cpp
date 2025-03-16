// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// module
#include "x/cpp/xargs/xargs.h"

/// internal
#include "driver/cmd/cmd.h"

int cmd::sub::start(xargs::Parser &args) {
    bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    if (args.error()) {
        LOG(ERROR) << "[driver] invalid arguments: " << args.error();
        return 1;
    }
    LOG(INFO) << xlog::BLUE() << "[driver] starting synnax driver " << cmd::version() << xlog::RESET();
    rack::Rack r;
    r.start(args);
    xshutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    LOG(INFO) << xlog::BLUE() << "[driver] received shutdown signal. Gracefully stopping driver. This can take up to 5 seconds. Please be patient" << xlog::RESET();
    if (const auto err = r.stop())
        LOG(ERROR) << "[driver] stopped with error: " << err;
    else LOG(INFO) << xlog::BLUE() << "[driver] stopped" << xlog::RESET();
    return 0;
}
