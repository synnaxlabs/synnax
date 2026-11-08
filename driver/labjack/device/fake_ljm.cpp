// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>

#include "driver/labjack/device/fake_ljm.h"

namespace {
constexpr int FAILURE = 1;
/// @brief LJME_USING_DEFAULT_CALIBRATION, a warning.
constexpr int USING_DEFAULT_CALIBRATION = 203;
}

#pragma GCC visibility push(default)
extern "C" {
int LJM_eStreamStart(const int handle, int, int, const int *, double *) {
    return handle == driver::labjack::device::FAILING_START_HANDLE ? FAILURE : 0;
}

int LJM_eStreamStop(const int handle) {
    return handle == driver::labjack::device::FAILING_STOP_HANDLE ? FAILURE : 0;
}

int LJM_Close(int) {
    return 0;
}

int LJM_Open(int, int, const char *identifier, int *handle) {
    if (std::strcmp(identifier, driver::labjack::device::DEFAULT_CALIBRATION_SERIAL) ==
        0) {
        *handle = driver::labjack::device::DEFAULT_CALIBRATION_HANDLE;
        return USING_DEFAULT_CALIBRATION;
    }
    *handle = 0;
    return 0;
}

int LJM_eReadName(const int handle, const char *, double *value) {
    *value = driver::labjack::device::READ_VALUE;
    return handle == driver::labjack::device::DEFAULT_CALIBRATION_HANDLE
             ? USING_DEFAULT_CALIBRATION
             : 0;
}

void LJM_ErrorToString(const int err, char *err_string) {
    if (err == driver::labjack::device::STREAM_BUFFER_FULL_CODE)
        std::strcpy(err_string, "STREAM_BUFFER_FULL");
    else
        std::strcpy(err_string, "FAKE_FAILURE");
}
}
#pragma GCC visibility pop
