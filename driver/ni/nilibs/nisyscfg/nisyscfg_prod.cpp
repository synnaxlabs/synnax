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

/// internal.
#include "driver/ni/nilibs/nisyscfg/nisyscfg.h"
#include "driver/ni/nilibs/nisyscfg/nisyscfg_prod.h"
#include "driver/ni/nilibs/nisyscfg/nisyscfg_errors.h"
#include "driver/ni/nilibs/shared/shared_library.h"

#ifdef _WIN32
static const char* kLibraryName = "nisyscfg.dll";
#else
static const char *kLibraryName = "libnisyscfg.so";
#endif


SysCfgProd::SysCfgProd(std::shared_ptr<SharedLibrary> library) : shared_library_(std::move(library)) {
    shared_library_->set_library_name(kLibraryName);
    shared_library_->load();
    bool loaded = shared_library_->is_loaded();
    memset(&function_pointers_, 0, sizeof(function_pointers_));
    if (!loaded) {
        return;
    }

    // Initialize function pointers
    function_pointers_.InitializeSession = reinterpret_cast<InitializeSessionPtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgInitializeSession")));

    function_pointers_.CreateFilter = reinterpret_cast<CreateFilterPtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgCreateFilter")));

    function_pointers_.SetFilterProperty = reinterpret_cast<SetFilterPropertyPtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgSetFilterProperty")));

    function_pointers_.CloseHandle = reinterpret_cast<CloseHandlePtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgCloseHandle")));

    function_pointers_.FindHardware = reinterpret_cast<FindHardwarePtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgFindHardware")));

    function_pointers_.NextResource = reinterpret_cast<NextResourcePtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgNextResource")));

    function_pointers_.GetResourceProperty = reinterpret_cast<GetResourcePropertyPtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgGetResourceProperty")));

    function_pointers_.GetResourceIndexedProperty = reinterpret_cast<GetResourceIndexedPropertyPtr>(
        const_cast<void*>(shared_library_->get_function_pointer("NISysCfgGetResourceIndexedProperty")));
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
    return function_pointers_.SetFilterProperty(filterHandle, propertyID);
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