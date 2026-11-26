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
#include <span>
#include <system_error>
#include <vector>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/serial/errors.h"
#include "driver/serial/native.h"
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
        const auto kept = std::span(chunk).first(size);
        out.insert(out.end(), kept.begin(), kept.end());
        errors += dropped;
    }
    return out;
}
}

/// @brief it should pass bytes without marks through.
TEST(Strip, PassesBytesWithoutMarksThrough) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 2, 3}}, errors), (Bytes{1, 2, 3}));
    EXPECT_EQ(errors, 0);
}

/// @brief it should turn an escaped 0xFF back into one byte.
TEST(Strip, UndoublesAnEscaped0xFF) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 0xFF, 0xFF, 2}}, errors), (Bytes{1, 0xFF, 2}));
    EXPECT_EQ(errors, 0);
}

/// @brief it should drop and count a byte with a parity or framing error.
TEST(Strip, DropsAndCountsAByteWithAParityOrFramingError) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{1, 0xFF, 0x00, 0x41, 2}}, errors), (Bytes{1, 2}));
    EXPECT_EQ(errors, 1);
}

/// @brief it should count a break.
TEST(Strip, CountsABreak) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(strip_all(mark, {{0xFF, 0x00, 0x00, 7}}, errors), (Bytes{7}));
    EXPECT_EQ(errors, 1);
}

/// @brief it should follow a mark that is split across reads.
TEST(Strip, FollowsAMarkSplitAcrossReads) {
    std::uint8_t mark = 0;
    std::size_t errors = 0;
    EXPECT_EQ(
        strip_all(mark, {{1, 0xFF}, {0x00}, {0x41, 2, 0xFF}, {0xFF, 3}}, errors),
        (Bytes{1, 2, 0xFF, 3})
    );
    EXPECT_EQ(errors, 1);
}

/// @brief it should drop a lone 0xFF without writing ahead of the input.
TEST(Strip, DropsALone0xFFWithoutWritingAheadOfTheInput) {
    std::uint8_t mark = 1;
    Bytes chunk{0x42, 0x43};
    const auto [size, marks] = strip(mark, chunk);
    EXPECT_EQ(size, 2);
    EXPECT_EQ(marks, 0);
    EXPECT_EQ(chunk, (Bytes{0x42, 0x43}));
    EXPECT_EQ(mark, 0);
}

#ifdef __linux__
/// @brief it should keep the stripped bytes when the overrun count fails.
TEST(Receive, KeepsTheBytesWhenTheOverrunCountFails) {
    Line line;
    Bytes chunk{1, 0xFF, 0xFF, 2};
    const auto [received, ec] = receive(-1, line, chunk);
    EXPECT_EQ(ec, std::errc::bad_file_descriptor);
    EXPECT_EQ(received.size, 3);
    EXPECT_EQ(std::span(chunk).first(received.size)[1], 0xFF);
}
#endif

/// @brief it should start from the first count that it sees.
TEST(Overruns, StartsFromTheFirstCountItSees) {
    std::optional<std::uint32_t> last;
    EXPECT_EQ(overruns_since(last, 17), 0);
    EXPECT_EQ(overruns_since(last, 17), 0);
    EXPECT_EQ(overruns_since(last, 20), 3);
}

/// @brief it should count across the wrap of the 32-bit counter.
TEST(Overruns, CountsAcrossTheWrapOfThe32BitCounter) {
    std::optional<std::uint32_t> last;
    EXPECT_EQ(overruns_since(last, 0xFFFFFFFE), 0);
    EXPECT_EQ(overruns_since(last, 1), 3);
}

/// @brief it should report no overruns for a port without counters.
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

/// @brief it should report a missing, gone, or closed port as unreachable.
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

/// @brief it should report a busy port at open as in use by another program.
TEST(TranslateOpen, ReportsABusyPortAsInUseByAnotherProgram) {
    const auto err = translate_open("/dev/ttyUSB0", os(EBUSY));
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port /dev/ttyUSB0: the port is in use by another "
        "program"
    );
}

/// @brief it should map every other open error through translate.
TEST(TranslateOpen, MapsEveryOtherErrorThroughTranslate) {
    const auto err = translate_open("/dev/ttyUSB0", os(ENOENT));
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open serial port /dev/ttyUSB0: " + os(ENOENT).message()
    );
}

/// @brief it should report a busy port outside an open with the system message.
TEST(Translate, ReportsABusyPortOutsideAnOpenWithTheSystemMessage) {
    const auto err = translate(
        "failed to read from serial port /dev/ttyUSB0",
        os(EBUSY)
    );
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to read from serial port /dev/ttyUSB0: " + os(EBUSY).message()
    );
}

/// @brief it should report a port that the Driver may not use as a configuration error.
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

/// @brief it should report an exhausted host or an unknown error as critical.
TEST(Translate, ReportsAnExhaustedHostOrAnUnknownErrorAsCritical) {
    for (const int code: {EMFILE, ENFILE, ENOMEM, ENOBUFS, ENOTRECOVERABLE}) {
        SCOPED_TRACE(os(code).message());
        ASSERT_OCCURRED_AS(
            translate("failed to open", os(code)),
            transport::CRITICAL_ERROR
        );
    }
}

/// @brief it should map an errno from the generic category.
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

/// @brief it should put the context before the system message.
TEST(Translate, PutsTheContextBeforeTheSystemMessage) {
    EXPECT_EQ(
        translate("failed to open serial port /dev/ttyUSB0", os(ENOENT)).data,
        "failed to open serial port /dev/ttyUSB0: " + os(ENOENT).message()
    );
}
}
