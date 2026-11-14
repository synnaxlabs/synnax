// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::errors {
/// @brief it should give the CoE meaning of a known abort code.
TEST(SDOAbortReasonTest, DescribesAKnownCode) {
    EXPECT_EQ(
        sdo_abort_reason(0x06020000),
        "Object does not exist in the object dictionary"
    );
}

/// @brief it should report no response for an abort code of 0.
TEST(SDOAbortReasonTest, ReportsNoResponseForZero) {
    EXPECT_EQ(sdo_abort_reason(0), "no response");
}

/// @brief it should give the hexadecimal value of an abort code it does not know.
TEST(SDOAbortReasonTest, GivesTheValueOfAnUnknownCode) {
    EXPECT_EQ(sdo_abort_reason(0x0BADC0DE), "unknown abort code 0x0BADC0DE");
}

/// @brief it should give the reason of each kind of SDO failure.
TEST(SDOFailureTest, GivesTheReasonOfEachKind) {
    using Kind = SDOFailure::Kind;
    EXPECT_EQ(SDOFailure{}.reason(), "no response");
    EXPECT_EQ(
        (SDOFailure{.kind = Kind::ABORTED, .abort_code = 0x06090011}).reason(),
        "Subindex does not exist"
    );
    EXPECT_EQ(
        SDOFailure{.kind = Kind::UNEXPECTED_FRAME}.reason(),
        "the slave sent an unexpected reply"
    );
    EXPECT_EQ(
        SDOFailure{.kind = Kind::ANSWER_TOO_LARGE}.reason(),
        "the reply is larger than the read allows"
    );
}
}
