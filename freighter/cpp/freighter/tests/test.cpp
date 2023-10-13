// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "freighter/freighter.h"

class BasicMiddleware : public Freighter::PassthroughMiddleware {
private:
    std::string value;
public:
    explicit BasicMiddleware(std::string value) : value(std::move(value)) {}

    std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context context) override {
        context.set("test", value);
        return Freighter::PassthroughMiddleware::operator()(context);
    }
};

class BasicFinalizer : public Freighter::PassthroughMiddleware {
public:
    std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context context) override {
        return {context, nullptr};
    }
};

TEST(testFreighter, testMiddleware) {
    auto middleware = BasicMiddleware("5");
    auto finalizer = BasicFinalizer();
    middleware.setNext(&finalizer);
    auto context = Freighter::Context("test", "1");
    auto result = middleware(context);
    ASSERT_EQ(result.first.get("test"), "5");
}

TEST(testFreighter, testMiddlewareCollector) {
    auto collector = Freighter::MiddlewareCollector();
    auto mw1 = BasicMiddleware("5");
    auto mw2 = BasicMiddleware("6");
    auto f = BasicFinalizer();
    collector.use(&mw1);
    collector.use(&mw2);
    auto result = collector.exec(Freighter::Context("test", "1"), &f);
    ASSERT_EQ(result.first.get("test"), "6");
}