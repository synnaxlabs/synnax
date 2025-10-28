// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std. lib.
#include <cstdarg>
#include <cstring>
#include <iostream>

#include <stdio.h>

/// internal.
#include "x/cpp/xos/xos.h"

#include "driver/errors/errors.h"
#include "driver/ni/errors.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/nisyscfg_errors.h"
#include "driver/ni/syscfg/prod.h"

#ifdef _WIN32
static const std::string LIB_NAME = "nisyscfg.dll";
#else
static const std::string LIB_NAME = "libnisyscfg.so";
#endif

namespace syscfg {
const auto LOAD_ERROR = driver::missing_lib(ni::NI_SYSCFG);

std::pair<std::shared_ptr<API>, xerrors::Error> ProdAPI::load() {
    if (xos::get() == xos::MACOS_NAME) return {nullptr, xerrors::NIL};
    auto lib = std::make_unique<xlib::SharedLib>(LIB_NAME);
    if (!lib->load()) return {nullptr, LOAD_ERROR};
    return {std::make_shared<ProdAPI>(lib), xerrors::NIL};
}

ProdAPI::ProdAPI(std::unique_ptr<xlib::SharedLib> &lib_): lib(std::move(lib_)) {
    memset(&function_pointers_, 0, sizeof(function_pointers_));
    function_pointers_.InitializeSession = reinterpret_cast<InitializeSessionPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgInitializeSession"))
    );

    function_pointers_.CreateFilter = reinterpret_cast<CreateFilterPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgCreateFilter"))
    );

    function_pointers_.SetFilterPropertyV = reinterpret_cast<SetFilterPropertyVPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgSetFilterPropertyV"))
    );

    function_pointers_.CloseHandle = reinterpret_cast<CloseHandlePtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgCloseHandle"))
    );

    function_pointers_.FindHardware = reinterpret_cast<FindHardwarePtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgFindHardware"))
    );

    function_pointers_.NextResource = reinterpret_cast<NextResourcePtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgNextResource"))
    );

    function_pointers_.GetResourceProperty = reinterpret_cast<GetResourcePropertyPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgGetResourceProperty"))
    );

    function_pointers_
        .GetResourceIndexedProperty = reinterpret_cast<GetResourceIndexedPropertyPtr>(
        const_cast<void *>(
            this->lib->get_func_ptr("NISysCfgGetResourceIndexedProperty")
        )
    );

    function_pointers_
        .GetStatusDescriptionW = reinterpret_cast<GetStatusDescriptionWPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgGetStatusDescriptionW"))
    );

    function_pointers_.FreeDetailedStringW = reinterpret_cast<FreeDetailedStringWPtr>(
        const_cast<void *>(this->lib->get_func_ptr("NISysCfgFreeDetailedStringW"))
    );
}

NISYSCFGCFUNC ProdAPI::InitializeSession(
    const char *targetName,
    const char *username,
    const char *password,
    NISysCfgLocale language,
    NISysCfgBool forcePropertyRefresh,
    unsigned int connectTimeoutMsec,
    NISysCfgEnumExpertHandle *expertEnumHandle,
    NISysCfgSessionHandle *sessionHandle
) {
    return function_pointers_.InitializeSession(
        targetName,
        username,
        password,
        language,
        forcePropertyRefresh,
        connectTimeoutMsec,
        expertEnumHandle,
        sessionHandle
    );
}

NISYSCFGCFUNC ProdAPI::CreateFilter(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterHandle *filterHandle
) {
    return function_pointers_.CreateFilter(sessionHandle, filterHandle);
}

NISYSCFGCDECL ProdAPI::SetFilterPropertyV(
    NISysCfgFilterHandle filterHandle,
    NISysCfgFilterProperty propertyID,
    va_list args
) {
    return function_pointers_.SetFilterPropertyV(filterHandle, propertyID, args);
}

NISYSCFGCDECL ProdAPI::SetFilterProperty(
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
    NISysCfgStatus status = SetFilterPropertyV(filterHandle, propertyID, args);
    va_end(args);
    return status;
}

NISYSCFGCFUNC ProdAPI::CloseHandle(void *syscfgHandle) {
    return function_pointers_.CloseHandle(syscfgHandle);
}

NISYSCFGCFUNC ProdAPI::FindHardware(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterMode filterMode,
    NISysCfgFilterHandle filterHandle,
    const char *expertNames,
    NISysCfgEnumResourceHandle *resourceEnumHandle
) {
    return function_pointers_.FindHardware(
        sessionHandle,
        filterMode,
        filterHandle,
        expertNames,
        resourceEnumHandle
    );
}

NISYSCFGCFUNC ProdAPI::NextResource(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgEnumResourceHandle resourceEnumHandle,
    NISysCfgResourceHandle *resourceHandle
) {
    return function_pointers_
        .NextResource(sessionHandle, resourceEnumHandle, resourceHandle);
}

NISYSCFGCFUNC ProdAPI::GetResourceProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgResourceProperty propertyID,
    void *value
) {
    return function_pointers_.GetResourceProperty(resourceHandle, propertyID, value);
}

NISYSCFGCFUNC ProdAPI::GetResourceIndexedProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgIndexedProperty propertyID,
    unsigned int index,
    void *value
) {
    return function_pointers_
        .GetResourceIndexedProperty(resourceHandle, propertyID, index, value);
}

NISysCfgStatus ProdAPI::GetStatusDescriptionW(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgStatus status,
    wchar_t **detailedDescription
) {
    return function_pointers_
        .GetStatusDescriptionW(sessionHandle, status, detailedDescription);
}

NISYSCFGCFUNC ProdAPI::FreeDetailedStringW(wchar_t str[]) {
    return function_pointers_.FreeDetailedStringW(str);
}
}
