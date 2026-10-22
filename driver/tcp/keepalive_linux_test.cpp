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
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <sys/socket.h>

#include "driver/tcp/keepalive.h"

namespace driver::tcp::keepalive {
namespace {
int option(const int socket, const int level, const int name) {
    int value = -1;
    socklen_t size = sizeof(value);
    EXPECT_EQ(::getsockopt(socket, level, name, &value, &size), 0);
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
    const int fd = socket.native_handle();
    EXPECT_NE(option(fd, SOL_SOCKET, SO_KEEPALIVE), 0);
    EXPECT_EQ(option(fd, IPPROTO_TCP, TCP_KEEPIDLE), 10);
    EXPECT_EQ(option(fd, IPPROTO_TCP, TCP_KEEPINTVL), 5);
    EXPECT_EQ(option(fd, IPPROTO_TCP, TCP_KEEPCNT), 3);
}

TEST(Keepalive, ReportsAnOptionTheSocketRejects) {
    EXPECT_EQ(enable(-1), std::errc::bad_file_descriptor);
}
}
