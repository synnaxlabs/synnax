// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "x/cpp/test/test.h"

#include "driver/ethercat/mock/pool.h"

namespace driver::ethercat::mock {

TEST(PoolConfiguration, ConfigureMasterAddsToEnumerate) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    auto infos = pool.enumerate();
    ASSERT_EQ(infos.size(), 1);
    EXPECT_EQ(infos[0].key, "eth0");
}

TEST(PoolConfiguration, ConfigureMasterAllowsAcquire) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    auto [engine, err] = pool.acquire("eth0");
    ASSERT_NIL(err);
    EXPECT_NE(engine, nullptr);
}

TEST(PoolAcquire, CreatesEngine) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    EXPECT_EQ(pool.get_engine("eth0"), nullptr);
    auto [engine, err] = pool.acquire("eth0");
    ASSERT_NIL(err);
    EXPECT_NE(pool.get_engine("eth0"), nullptr);
}

TEST(PoolAcquire, ReturnsSameEngine) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    auto [engine1, err1] = pool.acquire("eth0");
    ASSERT_NIL(err1);
    auto [engine2, err2] = pool.acquire("eth0");
    ASSERT_NIL(err2);
    EXPECT_EQ(engine1, engine2);
}

TEST(PoolAcquire, ReturnsErrorForUnconfigured) {
    Pool pool;
    ASSERT_OCCURRED_AS_P(pool.acquire("unknown"), errors::MASTER_INIT_ERROR);
}

TEST(PoolAcquire, WithInjectedError) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);
    pool.inject_acquire_error(x::errors::Error(errors::MASTER_INIT_ERROR, "injected"));

    ASSERT_OCCURRED_AS_P(pool.acquire("eth0"), errors::MASTER_INIT_ERROR);
}

TEST(PoolIsActive, ReturnsFalseInitially) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    EXPECT_FALSE(pool.is_active("eth0"));
}

TEST(PoolIsActive, ReturnsFalseForUnconfigured) {
    Pool pool;
    EXPECT_FALSE(pool.is_active("unknown"));
}

TEST(PoolGetSlaves, ReturnsEmptyForUnconfigured) {
    Pool pool;
    auto slaves = pool.get_slaves("unknown");
    EXPECT_TRUE(slaves.empty());
}

TEST(PoolGetSlaves, ReturnsMasterSlaves) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    master->add_slave(
        slave::Properties{.position = 0, .vendor_id = 0x100, .name = "Slave0"}
    );
    master->add_slave(
        slave::Properties{.position = 1, .vendor_id = 0x101, .name = "Slave1"}
    );
    pool.configure_master("eth0", master);

    auto slaves = pool.get_slaves("eth0");
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[0].properties.vendor_id, 0x100u);
    EXPECT_EQ(slaves[1].properties.position, 1);
    EXPECT_EQ(slaves[1].properties.vendor_id, 0x101u);
}

TEST(PoolGetMaster, ReturnsConfiguredMaster) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    EXPECT_EQ(pool.get_master("eth0"), master);
}

TEST(PoolGetMaster, ReturnsNullForUnconfigured) {
    Pool pool;
    EXPECT_EQ(pool.get_master("unknown"), nullptr);
}

TEST(PoolGetEngine, ReturnsNullBeforeAcquire) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    EXPECT_EQ(pool.get_engine("eth0"), nullptr);
}

TEST(PoolGetEngine, ReturnsEngineAfterAcquire) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);

    auto [engine, err] = pool.acquire("eth0");
    ASSERT_NIL(err);
    EXPECT_EQ(pool.get_engine("eth0"), engine);
}

TEST(PoolErrorInjection, ClearInjectedErrorsResetsAcquireError) {
    Pool pool;
    auto master = std::make_shared<Master>("eth0");
    pool.configure_master("eth0", master);
    pool.inject_acquire_error(x::errors::Error(errors::MASTER_INIT_ERROR, "injected"));
    pool.clear_injected_errors();

    auto [engine, err] = pool.acquire("eth0");
    ASSERT_NIL(err);
    EXPECT_NE(engine, nullptr);
}

TEST(PoolMultipleMasters, ConfigureAndAcquireMultiple) {
    Pool pool;
    auto master1 = std::make_shared<Master>("eth0");
    auto master2 = std::make_shared<Master>("eth1");
    pool.configure_master("eth0", master1);
    pool.configure_master("eth1", master2);

    auto [engine1, err1] = pool.acquire("eth0");
    ASSERT_NIL(err1);
    auto [engine2, err2] = pool.acquire("eth1");
    ASSERT_NIL(err2);

    EXPECT_NE(engine1, engine2);
    EXPECT_EQ(pool.get_engine("eth0"), engine1);
    EXPECT_EQ(pool.get_engine("eth1"), engine2);
}

}
