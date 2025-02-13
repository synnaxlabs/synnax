// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std. lib.
#include <stdio.h>
#include <iostream>
#include <cstring>
#include <cstdarg>

/// internal.
#include "driver/ni/nisyscfg/nisyscfg.h"
#include "driver/ni/nisyscfg/nisyscfg_prod.h"
#include "driver/ni/nisyscfg/nisyscfg_errors.h"
#include "x/cpp/xos/xos.h"

#ifdef _WIN32
static const std::string LIB_NAME = "nisyscfg.dll";
#else
static const std::string LIB_NAME = "libnisyscfg.so";
#endif

const auto LOAD_ERROR = xerrors::Error(
    xlib::LOAD_ERROR,
    "failed to load ni system configuration library. is it installed?"
);

std::pair<std::shared_ptr<SysCfg>, xerrors::Error> SysCfgProd::load() {
    if (xos::get() == "macOS") return {nullptr, xerrors::NIL};
    auto lib = std::make_unique<xlib::SharedLib>(LIB_NAME);
    if (!lib->load()) return {nullptr, LOAD_ERROR};
    return {std::make_shared<SysCfgProd>(lib), xerrors::NIL};
}

SysCfgProd::SysCfgProd(std::unique_ptr<xlib::SharedLib> &lib_) : lib(std::move(lib_)) {
    memset(&function_pointers_, 0, sizeof(function_pointers_));
    function_pointers_.InitializeSession = reinterpret_cast<InitializeSessionPtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgInitializeSession")));

    function_pointers_.CreateFilter = reinterpret_cast<CreateFilterPtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgCreateFilter")));

    function_pointers_.SetFilterPropertyV = reinterpret_cast<SetFilterPropertyVPtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgSetFilterPropertyV")));

    function_pointers_.CloseHandle = reinterpret_cast<CloseHandlePtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgCloseHandle")));

    function_pointers_.FindHardware = reinterpret_cast<FindHardwarePtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgFindHardware")));

    function_pointers_.NextResource = reinterpret_cast<NextResourcePtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgNextResource")));

    function_pointers_.GetResourceProperty = reinterpret_cast<GetResourcePropertyPtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgGetResourceProperty")));

    function_pointers_.GetResourceIndexedProperty = reinterpret_cast<GetResourceIndexedPropertyPtr>(
        const_cast<void*>(this->lib->get_func_ptr("NISysCfgGetResourceIndexedProperty")));
}

NISYSCFGCFUNC SysCfgProd::InitializeSession(
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

NISYSCFGCFUNC SysCfgProd::CreateFilter(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterHandle *filterHandle
) {
    return function_pointers_.CreateFilter(sessionHandle, filterHandle);
}

NISYSCFGCDECL SysCfgProd::SetFilterProperty(
    NISysCfgFilterHandle filterHandle,
    NISysCfgFilterProperty propertyID,
    ...
) {
  va_list args;
  va_start(args, propertyID);
  NISysCfgStatus status = function_pointers_.SetFilterPropertyV(filterHandle, propertyID, args);
  va_end(args);
  return status;
}

NISYSCFGCFUNC SysCfgProd::CloseHandle(
    void *syscfgHandle
) {
    return function_pointers_.CloseHandle(syscfgHandle);
}

NISYSCFGCFUNC SysCfgProd::FindHardware(
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

NISYSCFGCFUNC SysCfgProd::NextResource(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgEnumResourceHandle resourceEnumHandle,
    NISysCfgResourceHandle *resourceHandle
) {
    return function_pointers_.NextResource(sessionHandle, resourceEnumHandle, resourceHandle);
}

NISYSCFGCFUNC SysCfgProd::GetResourceProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgResourceProperty propertyID,
    void *value
) {
    return function_pointers_.GetResourceProperty(resourceHandle, propertyID, value);
}

NISYSCFGCFUNC SysCfgProd::GetResourceIndexedProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgIndexedProperty propertyID,
    unsigned int index,
    void *value
) {
    return function_pointers_.GetResourceIndexedProperty(resourceHandle, propertyID, index, value);
}
