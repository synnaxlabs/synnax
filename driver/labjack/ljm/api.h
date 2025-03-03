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
#include "x/cpp/xerrors/errors.h"

namespace ljm {
class API {
    LJM_ERROR_RETURN LJM_eStreamRead(
        int Handle,
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    );
    LJM_ERROR_RETURN LJM_eStreamStop(int Handle);
    LJM_ERROR_RETURN LJM_eWriteAddress(int Handle, int Address, int Type, double Value);
    LJM_ERROR_RETURN LJM_eWriteAddresses(
        int Handle, int NumFrames,
	const int * aAddresses, const int * aTypes, const double * aValues,
	int * ErrorAddress);
    LJM_ERROR_RETURN LJM_StartInterval(int IntervalHandle, int Microseconds);
    LJM_ERROR_RETURN LJM_eWriteName(int Handle, const char *Name, double Value);
    LJM_ERROR_RETURN LJM_NamesToAddresses(int NumFrames, const char **aNames, int *aAddresses, int *aTypes);
};

class SugaredAPI {
public:
     xerrors::Error LJM_eStreamRead(
        int Handle,
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    );
    xerrors::Error LJM_eStreamStop(int Handle);
    xerrors::Error LJM_eWriteAddress(int Handle, int Address, int Type, double Value);
        xerrors::Error LJM_eWriteAddresses(
        int Handle, int NumFrames,
	const int * aAddresses, const int * aTypes, const double * aValues,
	int * ErrorAddress);
    xerrors::Error LJM_StartInterval(int IntervalHandle, int Microseconds);
    xerrors::Error LJM_eWriteName(int Handle, const char *Name, double Value);
    xerrors::Error LJM_NamesToAddresses(int NumFrames, const char **aNames, int *aAddresses, int *aTypes);
    xerrors::Error LJM_WaitForNextInterval(int IntervalHandle, int *skippedIntervals);
    xerrors::Error LJM_eReadNames(int Handle, int NumFrames,
	const char ** aNames, double * aValues, int * ErrorAddress);
    xerrors::Error LJM_eStreamStart(int Handle, int ScansPerRead,
	int NumAddresses, const int * aScanList, double * ScanRate);
};
}
