#include <iostream>
#include <windows.h>
#include "glog/logging.h"

bool does_dll_exist(const char *dll_path) {
    HMODULE hModule = LoadLibrary(dll_path);
    if (hModule == NULL) {
        LOG(ERROR) << "[ni] " << dll_path << " not found";
        return false;
    }
    FreeLibrary(hModule);
    return true;
}

void log_dll_error(const std::shared_ptr<task::Context> &ctx,
                   const synnax::Task &task) {
    LOG(ERROR) << "[ni] Required NI DLLs not found, cannot configure task." <<
            std::endl;
    json j = {
        {"error", "Required NI DLLs not found."}
    };
    ctx->setState({
        .task = task.key,
        .variant = "error",
        .details = j
    });
}
