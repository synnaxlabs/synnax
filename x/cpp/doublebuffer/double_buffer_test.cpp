// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/doublebuffer/double_buffer.h"
#include "gtest/gtest.h"
#include <thread>
#include <vector>

// Simple struct for testing
struct TestData {
    int value;
    explicit TestData(int v = 0) : value(v) {}
};

class DoubleBufferTest : public ::testing::Test {
protected:
    DoubleBuffer<TestData> buffer{TestData(0), TestData(0)};
};

// Test basic read/write operations
TEST_F(DoubleBufferTest, BasicReadWrite) {
    auto* write_buf = buffer.curr_write();
    ASSERT_NE(write_buf, nullptr);
    write_buf->value = 42;

    // Before exchange, read should return initial value
    auto [read_buf, has_new] = buffer.curr_read();
    ASSERT_FALSE(has_new);
    ASSERT_EQ(read_buf, nullptr);

    // After exchange, read should return new value
    buffer.exchange();
    auto [new_read_buf, has_new2] = buffer.curr_read();
    ASSERT_TRUE(has_new2);
    ASSERT_NE(new_read_buf, nullptr);
    ASSERT_EQ(new_read_buf->value, 42);
}

// Test that consecutive reads without exchange return no new data
TEST_F(DoubleBufferTest, ConsecutiveReads) {
    auto* write_buf = buffer.curr_write();
    write_buf->value = 42;
    buffer.exchange();

    // First read should succeed
    auto [read_buf1, has_new1] = buffer.curr_read();
    ASSERT_TRUE(has_new1);
    ASSERT_EQ(read_buf1->value, 42);

    // Second read should indicate no new data
    auto [read_buf2, has_new2] = buffer.curr_read();
    ASSERT_FALSE(has_new2);
    ASSERT_EQ(read_buf2, nullptr);
}

// Test alternating writes between buffers
TEST_F(DoubleBufferTest, AlternatingWrites) {
    for (int i = 0; i < 4; i++) {
        auto* write_buf = buffer.curr_write();
        write_buf->value = i;
        buffer.exchange();

        auto [read_buf, has_new] = buffer.curr_read();
        ASSERT_TRUE(has_new);
        ASSERT_EQ(read_buf->value, i);
    }
}


// Test that the buffer properly handles rapid exchanges
TEST_F(DoubleBufferTest, RapidExchanges) {
    constexpr int NUM_EXCHANGES = 10000;
    
    for (int i = 0; i < NUM_EXCHANGES; i++) {
        auto* write_buf = buffer.curr_write();
        write_buf->value = i;
        buffer.exchange();
    }

    auto [read_buf, has_new] = buffer.curr_read();
    ASSERT_TRUE(has_new);
    ASSERT_EQ(read_buf->value, NUM_EXCHANGES - 1);
}
