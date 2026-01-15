// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/ni/syscfg/api.h"
#include "driver/ni/syscfg/sugared.h"

namespace {
/// @brief Mock implementation of syscfg::API for testing.
class MockAPI : public syscfg::API {
public:
    NISysCfgStatus close_handle_status = NISysCfg_OK;

    NISYSCFGCFUNC InitializeSession(
        const char *,
        const char *,
        const char *,
        NISysCfgLocale,
        NISysCfgBool,
        unsigned int,
        NISysCfgEnumExpertHandle *,
        NISysCfgSessionHandle *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CreateFilter(NISysCfgSessionHandle, NISysCfgFilterHandle *) override {
        return NISysCfg_OK;
    }

    NISYSCFGCDECL
    SetFilterProperty(NISysCfgFilterHandle, NISysCfgFilterProperty, ...) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CloseHandle(void *) override { return close_handle_status; }

    NISYSCFGCFUNC FindHardware(
        NISysCfgSessionHandle,
        NISysCfgFilterMode,
        NISysCfgFilterHandle,
        const char *,
        NISysCfgEnumResourceHandle *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle,
        NISysCfgEnumResourceHandle,
        NISysCfgResourceHandle *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    GetResourceProperty(
        NISysCfgResourceHandle,
        NISysCfgResourceProperty,
        void *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle,
        NISysCfgIndexedProperty,
        unsigned int,
        void *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    GetStatusDescriptionW(NISysCfgSessionHandle, NISysCfgStatus, wchar_t **) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    SetFilterPropertyV(NISysCfgFilterHandle, NISysCfgFilterProperty, va_list) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC FreeDetailedStringW(wchar_t[]) override { return NISysCfg_OK; }
};
}

/// @brief CloseHandle should return NIL when the underlying API returns OK.
TEST(SugaredAPITest, CloseHandleReturnsNilOnSuccess) {
    auto mock = std::make_shared<MockAPI>();
    mock->close_handle_status = NISysCfg_OK;
    syscfg::SugaredAPI api(mock);

    ASSERT_NIL(api.CloseHandle(nullptr));
}

/// @brief CloseHandle should return NIL when the underlying API returns EndOfEnum.
/// This is the key fix - EndOfEnum is normal when closing an exhausted enumeration.
TEST(SugaredAPITest, CloseHandleReturnsNilOnEndOfEnum) {
    auto mock = std::make_shared<MockAPI>();
    mock->close_handle_status = NISysCfg_EndOfEnum;
    syscfg::SugaredAPI api(mock);

    ASSERT_NIL(api.CloseHandle(nullptr));
}

/// @brief CloseHandle should propagate other errors normally.
TEST(SugaredAPITest, CloseHandlePropagatesOtherErrors) {
    auto mock = std::make_shared<MockAPI>();
    mock->close_handle_status = NISysCfg_InvalidArg;
    syscfg::SugaredAPI api(mock);

    auto err = api.CloseHandle(nullptr);
    EXPECT_TRUE(err) << "Expected an error for InvalidArg";
}
