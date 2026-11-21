// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/payload/proto.gen.h"
#include "x/cpp/testutil/testutil.h"

namespace x::payload {
/// @brief it should reject a protobuf time format with no value.
TEST(TimeFormat, RejectsOutOfRangeFromProto) {
    ASSERT_OCCURRED_AS_P(
        time_format_from_pb(static_cast<pb::TimeFormat>(99)),
        x::errors::Error("unrecognized TimeFormat protobuf value")
    );
}

/// @brief it should reject an unknown kind.
TEST(Kind, RejectsUnknownToProto) {
    ASSERT_OCCURRED_AS_P(
        kind_to_pb("complex"),
        x::errors::Error("unrecognized Kind value: complex")
    );
}
}
