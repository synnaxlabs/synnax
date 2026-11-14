// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <optional>

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/testutil/testutil.h"

namespace freighter {
class BasicMiddleware final : public PassthroughMiddleware {
    std::string value;

public:
    explicit BasicMiddleware(std::string value): value(std::move(value)) {}

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        context.set("test", value);
        return next(context);
    }
};

class BasicFinalizer final : public Finalizer<int, int> {
public:
    freighter::FinalizerReturn<int>
    operator()(const freighter::Context context, int &req) override {
        return {context, x::errors::NIL, req + 1};
    }
};

/// @brief it should execute middleware chain and return incremented result.
TEST(testFreighter, testMiddlewareCollector) {
    auto collector = MiddlewareCollector<int, int>();
    const auto mw1 = std::make_shared<BasicMiddleware>("5");
    const auto mw2 = std::make_shared<BasicMiddleware>("6");
    auto f = BasicFinalizer();
    collector.use(mw1);
    collector.use(mw2);
    const auto ctx = freighter::Context("test", x::url::URL("1"), freighter::UNARY);
    auto req = 1;
    const auto res = ASSERT_NIL_P(collector.exec(ctx, &f, req));
    ASSERT_EQ(res, 2);
}

/// @brief records the deadline of the last request.
class DeadlineFinalizer final : public Finalizer<int, int> {
public:
    std::optional<std::chrono::system_clock::time_point> deadline;

    freighter::FinalizerReturn<int>
    operator()(const freighter::Context context, int &req) override {
        this->deadline = context.deadline;
        return {context, x::errors::NIL, req};
    }
};

/// @brief it should set a deadline on a unary request and leave a stream unchanged.
TEST(testFreighter, testDeadlineSetsOnlyUnary) {
    auto collector = MiddlewareCollector<int, int>();
    collector.use(std::make_shared<Deadline>(std::chrono::seconds(60)));
    DeadlineFinalizer f;
    auto req = 1;
    const auto before = std::chrono::system_clock::now();
    ASSERT_NIL_P(collector.exec(Context("test", x::url::URL("1"), UNARY), &f, req));
    EXPECT_GE(
        f.deadline.value_or(std::chrono::system_clock::time_point::min()),
        before + std::chrono::seconds(60)
    );
    ASSERT_NIL_P(collector.exec(Context("test", x::url::URL("1"), STREAM), &f, req));
    EXPECT_FALSE(f.deadline.has_value());
}

/// @brief it should let a later deadline in the chain replace an earlier one.
TEST(testFreighter, testLaterDeadlineReplacesEarlier) {
    auto collector = MiddlewareCollector<int, int>();
    collector.use(std::make_shared<Deadline>(std::chrono::seconds(60)));
    collector.use(std::make_shared<Deadline>(std::chrono::seconds(5)));
    DeadlineFinalizer f;
    auto req = 1;
    ASSERT_NIL_P(collector.exec(Context("test", x::url::URL("1"), UNARY), &f, req));
    EXPECT_LT(
        f.deadline.value_or(std::chrono::system_clock::time_point::max()),
        std::chrono::system_clock::now() + std::chrono::seconds(10)
    );
}
}
