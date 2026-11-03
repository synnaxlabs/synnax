// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <functional>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include "open62541/client_subscriptions.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::connection {
const std::string BREAKER_OPEN_TEXT = "The Driver paused new connections to the server "
                                      "after too many failed attempts in a row.";

class ConnectionPoolTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = testutil::ServerConfig::create_default();
        server_cfg_.port = 4845;
        server_ = std::make_unique<testutil::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());

        conn_cfg_.endpoint = "opc.tcp://localhost:4845";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) { server_->stop(); }
    }

    /// @brief starts the server again with server_cfg_.
    void restart_server() {
        server_->stop();
        server_ = std::make_unique<testutil::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());
    }

    testutil::ServerConfig server_cfg_;
    std::unique_ptr<testutil::Server> server_;
    Config conn_cfg_;
};

/// @brief it should acquire a new connection from empty pool.
TEST_F(ConnectionPoolTest, AcquireNewConnection) {
    Pool pool(testutil::create_store());

    const auto connection = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_NE(connection.get(), nullptr);

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should reuse released connection from pool.
TEST_F(ConnectionPoolTest, ReuseConnection) {
    Pool pool(testutil::create_store());

    { auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

/// @brief it should create multiple simultaneous connections.
TEST_F(ConnectionPoolTest, MultipleSimultaneousConnections) {
    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(pool.size(), 2);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);

    EXPECT_NE(conn1.get(), conn2.get());
}

/// @brief it should create separate connections for different endpoints.
TEST_F(ConnectionPoolTest, DifferentEndpoints) {
    testutil::ServerConfig server2_cfg = testutil::ServerConfig::create_default();
    server2_cfg.port = 4846;
    testutil::Server server2(server2_cfg);
    server2.start();
    ASSERT_TRUE(server2.wait_until_ready());

    Config cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn2 = ASSERT_NIL_P(pool.acquire(cfg2, "[test] "));

    EXPECT_EQ(pool.size(), 2);
    EXPECT_NE(conn1.get(), conn2.get());

    server2.stop();
}

/// @brief it should properly transfer ownership with move semantics.
// NOLINTBEGIN(bugprone-use-after-move): the tests check the moved-from state.
TEST_F(ConnectionPoolTest, MoveSemantics) {
    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    auto *original_ptr = conn1.get();

    Pool::Connection conn2 = std::move(conn1);
    EXPECT_EQ(conn2.get(), original_ptr);
    EXPECT_FALSE(conn1);

    Pool::Connection conn3(std::move(conn2));
    EXPECT_EQ(conn3.get(), original_ptr);
    EXPECT_FALSE(conn2);
}
// NOLINTEND(bugprone-use-after-move)

/// @brief it should handle concurrent access from multiple threads safely.
TEST_F(ConnectionPoolTest, ThreadSafety) {
    Pool pool(testutil::create_store());
    const int num_threads = 10;
    const int acquisitions_per_thread = 5;

    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    threads.reserve(num_threads);
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

/// @brief it should connect a closed connection again in place.
TEST_F(ConnectionPoolTest, ConnectionInvalidation) {
    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client_ptr = conn1.shared();

    { Pool::Connection temp = std::move(conn1); }

    UA_Client_disconnect(client_ptr.get());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(conn2.get(), client_ptr.get());
    ASSERT_NIL(lost(conn2.get()));
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should create separate connections for different credentials.
TEST_F(ConnectionPoolTest, DifferentCredentials) {
    Pool pool(testutil::create_store());

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

/// @brief it should not reuse an idle connection for a config with other timeouts.
TEST_F(ConnectionPoolTest, DifferentTimeouts) {
    for (const auto &change: std::vector<std::function<void(Config &)>>{
             [](Config &c) { c.client_timeout_ms = 15000; },
             [](Config &c) { c.session_timeout_ms = 60000; },
             [](Config &c) { c.secure_channel_lifetime_ms = 30000; },
         }) {
        Pool pool(testutil::create_store());
        { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
        Config other = conn_cfg_;
        change(other);
        const auto conn = ASSERT_NIL_P(pool.acquire(other, "[test] "));
        EXPECT_EQ(pool.size(), 2);
    }
}

/// @brief it should return error when connecting to unavailable server.
TEST_F(ConnectionPoolTest, AcquireFromBadServer) {
    Pool pool(testutil::create_store());
    Config bad_cfg = conn_cfg_;
    bad_cfg.endpoint = "opc.tcp://localhost:9999";

    auto [connection, err] = pool.acquire(bad_cfg, "[test] ");
    ASSERT_TRUE(err);
    EXPECT_EQ(pool.size(), 0);
}

/// @brief it should automatically reconnect when stale connection is detected.
TEST_F(ConnectionPoolTest, StaleConnectionAutoReconnect) {
    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = Pool::Connection(nullptr, nullptr, "");
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    server_->stop();
    server_.reset();

    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
}

/// @brief it should create new connection after server restart.
TEST_F(ConnectionPoolTest, NewConnectionAfterServerRestart) {
    Pool pool(testutil::create_store());

    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    conn1 = Pool::Connection(nullptr, nullptr, "");

    server_->stop();
    server_.reset();

    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
}

/// @brief it should connect a cached connection that fails its probe again in place.
TEST_F(ConnectionPoolTest, RunIterateFailureFallsThrough) {
    Pool pool(testutil::create_store());

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = Pool::Connection(nullptr, nullptr, "");

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Manually disconnect to simulate a connection that will fail run_iterate
    UA_Client_disconnect(client1_ptr.get());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(conn2.get(), client1_ptr.get());
    ASSERT_NIL(lost(conn2.get()));
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should connect a second connection while the first is in use.
TEST_F(ConnectionPoolTest, AllCachedFailCreateNew) {
    Pool pool(testutil::create_store());

    // Create and release a connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto client1_ptr = conn1.shared();
    conn1 = Pool::Connection(nullptr, nullptr, "");

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Manually disconnect to simulate broken cached connection
    UA_Client_disconnect(client1_ptr.get());

    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    auto conn3 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    EXPECT_EQ(conn2.get(), client1_ptr.get());
    EXPECT_NE(conn3.get(), client1_ptr.get());
    EXPECT_EQ(pool.size(), 2);
}

/// @brief it should return an error when the server stops, and keep the lost connection
/// to connect it again later.
TEST_F(ConnectionPoolTest, ServerStopsDuringAcquire) {
    Pool pool(testutil::create_store());

    // Create and release connection
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Stop server
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    ASSERT_OCCURRED_AS_P(pool.acquire(conn_cfg_, "[test] "), errors::UNREACHABLE);
    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);
}

/// @brief Tests that after a server restart, the pool can recover and
/// provide working connections.
TEST_F(ConnectionPoolTest, ServerRestartRecovery) {
    Pool pool(testutil::create_store());

    // Create and hold connection
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Restart server while connection is held
    server_->stop();
    server_.reset();

    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Release the now-broken connection
    conn1 = Pool::Connection(nullptr, nullptr, "");

    // Should be able to acquire new working connection
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));

    // Verify connection is actually functional by checking session state
    UA_SessionState session_state = UA_SESSIONSTATE_CLOSED;
    UA_SecureChannelState channel_state = UA_SECURECHANNELSTATE_CLOSED;
    UA_Client_getState(conn2.get(), &channel_state, &session_state, nullptr);
    EXPECT_EQ(session_state, UA_SESSIONSTATE_ACTIVATED);
}

/// @brief Tests that connection errors are properly propagated with the
/// correct error type.
TEST_F(ConnectionPoolTest, ErrorStatusPropagation) {
    Pool pool(testutil::create_store());

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
    Pool pool(testutil::create_store());

    // Create initial connections
    {
        auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 2);

    // Restart server
    server_->stop();
    server_.reset();

    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Multiple threads should all succeed with new connections
    const int num_threads = 5;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    threads.reserve(num_threads);
    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&pool, &success_count, this]() {
            auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
            if (!err && conn) {
                // Verify connection is functional
                UA_SessionState session_state = UA_SESSIONSTATE_CLOSED;
                UA_SecureChannelState channel_state = UA_SECURECHANNELSTATE_CLOSED;
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
/// We inject an artificial 2s delay into the mock server's iterate cycle via
/// server_->probe_delay to simulate a slow server (TCP SYN hang). Without
/// this, localhost probes return instantly and mutex contention is unobservable.
/// After thread A enters its delayed probe, we clear the server delay so
/// thread B's probe against server2 completes instantly.
///
/// Expected timings:
///   main:   thread B waits ~2s behind thread A's lock      → FAIL
///   update: thread B runs concurrently, completes quickly   → PASS
TEST_F(ConnectionPoolTest, HealthyEndpointNotBlockedBySlowProbe) {
    testutil::ServerConfig server2_cfg = testutil::ServerConfig::create_default();
    server2_cfg.port = 4846;
    testutil::Server server2(server2_cfg);
    server2.start();
    ASSERT_TRUE(server2.wait_until_ready());

    Config cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    Pool pool(testutil::create_store());

    // Prime both endpoints with cached connections
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
    { auto conn = ASSERT_NIL_P(pool.acquire(cfg2, "[test] ")); }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);
    EXPECT_EQ(pool.available_count(cfg2.endpoint), 1);

    // Inject a 2-second delay into the mock server's iterate cycle to simulate
    // a slow/unreachable server where health probes hang.
    server_->probe_delay = 2 * x::telem::SECOND;

    // Thread A: acquire from server1 (will hit the slow probe on cached conn)
    std::thread thread_a([&pool, this]() {
        auto [conn, err] = pool.acquire(conn_cfg_, "[A] ");
        (void) conn;
        (void) err;
    });

    // Give thread A time to grab its candidate and enter the delayed probe
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    // Clear the server delay so subsequent iterate cycles are fast.
    // Thread A is already blocked waiting for server1's response.
    server_->probe_delay = x::telem::TimeSpan{};

    std::chrono::milliseconds thread_b_elapsed{0};

    // Thread B: acquire from server2 (should not be blocked by A's probe)
    std::thread thread_b([&pool, &cfg2, &thread_b_elapsed]() {
        auto start = std::chrono::steady_clock::now();
        auto [conn, err] = pool.acquire(cfg2, "[B] ");
        thread_b_elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::steady_clock::now() - start
        );
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
    // takes >= 2 seconds (the injected probe delay).
    EXPECT_LT(thread_b_elapsed.count(), 1500)
        << "Thread B took " << thread_b_elapsed.count()
        << "ms, suggesting it was blocked by thread A's slow health probe. "
        << "The health probe should run outside the mutex.";

    server2.stop();
}

/// @brief Tests that the default client timeout is 5 seconds when
/// client_timeout_ms is 0 (the default).
TEST_F(ConnectionPoolTest, DefaultClientTimeout) {
    Pool pool(testutil::create_store());
    Config cfg = conn_cfg_;
    cfg.client_timeout_ms = 0; // Use default

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->timeout, 5000);
}

/// @brief Tests that custom client timeout overrides the default.
TEST_F(ConnectionPoolTest, CustomClientTimeout) {
    Pool pool(testutil::create_store());
    Config cfg = conn_cfg_;
    cfg.client_timeout_ms = 15000;

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->timeout, 15000);
}

/// @brief Tests that the default secure channel lifetime is 10 minutes
/// and session timeout is 20 minutes.
TEST_F(ConnectionPoolTest, DefaultSessionAndChannelTimeouts) {
    Pool pool(testutil::create_store());
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
    Pool pool(testutil::create_store());
    Config cfg = conn_cfg_;
    cfg.secure_channel_lifetime_ms = 30000;
    cfg.session_timeout_ms = 60000;

    auto conn = ASSERT_NIL_P(pool.acquire(cfg, "[test] "));
    auto *config = UA_Client_getConfig(conn.get());
    EXPECT_EQ(config->secureChannelLifeTime, 30000);
    EXPECT_EQ(config->requestedSessionTimeout, 60000);
}

/// @brief it should read the namespace array and operation limits of the server.
TEST_F(ConnectionPoolTest, ReadsServerFacts) {
    server_cfg_.namespaces = {"urn:synnax:test"};
    server_cfg_.max_nodes_per_read = 7;
    server_cfg_.max_nodes_per_write = 3;
    server_cfg_.max_nodes_per_browse = 5;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto &server = conn.server();
    EXPECT_EQ(
        server.namespaces,
        std::vector<std::string>(
            {"http://opcfoundation.org/UA/",
             "urn:open62541.server.application",
             "urn:synnax:test"}
        )
    );
    EXPECT_EQ(server.max_nodes_per_read, 7);
    EXPECT_EQ(server.max_nodes_per_write, 3);
    EXPECT_EQ(server.max_nodes_per_browse, 5);
}

/// @brief it should read the server facts from a server that allows one node per Read
/// request.
TEST_F(ConnectionPoolTest, ReadsServerFactsOneNodePerRead) {
    server_cfg_.namespaces = {"urn:synnax:test"};
    server_cfg_.max_nodes_per_read = 1;
    server_cfg_.max_nodes_per_write = 3;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.server().namespaces.size(), 3);
    EXPECT_EQ(conn.server().namespaces[2], "urn:synnax:test");
    EXPECT_EQ(conn.server().max_nodes_per_read, 1);
    EXPECT_EQ(conn.server().max_nodes_per_write, 3);
}

/// @brief it should keep the server facts of an idle connection that it reuses.
TEST_F(ConnectionPoolTest, ReuseKeepsServerFacts) {
    server_cfg_.namespaces = {"urn:synnax:test"};
    restart_server();
    Pool pool(testutil::create_store());
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 1);
    ASSERT_EQ(conn.server().namespaces.size(), 3);
    EXPECT_EQ(conn.server().namespaces[2], "urn:synnax:test");
}

/// @brief it should read the server facts again when it connects a lost connection
/// again, since a restarted server can order its namespaces differently.
TEST_F(ConnectionPoolTest, ReconnectReadsServerFactsAgain) {
    server_cfg_.namespaces = {"urn:a", "urn:b"};
    restart_server();
    Pool pool(testutil::create_store());
    {
        const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        EXPECT_EQ(conn.server().namespaces[2], "urn:a");
    }
    server_cfg_.namespaces = {"urn:b", "urn:a"};
    restart_server();
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 1);
    ASSERT_EQ(conn.server().namespaces.size(), 4);
    EXPECT_EQ(conn.server().namespaces[2], "urn:b");
    EXPECT_EQ(conn.server().namespaces[3], "urn:a");
}

/// @brief it should split a read that names more nodes than the server allows, and
/// return the results in the order of the request.
TEST_F(ConnectionPoolTest, ReadSplitsAtServerLimit) {
    server_cfg_.max_nodes_per_read = 2;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const std::vector<std::string> names = {
        "TestUInt16",
        "TestUInt32",
        "TestInt16",
        "TestInt32",
        "TestDouble",
    };
    std::vector<types::NodeId> ids;
    ids.reserve(names.size());
    types::ReadRequestBuilder builder;
    for (const auto &name: names) {
        ids.push_back(ASSERT_NIL_P(types::NodeId::parse("ns=1;s=" + name)));
        builder.add_node(ids.back().resolve({}).value());
    }
    const auto res = ASSERT_NIL_P(conn.read(builder.build()));
    const auto &raw = res.get();
    ASSERT_EQ(raw.resultsSize, names.size());
    for (size_t i = 0; i < raw.resultsSize; ++i)
        EXPECT_EQ(raw.results[i].status, UA_STATUSCODE_GOOD) << names[i];
    EXPECT_EQ(*static_cast<UA_UInt16 *>(raw.results[0].value.data), 42);
    EXPECT_EQ(*static_cast<UA_UInt32 *>(raw.results[1].value.data), 12345);
    EXPECT_EQ(*static_cast<UA_Int16 *>(raw.results[2].value.data), 42);
    EXPECT_EQ(*static_cast<UA_Int32 *>(raw.results[3].value.data), 12345);
    EXPECT_DOUBLE_EQ(*static_cast<UA_Double *>(raw.results[4].value.data), 2.71828);
}

/// @brief it should split a write that names more nodes than the server allows.
TEST_F(ConnectionPoolTest, WriteSplitsAtServerLimit) {
    server_cfg_.max_nodes_per_write = 2;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto id = ASSERT_NIL_P(types::NodeId::parse("ns=1;s=TestUInt32"));
    types::WriteRequestBuilder builder;
    for (const uint32_t value: {1U, 2U, 3U})
        ASSERT_NIL(builder.add_value(
            id.resolve({}).value(),
            x::telem::Series(value, x::telem::UINT32_T),
            nullptr
        ));
    const auto res = ASSERT_NIL_P(conn.write(builder.build()));
    ASSERT_EQ(res.get().resultsSize, 3);
    for (size_t i = 0; i < 3; ++i)
        EXPECT_EQ(res.get().results[i], UA_STATUSCODE_GOOD);
    const auto value = ASSERT_NIL_P(
        testutil::simple_read(conn.shared(), "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(value.at<uint32_t>(0), 3);
}

/// @returns the browse names of the children in result.
std::vector<std::string> names(const types::BrowseResult &result) {
    std::vector<std::string> out;
    out.reserve(result.references.size());
    for (const auto &ref: result.references)
        out.push_back(ref.browse_name);
    return out;
}

/// @brief it should list the forward hierarchical children of a node, without the
/// target of its HasTypeDefinition reference.
TEST_F(ConnectionPoolTest, BrowseListsChildren) {
    server_cfg_.generated_variables = 2;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto results = ASSERT_NIL_P(
        conn.browse({UA_NODEID_STRING(1, const_cast<char *>("Generated"))})
    );
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL(results[0].err);
    EXPECT_EQ(names(results[0]), std::vector<std::string>({"V0", "V1", "Inner"}));
    const auto &v0 = results[0].references[0];
    EXPECT_EQ(v0.node_class, UA_NODECLASS_VARIABLE);
    EXPECT_EQ(v0.node.to_string(), "NS=1;S=Generated.V0");
    EXPECT_EQ(results[0].references[2].node_class, UA_NODECLASS_OBJECT);
}

/// @brief it should follow continuation points until the server returns every child.
TEST_F(ConnectionPoolTest, BrowseFollowsContinuationPoints) {
    server_cfg_.generated_variables = 7;
    server_cfg_.max_references_per_node = 2;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto results = ASSERT_NIL_P(
        conn.browse({UA_NODEID_STRING(1, const_cast<char *>("Generated"))})
    );
    ASSERT_NIL(results[0].err);
    EXPECT_EQ(
        names(results[0]),
        std::vector<std::string>({"V0", "V1", "V2", "V3", "V4", "V5", "V6", "Inner"})
    );
    EXPECT_FALSE(results[0].truncated);
}

/// @brief it should stop when the children of all of its nodes reach MAX_CHILDREN, and
/// mark each node that it cut short.
TEST_F(ConnectionPoolTest, BrowseStopsAtChildLimit) {
    server_cfg_.generated_variables = MAX_CHILDREN + 1;
    server_cfg_.max_nodes_per_browse = 1;
    server_cfg_.max_references_per_node = 2500;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto inner = UA_NODEID_STRING(1, const_cast<char *>("Generated.Inner"));
    const auto results = ASSERT_NIL_P(
        conn.browse({UA_NODEID_STRING(1, const_cast<char *>("Generated")), inner})
    );
    ASSERT_EQ(results[0].references.size(), MAX_CHILDREN);
    EXPECT_TRUE(results[0].truncated);
    EXPECT_TRUE(results[1].references.empty());
    EXPECT_TRUE(results[1].truncated);
    const auto again = ASSERT_NIL_P(conn.browse({inner}));
    EXPECT_EQ(names(again[0]), std::vector<std::string>({"Deep", "Generated"}));
    EXPECT_FALSE(again[0].truncated);
}

/// @brief it should stop following continuation points once its deadline passes, and
/// mark each node that it cut short.
TEST_F(ConnectionPoolTest, BrowseStopsAtDeadline) {
    server_cfg_.generated_variables = 50;
    server_cfg_.max_nodes_per_browse = 1;
    server_cfg_.max_references_per_node = 1;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    server_->probe_delay = 100 * x::telem::MILLISECOND;
    const auto start = std::chrono::steady_clock::now();
    const auto results = ASSERT_NIL_P(conn.browse(
        {UA_NODEID_STRING(1, const_cast<char *>("Generated")),
         UA_NODEID_STRING(1, const_cast<char *>("Generated.Inner"))},
        start + std::chrono::seconds(1)
    ));
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(5));
    EXPECT_GT(results[0].references.size(), 0);
    EXPECT_LT(results[0].references.size(), 51);
    EXPECT_TRUE(results[0].truncated);
    EXPECT_TRUE(results[1].references.empty());
    EXPECT_TRUE(results[1].truncated);
    server_->probe_delay = x::telem::TimeSpan{};
}

/// @brief it should split a browse that names more nodes than the server allows, and
/// keep the error of a node that fails without failing the others.
TEST_F(ConnectionPoolTest, BrowseSplitsAtServerLimit) {
    server_cfg_.generated_variables = 1;
    server_cfg_.max_nodes_per_browse = 1;
    server_cfg_.max_references_per_node = 1;
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto results = ASSERT_NIL_P(conn.browse({
        UA_NODEID_STRING(1, const_cast<char *>("Generated")),
        UA_NODEID_STRING(1, const_cast<char *>("Missing")),
        UA_NODEID_STRING(1, const_cast<char *>("Generated.Inner")),
    }));
    ASSERT_EQ(results.size(), 3);
    EXPECT_EQ(names(results[0]), std::vector<std::string>({"V0", "Inner"}));
    ASSERT_OCCURRED_AS(results[1].err, errors::CRITICAL.sub("BadNodeIdUnknown"));
    EXPECT_EQ(
        results[1].err.data,
        "The node id refers to a node that does not exist in the server address space."
    );
    EXPECT_TRUE(results[1].references.empty());
    EXPECT_EQ(names(results[2]), std::vector<std::string>({"Deep", "Generated"}));
}

/// @brief an event that a subscription delivered.
struct Received {
    std::string message;
    std::optional<UA_Double> level;
};

/// @brief the fields that the event tests select: Message and 1:Level.
const std::vector<std::vector<UA_QualifiedName>> EVENT_FIELDS = {
    {UA_QUALIFIEDNAME(0, const_cast<char *>("Message"))},
    {UA_QUALIFIEDNAME(1, const_cast<char *>("Level"))},
};

/// @returns a handler that adds each event to received.
EventHandler collect(std::vector<Received> &received) {
    return [&received](const UA_KeyValueMap &fields) {
        Received r;
        const auto &message = fields.map[0].value;
        if (UA_Variant_hasScalarType(&message, &UA_TYPES[UA_TYPES_LOCALIZEDTEXT])) {
            const auto &text = static_cast<const UA_LocalizedText *>(message.data)
                                   ->text;
            r.message = std::string(reinterpret_cast<char *>(text.data), text.length);
        }
        const auto &level = fields.map[1].value;
        if (UA_Variant_hasScalarType(&level, &UA_TYPES[UA_TYPES_DOUBLE]))
            r.level = *static_cast<const UA_Double *>(level.data);
        received.push_back(std::move(r));
    };
}

/// @returns whether client holds the subscription with id.
bool holds(UA_Client *client, const UA_UInt32 id) {
    void *context = nullptr;
    return UA_Client_Subscriptions_getContext(client, id, &context) ==
           UA_STATUSCODE_GOOD;
}

/// @brief iterates conn until received holds count events, or 5 seconds pass.
void iterate_until(
    const Pool::Connection &conn,
    const std::vector<Received> &received,
    const size_t count
) {
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(5);
    while (received.size() < count && std::chrono::steady_clock::now() < deadline)
        ASSERT_NIL(conn.iterate(x::telem::MILLISECOND * 100));
    ASSERT_EQ(received.size(), count);
}

/// @brief it should deliver the events of a notifier that pass the filter, with the
/// selected fields in order.
TEST_F(ConnectionPoolTest, SubscribeDeliversFilteredEvents) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const types::EventFilter filter(EVENT_FIELDS, {testutil::TEST_EVENT_TYPE});
    std::vector<Received> received;
    const auto results = ASSERT_NIL_P(
        conn.subscribe({testutil::PUMP}, filter.get(), collect(received))
    );
    ASSERT_EQ(results.size(), 1);
    ASSERT_NIL(results[0]);
    EXPECT_NE(conn.subscription_id(), 0);
    server_->emit({.source = testutil::PUMP, .message = "filtered out"});
    server_->emit({
        .source = testutil::PUMP,
        .type = testutil::TEST_EVENT_TYPE,
        .message = "overpressure",
        .level = 3.5,
    });
    iterate_until(conn, received, 1);
    EXPECT_EQ(received[0].message, "overpressure");
    EXPECT_EQ(received[0].level, 3.5);
}

/// @brief it should monitor the notifiers that exist, and return the error of each
/// notifier that the server does not monitor.
TEST_F(ConnectionPoolTest, SubscribeReportsEachNotifier) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const types::EventFilter filter(EVENT_FIELDS, {});
    std::vector<Received> received;
    const auto results = ASSERT_NIL_P(conn.subscribe(
        {UA_NODEID_STRING(1, const_cast<char *>("Missing")), testutil::PUMP},
        filter.get(),
        collect(received)
    ));
    ASSERT_EQ(results.size(), 2);
    ASSERT_OCCURRED_AS(results[0], errors::CRITICAL.sub("BadNodeIdUnknown"));
    ASSERT_NIL(results[1]);
    server_->emit({.source = testutil::PUMP, .message = "started"});
    iterate_until(conn, received, 1);
    EXPECT_EQ(received[0].message, "started");
    EXPECT_FALSE(received[0].level.has_value());
}

/// @brief it should delete the earlier subscription when the connection subscribes
/// again.
TEST_F(ConnectionPoolTest, SubscribeReplacesSubscription) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const types::EventFilter filter(EVENT_FIELDS, {});
    std::vector<Received> first;
    ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(first)));
    const auto replaced = conn.subscription_id();
    std::vector<Received> second;
    ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(second)));
    EXPECT_NE(conn.subscription_id(), replaced);
    EXPECT_FALSE(holds(conn.get(), replaced));
    server_->emit({.source = testutil::PUMP, .message = "once"});
    iterate_until(conn, second, 1);
    EXPECT_TRUE(first.empty());
}

/// @brief it should delete the subscription when the connection goes back to the pool,
/// and keep the Session for the next caller.
TEST_F(ConnectionPoolTest, ReleaseDeletesSubscription) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    const types::EventFilter filter(EVENT_FIELDS, {});
    UA_UInt32 id = 0;
    UA_Client *client = nullptr;
    {
        std::vector<Received> received;
        auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(received)));
        id = conn.subscription_id();
        client = conn.get();
    }
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.get(), client);
    ASSERT_NIL(lost(conn.get()));
    EXPECT_FALSE(holds(conn.get(), id));
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should report a lost connection while it waits for events, and delete the
/// subscription of the lost connection once the pool connects again.
TEST_F(ConnectionPoolTest, IterateReportsLostConnection) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    const types::EventFilter filter(EVENT_FIELDS, {});
    UA_UInt32 id = 0;
    UA_Client *client = nullptr;
    {
        std::vector<Received> received;
        auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(received)));
        id = conn.subscription_id();
        client = conn.get();
        server_->stop();
        x::errors::Error err;
        const auto deadline = std::chrono::steady_clock::now() +
                              std::chrono::seconds(5);
        while (!err && std::chrono::steady_clock::now() < deadline)
            err = conn.iterate(x::telem::MILLISECOND * 100);
        ASSERT_OCCURRED_AS(err, errors::NO_CONNECTION);
    }
    EXPECT_TRUE(holds(client, id));
    restart_server();
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.get(), client);
    EXPECT_FALSE(holds(client, id));
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should keep the Session of a connection whose channel drops while it holds
/// a subscription, and delete the subscription once the Session is active again.
TEST_F(ConnectionPoolTest, LostChannelKeepsSession) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    const types::EventFilter filter(EVENT_FIELDS, {});
    UA_UInt32 id = 0;
    UA_Client *client = nullptr;
    {
        std::vector<Received> received;
        auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(received)));
        id = conn.subscription_id();
        client = conn.get();
        UA_Client_disconnectSecureChannel(client);
    }
    EXPECT_TRUE(holds(client, id));
    // The handler of the subscription is gone, so this event must go nowhere.
    server_->emit({.source = testutil::PUMP, .message = "orphaned"});
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.get(), client);
    EXPECT_FALSE(holds(client, id));
    EXPECT_EQ(pool.size(), 1);
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    EXPECT_EQ(server_->sessions_created, 1);
}

/// @brief it should wait on a server that stops answering only once when a connection
/// with a subscription goes back to the pool, and keep its Session.
TEST_F(ConnectionPoolTest, SilentServerKeepsSession) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    conn_cfg_.client_timeout_ms = 400;
    Pool pool(testutil::create_store());
    const types::EventFilter filter(EVENT_FIELDS, {});
    UA_UInt32 id = 0;
    UA_Client *client = nullptr;
    std::chrono::steady_clock::duration released{};
    {
        std::vector<Received> received;
        auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(received)));
        id = conn.subscription_id();
        client = conn.get();
        types::ReadRequestBuilder builder;
        builder.add_node(
            UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME)
        );
        server_->probe_delay = 2 * x::telem::SECOND;
        // The server answers this read, then waits before it handles the next request.
        ASSERT_NIL_P(conn.read(builder.build()));
        const auto start = std::chrono::steady_clock::now();
        conn = Pool::Connection(nullptr, nullptr, "");
        released = std::chrono::steady_clock::now() - start;
    }
    EXPECT_LT(released, std::chrono::milliseconds(700));
    ASSERT_OCCURRED_AS(lost(client), errors::NO_CONNECTION);
    EXPECT_TRUE(holds(client, id));
    server_->probe_delay = x::telem::TimeSpan();
    std::this_thread::sleep_for(std::chrono::milliseconds(2100));
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.get(), client);
    EXPECT_FALSE(holds(client, id));
    EXPECT_EQ(server_->sessions_created, 1);
}

/// @brief it should not wait on the server when a connection with a subscription goes
/// back to the pool after a call timed out, and keep its Session.
TEST_F(ConnectionPoolTest, TimeoutKeepsSession) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    conn_cfg_.client_timeout_ms = 250;
    Pool pool(testutil::create_store());
    const types::EventFilter filter(EVENT_FIELDS, {});
    UA_UInt32 id = 0;
    UA_Client *client = nullptr;
    std::chrono::steady_clock::duration released{};
    {
        std::vector<Received> received;
        auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_NIL_P(conn.subscribe({testutil::PUMP}, filter.get(), collect(received)));
        id = conn.subscription_id();
        client = conn.get();
        types::ReadRequestBuilder builder;
        builder.add_node(
            UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME)
        );
        server_->probe_delay = x::telem::SECOND;
        // The server answers this read, then waits before it handles the next one.
        ASSERT_NIL_P(conn.read(builder.build()));
        ASSERT_OCCURRED_AS_P(conn.read(builder.build()), errors::UNREACHABLE);
        const auto start = std::chrono::steady_clock::now();
        conn = Pool::Connection(nullptr, nullptr, "");
        released = std::chrono::steady_clock::now() - start;
    }
    EXPECT_LT(released, std::chrono::milliseconds(100));
    EXPECT_TRUE(holds(client, id));
    server_->probe_delay = x::telem::TimeSpan();
    std::this_thread::sleep_for(std::chrono::milliseconds(1100));
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_EQ(conn.get(), client);
    EXPECT_FALSE(holds(client, id));
    EXPECT_EQ(server_->sessions_created, 1);
}

/// @returns a request to call method on the Pump object with args.
UA_CallMethodRequest
call_request(const UA_NodeId &method, std::vector<UA_Variant> &args) {
    UA_CallMethodRequest req;
    UA_CallMethodRequest_init(&req);
    req.objectId = testutil::PUMP;
    req.methodId = method;
    req.inputArguments = args.data();
    req.inputArgumentsSize = args.size();
    return req;
}

/// @brief it should call a method with its arguments.
TEST_F(ConnectionPoolTest, CallRunsMethod) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    UA_Double speed = 42;
    UA_String mode = UA_STRING(const_cast<char *>("auto"));
    std::vector<UA_Variant> args(2);
    UA_Variant_setScalar(args.data(), &mode, &UA_TYPES[UA_TYPES_STRING]);
    UA_Variant_setScalar(&args[1], &speed, &UA_TYPES[UA_TYPES_DOUBLE]);
    const auto res = ASSERT_NIL_P(
        conn.call(call_request(testutil::PUMP_CONFIGURE, args))
    );
    EXPECT_EQ(res.get().results[0].statusCode, UA_STATUSCODE_GOOD);
    types::ReadRequestBuilder builder;
    const auto speed_node = UA_NODEID_STRING(1, const_cast<char *>("Pump.Speed"));
    const auto mode_node = UA_NODEID_STRING(1, const_cast<char *>("Pump.Mode"));
    builder.add_node(speed_node).add_node(mode_node);
    const auto read = ASSERT_NIL_P(conn.read(builder.build()));
    EXPECT_EQ(*static_cast<UA_Double *>(read.get().results[0].value.data), 42);
    const auto *got = static_cast<UA_String *>(read.get().results[1].value.data);
    EXPECT_EQ(std::string(reinterpret_cast<char *>(got->data), got->length), "auto");
}

/// @brief it should return the status of a method that fails as its result.
TEST_F(ConnectionPoolTest, CallReturnsMethodStatus) {
    server_cfg_.pump = testutil::Pump{};
    restart_server();
    Pool pool(testutil::create_store());
    const auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    std::vector<UA_Variant> args;
    const auto res = ASSERT_NIL_P(conn.call(call_request(testutil::PUMP_FAIL, args)));
    EXPECT_EQ(res.get().results[0].statusCode, UA_STATUSCODE_BADINVALIDSTATE);
}

class SessionLimitPoolTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = testutil::ServerConfig::create_default();
        server_cfg_.port = 4860;
        server_cfg_.max_sessions = 2;
        server_ = std::make_unique<testutil::Server>(server_cfg_);
        server_->start();
        ASSERT_TRUE(server_->wait_until_ready());

        conn_cfg_.endpoint = "opc.tcp://localhost:4860";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) server_->stop();
    }

    testutil::ServerConfig server_cfg_;
    std::unique_ptr<testutil::Server> server_;
    Config conn_cfg_;
};

/// @brief it should reject connections when server session limit is reached.
TEST_F(SessionLimitPoolTest, ServerRejectsWhenSessionLimitReached) {
    Pool pool(testutil::create_store());

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
    Pool pool(testutil::create_store());

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

/// @brief it should connect lost connections again in place after the server restarts,
/// so the pool stays within the server's Session limit.
TEST_F(SessionLimitPoolTest, ReconnectKeepsSessionCount) {
    Pool pool(testutil::create_store());

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
    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // Each acquire connects one lost entry again, which opens one Session on the new
    // server.
    auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
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

/// @brief it should activate the same Session again after the channel drops, so a flaky
/// network does not fill the server's Session table.
TEST_F(SessionLimitPoolTest, ReconnectKeepsSession) {
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto client = conn.shared();
    for (int i = 0; i < 4; ++i) {
        UA_Client_disconnectSecureChannel(client.get());
        conn = Pool::Connection(nullptr, nullptr, "");
        conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        ASSERT_EQ(conn.get(), client.get());
    }
    auto other = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(pool.size(), 2);
}

/// @brief it should leave a connection that the server dropped lost while the client
/// handles network events, so only the Pool connects again.
TEST_F(SessionLimitPoolTest, DroppedConnectionStaysLost) {
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    server_->stop();
    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());
    for (int i = 0; i < 20; ++i)
        UA_Client_run_iterate(conn.get(), 50);
    const auto err = lost(conn.get());
    ASSERT_OCCURRED_AS(err, errors::NO_CONNECTION);
    EXPECT_EQ(err.data, "The connection to the server is lost.");
}

/// @brief it should double the cooldown each time the breaker opens again.
TEST_F(SessionLimitPoolTest, CooldownDoubles) {
    Pool pool(
        testutil::create_store(),
        {.min_cooldown = std::chrono::milliseconds(200),
         .max_cooldown = std::chrono::seconds(1)}
    );
    server_->stop();
    server_.reset();
    for (int i = 0; i < 3; ++i)
        ASSERT_OCCURRED_AS_P(pool.acquire(conn_cfg_, "[test] "), errors::UNREACHABLE);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    ASSERT_OCCURRED_AS_P(pool.acquire(conn_cfg_, "[test] "), errors::UNREACHABLE);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
    EXPECT_EQ(err.data, BREAKER_OPEN_TEXT);
}

/// @brief it should close a connection that no one used for the idle timeout, with no
/// call to the Pool.
TEST_F(SessionLimitPoolTest, ClosesIdleConnections) {
    Pool pool(
        testutil::create_store(),
        {.idle_timeout = std::chrono::milliseconds(200)}
    );
    auto held = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
    EXPECT_EQ(pool.size(), 2);
    ASSERT_EVENTUALLY_EQ(pool.size(), 1);
    held = Pool::Connection(nullptr, nullptr, "");
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);
    ASSERT_EVENTUALLY_EQ(pool.size(), 0);
}

/// @brief it should close the channel of a call that the server does not answer in
/// time, so the next acquire connects again with the same Session instead of probing
/// first.
TEST_F(SessionLimitPoolTest, TimeoutClosesChannel) {
    Pool pool(testutil::create_store());
    conn_cfg_.client_timeout_ms = 250;
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    const auto client = conn.shared();
    types::ReadRequestBuilder builder;
    builder.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME));
    server_->probe_delay = x::telem::SECOND;
    // The server answers this read, then waits before it handles the next one.
    ASSERT_NIL_P(conn.read(builder.build()));
    ASSERT_OCCURRED_AS_P(conn.read(builder.build()), errors::UNREACHABLE);
    ASSERT_OCCURRED_AS(lost(conn.get()), errors::NO_CONNECTION);
    server_->probe_delay = x::telem::TimeSpan();
    std::this_thread::sleep_for(std::chrono::milliseconds(1100));
    conn = Pool::Connection(nullptr, nullptr, "");
    conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_EQ(conn.get(), client.get());
    EXPECT_EQ(pool.size(), 1);
}

/// @brief it should record the time of the last reply from the server.
TEST_F(SessionLimitPoolTest, LastReply) {
    Pool pool(testutil::create_store());
    EXPECT_EQ(pool.last_reply(conn_cfg_), x::telem::TimeStamp());
    const auto before = x::telem::TimeStamp::now();
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    EXPECT_GE(pool.last_reply(conn_cfg_), before);
    Config other = conn_cfg_;
    other.password = "other";
    EXPECT_EQ(pool.last_reply(other), x::telem::TimeStamp());
}

/// @brief it should not send a request on a lost connection, since open62541 would
/// connect again inside the call.
TEST_F(SessionLimitPoolTest, ReadOnLostConnection) {
    Pool pool(testutil::create_store());
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    UA_Client_disconnectSecureChannel(conn.get());
    types::ReadRequestBuilder builder;
    builder.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME));
    ASSERT_OCCURRED_AS_P(conn.read(builder.build()), errors::NO_CONNECTION);
    ASSERT_OCCURRED_AS(lost(conn.get()), errors::NO_CONNECTION);
}

/// @brief After 3 consecutive connection failures, the circuit breaker trips
/// and rejects acquire() immediately without attempting network I/O.
///
/// On main: no circuit breaker exists, so every acquire() attempts connect()
/// even when the server is down, hammering the endpoint.
///
/// With the update: after three failures, the breaker opens and acquire() returns an
/// error instantly during the cooldown window.
TEST_F(SessionLimitPoolTest, CircuitBreakerTripsAfterFailures) {
    Pool pool(testutil::create_store());

    // Stop server so all connection attempts fail
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    Config bad_cfg = conn_cfg_;

    // First 3 failures should attempt connection (and fail)
    for (int i = 0; i < 3; ++i) {
        auto [conn, err] = pool.acquire(bad_cfg, "[test] ");
        ASSERT_TRUE(err) << "Attempt " << i << " should fail (server is down)";
    }

    // 4th attempt: breaker should be tripped — reject instantly
    auto start = std::chrono::steady_clock::now();
    auto [conn, err] = pool.acquire(bad_cfg, "[test] ");
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now() - start
    );

    ASSERT_TRUE(err);
    EXPECT_FALSE(conn);

    // Should complete in <500ms (no network I/O), proving breaker rejected it
    EXPECT_LT(elapsed.count(), 500)
        << "Breaker should reject instantly, but took " << elapsed.count() << "ms";

    // Error message should mention circuit breaker
    EXPECT_EQ(err.data, BREAKER_OPEN_TEXT);
}

/// @brief it should reject an acquire at once after the breaker opens, even when the
/// pool holds a lost connection, since connecting that connection again waits on the
/// network.
TEST_F(SessionLimitPoolTest, CircuitBreakerSkipsCleanupWhenTripped) {
    Pool pool(testutil::create_store());

    // Prime pool with one cached connection
    { auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] ")); }
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    // Stop the server — cached connection is now stale, new connections fail
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Trip the breaker via three consecutive failures
    for (int i = 0; i < 3; ++i) {
        auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
        ASSERT_TRUE(err) << "Attempt " << i << " should fail (server is down)";
    }

    // The breaker is open, and the pool still holds the lost entry.
    auto start = std::chrono::steady_clock::now();
    auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now() - start
    );

    ASSERT_TRUE(err);
    EXPECT_FALSE(conn);
    EXPECT_LT(elapsed.count(), 500)
        << "Breaker should reject before reconnecting, but took " << elapsed.count()
        << "ms";
    EXPECT_EQ(err.data, BREAKER_OPEN_TEXT);
}

/// @brief After the circuit breaker trips and the cooldown expires, a
/// successful probe connection resets the breaker and restores normal
/// operation.
///
/// On main: no breaker, so this test structure doesn't apply — but it
/// validates that the breaker doesn't permanently lock out an endpoint.
TEST_F(SessionLimitPoolTest, CircuitBreakerResetsOnSuccess) {
    Pool pool(
        testutil::create_store(),
        {.min_cooldown = std::chrono::milliseconds(200)}
    );

    // Stop server to trip the breaker
    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Trip the breaker: 3 consecutive failures
    for (int i = 0; i < 3; ++i) {
        auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
        ASSERT_TRUE(err);
    }

    // Verify breaker is tripped
    {
        auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
        ASSERT_TRUE(err);
        EXPECT_EQ(err.data, BREAKER_OPEN_TEXT);
    }

    // Restart server
    server_cfg_.max_sessions = 2;
    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    // Now acquire should succeed — breaker enters half-open, probe succeeds,
    // breaker resets
    auto conn = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_NE(conn.get(), nullptr);

    // Verify connection is functional
    UA_SessionState ss = UA_SESSIONSTATE_CLOSED;
    UA_SecureChannelState cs = UA_SECURECHANNELSTATE_CLOSED;
    UA_Client_getState(conn.get(), &cs, &ss, nullptr);
    EXPECT_EQ(ss, UA_SESSIONSTATE_ACTIVATED);

    // Subsequent acquires should also succeed (breaker fully reset)
    auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    ASSERT_NE(conn2.get(), nullptr);
}

/// @brief 4 threads race to reconnect after a server restart with
/// max_sessions=2. Serialized connection creation should prevent all 4
/// from calling connect() simultaneously and overwhelming the session table.
///
/// On main: all 4 threads call connect() concurrently, potentially
/// creating 4 sessions and exceeding the limit.
///
/// With the update: only one thread connects at a time (serialized via
/// connect_cv_). Other threads wait and reuse the newly created connection.
TEST_F(SessionLimitPoolTest, ConcurrentChurnExhaustsSessions) {
    Pool pool(testutil::create_store());

    // Prime with 2 cached connections
    {
        auto conn1 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
        auto conn2 = ASSERT_NIL_P(pool.acquire(conn_cfg_, "[test] "));
    }
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 2);

    // Restart server — cached connections become stale
    server_->stop();
    server_.reset();

    server_cfg_.max_sessions = 2;
    server_ = std::make_unique<testutil::Server>(server_cfg_);
    server_->start();
    ASSERT_TRUE(server_->wait_until_ready());

    // 4 threads race to acquire — serialization should keep us within
    // the session limit
    const int num_threads = 4;
    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};
    std::atomic<int> error_count{0};

    threads.reserve(num_threads);
    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&pool, &success_count, &error_count, this]() {
            auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
            if (!err && conn) {
                success_count++;
                // Hold connection briefly to simulate task usage
                std::this_thread::sleep_for(std::chrono::milliseconds(100));
            } else {
                error_count++;
            }
        });
    }

    for (auto &t: threads)
        t.join();

    // With serialization: threads connect one at a time and share
    // connections. At least 2 should succeed (the session limit).
    // Without serialization: all 4 race and some may get rejected.
    EXPECT_GE(success_count.load(), 2)
        << "At least 2 threads should succeed (session limit is 2)";
}

}
