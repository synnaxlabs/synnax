// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstdint>
#include <span>
#include <string>
#include <thread>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/contract.h"
#include "driver/transport/errors.h"

namespace driver::bus::testutil {
namespace {
std::span<const std::uint8_t> bytes(const std::string &s) {
    return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}

/// @brief reads from the transport until n bytes arrive or a second passes.
/// @returns the bytes in the order they arrived, and the first error.
std::pair<std::string, x::errors::Error>
read(Transport &transport, const std::size_t n) {
    std::string got;
    const auto deadline = x::telem::TimeStamp::now() + x::telem::SECOND;
    while (got.size() < n && x::telem::TimeStamp::now() < deadline) {
        auto [chunk, err] = transport.read(20 * x::telem::MILLISECOND);
        if (err) return {got, err};
        got.append(
            reinterpret_cast<const char *>(chunk.data.data()),
            chunk.data.size()
        );
    }
    return {got, x::errors::NIL};
}
}

GTEST_ALLOW_UNINSTANTIATED_PARAMETERIZED_TEST(StreamContract);

TEST_P(TransportContract, ReturnsAnEmptyChunkWhenNothingArrivesBeforeTheTimeout) {
    const auto transport = ASSERT_NIL_P(this->rig->opener()());
    const x::telem::Stopwatch sw;
    const auto chunk = ASSERT_NIL_P(transport->read(50 * x::telem::MILLISECOND));
    EXPECT_TRUE(chunk.data.empty());
    EXPECT_GE(sw.elapsed(), 50 * x::telem::MILLISECOND);
}

TEST_P(TransportContract, DeliversBytesInTheOrderTheyWereWritten) {
    const auto transport = ASSERT_NIL_P(this->rig->opener()());
    ASSERT_NIL(transport->write(bytes("first,"), x::telem::SECOND));
    ASSERT_NIL(transport->write(bytes("second"), x::telem::SECOND));
    EXPECT_EQ(this->rig->receive(12), "first,second");
    this->rig->send("third,");
    this->rig->send("fourth");
    const auto [got, err] = read(*transport, 12);
    ASSERT_NIL(err);
    EXPECT_EQ(got, "third,fourth");
}

TEST_P(TransportContract, ReportsAnAbsentDeviceAsUnreachable) {
    ASSERT_OCCURRED_AS_P(this->rig->absent()(), transport::UNREACHABLE_ERROR);
}

TEST_P(TransportContract, OpensAgainOnceTheDeviceReturns) {
    const auto opener = this->rig->opener();
    {
        const auto first = ASSERT_NIL_P(opener());
        this->rig->hang_up();
    }
    this->rig->restore();
    const auto second = ASSERT_NIL_P(opener());
    ASSERT_NIL(second->write(bytes("back?"), x::telem::SECOND));
    EXPECT_EQ(this->rig->receive(5), "back?");
    this->rig->send("back");
    const auto [got, err] = read(*second, 4);
    ASSERT_NIL(err);
    EXPECT_EQ(got, "back");
}

TEST_P(StreamContract, ReportsUnreachableOnReadOnceThePeerCloses) {
    const auto transport = ASSERT_NIL_P(this->rig->opener()());
    this->rig->hang_up();
    const auto [got, err] = read(*transport, 1);
    EXPECT_TRUE(got.empty());
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
}

TEST_P(StreamContract, ReportsUnreachableOnWriteOnceThePeerCloses) {
    const auto transport = ASSERT_NIL_P(this->rig->opener()());
    this->rig->hang_up();
    // TCP takes the first write after the peer closes and learns of the close from the
    // reset that the peer sends back.
    auto err = x::errors::NIL;
    for (int i = 0; i < 50 && !err; i++) {
        err = transport->write(bytes("ping"), x::telem::SECOND);
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    ASSERT_OCCURRED_AS(err, transport::UNREACHABLE_ERROR);
}
}
