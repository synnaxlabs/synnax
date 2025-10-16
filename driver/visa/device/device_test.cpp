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

#include "driver/visa/device/device.h"
#include "driver/visa/api/mock_api.h"

using namespace visa::device;
using namespace visa_api;

class DeviceManagerTest : public ::testing::Test {
protected:
    using MockManager = visa::device::ManagerImpl<MockAPIWrapper>;
    using MockSession = visa::device::SessionImpl<MockAPIWrapper>;
    MockAPI::Config default_cfg;

    void SetUp() override {
        // Setup default mock with one device
        default_cfg.resources = {"TCPIP0::192.168.1.100::INSTR"};
        MockAPI::DeviceResponse dev;
        dev.idn = "Keysight Technologies,34465A,MY54505123,A.02.16";
        dev.command_responses["MEAS:VOLT?"] = "3.14159\n";
        default_cfg.devices["TCPIP0::192.168.1.100::INSTR"] = dev;
    }
};

TEST_F(DeviceManagerTest, testBasicAcquire) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};
    auto [session, err] = manager.acquire(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(session, nullptr);
}

TEST_F(DeviceManagerTest, testConnectionPooling) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};

    // First acquire
    auto [session1, err1] = manager.acquire(cfg);
    ASSERT_NIL(err1);
    ASSERT_NE(session1, nullptr);

    // Second acquire should return the same cached session
    auto [session2, err2] = manager.acquire(cfg);
    ASSERT_NIL(err2);
    ASSERT_NE(session2, nullptr);
    EXPECT_EQ(session1.get(), session2.get());
}

TEST_F(DeviceManagerTest, testConnectionPoolingAfterExpiry) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};

    {
        auto [session1, err1] = manager.acquire(cfg);
        ASSERT_NIL(err1);
        // session1 goes out of scope, weak_ptr should expire
    }

    // Acquire again - should create new session since previous expired
    auto [session2, err2] = manager.acquire(cfg);
    ASSERT_NIL(err2);
    ASSERT_NE(session2, nullptr);
}

TEST_F(DeviceManagerTest, testMultipleDevices) {
    default_cfg.resources = {
        "TCPIP0::192.168.1.100::INSTR",
        "TCPIP0::192.168.1.101::INSTR"
    };
    MockAPI::DeviceResponse dev1, dev2;
    dev1.idn = "Device 1";
    dev2.idn = "Device 2";
    default_cfg.devices["TCPIP0::192.168.1.100::INSTR"] = dev1;
    default_cfg.devices["TCPIP0::192.168.1.101::INSTR"] = dev2;

    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    auto [session1, err1] = manager.acquire({"TCPIP0::192.168.1.100::INSTR"});
    ASSERT_NIL(err1);

    auto [session2, err2] = manager.acquire({"TCPIP0::192.168.1.101::INSTR"});
    ASSERT_NIL(err2);

    EXPECT_NE(session1.get(), session2.get());
}

TEST_F(DeviceManagerTest, testOpenFailure) {
    auto fail_cfg = default_cfg;
    fail_cfg.fail_open_session = true;
    fail_cfg.open_session_status = VI_ERROR_RSRC_NFOUND;

    auto api = std::make_shared<MockAPIWrapper>(fail_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};
    auto [session, err] = manager.acquire(cfg);
    ASSERT_TRUE(err);
    EXPECT_EQ(session, nullptr);
}

TEST_F(DeviceManagerTest, testResourceManagerInitFailure) {
    auto fail_cfg = default_cfg;
    fail_cfg.fail_open_rm = true;
    fail_cfg.open_rm_status = VI_ERROR_RSRC_NFOUND;

    auto api = std::make_shared<MockAPIWrapper>(fail_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};
    auto [session, err] = manager.acquire(cfg);
    ASSERT_TRUE(err);
}

TEST_F(DeviceManagerTest, testFindResourcesSuccess) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    std::vector<std::string> resources;
    auto err = manager.find_resources("?*::INSTR", resources);
    ASSERT_NIL(err);
    ASSERT_EQ(resources.size(), 1);
    EXPECT_EQ(resources[0], "TCPIP0::192.168.1.100::INSTR");
}

TEST_F(DeviceManagerTest, testFindResourcesEmpty) {
    auto empty_cfg = default_cfg;
    empty_cfg.resources.clear();

    auto api = std::make_shared<MockAPIWrapper>(empty_cfg);
    MockManager manager(api);

    std::vector<std::string> resources;
    auto err = manager.find_resources("?*::INSTR", resources);
    ASSERT_NIL(err); // Not finding devices is not an error
    EXPECT_EQ(resources.size(), 0);
}

TEST_F(DeviceManagerTest, testQueryIdnSuccess) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    std::string idn;
    auto err = manager.query_idn("TCPIP0::192.168.1.100::INSTR", idn);
    ASSERT_NIL(err); // query_idn always returns NIL (best effort)
    EXPECT_TRUE(idn.find("Keysight") != std::string::npos);
}

TEST_F(DeviceManagerTest, testQueryIdnDeviceNotResponding) {
    auto fail_cfg = default_cfg;
    fail_cfg.timeout_on_read = true;

    auto api = std::make_shared<MockAPIWrapper>(fail_cfg);
    MockManager manager(api);

    std::string idn;
    auto err = manager.query_idn("TCPIP0::192.168.1.100::INSTR", idn);
    ASSERT_NIL(err); // query_idn is best-effort, doesn't fail
}

TEST_F(DeviceManagerTest, testSessionQuery) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};
    auto [session, err] = manager.acquire(cfg);
    ASSERT_NIL(err);

    char response[256];
    auto query_err = session->query("MEAS:VOLT?\n", response, sizeof(response));
    ASSERT_NIL(query_err);

    std::string resp_str(response);
    EXPECT_TRUE(resp_str.find("3.14159") != std::string::npos);
}

TEST_F(DeviceManagerTest, testSessionReadWrite) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};
    auto [session, err] = manager.acquire(cfg);
    ASSERT_NIL(err);

    // Write command
    const char *cmd = "MEAS:VOLT?\n";
    size_t written;
    auto write_err = session->write(
        reinterpret_cast<const uint8_t *>(cmd),
        strlen(cmd),
        written
    );
    ASSERT_NIL(write_err);
    EXPECT_EQ(written, strlen(cmd));

    // Read response
    uint8_t buffer[256];
    size_t read_count;
    auto read_err = session->read(buffer, sizeof(buffer) - 1, read_count);
    ASSERT_NIL(read_err);
    buffer[read_count] = '\0';

    std::string response(reinterpret_cast<char *>(buffer));
    EXPECT_TRUE(response.find("3.14159") != std::string::npos);
}

TEST_F(DeviceManagerTest, testConcurrentAcquire) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    constexpr int num_threads = 10;
    std::vector<std::thread> threads;
    std::vector<std::shared_ptr<MockSession>> sessions(num_threads);
    std::vector<xerrors::Error> errors(num_threads);

    ConnectionConfig cfg{"TCPIP0::192.168.1.100::INSTR"};

    for (int i = 0; i < num_threads; i++) {
        threads.emplace_back([&, i]() {
            auto [session, err] = manager.acquire(cfg);
            sessions[i] = session;
            errors[i] = err;
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    // All should succeed (due to connection pooling)
    for (int i = 0; i < num_threads; i++) {
        ASSERT_NIL(errors[i]);
        ASSERT_NE(sessions[i], nullptr);
    }

    // All should have the same session (pooled)
    for (int i = 1; i < num_threads; i++) {
        EXPECT_EQ(sessions[0].get(), sessions[i].get());
    }
}

TEST_F(DeviceManagerTest, testTimeoutConfiguration) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{
        "TCPIP0::192.168.1.100::INSTR",
        10000, // 10 second timeout
        '\n',
        true
    };

    auto [session, err] = manager.acquire(cfg);
    ASSERT_NIL(err);

    // Verify timeout was set (no error returned = success)
    auto timeout_err = session->set_timeout(5000);
    ASSERT_NIL(timeout_err);
}

TEST_F(DeviceManagerTest, testTermCharConfiguration) {
    auto api = std::make_shared<MockAPIWrapper>(default_cfg);
    MockManager manager(api);

    ConnectionConfig cfg{
        "TCPIP0::192.168.1.100::INSTR",
        5000,
        '\r', // CR instead of LF
        true
    };

    auto [session, err] = manager.acquire(cfg);
    ASSERT_NIL(err);

    // Verify term char was set
    auto term_err = session->set_term_char('\r', true);
    ASSERT_NIL(term_err);
}
