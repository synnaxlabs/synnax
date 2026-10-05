// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>

#include "gtest/gtest.h"
#include "modbus/modbus.h"

#include "x/cpp/test/test.h"

#include "driver/modbus/mock/slave.h"

namespace driver::modbus::mock {
namespace {
using Client = std::unique_ptr<modbus_t, decltype(&modbus_free)>;

Client connect(const int port) {
    Client client(modbus_new_tcp("127.0.0.1", port), modbus_free);
    EXPECT_NE(client, nullptr);
    EXPECT_EQ(modbus_connect(client.get()), 0) << modbus_strerror(errno);
    return client;
}

void expect_coil_set(const Client &client) {
    uint8_t bit = 0;
    EXPECT_EQ(modbus_read_bits(client.get(), 0, 1, &bit), 1) << modbus_strerror(errno);
    EXPECT_EQ(bit, 1);
}
}

TEST(Slave, ServesNewClientAfterClientDisconnects) {
    SlaveConfig config;
    config.port = 1560;
    config.coils[0] = 1;
    Slave slave(config);
    ASSERT_NIL(slave.start());

    auto first = connect(config.port);
    expect_coil_set(first);
    modbus_close(first.get());

    const auto second = connect(config.port);
    expect_coil_set(second);
    modbus_close(second.get());
}

TEST(Slave, RestartsOnTheSamePortAfterStop) {
    SlaveConfig config;
    config.port = 1561;
    config.coils[0] = 1;
    Slave slave(config);
    ASSERT_NIL(slave.start());
    slave.stop();
    ASSERT_NIL(slave.start());

    const auto client = connect(config.port);
    expect_coil_set(client);
    modbus_close(client.get());
}
}
