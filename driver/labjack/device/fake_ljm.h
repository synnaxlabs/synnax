// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

namespace driver::labjack::device {
/// @brief a device handle whose stream starts fail in the fake LJM library.
constexpr int FAILING_START_HANDLE = 1;
/// @brief a device handle whose stream stops fail in the fake LJM library.
constexpr int FAILING_STOP_HANDLE = 2;
/// @brief an error code that the fake library names STREAM_BUFFER_FULL, a device error.
constexpr int STREAM_BUFFER_FULL_CODE = 2900;
/// @brief a device handle whose reads return LJME_USING_DEFAULT_CALIBRATION.
constexpr int DEFAULT_CALIBRATION_HANDLE = 3;
/// @brief a serial number whose open returns LJME_USING_DEFAULT_CALIBRATION.
constexpr const char *DEFAULT_CALIBRATION_SERIAL = "470000003";
/// @brief the value that every read of the fake library returns.
constexpr double READ_VALUE = 1.5;
/// @brief a device handle whose first LATE_SCANS_EMPTY_READS stream reads return
/// LJME_NO_SCANS_RETURNED.
constexpr int LATE_SCANS_HANDLE = 4;
constexpr int LATE_SCANS_EMPTY_READS = 3;
/// @brief a device handle whose first stream read returns LJME_NO_SCANS_RETURNED, and
/// whose next reads return LJME_NO_RESPONSE_BYTES_RECEIVED.
constexpr int LOST_STREAM_HANDLE = 5;
}
