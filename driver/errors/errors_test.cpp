// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <curl/curl.h>
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
    for (const auto &lib : ALL_LIBS) {
        EXPECT_FALSE(lib.name.empty()) << "Library name should not be empty";
        EXPECT_FALSE(lib.url.empty()) << "Library URL should not be empty for " << lib.name;
    }
}

TEST(ErrorsTest, URLsAreWellFormed) {
    auto is_valid_url = [](const std::string &url) {
        if (url.length() < 12) return false;
        if (url.substr(0, 8) != "https://") return false;
        if (url.find(' ') != std::string::npos) return false;
        return true;
    };

    for (const auto &lib : ALL_LIBS) {
        EXPECT_TRUE(is_valid_url(lib.url)) << "URL is not well-formed for " << lib.name;
    }
}

size_t write_callback(void *contents, size_t size, size_t nmemb, void *userp) {
    return size * nmemb;
}

TEST(ErrorsTest, URLsAreReachable) {
    curl_global_init(CURL_GLOBAL_DEFAULT);

    for (const auto &lib : ALL_LIBS) {
        CURL *curl = curl_easy_init();
        ASSERT_NE(curl, nullptr) << "Failed to initialize curl for " << lib.name;

        curl_easy_setopt(curl, CURLOPT_URL, lib.url.c_str());
        curl_easy_setopt(curl, CURLOPT_NOBODY, 1L);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, write_callback);

        CURLcode res = curl_easy_perform(curl);

        long response_code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &response_code);

        curl_easy_cleanup(curl);

        EXPECT_EQ(res, CURLE_OK) << "Failed to reach " << lib.name << " at " << lib.url
                                  << " (curl error: " << curl_easy_strerror(res) << ")";

        if (res == CURLE_OK) {
            EXPECT_NE(response_code, 404) << lib.name << " URL returned 404: " << lib.url;
            EXPECT_GE(response_code, 200) << lib.name << " URL returned error code: " << response_code;
            EXPECT_LT(response_code, 400) << lib.name << " URL returned error code: " << response_code;
        }
    }

    curl_global_cleanup();
}

TEST(ErrorsTest, LibraryInfoWithoutURL) {
    driver::LibraryInfo no_url = {
        "Test Library",
        ""
    };

    auto err = driver::missing_lib(no_url);

    EXPECT_TRUE(err.matches(xlib::LOAD_ERROR));
    EXPECT_NE(err.data.find("Test Library"), std::string::npos);
    EXPECT_NE(err.data.find("is not installed"), std::string::npos);
    EXPECT_EQ(err.data.find("Download here:"), std::string::npos);
}
