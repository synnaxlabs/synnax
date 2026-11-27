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

/// @brief calls the next middleware twice, as a middleware that retries does.
class RetryMiddleware final : public PassthroughMiddleware {
public:
    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        next(context);
        return next(context);
    }
};

/// @brief counts the times it runs.
class CountMiddleware final : public PassthroughMiddleware {
public:
    int calls = 0;

    std::pair<freighter::Context, x::errors::Error>
    operator()(freighter::Context context, freighter::Next &next) override {
        ++this->calls;
        return next(context);
    }
};

/// @brief it should run the rest of the chain each time a middleware calls next.
TEST(testFreighter, testMiddlewareCollectorRepeatedNext) {
    auto collector = MiddlewareCollector<int, int>();
    const auto count = std::make_shared<CountMiddleware>();
    auto f = BasicFinalizer();
    collector.use(std::make_shared<RetryMiddleware>());
    collector.use(count);
    const auto ctx = freighter::Context("test", x::url::URL("1"), freighter::UNARY);
    auto req = 1;
    const auto res = ASSERT_NIL_P(collector.exec(ctx, &f, req));
    ASSERT_EQ(res, 2);
    ASSERT_EQ(count->calls, 2);
}

/// @brief it should replace the params of the target when assigning a context.
TEST(testFreighter, testContextAssignReplacesParams) {
    auto target = freighter::Context("test", x::url::URL("1"), freighter::UNARY);
    target.set("stale", "1");
    auto source = freighter::Context("test", x::url::URL("1"), freighter::UNARY);
    source.set("fresh", "2");
    target = source;
    ASSERT_FALSE(target.has("stale"));
    ASSERT_TRUE(target.has("fresh"));
}

/// @brief it should call every function that a canceler holds when it cancels, and
/// none that it released.
TEST(testFreighter, testCancelerEndsEveryHeldRequest) {
    Canceler canceler;
    int first = 0;
    int second = 0;
    int released = 0;
    const auto a = canceler.hold([&] { first++; });
    const auto b = canceler.hold([&] { second++; });
    const auto c = canceler.hold([&] { released++; });
    ASSERT_TRUE(a.has_value() && b.has_value() && c.has_value());
    canceler.release(*c);
    canceler.cancel();
    ASSERT_EQ(first, 1);
    ASSERT_EQ(second, 1);
    ASSERT_EQ(released, 0);
    ASSERT_TRUE(canceler.is_cancelled());
    ASSERT_FALSE(canceler.hold([] {}).has_value());
}

/// @brief it should fail a send_cancellable whose canceler cancelled without sending,
/// and send otherwise, for a client that keeps the default send_cancellable.
TEST(testFreighter, testDefaultSendCancellable) {
    class Client final : public UnaryClient<int, int> {
    public:
        int sent = 0;

        void use(std::shared_ptr<Middleware>) override {}

        std::pair<int, x::errors::Error> send(const std::string &, int &req) override {
            this->sent++;
            return {req, x::errors::NIL};
        }
    };
    Client client;
    int req = 1;
    const auto cancelled = std::make_shared<Canceler>();
    cancelled->cancel();
    ASSERT_OCCURRED_AS_P(
        client.send_cancellable("", req, cancelled),
        x::errors::CANCELLED
    );
    ASSERT_EQ(client.sent, 0);
    ASSERT_EQ(ASSERT_NIL_P(client.send_cancellable("", req, nullptr)), 1);
}
}
