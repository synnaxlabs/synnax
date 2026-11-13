// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <cstring>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ni/alias.h"
#include "driver/ni/syscfg/api.h"

namespace driver::ni {
namespace {
/// @brief an in-memory NI System Configuration API that enumerates devices by their
/// resource name and alias, and counts the handles it leaves open.
class FakeAPI final : public syscfg::API {
public:
    struct Resource {
        std::string resource_name;
        std::string alias;
    };

    int open_handles = 0;
    /// @brief the call that fails with NISysCfg_Timeout, or empty when none fails.
    std::string failing_call;

private:
    std::vector<Resource> resources;
    std::size_t next = 0;

public:
    explicit FakeAPI(std::vector<Resource> resources):
        resources(std::move(resources)) {}

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
        if (this->failing_call == "InitializeSession") return NISysCfg_Timeout;
        this->open_handles++;
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CreateFilter(NISysCfgSessionHandle, NISysCfgFilterHandle *) override {
        if (this->failing_call == "CreateFilter") return NISysCfg_Timeout;
        this->open_handles++;
        return NISysCfg_OK;
    }

    NISYSCFGCDECL
    SetFilterProperty(NISysCfgFilterHandle, NISysCfgFilterProperty, ...) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CloseHandle(void *) override {
        this->open_handles--;
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC FindHardware(
        NISysCfgSessionHandle,
        NISysCfgFilterMode,
        NISysCfgFilterHandle,
        const char *,
        NISysCfgEnumResourceHandle *
    ) override {
        if (this->failing_call == "FindHardware") return NISysCfg_Timeout;
        if (this->resources.empty()) return NISysCfg_EndOfEnum;
        this->open_handles++;
        this->next = 0;
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle,
        NISysCfgEnumResourceHandle,
        NISysCfgResourceHandle *handle
    ) override {
        if (this->failing_call == "NextResource") return NISysCfg_Timeout;
        if (this->next == this->resources.size()) return NISysCfg_EndOfEnum;
        this->open_handles++;
        *handle = reinterpret_cast<NISysCfgResourceHandle>(++this->next);
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    GetResourceProperty(
        NISysCfgResourceHandle,
        NISysCfgResourceProperty,
        void *
    ) override {
        return NISysCfg_PropDoesNotExist;
    }

    NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle handle,
        const NISysCfgIndexedProperty property,
        unsigned int,
        void *value
    ) override {
        const bool alias = property == NISysCfgIndexedPropertyExpertUserAlias;
        if (this->failing_call == (alias ? "Alias" : "ResourceName"))
            return NISysCfg_Timeout;
        const auto &res = this->resources[reinterpret_cast<std::uintptr_t>(handle) - 1];
        const auto v = property == NISysCfgIndexedPropertyExpertUserAlias
                         ? res.alias
                         : "\"" + res.resource_name + "\"";
        std::strcpy(static_cast<char *>(value), v.c_str());
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    GetStatusDescriptionW(NISysCfgSessionHandle, NISysCfgStatus, wchar_t **) override {
        return NISysCfg_NotImplemented;
    }

    NISYSCFGCFUNC
    SetFilterPropertyV(NISysCfgFilterHandle, NISysCfgFilterProperty, va_list) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC FreeDetailedStringW(wchar_t[]) override { return NISysCfg_OK; }
};
}

/// @brief it should find the alias of the device with the resource name.
TEST(FindAlias, FindsAliasByResourceName) {
    const auto api = std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{
        {.resource_name = "3BA1A71A", .alias = "Dev1"},
        {.resource_name = "7C2D9E0F", .alias = "SY2Mod1"},
    });
    syscfg::SugaredAPI syscfg(api);
    ASSERT_EQ(ASSERT_NIL_P(find_alias(syscfg, "7C2D9E0F")), "SY2Mod1");
    ASSERT_EQ(api->open_handles, 0);
}

/// @brief it should return NOT_FOUND when no present device has the resource name.
TEST(FindAlias, ReturnsNotFoundForUnknownResourceName) {
    const auto api = std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{
        {.resource_name = "3BA1A71A", .alias = "Dev1"},
    });
    syscfg::SugaredAPI syscfg(api);
    ASSERT_OCCURRED_AS_P(find_alias(syscfg, "7C2D9E0F"), x::errors::NOT_FOUND);
    ASSERT_EQ(api->open_handles, 0);
}

/// @brief it should return NOT_FOUND when no device is present.
TEST(FindAlias, ReturnsNotFoundWithNoDevices) {
    const auto api = std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{});
    syscfg::SugaredAPI syscfg(api);
    ASSERT_OCCURRED_AS_P(find_alias(syscfg, "7C2D9E0F"), x::errors::NOT_FOUND);
    ASSERT_EQ(api->open_handles, 0);
}

namespace {
/// @brief asserts that find_alias returns the error of the failing call, not
/// NOT_FOUND, and closes every handle it opened.
void assert_fails_on(const std::string &call) {
    const auto api = std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{
        {.resource_name = "7C2D9E0F", .alias = "SY2Mod1"},
    });
    api->failing_call = call;
    syscfg::SugaredAPI syscfg(api);
    const auto [alias, err] = find_alias(syscfg, "7C2D9E0F");
    ASSERT_TRUE(err) << call;
    ASSERT_FALSE(err.matches(x::errors::NOT_FOUND)) << call;
    ASSERT_EQ(api->open_handles, 0) << call;
}
}

/// @brief it should return the error of a failed System Configuration call.
TEST(FindAlias, ReturnsSystemConfigurationErrors) {
    for (const auto *call: {
             "InitializeSession",
             "CreateFilter",
             "FindHardware",
             "NextResource",
             "ResourceName",
             "Alias",
         })
        assert_fails_on(call);
}
}
