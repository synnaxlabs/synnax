// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/mock/server.h"

namespace driver::opc::connection {
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
    Config conn_cfg_;
};

/// @brief it should acquire a new connection from empty pool.
TEST_F(ConnectionPoolTest, AcquireNewConnection) {
    Pool pool;

    const auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_NE(connection.get(), nullptr);

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should reuse released connection from pool.
TEST_F(ConnectionPoolTest, ReuseConnection) {
    Pool pool;

    { auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should create multiple simultaneous connections.
TEST_F(ConnectionPoolTest, MultipleSimultaneousConnections) {
    Pool pool;

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

    Config cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(cfg2, "[test] "));

    EXPECT_EQ(pool.size(), 2);
    EXPECT_NE(conn1.get(), conn2.get());

    server2.stop();
}

/// @brief it should properly transfer ownership with move semantics.
TEST_F(ConnectionPoolTest, MoveSemantics) {
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    auto *original_ptr = conn1.get();

    Pool::Connection conn2 = std::move(conn1);
    EXPECT_EQ(conn2.get(), original_ptr);
    EXPECT_FALSE(conn1);

    Pool::Connection conn3(std::move(conn2));
    EXPECT_EQ(conn3.get(), original_ptr);
    EXPECT_FALSE(conn2);
}

/// @brief it should handle concurrent access from multiple threads safely.
TEST_F(ConnectionPoolTest, ThreadSafety) {
    Pool pool;
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
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client_ptr = conn1.shared();

    { Pool::Connection temp = std::move(conn1); }

    UA_Client_disconnect(client_ptr.get());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_NE(conn2.get(), client_ptr.get());
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should create separate connections for different credentials.
TEST_F(ConnectionPoolTest, DifferentCredentials) {
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    Config cfg_with_user = conn_cfg_;
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
    Pool pool;
    Config bad_cfg = conn_cfg_;
    bad_cfg.endpoint = "opc.tcp://localhost:9999";

    auto [connection, err] = pool.acquire(bad_cfg, "[test] ");
    ASSERT_TRUE(err);
    EXPECT_EQ(pool.size(), 0);
}

/// @brief it should automatically reconnect when stale connection is detected.
TEST_F(ConnectionPoolTest, StaleConnectionAutoReconnect) {
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = Pool::Connection(nullptr, nullptr, "");
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
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = Pool::Connection(nullptr, nullptr, "");

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
    Pool pool;

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = Pool::Connection(nullptr, nullptr, "");

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
    Pool pool;

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = Pool::Connection(nullptr, nullptr, "");

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
    Pool pool;

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
    Pool pool;

    // Create and hold connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Restart server while connection is held
    server_->stop();
    server_.reset();

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Release the now-broken connection
    conn1 = Pool::Connection(nullptr, nullptr, "");

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
    Pool pool;

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
    Pool pool;

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

/// @brief One endpoint's slow/failed health probe must not block acquisition
/// from a different healthy endpoint.
///
/// On main: health probe runs inside the mutex, so a thread probing an
/// endpoint holds the lock for the entire probe duration. Any other thread
/// trying to acquire from a different endpoint must wait behind that lock.
///
/// With the update: health probe runs outside the mutex, so the second
/// thread grabs the lock, picks its candidate, releases the lock, and
/// probes its connection concurrently.
///
/// We inject an artificial 2s probe delay via test_probe_delay_ms_ to
/// simulate a network-unreachable server (TCP SYN hang). Without this,
/// localhost probes return instantly and mutex contention is unobservable.
/// After thread A enters its delayed probe, we clear the delay so thread B's
/// probe completes instantly.
///
/// Expected timings:
///   main:   thread B waits ~2s behind thread A's lock      → FAIL
///   update: thread B runs concurrently, completes quickly   → PASS
TEST_F(ConnectionPoolTest, HealthyEndpointNotBlockedBySlowProbe) {
    mock::ServerConfig server2_cfg = mock::ServerConfig::create_default();
    server2_cfg.port = 4846;
    mock::Server server2(server2_cfg);
    server2.start();
    ASSERT_TRUE(server2.wait_until_ready());

    Config cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    Pool pool;

    // Prime both endpoints with cached connections
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
    { auto conn = ASSERT_NIL_P(pool.acquire(cfg2, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);
    EXPECT_EQ(pool.available_count(cfg2.endpoint), 1);

    // Inject a 2-second delay into the health probe
    const int probe_delay_ms = 2000;
    pool.test_probe_delay_ms_.store(probe_delay_ms, std::memory_order_relaxed);

    // Thread A: acquire from server1 (will hit the slow probe on cached conn)
    std::thread thread_a([&pool, this]() {
        auto [conn, err] = pool.acquire(conn_cfg_, "[A] ");
        (void)conn;
        (void)err;
    });

    // Give thread A time to grab its candidate and enter the delayed probe
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    // Clear the delay so thread B's own health probe completes instantly.
    // Thread A is already sleeping inside its probe.
    pool.test_probe_delay_ms_.store(0, std::memory_order_relaxed);

    std::chrono::milliseconds thread_b_elapsed{0};

    // Thread B: acquire from server2 (should not be blocked by A's probe)
    std::thread thread_b([&pool, &cfg2, &thread_b_elapsed]() {
        auto start = std::chrono::steady_clock::now();
        auto [conn, err] = pool.acquire(cfg2, "[B] ");
        thread_b_elapsed = std::chrono::duration_cast<
            std::chrono::milliseconds
        >(std::chrono::steady_clock::now() - start);
    });

    thread_a.join();
    thread_b.join();

    // With the update (probe outside lock): thread A holds the lock only
    // briefly to find its candidate, then releases and enters the slow
    // probe. Thread B acquires the lock immediately, finds its candidate,
    // and runs its own (instant) probe. Total: well under 1 second.
    //
    // On main (probe inside lock): thread A holds the lock for the entire
    // 2-second probe. Thread B blocks on the lock until A finishes, so it
    // takes >= probe_delay_ms.
    EXPECT_LT(thread_b_elapsed.count(), probe_delay_ms - 500)
        << "Thread B took " << thread_b_elapsed.count()
        << "ms, suggesting it was blocked by thread A's slow health probe. "
        << "The health probe should run outside the mutex.";

    server2.stop();
}

/// @brief Tests that the default client timeout is 5 seconds when
/// client_timeout_ms is 0 (the default).
TEST_F(ConnectionPoolTest, DefaultClientTimeout) {
    Pool pool;
    Config cfg = conn_cfg_;
    cfg.client_timeout_ms = 0; // Use default

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->timeout, 5000);
}

/// @brief Tests that custom client timeout overrides the default.
TEST_F(ConnectionPoolTest, CustomClientTimeout) {
    Pool pool;
    Config cfg = conn_cfg_;
    cfg.client_timeout_ms = 15000;

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->timeout, 15000);
}

/// @brief Tests that the default secure channel lifetime is 10 minutes
/// and session timeout is 20 minutes.
TEST_F(ConnectionPoolTest, DefaultSessionAndChannelTimeouts) {
    Pool pool;
    Config cfg = conn_cfg_;
    cfg.secure_channel_lifetime_ms = 0;
    cfg.session_timeout_ms = 0;

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->secureChannelLifeTime, 600000); // 10 minutes
    EXPECT_EQ(config->requestedSessionTimeout, 1200000); // 20 minutes
}

/// @brief Tests that custom secure channel and session timeouts override defaults.
TEST_F(ConnectionPoolTest, CustomSessionAndChannelTimeouts) {
    Pool pool;
    Config cfg = conn_cfg_;
    cfg.secure_channel_lifetime_ms = 30000;
    cfg.session_timeout_ms = 60000;

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->secureChannelLifeTime, 30000);
    EXPECT_EQ(config->requestedSessionTimeout, 60000);
}

class SessionLimitPoolTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = mock::ServerConfig::create_default();
        server_cfg_.port = 4860;
        server_cfg_.max_sessions = 2;
        server_ = std::make_unique<mock::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());

        conn_cfg_.endpoint = "opc.tcp://localhost:4860";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) server_->stop();
    }

    mock::ServerConfig server_cfg_;
    std::unique_ptr<mock::Server> server_;
    Config conn_cfg_;
};

/// @brief it should reject connections when server session limit is reached.
TEST_F(SessionLimitPoolTest, ServerRejectsWhenSessionLimitReached) {
    Pool pool;

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Third connection should fail — server only allows 2 sessions
    auto [conn3, err3] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_TRUE(err3);
    EXPECT_FALSE(conn3);

    // Pool should only have the 2 successful connections
    EXPECT_EQ(pool.size(), 2);
}

/// @brief it should reuse connections within session limit instead of creating
/// new ones.
TEST_F(SessionLimitPoolTest, ReusePreventsSessionExhaustion) {
    Pool pool;

    // Acquire and release — connection goes back to pool as cached
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Acquire again — should reuse, not create a new session
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 1);

    // Can still acquire a second (limit is 2)
    auto conn3 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 2);
}

/// @brief it should exhaust sessions when connections are discarded and
/// recreated rapidly (documents the churn problem).
TEST_F(SessionLimitPoolTest, ConnectionChurnExhaustsSessions) {
    Pool pool;

    // Acquire and release 2 connections — they go back to pool as cached
    {
        auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    }
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 2);

    // Simulate broken cached connections by disconnecting them
    // (mirrors what happens when a PLC becomes unresponsive and health
    // probes fail)
    server_->stop();
    server_.reset();

    server_cfg_.max_sessions = 2;
    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // First acquire: health probe on cached connection fails, pool discards
    // it and creates a new connection (session 1 on new server)
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Second acquire: same thing — discard cached, create new (session 2)
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Both should work since we're within the limit
    EXPECT_EQ(pool.size(), 2);

    // Release both
    conn1 = Pool::Connection(nullptr, nullptr, "");
    conn2 = Pool::Connection(nullptr, nullptr, "");

    // Verify they're reusable (no churn needed)
    auto conn3 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn4 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 2);
}

}
