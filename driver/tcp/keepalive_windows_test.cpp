// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "gtest/gtest.h"
#include <winsock2.h>
#include <ws2tcpip.h>

#include "driver/tcp/keepalive.h"

namespace driver::tcp::keepalive {
namespace {
DWORD option(const SOCKET socket, const int level, const int name) {
    DWORD value = 0;
    int size = sizeof(value);
    EXPECT_EQ(
        ::getsockopt(socket, level, name, reinterpret_cast<char *>(&value), &size),
        0
    );
    return value;
}
}

TEST(Keepalive, SetsTheProbesOnALoopbackConnection) {
    asio::io_context ctx;
    asio::ip::tcp::acceptor acceptor(
        ctx,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    );
    asio::ip::tcp::socket socket(ctx);
    socket.connect(acceptor.local_endpoint());
    ASSERT_FALSE(enable(socket.native_handle()));
    const auto s = socket.native_handle();
    EXPECT_NE(option(s, SOL_SOCKET, SO_KEEPALIVE), 0);
    EXPECT_EQ(option(s, IPPROTO_TCP, TCP_KEEPIDLE), 10);
    EXPECT_EQ(option(s, IPPROTO_TCP, TCP_KEEPINTVL), 5);
    EXPECT_EQ(option(s, IPPROTO_TCP, TCP_KEEPCNT), 3);
}

TEST(Keepalive, ReportsAnOptionTheSocketRejects) {
    EXPECT_TRUE(enable(INVALID_SOCKET));
}
}
