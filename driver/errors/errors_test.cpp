// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/errors/errors.h"
#include "driver/labjack/errors.h"
#include "driver/ni/errors.h"

const std::vector<driver::LibraryInfo> ALL_LIBS = {
    labjack::LABJACK_LJM,
    ni::NI_DAQMX,
    ni::NI_SYSCFG,
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
    const driver::LibraryInfo no_url = {"Test Library", ""};

    auto err = driver::missing_lib(no_url);

    ASSERT_MATCHES(err, xlib::LOAD_ERROR);
    EXPECT_NE(err.data.find("Test Library"), std::string::npos);
    EXPECT_NE(err.data.find("is not installed"), std::string::npos);
    EXPECT_EQ(err.data.find("Download here:"), std::string::npos);
}
