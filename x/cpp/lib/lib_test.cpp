// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/test/test.h"

namespace x::lib {
namespace {
#ifdef _WIN32
const std::string SYSTEM_LIBRARY = "kernel32.dll";
const std::string SYSTEM_FUNCTION = "GetTickCount";
#elif defined(__APPLE__)
const std::string SYSTEM_LIBRARY = "/usr/lib/libSystem.B.dylib";
const std::string SYSTEM_FUNCTION = "strlen";
#else
const std::string SYSTEM_LIBRARY = "libc.so.6";
const std::string SYSTEM_FUNCTION = "strlen";
#endif

const errors::Error TEST_ERROR = errors::SY.sub("test");

using Fn = std::size_t (*)(const char *);
}

TEST(Symbols, ResolvesTheFunctionsALibraryHas) {
    Shared lib(SYSTEM_LIBRARY);
    ASSERT_TRUE(lib.load());
    Symbols symbols(lib);
    Fn fn = nullptr;
    symbols.resolve(SYSTEM_FUNCTION, fn);
    EXPECT_NE(fn, nullptr);
    ASSERT_NIL(symbols.error(TEST_ERROR, "system"));
}

TEST(Symbols, NamesEveryFunctionALibraryLacks) {
    Shared lib(SYSTEM_LIBRARY);
    ASSERT_TRUE(lib.load());
    Symbols symbols(lib);
    Fn found = nullptr;
    Fn first = nullptr;
    Fn second = nullptr;
    symbols.resolve("synnax_missing_one", first);
    symbols.resolve(SYSTEM_FUNCTION, found);
    symbols.resolve("synnax_missing_two", second);
    EXPECT_EQ(first, nullptr);
    EXPECT_EQ(second, nullptr);
    const auto err = symbols.error(TEST_ERROR, "system");
    ASSERT_MATCHES(err, TEST_ERROR);
    EXPECT_EQ(
        err.data,
        "the installed system library lacks synnax_missing_one, synnax_missing_two. "
        "Install a newer version"
    );
}

TEST(Symbols, ResolvesNothingFromALibraryThatDidNotLoad) {
    const Shared lib("synnax_missing_library");
    Symbols symbols(lib);
    Fn fn = nullptr;
    symbols.resolve(SYSTEM_FUNCTION, fn);
    EXPECT_EQ(fn, nullptr);
    ASSERT_MATCHES(symbols.error(TEST_ERROR, "missing"), TEST_ERROR);
}
}
