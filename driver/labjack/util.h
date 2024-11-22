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
// An internal namespace for special labjack methods that cannot be called concurrently.
namespace locked {
// This mutex is reserved for internal use to the namespace only.
inline std::mutex _priv_device_mutex;

inline int LJM_ListAll_wrapped(int DeviceType, int ConnectionType,
                               int *NumFound, int *aDeviceTypes, int *aConnectionTypes,
                               int *aSerialNumbers, int *aIPAddresses) {
    std::lock_guard<std::mutex> lock(_priv_device_mutex);
    return LJM_ListAll(
        DeviceType,
        ConnectionType,
        NumFound,
        aDeviceTypes,
        aConnectionTypes,
        aSerialNumbers,
        aIPAddresses
    );
}

inline int LJM_Open_wrapped(int DeviceType, int ConnectionType,
                            const char *Identifier, int *Handle) {
    std::lock_guard<std::mutex> lock(_priv_device_mutex);
    return LJM_Open(DeviceType, ConnectionType, Identifier, Handle);
}
}

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
    const auto &error_map = GetErrorDescriptions(); // Changed this line
    if (auto it = error_map.find(err_msg); it != error_map.end()) {
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
        std::lock_guard<std::mutex> lock(mu);
        if (this->device_handles.find(serial_number) == device_handles.end()) {
            int handle;
            int err = locked::LJM_Open_wrapped(LJM_dtANY, LJM_ctANY, serial_number.c_str(), &handle);
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
    std::mutex mu;
    std::map<std::string, int> device_handles;
};
}
