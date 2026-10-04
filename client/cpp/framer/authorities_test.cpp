// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/framer/framer.h"
#include "x/cpp/test/test.h"

namespace synnax::framer {
/// @brief validate_authorities should return nil for empty authorities.
TEST(ValidateAuthorities, EmptyAuthorities) {
    ASSERT_NIL(validate_authorities({}, {}));
}

/// @brief validate_authorities should return nil for a single authority broadcast.
TEST(ValidateAuthorities, SingleAuthorityBroadcast) {
    ASSERT_NIL(validate_authorities({1, 2, 3}, {200}));
}

/// @brief validate_authorities should return nil when keys and authorities match.
TEST(ValidateAuthorities, MatchingSizes) {
    ASSERT_NIL(validate_authorities({1, 2, 3}, {100, 200, 255}));
}

/// @brief validate_authorities should return nil for single authority with no keys.
TEST(ValidateAuthorities, SingleAuthorityNoKeys) {
    ASSERT_NIL(validate_authorities({}, {200}));
}

/// @brief validate_authorities should reject mismatched keys and authorities sizes.
TEST(ValidateAuthorities, MismatchedSizes) {
    ASSERT_OCCURRED_AS(
        validate_authorities({1, 2, 3}, {100, 200}),
        x::errors::VALIDATION
    );
}

/// @brief validate_authorities should reject multiple authorities when keys is empty.
TEST(ValidateAuthorities, MultipleAuthoritiesNoKeys) {
    ASSERT_OCCURRED_AS(validate_authorities({}, {100, 200}), x::errors::VALIDATION);
}
}
