// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <memory>

#include "x/cpp/xargs/xargs.h"

#include "driver/cmd/cmd.h"

int cmd::sub::start(xargs::Parser &args) {
    LOG(INFO) << xlog::BLUE() << "starting synnax driver " << cmd::version()
              << xlog::RESET();

    const bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    VLOG(1) << "stdin stop " << (stdin_stop_enabled ? "enabled" : "disabled");

    const bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    VLOG(1) << "sig stop " << (sig_stop_enabled ? "enabled" : "disabled");

    if (args.error()) {
        LOG(ERROR) << "invalid arguments: " << args.error();
        return 1;
    }

    rack::Rack r;

    // Register an early shutdown handler to stop the driver when the process encounters
    // an error.
    auto early_shutdown = std::make_shared<std::atomic<bool>>(false);
    const std::function on_shutdown = [early_shutdown] {
        xshutdown::signal_shutdown();
        early_shutdown->store(true);
    };

    r.start(args, on_shutdown);

    // Register a signal handler to stop the driver when the process receives a signal.
    xshutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    if (!early_shutdown->load())
        LOG(INFO) << xlog::BLUE()
                  << "received shutdown signal. Gracefully stopping driver. "
                     "This can take up to 5 seconds. Please be patient"
                  << xlog::RESET();
    else
        LOG(WARNING) << "unexpected early shutdown";
    if (const auto err = r.stop())
        LOG(ERROR) << "stopped with error: " << err;
    else
        LOG(INFO) << xlog::BLUE() << "stopped" << xlog::RESET();
    return 0;
}
