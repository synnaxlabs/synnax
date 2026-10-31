// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <cstdarg>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"
#include "open62541/client.h"
#include "open62541/client_config_default.h"
#include "open62541/plugin/accesscontrol_default.h"
#include "open62541/server.h"
#include "open62541/server_config_default.h"

namespace {
struct Log {
    std::mutex mu;
    std::vector<std::string> messages;

    bool contains(const std::string &text) {
        std::lock_guard lock(this->mu);
        for (const auto &m: this->messages)
            if (m.contains(text)) return true;
        return false;
    }
};

void record(void *ctx, UA_LogLevel, UA_LogCategory, const char *msg, va_list args) {
    char buffer[1024] = {};
    UA_String out{
        .length = sizeof(buffer),
        .data = reinterpret_cast<UA_Byte *>(buffer)
    };
    UA_String_vformat(&out, msg, args);
    auto *log = static_cast<Log *>(ctx);
    std::lock_guard lock(log->mu);
    log->messages.emplace_back(buffer);
}
}

/// @brief it should close the Session on the server when the server rejects the
/// login.
TEST(PatchTest, ClosesRejectedSession) {
    Log log;
    UA_Server *server = UA_Server_new();
    UA_ServerConfig *server_cfg = UA_Server_getConfig(server);
    ASSERT_EQ(
        UA_ServerConfig_setMinimal(server_cfg, 4880, nullptr),
        UA_STATUSCODE_GOOD
    );
    server_cfg->logging->log = record;
    server_cfg->logging->context = &log;
    server_cfg->allowNonePolicyPassword = true;
    const UA_UsernamePasswordLogin login{
        UA_STRING_STATIC("user"),
        UA_STRING_STATIC("password")
    };
    server_cfg->accessControl.clear(&server_cfg->accessControl);
    const UA_String policy = server_cfg->securityPolicies[0].policyUri;
    ASSERT_EQ(
        UA_AccessControl_default(server_cfg, false, &policy, 1, &login),
        UA_STATUSCODE_GOOD
    );
    ASSERT_EQ(UA_Server_run_startup(server), UA_STATUSCODE_GOOD);
    std::atomic running = true;
    std::thread loop([&] {
        while (running)
            UA_Server_run_iterate(server, true);
    });

    UA_Client *client = UA_Client_new();
    UA_ClientConfig *client_cfg = UA_Client_getConfig(client);
    client_cfg->allowNonePolicyPassword = true;
    UA_ClientConfig_setAuthenticationUsername(client_cfg, "user", "wrong");
    EXPECT_NE(
        UA_Client_connect(client, "opc.tcp://localhost:4880"),
        UA_STATUSCODE_GOOD
    );
    UA_Client_delete(client);

    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(5);
    while (!log.contains("Closing the Session") &&
           std::chrono::steady_clock::now() < deadline)
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    EXPECT_TRUE(log.contains("Closing the Session"));

    running = false;
    loop.join();
    UA_Server_run_shutdown(server);
    UA_Server_delete(server);
}
