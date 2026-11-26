// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <system_error>

#include "asio/error.hpp"
#include "gtest/gtest.h"
#include <windows.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/errors.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @returns the Win32 error as Asio reports it.
std::error_code os(const DWORD code) {
    return {static_cast<int>(code), asio::system_category()};
}
}

TEST(Translate, ReportsAMissingGoneOrClosedPortAsUnreachable) {
    for (const DWORD code:
         {ERROR_FILE_NOT_FOUND,
          ERROR_PATH_NOT_FOUND,
          ERROR_INVALID_HANDLE,
          ERROR_BAD_COMMAND,
          ERROR_GEN_FAILURE,
          ERROR_DEV_NOT_EXIST,
          ERROR_SEM_TIMEOUT,
          ERROR_NO_SUCH_DEVICE,
          ERROR_OPERATION_ABORTED,
          ERROR_IO_DEVICE,
          ERROR_COUNTER_TIMEOUT,
          ERROR_DEVICE_NOT_CONNECTED,
          ERROR_DEVICE_REMOVED}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to read", os(code)),
            transport::UNREACHABLE_ERROR
        );
    }
    ASSERT_OCCURRED_AS(
        translate("failed to read", asio::error::eof),
        transport::UNREACHABLE_ERROR
    );
    ASSERT_OCCURRED_AS(
        translate("failed to read", asio::error::bad_descriptor),
        transport::UNREACHABLE_ERROR
    );
}

TEST(Translate, ReportsADeniedPortAsInUseByAnotherProgram) {
    for (const DWORD code: {ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION}) {
        SCOPED_TRACE(os(code).message());
        const auto err = translate("failed to open serial port COM3", os(code));
        ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
        EXPECT_EQ(
            err.data,
            "failed to open serial port COM3: the port is in use by another program"
        );
    }
}

TEST(Translate, ReportsAPortTheDriverMayNotUseAsAConfigError) {
    for (const DWORD code:
         {ERROR_INVALID_FUNCTION,
          ERROR_NOT_SUPPORTED,
          ERROR_INVALID_PARAMETER,
          ERROR_INVALID_NAME,
          ERROR_BAD_PATHNAME}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to open", os(code)),
            transport::CONFIG_ERROR
        );
    }
}

TEST(Translate, ReportsAnExhaustedHostOrAnUnknownErrorAsCritical) {
    for (const DWORD code:
         {ERROR_TOO_MANY_OPEN_FILES,
          ERROR_NOT_ENOUGH_MEMORY,
          ERROR_OUTOFMEMORY,
          ERROR_NO_SYSTEM_RESOURCES,
          ERROR_INVALID_DATA}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to open", os(code)),
            transport::CRITICAL_ERROR
        );
    }
}

TEST(Translate, PutsTheContextBeforeTheSystemMessage) {
    EXPECT_EQ(
        translate("failed to open serial port COM3", os(ERROR_FILE_NOT_FOUND)).data,
        "failed to open serial port COM3: " + os(ERROR_FILE_NOT_FOUND).message()
    );
}

TEST(Translate, MapsWin32ErrorsFromTheSystemCategory) {
    ASSERT_OCCURRED_AS(
        translate(
            "failed to set the line settings",
            {ERROR_GEN_FAILURE, std::system_category()}
        ),
        transport::UNREACHABLE_ERROR
    );
}
}
