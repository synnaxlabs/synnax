// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstdint>
#include <optional>
#include <string>
#include <system_error>
#include <vector>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/serial/errors.h"
#include "driver/serial/posix.h"
#include "driver/transport/errors.h"

namespace driver::serial::native {
namespace {
using Bytes = std::vector<std::uint8_t>;

/// @brief strips each chunk in turn and returns the bytes that remain.
Bytes strip_all(
    std::uint8_t &mark,
    const std::vector<Bytes> &chunks,
    std::size_t &errors
) {
    Bytes out;
    for (auto chunk: chunks) {
        const auto [size, dropped] = strip(mark, chunk);
        out.insert(out.end(), chunk.begin(), chunk.begin() + size);
        errors += dropped;
    }
    return out;
}
}

TEST(Strip, PassesBytesWithoutMarksThrough) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 2, 3}}, errors), (Bytes{1, 2, 3}));
    EXPECT_EQ(errors, 0);
}

TEST(Strip, UndoublesAnEscaped0xFF) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 0xFF, 0xFF, 2}}, errors), (Bytes{1, 0xFF, 2}));
    EXPECT_EQ(errors, 0);
}

TEST(Strip, DropsAndCountsAByteWithAParityOrFramingError) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 0xFF, 0x00, 0x41, 2}}, errors), (Bytes{1, 2}));
    EXPECT_EQ(errors, 1);
}

TEST(Strip, CountsABreak) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{0xFF, 0x00, 0x00, 7}}, errors), (Bytes{7}));
    EXPECT_EQ(errors, 1);
}

TEST(Strip, FollowsAMarkSplitAcrossReads) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(
        strip_all(mark, {{1, 0xFF}, {0x00}, {0x41, 2, 0xFF}, {0xFF, 3}}, errors),
        (Bytes{1, 2, 0xFF, 3})
    );
    EXPECT_EQ(errors, 1);
}

TEST(Overruns, StartsFromTheFirstCountItSees) {
    std::optional<std::uint32_t> last;
    EXPECT_EQ(overruns_since(last, 17), 0);
    EXPECT_EQ(overruns_since(last, 17), 0);
    EXPECT_EQ(overruns_since(last, 20), 3);
}

TEST(Overruns, CountsAcrossTheWrapOfThe32BitCounter) {
    std::optional<std::uint32_t> last;
    EXPECT_EQ(overruns_since(last, 0xFFFFFFFE), 0);
    EXPECT_EQ(overruns_since(last, 1), 3);
}

TEST(Overruns, ReportsNoneForAPortWithoutCounters) {
    std::optional<std::uint32_t> last;
    EXPECT_EQ(overruns_since(last, std::nullopt), 0);
    EXPECT_EQ(overruns_since(last, std::nullopt), 0);
}
}

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
