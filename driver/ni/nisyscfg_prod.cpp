// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <stdio.h>
#include <iostream>
#include "nisyscfg.h"
#include "nisyscfg_api.h"
#include "nisyscfg_errors.h"

NISYSCFGCFUNC ni::NiSysCfgInterface::InitializeSession(
    const char *targetName,
    const char *username,
    const char *password,
    NISysCfgLocale language,
    NISysCfgBool forcePropertyRefresh,
    unsigned int connectTimeoutMsec,
    NISysCfgEnumExpertHandle *expertEnumHandle,
    NISysCfgSessionHandle *sessionHandle
) {
    return NISysCfgInitializeSession(
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

NISYSCFGCFUNC ni::NiSysCfgInterface::CreateFilter(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterHandle *filterHandle
) {
    return NISysCfgCreateFilter(sessionHandle, filterHandle);
}

NISYSCFGCDECL ni::NiSysCfgInterface::SetFilterProperty(
    NISysCfgFilterHandle filterHandle,
    NISysCfgFilterProperty propertyID,
    ...
) {
    return NISysCfgSetFilterProperty(filterHandle, propertyID);
}

NISYSCFGCFUNC ni::NiSysCfgInterface::CloseHandle(
    void *syscfgHandle
) {
    return NISysCfgCloseHandle(syscfgHandle);
}


NISYSCFGCFUNC ni::NiSysCfgInterface::FindHardware(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterMode filterMode,
    NISysCfgFilterHandle filterHandle,
    const char *expertNames,
    NISysCfgEnumResourceHandle *resourceEnumHandle
) {
    std::cout << "Hello" << std::endl;
    return NISysCfgFindHardware(
        sessionHandle,
        filterMode,
        filterHandle,
        expertNames,
        resourceEnumHandle
    );
}

NISYSCFGCFUNC ni::NiSysCfgInterface::NextResource(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgEnumResourceHandle resourceEnumHandle,
    NISysCfgResourceHandle *resourceHandle
) {
    return NISysCfgNextResource(sessionHandle, resourceEnumHandle, resourceHandle);
}

NISYSCFGCFUNC ni::NiSysCfgInterface::GetResourceProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgResourceProperty propertyID,
    void *value
) {
    return NISysCfgGetResourceProperty(resourceHandle, propertyID, value);
}

NISYSCFGCFUNC ni::NiSysCfgInterface::GetResourceIndexedProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgIndexedProperty propertyID,
    unsigned int index,
    void *value
) {
    return NISysCfgGetResourceIndexedProperty(resourceHandle, propertyID, index, value);
}
