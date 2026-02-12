// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/errors/errors.h"
#include "driver/labjack/ljm/api.h"
#include "driver/ni/daqmx/api.h"
#include "driver/ni/syscfg/api.h"

namespace driver::errors {
const std::vector ALL_LIBS = {
    labjack::ljm::LIBRARY_INFO,
    ni::daqmx::LIBRARY_INFO,
    ni::syscfg::LIBRARY_INFO,
};

/// @brief it should have non-empty names and URLs for all library info.
TEST(ErrorsTest, LibraryInfoNotEmpty) {
    for (const auto &lib: ALL_LIBS) {
        EXPECT_FALSE(lib.name.empty()) << "Library name should not be empty";
        EXPECT_FALSE(lib.url.empty())
            << "Library URL should not be empty for " << lib.name;
    }
}

/// @brief it should have well-formed HTTPS URLs for all libraries.
TEST(ErrorsTest, URLsAreWellFormed) {
    auto is_valid_url = [](const std::string &url) {
        if (url.length() < 12) return false;
        if (url.substr(0, 8) != "https://") return false;
        if (url.find(' ') != std::string::npos) return false;
        return true;
    };

    for (const auto &lib: ALL_LIBS)
        EXPECT_TRUE(is_valid_url(lib.url)) << "URL is not well-formed for " << lib.name;
}

/// @brief it should create a missing library error without download URL.
TEST(ErrorsTest, LibraryInfoWithoutURL) {
    const LibraryInfo no_url = {"Test Library", ""};

    auto err = missing_lib(no_url);

    ASSERT_MATCHES(err, x::lib::LOAD_ERROR);
    EXPECT_NE(err.data.find("Test Library"), std::string::npos);
    EXPECT_NE(err.data.find("is not installed"), std::string::npos);
    EXPECT_EQ(err.data.find("Download here:"), std::string::npos);
}

/// @brief it should wrap error with channel name and hardware location.
TEST(ErrorsTest, WrapChannelError) {
    auto base_err = x::errors::Error(CRITICAL_HARDWARE_ERROR, "some hardware error");

    auto wrapped = wrap_channel_error(base_err, "my_channel", "AIN0");

    EXPECT_EQ(wrapped.type, base_err.type);
    EXPECT_NE(wrapped.data.find("my_channel"), std::string::npos)
        << "Expected channel name in error. Got: " << wrapped.data;
    EXPECT_NE(wrapped.data.find("AIN0"), std::string::npos)
        << "Expected hardware location in error. Got: " << wrapped.data;
    EXPECT_NE(wrapped.data.find("some hardware error"), std::string::npos)
        << "Expected original error message in error. Got: " << wrapped.data;
}
}
