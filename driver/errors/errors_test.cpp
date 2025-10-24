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

const std::vector<driver::LibraryInfo> ALL_LIBS = {
    driver::lib::LABJACK_LJM,
    driver::lib::NI_DAQMX,
    driver::lib::NI_SYSCFG,
};

TEST(ErrorsTest, LibraryInfoNotEmpty) {
    for (const auto &lib: ALL_LIBS) {
        EXPECT_FALSE(lib.name.empty()) << "Library name should not be empty";
        EXPECT_FALSE(lib.url.empty())
            << "Library URL should not be empty for " << lib.name;
    }
}

TEST(ErrorsTest, URLsAreWellFormed) {
    auto is_valid_url = [](const std::string &url) {
        if (url.length() < 12) return false;
        if (url.substr(0, 8) != "https://") return false;
        if (url.find(' ') != std::string::npos) return false;
        return true;
    };

    for (const auto &lib: ALL_LIBS) {
        EXPECT_TRUE(is_valid_url(lib.url)) << "URL is not well-formed for " << lib.name;
    }
}

TEST(ErrorsTest, LibraryInfoWithoutURL) {
    driver::LibraryInfo no_url = {"Test Library", ""};

    auto err = driver::missing_lib(no_url);

    EXPECT_TRUE(err.matches(xlib::LOAD_ERROR));
    EXPECT_NE(err.data.find("Test Library"), std::string::npos);
    EXPECT_NE(err.data.find("is not installed"), std::string::npos);
    EXPECT_EQ(err.data.find("Download here:"), std::string::npos);
}
