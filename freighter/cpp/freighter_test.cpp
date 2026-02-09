// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/test/test.h"

class BasicMiddleware final : public freighter::PassthroughMiddleware {
    std::string value;

public:
    explicit BasicMiddleware(std::string value): value(std::move(value)) {}

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        context.set("test", value);
        return next(context);
    }
};

class BasicFinalizer final : public freighter::Finalizer<int, int> {
public:
    freighter::FinalizerReturn<int>
    operator()(const freighter::Context context, int &req) override {
        return {context, x::errors::NIL, req + 1};
    }
};

/// @brief it should execute middleware chain and return incremented result.
TEST(testFreighter, testMiddlewareCollector) {
    auto collector = freighter::MiddlewareCollector<int, int>();
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
