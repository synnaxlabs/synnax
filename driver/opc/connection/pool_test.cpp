// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/mock/server.h"

class ConnectionPoolTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = mock::ServerConfig::create_default();
        server_cfg_.port = 4845;
        server_ = std::make_unique<mock::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());

        conn_cfg_.endpoint = "opc.tcp://localhost:4845";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) { server_->stop(); }
    }

    mock::ServerConfig server_cfg_;
    std::unique_ptr<mock::Server> server_;
    opc::connection::Config conn_cfg_;
};

/// @brief it should acquire a new connection from empty pool.
TEST_F(ConnectionPoolTest, AcquireNewConnection) {
    opc::connection::Pool pool;

    auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_NE(connection.get(), nullptr);

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should reuse released connection from pool.
TEST_F(ConnectionPoolTest, ReuseConnection) {
    opc::connection::Pool pool;

    { auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should create multiple simultaneous connections.
TEST_F(ConnectionPoolTest, MultipleSimultaneousConnections) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(pool.size(), 2);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);

    EXPECT_NE(conn1.get(), conn2.get());
}

/// @brief it should create separate connections for different endpoints.
TEST_F(ConnectionPoolTest, DifferentEndpoints) {
    mock::ServerConfig server2_cfg = mock::ServerConfig::create_default();
    server2_cfg.port = 4846;
    mock::Server server2(server2_cfg);
    server2.start();
    ASSERT_TRUE(server2.wait_until_ready());

    opc::connection::Config cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(cfg2, "[test] "));

    EXPECT_EQ(pool.size(), 2);
    EXPECT_NE(conn1.get(), conn2.get());

    server2.stop();
}

/// @brief it should properly transfer ownership with move semantics.
TEST_F(ConnectionPoolTest, MoveSemantics) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    auto *original_ptr = conn1.get();

    opc::connection::Pool::Connection conn2 = std::move(conn1);
    EXPECT_EQ(conn2.get(), original_ptr);
    EXPECT_FALSE(conn1);

    opc::connection::Pool::Connection conn3(std::move(conn2));
    EXPECT_EQ(conn3.get(), original_ptr);
    EXPECT_FALSE(conn2);
}

/// @brief it should handle concurrent access from multiple threads safely.
TEST_F(ConnectionPoolTest, ThreadSafety) {
    opc::connection::Pool pool;
    const int num_threads = 10;
    const int acquisitions_per_thread = 5;

    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&pool, &success_count, this]() {
            for (int j = 0; j < acquisitions_per_thread; ++j) {
                auto [connection, err] = pool.acquire(conn_cfg_, "[test] ");
                if (!err && connection) {
                    success_count++;
                    std::this_thread::sleep_for(std::chrono::milliseconds(10));
                }
            }
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(success_count, num_threads * acquisitions_per_thread);
}

/// @brief it should replace invalidated connections with new ones.
TEST_F(ConnectionPoolTest, ConnectionInvalidation) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client_ptr = conn1.shared();

    { opc::connection::Pool::Connection temp = std::move(conn1); }

    UA_Client_disconnect(client_ptr.get());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_NE(conn2.get(), client_ptr.get());
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should create separate connections for different credentials.
TEST_F(ConnectionPoolTest, DifferentCredentials) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    opc::connection::Config cfg_with_user = conn_cfg_;
    cfg_with_user.security_mode = "Sign";
    cfg_with_user.security_policy = "Basic256";

    auto [conn2, err2] = pool.acquire(cfg_with_user, "[test] ");
    if (err2) {
        GTEST_SKIP()
            << "Skipping credentials test - server doesn't support alternate security: "
            << err2.message();
    }

    EXPECT_NE(conn1.get(), conn2.get());
    EXPECT_EQ(pool.size(), 2);
}

/// @brief it should return error when connecting to unavailable server.
TEST_F(ConnectionPoolTest, AcquireFromBadServer) {
    opc::connection::Pool pool;
    opc::connection::Config bad_cfg = conn_cfg_;
    bad_cfg.endpoint = "opc.tcp://localhost:9999";

    auto [connection, err] = pool.acquire(bad_cfg, "[test] ");
    ASSERT_TRUE(err);
    EXPECT_EQ(pool.size(), 0);
}

/// @brief it should automatically reconnect when stale connection is detected.
TEST_F(ConnectionPoolTest, StaleConnectionAutoReconnect) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = opc::connection::Pool::Connection(nullptr, nullptr, "");
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    server_->stop();
    server_.reset();

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
}

/// @brief it should create new connection after server restart.
TEST_F(ConnectionPoolTest, NewConnectionAfterServerRestart) {
    opc::connection::Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = opc::connection::Pool::Connection(nullptr, nullptr, "");

    server_->stop();
    server_.reset();

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
}

/// @brief Tests that when run_iterate fails on a cached connection, the pool
/// discards it and creates a new connection.
TEST_F(ConnectionPoolTest, RunIterateFailureFallsThrough) {
    opc::connection::Pool pool;

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = opc::connection::Pool::Connection(nullptr, nullptr, "");

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Manually disconnect to simulate a connection that will fail run_iterate
    UA_Client_disconnect(client1_ptr.get());

    // Next acquire should skip the broken connection and create a new one
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Should have gotten a NEW connection (broken one was discarded)
    EXPECT_NE(conn2.get(), client1_ptr.get());
    // Pool should have exactly 1 connection (the new one)
    EXPECT_EQ(pool.size(), 1);
}

/// @brief Tests that when all cached connections fail, a new connection is created.
TEST_F(ConnectionPoolTest, AllCachedFailCreateNew) {
    opc::connection::Pool pool;

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = opc::connection::Pool::Connection(nullptr, nullptr, "");

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Manually disconnect to simulate broken cached connection
    UA_Client_disconnect(client1_ptr.get());

    // Acquire should create new connection after cached one fails
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Should have created a new connection (old one removed)
    EXPECT_EQ(pool.size(), 1);
    EXPECT_NE(conn2.get(), client1_ptr.get());
}

/// @brief Tests that when the server stops, acquire returns an error and
/// cleans up the broken connection from the pool.
TEST_F(ConnectionPoolTest, ServerStopsDuringAcquire) {
    opc::connection::Pool pool;

    // Create and release connection
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Stop server
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Acquire should fail cleanly (no server, cached connection broken)
    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_TRUE(err2); // Should return error
    EXPECT_FALSE(conn2);

    // Pool should be empty (broken connection removed, new connection failed)
    EXPECT_EQ(pool.size(), 0);
}

/// @brief Tests that after a server restart, the pool can recover and
/// provide working connections.
TEST_F(ConnectionPoolTest, ServerRestartRecovery) {
    opc::connection::Pool pool;

    // Create and hold connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Restart server while connection is held
    server_->stop();
    server_.reset();

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Release the now-broken connection
    conn1 = opc::connection::Pool::Connection(nullptr, nullptr, "");

    // Should be able to acquire new working connection
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Verify connection is actually functional by checking session state
    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(conn2.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);
}

/// @brief Tests that connection errors are properly propagated with the
/// correct error type.
TEST_F(ConnectionPoolTest, ErrorStatusPropagation) {
    opc::connection::Pool pool;

    // Create and release connection
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    // Stop server to make cached connection fail
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Acquire should return appropriate error
    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_TRUE(err2);

    // Error should be present and have a meaningful message
    EXPECT_FALSE(err2.message().empty());
}

/// @brief Tests that multiple threads can recover after server restart.
TEST_F(ConnectionPoolTest, ConcurrentRecoveryAfterFailure) {
    opc::connection::Pool pool;

    // Create initial connections
    {
        auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 2);

    // Restart server
    server_->stop();
    server_.reset();

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Multiple threads should all succeed with new connections
    const int num_threads = 5;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&pool, &success_count, this]() {
            auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
            if (!err && conn) {
                // Verify connection is functional
                UA_SessionState session_state;
                UA_SecureChannelState channel_state;
                UA_Client_getState(conn.get(), &channel_state, &session_state, nullptr);
                if (session_state == UA_SESSIONSTATE_ACTIVATED) { success_count++; }
            }
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    EXPECT_EQ(success_count, num_threads);
}
