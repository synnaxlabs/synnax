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

/// @brief it should treat an abort about the value as a rejection.
TEST(StartupParameterErrorTest, RejectsAValueAbort) {
    EXPECT_EQ(
        startup_parameter_error(0x06090031).type,
        STARTUP_PARAMETER_REJECTED.type
    );
    EXPECT_EQ(
        startup_parameter_error(0x06020000).type,
        STARTUP_PARAMETER_REJECTED.type
    );
}

/// @brief it should treat no response and a temporary abort as a retryable error.
TEST(StartupParameterErrorTest, RetriesNoResponseAndTemporaryAborts) {
    EXPECT_EQ(startup_parameter_error(0).type, STARTUP_PARAMETER_ERROR.type);
    EXPECT_EQ(startup_parameter_error(0x05040000).type, STARTUP_PARAMETER_ERROR.type);
    EXPECT_EQ(startup_parameter_error(0x08000022).type, STARTUP_PARAMETER_ERROR.type);
}
}
