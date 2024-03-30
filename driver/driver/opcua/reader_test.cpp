/// external.
#include <gtest/gtest.h>
#include "nlohmann/json.hpp"

/// synnax internal.
#include "driver/driver/testutil/testutil.h"
#include "driver/driver/opcua/mock_server.h"

/// client internal.
#include "reader.h"
#include "mock_server.h"


using json = nlohmann::json;

class Reader {
};

TEST(OPCUAReaderTest, testReaderConfigurationFromJSON) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [idx, idx_err] = client->channels.create(
        "index1",
        TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(idx_err) << idx_err.message();

    auto [ch1, ch1_err] = client->channels.create(
        "node1",
        INT32,
        idx.key,
        false
    );
    ASSERT_FALSE(ch1_err) << ch1_err.message();

    auto [ch2, ch2_err] = client->channels.create(
        "node2",
        INT32,
        idx.key,
        false
    );
    ASSERT_FALSE(ch2_err) << ch2_err.message();

    auto j = json{
        {
            "connection", json{
                {"endpoint", "opc.tcp://0.0.0.0:4840"},
            }
        },
        {"rate", 22.5},
        {
            "channels", std::vector<json>{
                {{"ns", 1}, {"node", "node1"}, {"key", ch1.key}},
                {{"ns", 1}, {"node", "node2"}, {"key", ch2.key}}
            }
        }
    };

    auto t = synnax::Task(
        "my_task",
        "opcuaRead",
        to_string(j)
    );


    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto mock = MockServerConfig{
        {
            {1, "node1"},
            {1, "node2"}
        }
    };

    auto server = MockServer(mock);
    server.start();
    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    auto reader = opcua::Reader(mockCtx, t);
    ASSERT_EQ(mockCtx->states.size(), 0) << to_string(mockCtx->states[0].details);
    std::this_thread::sleep_for(std::chrono::seconds(30));
    reader.stop();
    server.stop();
}
