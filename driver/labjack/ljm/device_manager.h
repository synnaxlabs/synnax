// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <map>

#include "api.h"
#include "errors.h"
#include "driver/labjack/ljm/LabJackM.h"

namespace ljm {
class DeviceAPI {
    std::shared_ptr<ljm::API> ljm;

public:
    const int handle;

    DeviceAPI(const std::shared_ptr<ljm::API> &ljm, const int handle):
        ljm(ljm), handle(handle) {
    }

    xerrors::Error eStreamRead(
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    ) const {
        return parse_error(
            ljm,
            ljm->eStreamRead(
                handle,
                aData,
                DeviceScanBacklog,
                LJMScanBacklog
            )
        );
    }

    xerrors::Error eStreamStop() const {
        return parse_error(
            ljm,
            ljm->eStreamStop(handle)
        );
    }

    xerrors::Error eWriteAddress(
        const int Address,
        const int Type,
        const double Value
    ) const {
        return parse_error(
            ljm,
            ljm->eWriteAddress(
                handle,
                Address,
                Type,
                Value
            )
        );
    }

    xerrors::Error eWriteAddresses(
        const int NumFrames,
        const int *aAddresses,
        const int *aTypes,
        const double *aValues,
        int *ErrorAddress
    ) const {
        return parse_error(
            ljm,
            ljm->eWriteAddresses(
                handle,
                NumFrames,
                aAddresses,
                aTypes,
                aValues,
                ErrorAddress
            )
        );
    }

    xerrors::Error StartInterval(
        const int IntervalHandle,
        const int Microseconds
    ) const {
        return parse_error(
            ljm,
            ljm->StartInterval(
                IntervalHandle,
                Microseconds
            )
        );
    }

    xerrors::Error eWriteName(const char *Name, const double Value) const {
        return parse_error(
            ljm,
            ljm->eWriteName(
                handle,
                Name,
                Value
            )
        );
    }

    xerrors::Error eWriteNames(
        const int NumFrames,
        const char **aNames,
        const double *aValues,
        int *ErrorAddress
    ) const {
        return parse_error(
            ljm,
            ljm->eWriteNames(
                handle,
                NumFrames,
                aNames,
                aValues,
                ErrorAddress
            )
        );
    }

    xerrors::Error NamesToAddresses(
        const int NumFrames,
        const char **aNames,
        int *aAddresses,
        int *aTypes
    ) const {
        return parse_error(
            ljm,
            ljm->NamesToAddresses(
                NumFrames,
                aNames,
                aAddresses,
                aTypes
            )
        );
    }

    xerrors::Error WaitForNextInterval(
        const int IntervalHandle,
        int *skippedIntervals
    ) const {
        return parse_error(
            ljm,
            ljm->WaitForNextInterval(
                IntervalHandle,
                skippedIntervals
            )
        );
    }

    xerrors::Error eReadNames(
        const int NumFrames,
        const char **aNames,
        double *aValues,
        int *ErrorAddress
    ) const {
        return parse_error(
            ljm,
            ljm->eReadNames(
                handle,
                NumFrames,
                aNames,
                aValues,
                ErrorAddress
            )
        );
    }

    xerrors::Error eStreamStart(
        const int ScansPerRead,
        const int NumAddresses,
        const int *aScanList,
        double *ScanRate
    ) const {
        return parse_error(
            ljm,
            ljm->eStreamStart(
                handle,
                ScansPerRead,
                NumAddresses,
                aScanList,
                ScanRate
            )
        );
    }
};

class DeviceManager {
    std::mutex mu;
    std::map<std::string, std::shared_ptr<DeviceAPI>> device_handles;
    std::shared_ptr<ljm::API> ljm;
public:
    explicit DeviceManager(const std::shared_ptr<ljm::API> &ljm): ljm(ljm) {
    }

    xerrors::Error ListAll(
        const int DeviceType,
        const int ConnectionType,
        int *NumFound,
        int *aDeviceTypes,
        int *aConnectionTypes,
        int *aSerialNumbers,
        int *aIPAddresses
    ) const {
        return parse_error(
            ljm,
            ljm->ListAll(
                DeviceType,
                ConnectionType,
                NumFound,
                aDeviceTypes,
                aConnectionTypes,
                aSerialNumbers,
                aIPAddresses
            )
        );
    }


    std::pair<std::shared_ptr<DeviceAPI>, xerrors::Error> acquire(
        const std::string &serial_number
    ) {
        std::lock_guard lock(mu);
        const auto it = this->device_handles.find(serial_number);
        if (it == device_handles.end()) {
            int handle;
            const int err = ljm->Open(
                LJM_dtANY,
                LJM_ctANY,
                serial_number.c_str(),
                &handle
            );
            if (err != 0) return {nullptr, parse_error(ljm, err)};
            auto dev = std::make_shared<DeviceAPI>(ljm, handle);
            device_handles[serial_number] = dev;
            return {dev, xerrors::NIL};
        }
        return {it->second, xerrors::NIL};
    }

    xerrors::Error release(const std::string &serial_number) {
        std::lock_guard lock(mu);
        const auto it = this->device_handles.find(serial_number);
        if (it == device_handles.end()) return xerrors::NIL;
        const auto error = ljm->Close(it->second->handle);
        device_handles.erase(serial_number);
        return parse_error(ljm, error);
    }
};
}
