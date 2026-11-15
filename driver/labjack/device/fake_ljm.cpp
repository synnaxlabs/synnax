// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstring>
#include <mutex>

#include "driver/labjack/device/fake_ljm.h"

namespace {
constexpr int FAILURE = 1;
/// @brief LJME_USING_DEFAULT_CALIBRATION, a warning.
constexpr int USING_DEFAULT_CALIBRATION = 203;
/// @brief LJME_NO_SCANS_RETURNED.
constexpr int NO_SCANS_RETURNED = 1309;
/// @brief LJME_NO_RESPONSE_BYTES_RECEIVED, a lost connection.
constexpr int NO_RESPONSE_BYTES_RECEIVED = 1263;
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): LJM is a C library,
// so its callers reach its state only through global functions.
std::atomic<int> late_scans_reads = 0;
std::atomic<int> lost_stream_reads = 0;
std::atomic<int> stalled_stream_reads = 0;
std::mutex stalled_mu;
std::condition_variable stalled_cv;
bool stalled = false;
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)
}

#pragma GCC visibility push(default)
extern "C" {
int LJM_eStreamStart(const int handle, int, int, const int *, double *) {
    return handle == driver::labjack::device::FAILING_START_HANDLE ? FAILURE : 0;
}

int LJM_eStreamStop(const int handle) {
    return handle == driver::labjack::device::FAILING_STOP_HANDLE ? FAILURE : 0;
}

int LJM_WriteLibraryConfigS(const char *, double) {
    return 0;
}

int LJM_eStreamRead(const int handle, double *data, int *, int *) {
    if (handle == driver::labjack::device::LATE_SCANS_HANDLE &&
        late_scans_reads++ < driver::labjack::device::LATE_SCANS_EMPTY_READS)
        return NO_SCANS_RETURNED;
    if (handle == driver::labjack::device::STALLED_STREAM_HANDLE &&
        stalled_stream_reads++ < driver::labjack::device::STALLED_STREAM_EMPTY_READS) {
        {
            std::lock_guard lock(stalled_mu);
            stalled = true;
        }
        stalled_cv.notify_all();
        return NO_SCANS_RETURNED;
    }
    if (handle == driver::labjack::device::LOST_STREAM_HANDLE)
        return lost_stream_reads++ == 0 ? NO_SCANS_RETURNED
                                        : NO_RESPONSE_BYTES_RECEIVED;
    *data = driver::labjack::device::READ_VALUE;
    return 0;
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

bool Fake_WaitForStalledRead() {
    std::unique_lock lock(stalled_mu);
    return stalled_cv.wait_for(lock, std::chrono::seconds(5), [] { return stalled; });
}
}
#pragma GCC visibility pop
