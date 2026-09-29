// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/test/test.h"

#include "driver/arinc429/vendor.h"

namespace driver::arinc429::vendor {
namespace {
#if defined(_WIN32)
const std::string PRESENT = "kernel32.dll";
constexpr std::array<const char *, 1> FUNCTIONS = {"GetTickCount"};
#elif defined(__APPLE__)
const std::string PRESENT = "/usr/lib/libSystem.B.dylib";
constexpr std::array<const char *, 1> FUNCTIONS = {"malloc"};
#else
const std::string PRESENT = "libc.so.6";
constexpr std::array<const char *, 1> FUNCTIONS = {"malloc"};
#endif
}

TEST(Load, LoadsALibraryWithEveryFunction) {
    const auto lib = ASSERT_NIL_P(load(PRESENT, {"Present", ""}, FUNCTIONS));
    EXPECT_NE(lib, nullptr);
}

TEST(Load, ReturnsTheMissingLibraryErrorForAnAbsentFunction) {
    constexpr std::array<const char *, 1> functions = {"sy_absent_function"};
    const auto [lib, err] = load(PRESENT, {"Present", ""}, functions);
    ASSERT_OCCURRED_AS(err, x::lib::LOAD_ERROR);
    EXPECT_EQ(err.data, "Present library is not installed.");
}

TEST(Load, ReturnsTheMissingLibraryErrorForAnAbsentLibrary) {
    const auto [lib, err] = load("libabsent.so", {"Absent", ""});
    ASSERT_OCCURRED_AS(err, x::lib::LOAD_ERROR);
    EXPECT_EQ(err.data, "Absent library is not installed.");
}

TEST(Unsupported, NamesTheBackend) {
    const auto err = unsupported("Ballard ARINC 429");
    ASSERT_OCCURRED_AS(err, errors::CONFIGURATION_ERROR);
    EXPECT_EQ(
        err.data,
        "the Ballard ARINC 429 backend is not supported yet. Its vendor library "
        "loaded, but the Driver does not drive it"
    );
}
}
