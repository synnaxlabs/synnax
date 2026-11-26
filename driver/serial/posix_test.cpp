// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <system_error>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/errors.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @returns errno as Asio reports it.
std::error_code os(const int code) {
    return {code, asio::system_category()};
}
}

TEST(Translate, ReportsAMissingGoneOrClosedPortAsUnreachable) {
    for (const int code: {ENOENT, ENXIO, ENODEV, EIO, EPIPE, EBADF, ETIMEDOUT}) {
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
}

TEST(Translate, ReportsABusyPortAsInUseByAnotherProgram) {
    const auto err = translate("failed to open serial port /dev/ttyUSB0", os(EBUSY));
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port /dev/ttyUSB0: the port is in use by another "
        "program"
    );
}

TEST(Translate, ReportsAPortTheDriverMayNotUseAsAConfigError) {
    for (const int code:
         {EACCES, EPERM, EINVAL, ENOTTY, EISDIR, ENOTDIR, ENAMETOOLONG, ENOTSUP}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to open", os(code)),
            transport::CONFIG_ERROR
        );
    }
}

TEST(Translate, ReportsAnExhaustedHostOrAnUnknownErrorAsCritical) {
    for (const int code: {EMFILE, ENFILE, ENOMEM, ENOBUFS, ENOTRECOVERABLE}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to open", os(code)),
            transport::CRITICAL_ERROR
        );
    }
}

TEST(Translate, MapsErrnoFromTheGenericCategory) {
    ASSERT_OCCURRED_AS(
        translate("failed to set the line settings", {EIO, std::generic_category()}),
        transport::UNREACHABLE_ERROR
    );
    ASSERT_OCCURRED_AS(
        translate("failed to set the line settings", {EINVAL, std::generic_category()}),
        transport::CONFIG_ERROR
    );
}

TEST(Translate, PutsTheContextBeforeTheSystemMessage) {
    EXPECT_EQ(
        translate("failed to open serial port /dev/ttyUSB0", os(ENOENT)).data,
        "failed to open serial port /dev/ttyUSB0: " + os(ENOENT).message()
    );
}
}
