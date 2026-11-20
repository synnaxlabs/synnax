// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "x/cpp/args/args.h"
#include "x/cpp/log/log.h"
#include "x/cpp/shutdown/shutdown.h"

#include "driver/rack/rack.h"

namespace driver::cmd::sub {
std::string version();

int start(x::args::Parser &args) {
    LOG(INFO) << x::log::BLUE() << "starting Synnax Driver " << version()
              << x::log::RESET();

    const bool stdin_stop_enabled = !args.flag("--disable-stdin-stop");
    VLOG(1) << "stdin stop " << (stdin_stop_enabled ? "enabled" : "disabled");

    const bool sig_stop_enabled = !args.flag("--disable-sig-stop");
    VLOG(1) << "sig stop " << (sig_stop_enabled ? "enabled" : "disabled");

    if (args.error()) {
        LOG(ERROR) << "invalid arguments: " << args.error();
        return 1;
    }

    rack::Rack r;
    // Wakes the main thread when the rack stops by itself.
    r.start(args, x::shutdown::signal_shutdown);
    x::shutdown::listen(sig_stop_enabled, stdin_stop_enabled);
    LOG(INFO) << x::log::BLUE()
              << "stopping driver. This can take up to 5 seconds. Please be patient"
              << x::log::RESET();
    const auto err = r.stop();
    if (err)
        LOG(ERROR) << "stopped with error: " << err;
    else
        LOG(INFO) << x::log::BLUE() << "stopped" << x::log::RESET();
    // A nonzero code makes the service manager restart the Driver.
    return err ? 1 : 0;
}
}
