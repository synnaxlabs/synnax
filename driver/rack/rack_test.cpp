// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include "absl/crc/crc32c.h"
#include "gtest/gtest.h"

#include "client/cpp/auth/auth.h"
#include "client/cpp/testutil/pipe.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/kv/kv.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/journal/journal.h"
#include "driver/outbox/outbox.h"
#include "driver/rack/rack.h"

namespace driver::rack {
const std::string RACK_STATE_PATH = "/tmp/rack-test/state.json";

/// @brief writes a segment file start at path with format version 2, which this
/// Driver cannot read.
void write_future_segment(const std::filesystem::path &path) {
    std::string header(outbox::SEGMENT_MAGIC.data(), outbox::SEGMENT_MAGIC.size());
    header += std::string("\x02\x00", 2);
    header += std::string(journal::HEADER_SIZE - 14, '\0');
    const auto crc = static_cast<std::uint32_t>(absl::ComputeCrc32c(header));
    for (int i = 0; i < 4; i++)
        header.push_back(static_cast<char>(crc >> (8 * i)));
    std::ofstream(path, std::ios::binary) << header;
}

/// @brief returns arguments that point the rack at a fresh state file.
x::args::Parser fresh_args(std::vector<std::string> extra) {
    std::filesystem::remove(RACK_STATE_PATH);
    std::vector<std::string> argv{"program", "--state-file", RACK_STATE_PATH};
    argv.insert(argv.end(), extra.begin(), extra.end());
    return x::args::Parser(argv);
}

/// @brief it should stop without an error while it waits to reach the Core again.
TEST(RackTest, StopDuringReconnectReportsNoError) {
    auto args = fresh_args({"--port", "1"});
    std::atomic shut_down = false;
    Rack r;
    r.start(args, [&] { shut_down = true; });
    // The rack creates its state file just before it first tries to reach the Core.
    ASSERT_EVENTUALLY_TRUE(std::filesystem::exists(RACK_STATE_PATH));
    ASSERT_NIL(r.stop());
    ASSERT_FALSE(shut_down);
}

/// @brief it should call on_shutdown and return the error that stopped it.
TEST(RackTest, FatalErrorStopsRack) {
    const auto conn = test_client_config();
    auto args = fresh_args({
        "--host",
        conn.host,
        "--port",
        std::to_string(conn.port),
        "--username",
        "rack-test-nobody",
        "--password",
        "wrong",
    });
    std::atomic shut_down = false;
    Rack r;
    r.start(args, [&] { shut_down = true; });
    ASSERT_EVENTUALLY_TRUE(shut_down.load());
    ASSERT_OCCURRED_AS(r.stop(), synnax::auth::ERR_INVALID_CREDENTIALS);
}

/// @brief it should keep waiting for an unreachable Core past its retry limit after its
/// first run.
TEST(RackTest, WaitsForAnUnreachableCorePastTheRetryLimit) {
    const auto conn = test_client_config();
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("outage rack"));
    const auto cluster_key = client.connectivity->check().cluster_key;
    CorePipe pipe(conn.port);
    auto args = fresh_args({
        "--host",
        "127.0.0.1",
        "--port",
        std::to_string(pipe.port),
        "--username",
        conn.username,
        "--password",
        conn.password,
    });
    const auto kv = ASSERT_NIL_P(
        x::kv::JSONFile::open_file({
            .path = RACK_STATE_PATH,
            .dir_mode = std::filesystem::perms::owner_all,
            .file_mode = std::filesystem::perms::owner_read |
                         std::filesystem::perms::owner_write,
        })
    );
    ASSERT_NIL(kv->set(
        "remote_info",
        x::json::json{{"rack_key", rack.key}, {"cluster_key", cluster_key}}.dump()
    ));
    std::atomic shut_down = false;
    Rack r({
        .name = "driver",
        .base_interval = x::telem::MILLISECOND,
        .max_retries = 2,
        .scale = 1,
        .max_interval = x::telem::MILLISECOND,
    });
    r.start(args, [&] { shut_down = true; });
    const auto running = [&] {
        const synnax::rack::RetrieveOptions opts{.include_status = true};
        const auto [remote, err] = client.racks.retrieve(rack.key, opts);
        return !err && remote.status.has_value() && remote.status->details.running;
    };
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        running(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    pipe.pause();
    std::this_thread::sleep_for(std::chrono::seconds(1));
    ASSERT_FALSE(shut_down);
    pipe.resume();
    ASSERT_NIL(r.stop());
}

/// @brief returns arguments that log the rack in to the test Core.
x::args::Parser core_args() {
    const auto conn = test_client_config();
    return fresh_args({
        "--host",
        conn.host,
        "--port",
        std::to_string(conn.port),
        "--username",
        conn.username,
        "--password",
        conn.password,
    });
}

/// @brief returns the outbox directory of the rack under test.
std::filesystem::path outbox_dir() {
    return std::filesystem::path(RACK_STATE_PATH).parent_path() / DATA_DIR / OUTBOX_DIR;
}

/// @brief it should open the outbox in the data directory.
TEST(RackTest, OpensOutbox) {
    std::filesystem::remove_all(outbox_dir());
    auto args = core_args();
    Rack r;
    r.start(args);
    ASSERT_EVENTUALLY_TRUE(std::filesystem::exists(outbox_dir()));
    ASSERT_NIL(r.stop());
}

/// @brief it should keep running without the outbox when it cannot open, report why in
/// the rack status, and leave the outbox files in place.
TEST(RackTest, RunsWithoutOutbox) {
    const auto dir = outbox_dir();
    std::filesystem::remove_all(dir);
    std::filesystem::create_directories(dir);
    write_future_segment(dir / "0.calls");
    x::defer::defer cleanup([&] { std::filesystem::remove_all(dir); });
    const auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("outbox rack"));
    auto args = core_args();
    const auto kv = ASSERT_NIL_P(
        x::kv::JSONFile::open_file({
            .path = RACK_STATE_PATH,
            .dir_mode = std::filesystem::perms::owner_all,
            .file_mode = std::filesystem::perms::owner_read |
                         std::filesystem::perms::owner_write,
        })
    );
    ASSERT_NIL(kv->set(
        "remote_info",
        x::json::json{
            {"rack_key", rack.key},
            {"cluster_key", client.connectivity->check().cluster_key}
        }.dump()
    ));
    std::atomic shut_down = false;
    Rack r;
    r.start(args, [&] { shut_down = true; });
    const auto reported = [&] {
        const synnax::rack::RetrieveOptions opts{.include_status = true};
        const auto [remote, err] = client.racks.retrieve(rack.key, opts);
        return !err && remote.status.has_value() &&
               remote.status->message == OUTBOX_FAILURE_MSG &&
               remote.status->variant == synnax::status::VARIANT_ERROR &&
               remote.status->description.contains("Update the Driver") &&
               remote.status->details.running;
    };
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        reported(),
        std::chrono::seconds(30),
        std::chrono::milliseconds(100)
    );
    ASSERT_FALSE(shut_down);
    ASSERT_NIL(r.stop());
    ASSERT_EQ(std::filesystem::file_size(dir / "0.calls"), journal::HEADER_SIZE);
}
}
