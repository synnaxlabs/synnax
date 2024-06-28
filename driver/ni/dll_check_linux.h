#include <iostream>
#include "glog/logging.h"

bool does_dll_exist(const char *dll_path) {
    return false;
}

void log_dll_error(const std::shared_ptr<task::Context> &ctx,
                   const synnax::Task &task) {
    LOG(ERROR) << "[ni] NI acquisition and control not supported on Linux or MacOS" <<
            std::endl;
    json j = {
        {"error", " NI acquisition and control not supported on Linux or MacOS"}
    };
    ctx->setState({
        .task = task.key,
        .variant = "error",
        .details = j
    });
}
