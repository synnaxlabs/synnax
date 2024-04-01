// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "freighter/cpp/freighter/freighter.h"

class BasicMiddleware : public freighter::PassthroughMiddleware {
private:
    std::string value;
public:
    explicit BasicMiddleware(std::string value) : value(std::move(value)) {}

    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context context) override {
        context.set("test", value);
        return freighter::PassthroughMiddleware::operator()(context);
    }
};

class BasicFinalizer : public freighter::PassthroughMiddleware {
public:
    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context context) override {
        return {context, freighter::NIL};
    }
};

TEST(testFreighter, testMiddleware) {
    auto middleware = BasicMiddleware("5");
    auto finalizer = BasicFinalizer();
    middleware.setNext(&finalizer);
    auto context = freighter::Context("test", "1");
    auto result = middleware(context);
    ASSERT_EQ(result.first.get("test"), "5");
}


TEST(testFreighter, testMiddlewareCollector) {
    auto collector = freighter::MiddlewareCollector();
    auto mw1 = std::make_shared<BasicMiddleware>("5");
    auto mw2 = std::make_shared<BasicMiddleware>("6");
    auto f = BasicFinalizer();
    collector.use(mw1);
    collector.use(mw2);
    auto result = collector.exec(freighter::Context("test", "1"), &f);
    ASSERT_EQ(result.first.get("test"), "6");
}

TEST(testFreighter, testErrorConstructionFromString) {
    std::string error = "sy.validation---invalid key: 1000: validation error";
    auto err = freighter::Error(error);
}

TEST(testFreighter, testErrorEqualsExactlyEqual) {
    auto err1 = freighter::Error("test", "");
    auto err2 = freighter::Error("test", "");
    ASSERT_EQ(err1, err2);
}

TEST(testFreighter, testErrorHequalHasPrefix) {
    auto err1 = freighter::Error("test", "");
    auto err2 = freighter::Error("test-specific", "");
    ASSERT_TRUE(err2.matches(err1));
}