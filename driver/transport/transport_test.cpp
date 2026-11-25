// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <chrono>
#include <future>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

#include "asio/error.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/address.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/ip/udp.hpp"
#include "asio/post.hpp"
#include "asio/steady_timer.hpp"
#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::transport {
/// @brief it should return an operation that completes before the timeout.
TEST(RunFor, ReturnCompletion) {
    asio::io_context ctx;
    bool cancelled = false;
    const auto done = run_for(
        ctx,
        x::telem::SECOND,
        [&ctx](const Handler &handler) {
            asio::post(ctx, [handler] { handler({}, 5); });
        },
        [&cancelled] { cancelled = true; }
    );
    EXPECT_FALSE(cancelled);
    EXPECT_FALSE(done.timed_out());
    EXPECT_FALSE(done.ec);
    EXPECT_EQ(done.size, 5);
}

/// @brief it should cancel an operation at the timeout and report a timeout.
TEST(RunFor, ReportTimeout) {
    asio::io_context ctx;
    asio::steady_timer timer(ctx, std::chrono::hours(1));
    const auto done = run_for(
        ctx,
        10 * x::telem::MILLISECOND,
        [&timer](const Handler &handler) {
            timer.async_wait([handler](const std::error_code &ec) { handler(ec, 0); });
        },
        [&timer] { timer.cancel(); }
    );
    EXPECT_TRUE(done.timed_out());
    EXPECT_EQ(done.ec, asio::error::operation_aborted);
}

/// @brief it should report an abort that it did not cause as an error, not a timeout.
TEST(RunFor, ReportOwnAbortAsError) {
    asio::io_context ctx;
    bool cancelled = false;
    const auto done = run_for(
        ctx,
        x::telem::SECOND,
        [&ctx](const Handler &handler) {
            asio::post(ctx, [handler] { handler(asio::error::operation_aborted, 0); });
        },
        [&cancelled] { cancelled = true; }
    );
    EXPECT_FALSE(cancelled);
    EXPECT_FALSE(done.timed_out());
    EXPECT_EQ(done.ec, asio::error::operation_aborted);
}

/// @brief it should keep an operation that completes as the timeout cancels it.
TEST(RunFor, KeepCompletionDuringCancel) {
    asio::io_context ctx;
    Handler pending;
    const auto done = run_for(
        ctx,
        10 * x::telem::MILLISECOND,
        [&pending](Handler handler) { pending = std::move(handler); },
        [&ctx, &pending] { asio::post(ctx, [&pending] { pending({}, 3); }); }
    );
    EXPECT_FALSE(done.timed_out());
    EXPECT_FALSE(done.ec);
    EXPECT_EQ(done.size, 3);
}

/// @brief it should let an Asio system error compare equal to the standard codes.
TEST(FromAsio, ConvertSystemCategory) {
    const asio::error_code ec(ENOENT, asio::system_category());
    EXPECT_NE(ec, std::errc::no_such_file_or_directory);
    EXPECT_EQ(from_asio(ec), std::errc::no_such_file_or_directory);
    EXPECT_EQ(from_asio(ec).message(), ec.message());
}

/// @brief it should keep an error from another category.
TEST(FromAsio, KeepOtherCategory) {
    EXPECT_EQ(from_asio(asio::error::eof), asio::error::eof);
}

/// @brief it should match an error code that equals one of the codes.
TEST(Matches, AnyCode) {
    const auto ec = std::make_error_code(std::errc::connection_refused);
    EXPECT_TRUE(matches(ec, {std::errc::timed_out, std::errc::connection_refused}));
    EXPECT_FALSE(matches(ec, {std::errc::timed_out}));
}

/// @brief it should give an error of the type whose message is the context, then the
/// message of the code.
TEST(Error, PrefixContext) {
    const auto ec = std::make_error_code(std::errc::no_such_file_or_directory);
    const auto err = error(UNREACHABLE_ERROR, "failed to open /dev/ttyUSB0", ec);
    ASSERT_OCCURRED_AS(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "failed to open /dev/ttyUSB0: " + ec.message());
}

/// @brief it should return the result of a call that finishes in time.
TEST(AbandonAfter, ReturnResultInTime) {
    const auto [value, ec] = abandon_after(x::telem::SECOND, [] {
        return std::pair{std::string("resolved"), std::error_code{}};
    });
    EXPECT_FALSE(ec);
    EXPECT_EQ(value, "resolved");
}

/// @brief it should stop waiting for a call that blocks past the timeout.
TEST(AbandonAfter, StopWaitingPastTimeout) {
    std::promise<void> release;
    auto released = release.get_future().share();
    const x::telem::Stopwatch sw;
    const auto [value, ec] = abandon_after(20 * x::telem::MILLISECOND, [released] {
        released.wait();
        return std::pair{std::string("late"), std::error_code{}};
    });
    EXPECT_LT(sw.elapsed(), x::telem::SECOND);
    EXPECT_EQ(ec, std::errc::timed_out);
    EXPECT_TRUE(value.empty());
    release.set_value();
}

/// @brief it should resolve an IP address without waiting for getaddrinfo.
TEST(Resolve, ReturnIPAddressAtOnce) {
    const auto [v4, v4_err] = resolve<asio::ip::tcp>(
        "192.0.2.7",
        502,
        x::telem::TimeSpan::ZERO()
    );
    ASSERT_FALSE(v4_err) << v4_err.message();
    EXPECT_EQ(
        v4,
        std::vector{asio::ip::tcp::endpoint(asio::ip::make_address("192.0.2.7"), 502)}
    );
    const auto [v6, v6_err] = resolve<asio::ip::udp>(
        "::1",
        5000,
        x::telem::TimeSpan::ZERO()
    );
    ASSERT_FALSE(v6_err) << v6_err.message();
    EXPECT_EQ(
        v6,
        std::vector{asio::ip::udp::endpoint(asio::ip::make_address("::1"), 5000)}
    );
}

/// @brief it should resolve a name to its addresses on the port.
TEST(Resolve, ReturnAddressesOfName) {
    const auto [endpoints, err] = resolve<asio::ip::tcp>(
        "localhost",
        502,
        5 * x::telem::SECOND
    );
    ASSERT_FALSE(err) << err.message();
    ASSERT_FALSE(endpoints.empty());
    for (const auto &e: endpoints) {
        EXPECT_TRUE(e.address().is_loopback()) << e.address().to_string();
        EXPECT_EQ(e.port(), 502);
    }
}
}
