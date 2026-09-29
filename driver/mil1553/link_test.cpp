// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <chrono>
#include <future>
#include <memory>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/mil1553/link.h"
#include "driver/mil1553/simulated/backend.h"

namespace driver::mil1553 {
TEST(Link, RespondsWhileAReadWaits) {
    const auto backend = std::make_shared<simulated::Backend>();
    Link rt(
        backend,
        {
            .backend = synnax::mil1553::BACKEND_SIMULATED,
            .role = synnax::mil1553::ROLE_REMOTE_TERMINAL,
            .terminals = {5},
        }
    );
    ASSERT_NIL(rt.open());
    std::vector<Transfer> out(1);
    auto read = std::async(std::launch::async, [&] {
        return rt.read(out, x::telem::SECOND * 5);
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    const std::array<std::uint16_t, 1> response{0xBEEF};
    ASSERT_NIL(rt.respond(5, 1, response));
    const auto bc = ASSERT_NIL_P(backend->open({
        .backend = synnax::mil1553::BACKEND_SIMULATED,
        .role = synnax::mil1553::ROLE_BUS_CONTROLLER,
    }));
    const codec::mil1553::Command cmd{.rt = 5, .transmit = true, .subaddress = 1};
    ASSERT_NIL_P(bc->transact(cmd, {}));
    const auto batch = ASSERT_NIL_P(read.get());
    ASSERT_EQ(batch.count, 1);
    EXPECT_EQ(out[0].command, cmd);
    EXPECT_EQ(out[0].words[0], 0xBEEF);
}
}
