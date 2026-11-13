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
#include <cstring>
#include <thread>

#include "driver/labjack/device/fake_ljm.h"

namespace {
constexpr int FAILURE = 1;
/// @brief LJME_USING_DEFAULT_CALIBRATION, a warning.
constexpr int USING_DEFAULT_CALIBRATION = 203;
/// @brief LJME_NO_SCANS_RETURNED.
constexpr int NO_SCANS_RETURNED = 1309;
/// @brief LJME_NO_RESPONSE_BYTES_RECEIVED, a lost connection.
constexpr int NO_RESPONSE_BYTES_RECEIVED = 1263;
using StreamCallback = void (*)(void *);
std::atomic<StreamCallback> stream_callback = nullptr;
std::atomic<void *> stream_callback_arg = nullptr;
std::atomic<bool> late_scans_sent = false;
std::atomic<bool> late_scans_arrived = false;
std::atomic<int> lost_stream_reads = 0;
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

int LJM_SetStreamCallback(int, const StreamCallback callback, void *arg) {
    stream_callback = callback;
    stream_callback_arg = arg;
    return 0;
}

int LJM_eStreamRead(const int handle, double *data, int *, int *) {
    if (handle == driver::labjack::device::LATE_SCANS_HANDLE) {
        if (!late_scans_sent.exchange(true))
            std::thread([] {
                std::this_thread::sleep_for(std::chrono::milliseconds(10));
                late_scans_arrived = true;
                stream_callback.load()(stream_callback_arg.load());
            }).detach();
        if (!late_scans_arrived) return NO_SCANS_RETURNED;
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
}
#pragma GCC visibility pop
