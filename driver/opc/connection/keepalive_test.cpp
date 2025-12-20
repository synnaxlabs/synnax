// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"
#include "open62541/client.h"
#include "open62541/client_highlevel.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/mock/server.h"

using namespace opc::connection;

class ConnectionPoolKeepAliveTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = mock::ServerConfig::create_default();
        server_cfg_.port = 4847;
        server_ = std::make_unique<mock::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());

        conn_cfg_.endpoint = "opc.tcp://localhost:4847";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) { server_->stop(); }
    }

    mock::ServerConfig server_cfg_;
    std::unique_ptr<mock::Server> server_;
    Config conn_cfg_;
};

/// @brief it should keep connection alive when repeatedly acquired and released.
TEST_F(ConnectionPoolKeepAliveTest, RepeatedAcquireKeepsConnectionAlive) {
    Pool pool;

    // Acquire and release 100 times over ~10 seconds
    // This simulates normal task operation patterns
    for (int i = 0; i < 100; ++i) {
        auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

        // Verify connection is actually functional
        UA_SessionState session_state;
        UA_SecureChannelState channel_state;
        UA_Client_getState(connection.get(), &channel_state, &session_state, nullptr);

        EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED)
            << "Iteration " << i << ": Session not activated";

        // Simulate some work
        std::this_thread::sleep_for(std::chrono::milliseconds(100));

        // Connection auto-released when connection goes out of scope
    }

    // Connection should still be in pool and reusable
    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Final acquire should succeed
    auto final_conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
}

/// @brief it should survive idle periods between acquisitions.
TEST_F(ConnectionPoolKeepAliveTest, ConnectionSurvivesIdlePeriods) {
    Pool pool;

    // Initial acquisition
    { auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Wait 5 seconds (simulating idle time)
    std::this_thread::sleep_for(std::chrono::seconds(5));

    // Acquire again - should get same connection, still alive
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(conn2.get(), &channel_state, &session_state, nullptr);

    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);
    EXPECT_EQ(pool.size(), 1); // Same connection reused
}

/// @brief it should handle concurrent access with keep-alive enabled.
TEST_F(ConnectionPoolKeepAliveTest, ConcurrentAccessWithKeepAlive) {
    Pool pool;
    std::atomic<int> success_count{0};
    std::atomic<int> failure_count{0};

    constexpr int num_threads = 5;
    constexpr int iterations_per_thread = 20;

    std::vector<std::thread> threads;

    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&]() {
            for (int j = 0; j < iterations_per_thread; ++j) {
                auto [connection, err] = pool.acquire(conn_cfg_, "[test] ");
                if (err || !connection) {
                    ++failure_count;
                    continue;
                }

                UA_SessionState session_state;
                UA_SecureChannelState channel_state;
                UA_Client_getState(
                    connection.get(),
                    &channel_state,
                    &session_state,
                    nullptr
                );

                if (session_state == UA_SESSIONSTATE_ACTIVATED)
                    ++success_count;
                else
                    ++failure_count;

                std::this_thread::sleep_for(std::chrono::milliseconds(50));
            }
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(success_count, num_threads * iterations_per_thread);
    EXPECT_EQ(failure_count, 0);
}

/// @brief it should perform reads correctly after keep-alive iterations.
TEST_F(ConnectionPoolKeepAliveTest, CanPerformReadAfterKeepAlive) {
    Pool pool;

    // Acquire connection multiple times to trigger keep-alive
    for (int i = 0; i < 10; ++i) {
        auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

        // Try to read a node to verify connection is functional
        UA_Variant value;
        UA_Variant_init(&value);

        UA_StatusCode status = UA_Client_readValueAttribute(
            connection.get(),
            UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME),
            &value
        );

        EXPECT_EQ(status, UA_STATUSCODE_GOOD) << "Iteration " << i << ": Read failed";

        UA_Variant_clear(&value);

        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
}

/// @brief it should maintain connection alive with short timeout configuration.
TEST_F(ConnectionPoolKeepAliveTest, ShortTimeoutKeepAlive) {
    Pool pool;

    // Configure very short timeouts for testing
    Config short_cfg = conn_cfg_;
    short_cfg.secure_channel_lifetime_ms = 15000; // 15 seconds
    short_cfg.session_timeout_ms = 30000; // 30 seconds
    short_cfg.client_timeout_ms = 15000; // 15 seconds

    // Keep acquiring at regular intervals to trigger keep-alive
    // Interval is less than half the lifetime to ensure renewal happens
    constexpr int num_iterations = 8;
    constexpr int interval_seconds = 4; // 4s interval < 15s/2 lifetime

    for (int i = 0; i < num_iterations; ++i) {
        auto connection = ASSERT_NIL_P(pool.acquire(short_cfg, "[test] "));

        // Verify connection is active
        UA_SessionState session_state;
        UA_SecureChannelState channel_state;
        UA_Client_getState(connection.get(), &channel_state, &session_state, nullptr);

        EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED)
            << "Iteration " << i << ": Session not activated";

        // Wait before next iteration
        // Connection released when connection goes out of scope
        if (i < num_iterations - 1)
            std::this_thread::sleep_for(std::chrono::seconds(interval_seconds));
    }

    // Total time elapsed: 8 iterations * 4s = 32 seconds
    // SecureChannel would have expired at 15s without keep-alive
    // We've gone through 2+ full SecureChannel lifetimes successfully

    // Pool should have at most 1 connection (stale ones get cleaned up on acquire)
    EXPECT_LE(pool.size(), 1);

    // Verify we can still acquire successfully
    auto final_conn = ASSERT_NIL_P(pool.acquire(short_cfg, "[test] "));
}
