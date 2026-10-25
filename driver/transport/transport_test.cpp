// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <functional>
#include <future>
#include <string>
#include <system_error>

#include "asio/error.hpp"
#include "asio/io_context.hpp"
#include "asio/post.hpp"
#include "asio/steady_timer.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::transport {
using Handler = std::function<void(const std::error_code &, std::size_t)>;

TEST(RunFor, ReportsATimeoutWhenItCancelsTheOperation) {
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

TEST(RunFor, ReportsAnAbortThatItDidNotCauseAsAnError) {
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

TEST(RunFor, KeepsAnOperationThatCompletesAsTheTimeoutCancelsIt) {
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

TEST(FromAsio, LetsAnAsioSystemErrorCompareToTheStandardCodes) {
    const asio::error_code ec(ENOENT, asio::system_category());
    EXPECT_NE(ec, std::errc::no_such_file_or_directory);
    EXPECT_EQ(from_asio(ec), std::errc::no_such_file_or_directory);
    EXPECT_EQ(from_asio(ec).message(), ec.message());
}

TEST(FromAsio, KeepsAnErrorFromAnotherCategory) {
    EXPECT_EQ(from_asio(asio::error::eof), asio::error::eof);
}

TEST(AbandonAfter, ReturnsTheResultOfACallThatFinishesInTime) {
    const auto [value, ec] = abandon_after(x::telem::SECOND, [] {
        return std::pair{std::string("resolved"), std::error_code{}};
    });
    EXPECT_FALSE(ec);
    EXPECT_EQ(value, "resolved");
}

TEST(AbandonAfter, StopsWaitingForACallThatBlocksPastTheTimeout) {
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
}
