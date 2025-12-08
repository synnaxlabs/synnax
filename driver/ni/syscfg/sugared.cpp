// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/errors.h"
#include "driver/ni/syscfg/sugared.h"

namespace syscfg {
xerrors::Error SugaredAPI::process_error(NISysCfgStatus status) const {
    wchar_t *error_buf = nullptr;
    if (status == NISysCfg_OK) return xerrors::NIL;
    if (status == NISysCfg_EndOfEnum) return ni::END_OF_ENUM;
    const auto desc_status = this->syscfg
                                 ->GetStatusDescriptionW(nullptr, status, &error_buf);
    if (desc_status != NISysCfg_OK || error_buf == nullptr)
        return xerrors::Error(
            "failed to retrieve error message for status code " + std::to_string(status)
        );
    const auto str = std::wstring(error_buf);
    this->syscfg->FreeDetailedStringW(error_buf);
    return xerrors::Error(std::string(str.begin(), str.end()));
}

xerrors::Error SugaredAPI::InitializeSession(
    const char *targetName,
    const char *username,
    const char *password,
    NISysCfgLocale language,
    NISysCfgBool forcePropertyRefresh,
    unsigned int connectTimeoutMsec,
    NISysCfgEnumExpertHandle *expertEnumHandle,
    NISysCfgSessionHandle *sessionHandle
) {
    auto status = syscfg->InitializeSession(
        targetName,
        username,
        password,
        language,
        forcePropertyRefresh,
        connectTimeoutMsec,
        expertEnumHandle,
        sessionHandle
    );
    return process_error(status);
}

xerrors::Error SugaredAPI::CreateFilter(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterHandle *filterHandle
) {
    auto status = syscfg->CreateFilter(sessionHandle, filterHandle);
    return process_error(status);
}

xerrors::Error SugaredAPI::SetFilterProperty(
    NISysCfgFilterHandle filterHandle,
    NISysCfgFilterProperty propertyID,
    ...
) {
    va_list args;
    // Note: Enum types undergo default argument promotion to int in varargs.
    // This is a known limitation of C varargs but is how the NI API is designed.
    // Suppressing the warning to maintain semantic type safety of the enum.
#ifdef __clang__
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wvarargs"
#endif
    va_start(args, propertyID);
#ifdef __clang__
#pragma clang diagnostic pop
#endif
    auto status = syscfg->SetFilterPropertyV(filterHandle, propertyID, args);
    va_end(args);
    return process_error(status);
}

xerrors::Error SugaredAPI::CloseHandle(void *syscfgHandle) {
    auto status = syscfg->CloseHandle(syscfgHandle);
    return process_error(status);
}

xerrors::Error SugaredAPI::FindHardware(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterMode filterMode,
    NISysCfgFilterHandle filterHandle,
    const char *expertNames,
    NISysCfgEnumResourceHandle *resourceEnumHandle
) {
    auto status = syscfg->FindHardware(
        sessionHandle,
        filterMode,
        filterHandle,
        expertNames,
        resourceEnumHandle
    );
    return process_error(status);
}

xerrors::Error SugaredAPI::NextResource(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgEnumResourceHandle resourceEnumHandle,
    NISysCfgResourceHandle *resourceHandle
) {
    auto status = syscfg
                      ->NextResource(sessionHandle, resourceEnumHandle, resourceHandle);
    return process_error(status);
}

xerrors::Error SugaredAPI::GetResourceProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgResourceProperty propertyID,
    void *value
) {
    auto status = syscfg->GetResourceProperty(resourceHandle, propertyID, value);
    return process_error(status);
}

xerrors::Error SugaredAPI::GetResourceIndexedProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgIndexedProperty propertyID,
    unsigned int index,
    void *value
) {
    auto status = syscfg->GetResourceIndexedProperty(
        resourceHandle,
        propertyID,
        index,
        value
    );
    return process_error(status);
}
}
