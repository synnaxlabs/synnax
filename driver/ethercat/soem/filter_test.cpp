// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#endif

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/filter.h"

namespace driver::ethercat::soem {
#ifdef _WIN32
namespace {
bool elevated() {
    HANDLE token = nullptr;
    if (OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token) == 0) return false;
    TOKEN_ELEVATION elevation{};
    DWORD size = 0;
    const bool read = GetTokenInformation(
                          token,
                          TokenElevation,
                          &elevation,
                          sizeof(elevation),
                          &size
                      ) != 0;
    CloseHandle(token);
    return read && elevation.TokenIsElevated != 0;
}
}

/// @brief it should refuse an interface that is not a Windows network adapter.
TEST(DisableDewesoftFilterTest, RefusesAnInterfaceThatIsNotAnAdapter) {
    const auto err = disable_dewesoft_filter({"eth0"});
    ASSERT_OCCURRED_AS(err, errors::FILTER_ERROR);
    EXPECT_EQ(err.data, "eth0 is not a Windows network adapter.");
}

/// @brief it should change nothing and report an adapter that Windows does not have.
TEST(DisableDewesoftFilterTest, ReportsAnAdapterThatWindowsDoesNotHave) {
    if (!elevated()) GTEST_SKIP() << "the test would show an administrator prompt";
    const auto err = disable_dewesoft_filter(
        {"\\Device\\NPF_{00000000-0000-0000-0000-000000000000}"}
    );
    ASSERT_OCCURRED_AS(err, errors::FILTER_ERROR);
    EXPECT_EQ(
        err.data,
        "Windows has no network adapter for the interface. To turn it off by hand, "
        "clear \"EthercatDAQ Filter\" in the properties of the adapter."
    );
}
#else
/// @brief it should refuse to turn off the filter outside Windows.
TEST(DisableDewesoftFilterTest, RefusesOutsideWindows) {
    const auto err = disable_dewesoft_filter({"eth0"});
    ASSERT_OCCURRED_AS(err, errors::FILTER_ERROR);
    EXPECT_EQ(err.data, "Only a Driver on Windows can turn off the DewesoftX filter.");
}
#endif
}
