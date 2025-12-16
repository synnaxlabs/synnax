// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/cmd/cmd.h"

// Updated helper function with C++ strings
int exec_svc_cmd(
    const std::function<xerrors::Error()> &cmd,
    const std::string &action,
    const std::string &past_tense = ""
) {
    if (const auto err = cmd()) {
        LOG(ERROR) << "" << xlog::red() << "Failed to " << action << ": " << err
                   << xlog::reset();
        return 1;
    }
    if (!past_tense.empty()) {
        LOG(INFO) << "" << xlog::green() << past_tense << " successfully"
                  << xlog::reset();
    }
    return 0;
}

int cmd::sub::service_start(xargs::Parser &args) {
    return exec_svc_cmd(daemond::start_service, "start", "started");
}

int cmd::sub::service_stop(xargs::Parser &args) {
    return exec_svc_cmd(daemond::stop_service, "stop", "stopped");
}

int cmd::sub::service_restart(xargs::Parser &args) {
    return exec_svc_cmd(daemond::restart_service, "restart", "restarted");
}

int cmd::sub::service_install(xargs::Parser &args) {
    return exec_svc_cmd(daemond::install_service, "install", "installed");
}

int cmd::sub::service_uninstall(xargs::Parser &args) {
    return exec_svc_cmd(daemond::uninstall_service, "uninstall", "uninstalled");
}

int cmd::sub::service_view_logs(xargs::Parser &args) {
    return exec_svc_cmd(daemond::view_logs, "view logs");
}

int cmd::sub::service_status(xargs::Parser &args) {
    return exec_svc_cmd(daemond::status, "status");
}
