// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <mutex>
#include <map>
#include <_virtual_includes/glog/glog/logging.h>

#include "driver/labjack/ljm/errors.h"
#include "driver/labjack/ljm/LabJackM.h"

namespace labjack {
// An internal namespace for special labjack methods that cannot be called concurrently.
namespace locked {
// This mutex is reserved for internal use to the namespace only.

inline int LJM_ListAll_wrapped(
    int DeviceType,
    int ConnectionType,
    int *NumFound,
    int *aDeviceTypes,
    int *aConnectionTypes,
    int *aSerialNumbers,
    int *aIPAddresses
) {
    std::lock_guard lock(_priv_device_mutex);
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


class DeviceAPI {
public:
     xerrors::Error eStreamRead(
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    );
    xerrors::Error eStreamStop();
    xerrors::Error LJM_eWriteAddress(int Address, int Type, double Value);
    xerrors::Error LJM_eWriteAddresses(int NumFrames,
	const int * aAddresses, const int * aTypes, const double * aValues,
	int * ErrorAddress);
    xerrors::Error StartInterval(int IntervalHandle, int Microseconds);
    xerrors::Error LJM_eWriteName(const char *Name, double Value);
    xerrors::Error LJM_eWriteNames(int NumFrames,
	const char ** aNames, const double * aValues, int * ErrorAddress);
    xerrors::Error NamesToAddresses(int NumFrames, const char **aNames, int *aAddresses, int *aTypes);
    xerrors::Error LJM_WaitForNextInterval(int IntervalHandle, int *skippedIntervals);
    xerrors::Error LJM_eReadNames(int NumFrames,
	const char ** aNames, double * aValues, int * ErrorAddress);
    xerrors::Error eStreamStart(int ScansPerRead,
	int NumAddresses, const int * aScanList, double * ScanRate);
};

class DeviceManager {
    std::mutex mu;
    std::map<std::string, int> device_handles;
public:
    DeviceManager() {
    }

    std::pair<std::shared_ptr<DeviceAPI>, xerrors::Error> get_device_handle(std::string serial_number) {
        std::lock_guard lock(mu);
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
        std::lock_guard lock(mu);
        if (this->device_handles.find(serial_number) != device_handles.end()) {
            int handle = device_handles[serial_number];
            LJM_Close(handle);
            device_handles.erase(serial_number);
        }
    }
};
}
