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
    LOG(INFO) << xlog::BLUE() << "[driver] starting synnax driver " << cmd::version()
              << xlog::RESET();

    const bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    VLOG(1) << "[driver] stdin stop " << (stdin_stop_enabled ? "enabled" : "disabled");

    const bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    VLOG(1) << "[driver] sig stop " << (sig_stop_enabled ? "enabled" : "disabled");

    if (args.error()) {
        LOG(ERROR) << "[driver] invalid arguments: " << args.error();
        return 1;
    }

    rack::Rack r;

    // Register an early shutdown handler to stop the driver when the process encounters
    // an error.
    volatile bool early_shutdown = false;
    const std::function on_shutdown = [&early_shutdown] {
        xshutdown::signal_shutdown();
        early_shutdown = true;
    };

    r.start(args, on_shutdown);

    // Register a signal handler to stop the driver when the process receives a signal.
    xshutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    if (!early_shutdown)
        LOG(INFO) << xlog::BLUE()
                  << "[driver] received shutdown signal. Gracefully stopping driver. "
                     "This can take up to 5 seconds. Please be patient"
                  << xlog::RESET();
    else
        LOG(WARNING) << "[driver] unexpected early shutdown";
    if (const auto err = r.stop())
        LOG(ERROR) << "[driver] stopped with error: " << err;
    else
        LOG(INFO) << xlog::BLUE() << "[driver] stopped" << xlog::RESET();
    return 0;
}
