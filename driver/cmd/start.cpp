// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <memory>

#include "x/cpp/args/args.h"

#include "driver/cmd/cmd.h"

namespace driver::cmd::sub {
int start(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "starting Synnax Driver " << ::driver::cmd::version()
              << x::log::RESET();

    const bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    VLOG(1) << "stdin stop " << (stdin_stop_enabled ? "enabled" : "disabled");

    const bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    VLOG(1) << "sig stop " << (sig_stop_enabled ? "enabled" : "disabled");

    if (args.error()) {
        LOG(ERROR) << "invalid arguments: " << args.error();
        return 1;
    }

    driver::rack::Rack r;

    // Register an early shutdown handler to stop the driver when the process encounters
    // an error.
    auto early_shutdown = std::make_shared<std::atomic<bool>>(false);
    const std::function on_shutdown = [early_shutdown] {
        x::shutdown::signal_shutdown();
        early_shutdown->store(true);
    };

    r.start(args, on_shutdown);

    // Register a signal handler to stop the driver when the process receives a signal.
    x::shutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    if (!early_shutdown->load())
        LOG(INFO) << x::log::BLUE()
                  << "received shutdown signal. Gracefully stopping driver. "
                     "This can take up to 5 seconds. Please be patient"
                  << x::log::RESET();
    else
        LOG(WARNING) << "unexpected early shutdown";
    if (const auto err = r.stop())
        LOG(ERROR) << "stopped with error: " << err;
    else
        LOG(INFO) << x::log::BLUE() << "stopped" << x::log::RESET();
    return 0;
}
}
