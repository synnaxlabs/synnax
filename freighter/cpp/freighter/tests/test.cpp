#include <gtest/gtest.h>

#include "freighter/freighter.h"

class BasicMiddleware : public Freighter::BaseMiddleware {
private:
    std::string value;
public:
    explicit BasicMiddleware(std::string value) : value(std::move(value)) {}

    std::pair<Freighter::Context, std::exception *> operator()(Freighter::Context context) override {
        context.set("test", value);
        return Freighter::BaseMiddleware::operator()(context);
    }
};

class BasicFinalizer : public Freighter::BaseMiddleware {
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
