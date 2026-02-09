// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>

#include "gtest/gtest.h"

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xtest/xtest.h"

class XTestTest : public ::testing::Test {
protected:
    std::atomic<int> counter{0};

    void inc_counter() { ++this->counter; }

    void SetUp() override { this->counter = 0; }
};

/// @brief it should eventually reach an equal value across threads.
TEST_F(XTestTest, TestEventuallyEQ) {
    std::thread t([this] {
        for (int i = 0; i < 5; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            inc_counter();
        }
    });
    ASSERT_EVENTUALLY_EQ(counter.load(), 5);
    t.join();
}

/// @brief it should eventually reach a greater-than-or-equal value.
TEST_F(XTestTest, TestEventuallyGE) {
    std::thread t([this] {
        for (int i = 0; i < 10; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            inc_counter();
        }
    });
    ASSERT_EVENTUALLY_GE(counter.load(), 5);
    t.join();
}

/// @brief it should eventually reach a less-than-or-equal value.
TEST_F(XTestTest, TestEventuallyLE) {
    counter = 10;
    std::thread t([this] {
        for (int i = 0; i < 5; i++) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            --this->counter;
        }
    });
    ASSERT_EVENTUALLY_LE(counter.load(), 5);
    t.join();
}

/// @brief it should eventually reach an equal value with custom timeout.
TEST_F(XTestTest, TestEventuallyEQWithCustomTimeout) {
    std::thread t([this] {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 5;
    });

    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        counter.load(),
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

/// @brief it should eventually reach a greater-than-or-equal value with custom timeout.
TEST_F(XTestTest, TestEventuallyGEWithCustomTimeout) {
    std::thread t([this] {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 10;
    });

    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        counter.load(),
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

/// @brief it should eventually reach a less-than-or-equal value with custom timeout.
TEST_F(XTestTest, TestEventuallyLEWithCustomTimeout) {
    counter = 10;
    std::thread t([this] {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        counter = 3;
    });

    ASSERT_EVENTUALLY_LE_WITH_TIMEOUT(
        counter.load(),
        5,
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

/// @brief it should unwrap successful results with ASSERT_NIL_P.
TEST_F(XTestTest, TestMustSucceedSuccess) {
    auto successful_op = []() -> std::pair<int, xerrors::Error> {
        return {42, xerrors::NIL};
    };
    EXPECT_EQ(ASSERT_NIL_P(successful_op()), 42);
}

/// @brief it should eventually become true.
TEST_F(XTestTest, TestEventuallyTrue) {
    std::atomic<bool> flag{false};
    std::thread t([&flag] {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
        flag = true;
    });
    ASSERT_EVENTUALLY_TRUE(flag.load());
    t.join();
}

/// @brief it should eventually become false.
TEST_F(XTestTest, TestEventuallyFalse) {
    std::atomic<bool> flag{true};
    std::thread t([&flag] {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
        flag = false;
    });
    ASSERT_EVENTUALLY_FALSE(flag.load());
    t.join();
}

/// @brief it should eventually become true with custom timeout.
TEST_F(XTestTest, TestEventuallyTrueWithCustomTimeout) {
    std::atomic<bool> flag{false};
    std::thread t([&flag] {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        flag = true;
    });
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        flag.load(),
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

/// @brief it should eventually become false with custom timeout.
TEST_F(XTestTest, TestEventuallyFalseWithCustomTimeout) {
    std::atomic<bool> flag{true};
    std::thread t([&flag] {
        std::this_thread::sleep_for(std::chrono::milliseconds(150));
        flag = false;
    });
    ASSERT_EVENTUALLY_FALSE_WITH_TIMEOUT(
        flag.load(),
        std::chrono::milliseconds(200),
        std::chrono::milliseconds(10)
    );
    t.join();
}

/// @brief ASSERT_NIL should only evaluate the expression once.
TEST_F(XTestTest, TestAssertNilSingleEvaluation) {
    auto nil_with_side_effect = [this]() -> xerrors::Error {
        inc_counter();
        return xerrors::NIL;
    };
    ASSERT_NIL(nil_with_side_effect());
    EXPECT_EQ(counter.load(), 1);
}

/// @brief ASSERT_OCCURRED_AS should only evaluate the expression once.
TEST_F(XTestTest, TestAssertOccurredAsSingleEvaluation) {
    const auto expected = xerrors::Error("test error");
    auto error_with_side_effect = [this, &expected]() -> xerrors::Error {
        inc_counter();
        return expected;
    };
    ASSERT_OCCURRED_AS(error_with_side_effect(), expected);
    EXPECT_EQ(counter.load(), 1);
}
