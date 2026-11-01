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

void LJM_ErrorToString(int, char *err_string) {
    std::strcpy(err_string, "FAKE_FAILURE");
}
}
#pragma GCC visibility pop
