// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

namespace driver::cmd::sub {
// Updated helper function with C++ strings
int exec_svc_cmd(
    const std::function<x::errors::Error()> &cmd,
    const std::string &action,
    const std::string &past_tense = ""
) {
    if (const auto err = cmd()) {
        LOG(ERROR) << "" << x::log::RED() << "Failed to " << action << ": " << err
                   << x::log::RESET();
        return 1;
    }
    if (!past_tense.empty()) {
        LOG(INFO) << "" << x::log::GREEN() << past_tense << " successfully"
                  << x::log::RESET();
    }
    return 0;
}

int service_start(x::args::Parser &args) {
    return exec_svc_cmd(daemon::start_service, "start", "started");
}

int service_stop(x::args::Parser &args) {
    return exec_svc_cmd(daemon::stop_service, "stop", "stopped");
}

int service_restart(x::args::Parser &args) {
    return exec_svc_cmd(daemon::restart_service, "restart", "restarted");
}

int service_install(x::args::Parser &args) {
    return exec_svc_cmd(daemon::install_service, "install", "installed");
}

int service_uninstall(x::args::Parser &args) {
    return exec_svc_cmd(daemon::uninstall_service, "uninstall", "uninstalled");
}

int service_view_logs(x::args::Parser &args) {
    return exec_svc_cmd(daemon::view_logs, "view logs");
}

int service_status(x::args::Parser &args) {
    return exec_svc_cmd(daemon::status, "status");
}
}
