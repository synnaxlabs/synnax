// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <mutex>
#include <thread>
#include <map>

#include "driver/labjack/errors.h"
#include "LJM_Utilities.h"

namespace labjack {
inline std::mutex device_mutex;

inline int check_err_internal(
    int err,
    std::string caller,
    std::string prefix,
    std::shared_ptr<task::Context> ctx,
    bool &ok_state,
    synnax::TaskKey task_key
) {
    if (err == 0) return 0;

    char err_msg[LJM_MAX_NAME_SIZE];
    LJM_ErrorToString(err, err_msg);

    // Get additional description if available
    std::string description = "";
    if (auto it = ERROR_DESCRIPTIONS.find(err_msg); it != ERROR_DESCRIPTIONS.end()) {
        description = ": " + it->second;
    }
    ctx->set_state({
        .task = task_key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", std::string(err_msg) + description}
        }
    });

    LOG(ERROR) << "[labjack." << prefix << "] " << err_msg << "(" << err << ")" << description << " (" << caller << ")";

    ok_state = false;
    return -1;
}

class DeviceManager {
public:
    DeviceManager() : device_handles() {
    }

    int get_device_handle(std::string serial_number) {
        std::lock_guard<std::mutex> lock(device_mutex);
        if (this->device_handles.find(serial_number) == device_handles.end()) {
            int handle;
            int err = LJM_Open(LJM_dtANY, LJM_ctANY, serial_number.c_str(), &handle);
            if (err != 0) {
                char err_msg[LJM_MAX_NAME_SIZE];
                LJM_ErrorToString(err, err_msg);
                LOG(ERROR) << "[labjack.reader] LJM_Open error: " << err_msg << "(" << err << ")";
                return -1;
            }
            device_handles[serial_number] = handle;
        }
        return device_handles[serial_number];
    }

    void close_device(std::string serial_number) {
        std::lock_guard<std::mutex> lock(device_mutex);
        if (this->device_handles.find(serial_number) != device_handles.end()) {
            int handle = device_handles[serial_number];
            LJM_Close(handle);
            device_handles.erase(serial_number);
        }
    }

private:
    std::map<std::string, int> device_handles;
};
}
