// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/opc/mock/server.h"
#include "driver/opc/util/conn_pool.h"

using namespace util;

class ConnectionPoolTest : public ::testing::Test {
protected:
    void SetUp() override {
        server_cfg_ = mock::ServerConfig::create_default();
        server_cfg_.port = 4845;
        server_ = std::make_unique<mock::Server>(server_cfg_);
        server_->start();
        std::this_thread::sleep_for(std::chrono::milliseconds(500));

        conn_cfg_.endpoint = "opc.tcp://localhost:4845";
        conn_cfg_.security_mode = "None";
        conn_cfg_.security_policy = "None";
    }

    void TearDown() override {
        if (server_) { server_->stop(); }
    }

    mock::ServerConfig server_cfg_;
    std::unique_ptr<mock::Server> server_;
    ConnectionConfig conn_cfg_;
};

TEST_F(ConnectionPoolTest, AcquireNewConnection) {
    ConnectionPool pool;

    auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err) << err.message();
    ASSERT_TRUE(conn);
    ASSERT_NE(conn.get(), nullptr);

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

TEST_F(ConnectionPoolTest, ReuseConnection) {
    ConnectionPool pool;

    {
        auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
        ASSERT_FALSE(err1);
        ASSERT_TRUE(conn1);
    }

    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err2);
    ASSERT_TRUE(conn2);

    EXPECT_EQ(pool.size(), 1);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);
}

TEST_F(ConnectionPoolTest, MultipleSimultaneousConnections) {
    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);

    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err2);

    EXPECT_EQ(pool.size(), 2);
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 0);

    EXPECT_NE(conn1.get(), conn2.get());
}

TEST_F(ConnectionPoolTest, DifferentEndpoints) {
    mock::ServerConfig server2_cfg = mock::ServerConfig::create_default();
    server2_cfg.port = 4846;
    mock::Server server2(server2_cfg);
    server2.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    ConnectionConfig cfg2 = conn_cfg_;
    cfg2.endpoint = "opc.tcp://localhost:4846";

    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);

    auto [conn2, err2] = pool.acquire(cfg2, "[test] ");
    ASSERT_FALSE(err2);

    EXPECT_EQ(pool.size(), 2);
    EXPECT_NE(conn1.get(), conn2.get());

    server2.stop();
}

TEST_F(ConnectionPoolTest, MoveSemantics) {
    ConnectionPool pool;

    auto [conn1, err] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err);

    auto *original_ptr = conn1.get();

    ConnectionPool::Connection conn2 = std::move(conn1);
    EXPECT_EQ(conn2.get(), original_ptr);
    EXPECT_FALSE(conn1);

    ConnectionPool::Connection conn3(std::move(conn2));
    EXPECT_EQ(conn3.get(), original_ptr);
    EXPECT_FALSE(conn2);
}

TEST_F(ConnectionPoolTest, ThreadSafety) {
    ConnectionPool pool;
    const int num_threads = 10;
    const int acquisitions_per_thread = 5;

    std::vector<std::thread> threads;
    std::atomic<int> success_count{0};

    for (int i = 0; i < num_threads; ++i) {
        threads.emplace_back([&pool, &success_count, this]() {
            for (int j = 0; j < acquisitions_per_thread; ++j) {
                auto [conn, err] = pool.acquire(conn_cfg_, "[test] ");
                if (!err && conn) {
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

TEST_F(ConnectionPoolTest, ConnectionInvalidation) {
    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);
    auto client_ptr = conn1.shared();

    { ConnectionPool::Connection temp = std::move(conn1); }

    UA_Client_disconnect(client_ptr.get());

    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err2);

    EXPECT_NE(conn2.get(), client_ptr.get());
    EXPECT_EQ(pool.size(), 1);
}

TEST_F(ConnectionPoolTest, DifferentCredentials) {
    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);

    ConnectionConfig cfg_with_user = conn_cfg_;
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

TEST_F(ConnectionPoolTest, AcquireFromBadServer) {
    ConnectionPool pool;
    ConnectionConfig bad_cfg = conn_cfg_;
    bad_cfg.endpoint = "opc.tcp://localhost:9999";

    auto [conn, err] = pool.acquire(bad_cfg, "[test] ");
    ASSERT_TRUE(err);
    EXPECT_EQ(pool.size(), 0);
}

TEST_F(ConnectionPoolTest, StaleConnectionAutoReconnect) {
    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);

    conn1 = ConnectionPool::Connection(nullptr, nullptr, "");
    EXPECT_EQ(pool.available_count(conn_cfg_.endpoint), 1);

    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err2);
    ASSERT_TRUE(conn2);
}

TEST_F(ConnectionPoolTest, NewConnectionAfterServerRestart) {
    ConnectionPool pool;

    auto [conn1, err1] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err1);

    conn1 = ConnectionPool::Connection(nullptr, nullptr, "");

    server_->stop();
    server_.reset();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    server_ = std::make_unique<mock::Server>(server_cfg_);
    server_->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    auto [conn2, err2] = pool.acquire(conn_cfg_, "[test] ");
    ASSERT_FALSE(err2);
    ASSERT_TRUE(conn2);
}
