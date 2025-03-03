// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 3/2/25.
//

#pragma once

#include "driver/labjack/ljm/LabJackM.h"

namespace ljm {
class API {
public:
    LJM_ERROR_RETURN eStreamRead(
        int Handle,
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    );

    LJM_ERROR_RETURN eStreamStop(int Handle);

    LJM_ERROR_RETURN eWriteAddress(int Handle, int Address, int Type, double Value);

    LJM_ERROR_RETURN eWriteAddresses(
        int Handle,
        int NumFrames,
        const int *aAddresses,
        const int *aTypes,
        const double *aValues,
        int *ErrorAddress
    );

    LJM_ERROR_RETURN StartInterval(int IntervalHandle, int Microseconds);

    LJM_ERROR_RETURN eWriteName(int Handle, const char *Name, double Value);

    LJM_ERROR_RETURN NamesToAddresses(
        int NumFrames,
        const char **aNames,
        int *aAddresses,
        int *aTypes
    );

    LJM_VOID_RETURN ErrorToString(int ErrorCode, char *ErrorString);

    LJM_ERROR_RETURN eWriteNames(int Handle, int NumFrames,
	const char ** aNames, const double * aValues, int * ErrorAddress);


    LJM_ERROR_RETURN ListAll(
        int DeviceType,
        int ConnectionType,
        int *NumFound,
        int *aDeviceTypes,
        int *aConnectionTypes,
        int *aSerialNumbers,
        int *aIPAddresses
    );

    LJM_ERROR_RETURN Open(
        int DeviceType,
        int ConnectionType,
        const char *Identifier,
        int *Handle
    );


    LJM_ERROR_RETURN Close(int Handle);

    LJM_ERROR_RETURN eReadNames(int Handle, int NumFrames,
	const char ** aNames, double * aValues, int * ErrorAddress);


    LJM_ERROR_RETURN WaitForNextInterval(int IntervalHandle, int * SkippedIntervals);

    LJM_ERROR_RETURN eStreamStart(int Handle, int ScansPerRead,
	int NumAddresses, const int * aScanList, double * ScanRate);
};
}
