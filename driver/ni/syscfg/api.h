// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/nisyscfg_errors.h"

namespace syscfg {
class API {
public:
    virtual ~API() = default;

    virtual NISYSCFGCFUNC InitializeSession(
        const char *targetName,
        const char *username,
        const char *password,
        NISysCfgLocale language,
        NISysCfgBool forcePropertyRefresh,
        unsigned int connectTimeoutMsec,
        NISysCfgEnumExpertHandle *expertEnumHandle,
        NISysCfgSessionHandle *sessionHandle
    ) = 0;

    virtual NISYSCFGCFUNC CreateFilter(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterHandle *filterHandle
    ) = 0;

    virtual NISYSCFGCDECL SetFilterProperty(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        ...
    ) = 0;

    virtual NISYSCFGCFUNC CloseHandle(void *syscfgHandle) = 0;

    virtual NISYSCFGCFUNC FindHardware(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterMode filterMode,
        NISysCfgFilterHandle filterHandle,
        const char *expertNames,
        NISysCfgEnumResourceHandle *resourceEnumHandle
    ) = 0;

    virtual NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumResourceHandle resourceEnumHandle,
        NISysCfgResourceHandle *resourceHandle
    ) = 0;

    virtual NISYSCFGCFUNC GetResourceProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        void *value
    ) = 0;

    virtual NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgIndexedProperty propertyID,
        unsigned int index,
        void *value
    ) = 0;

    virtual NISYSCFGCFUNC GetStatusDescriptionW(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgStatus status,
        wchar_t **detailedDescription
    ) = 0;

    virtual NISYSCFGCFUNC SetFilterPropertyV(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        va_list args
    ) = 0;

    virtual NISYSCFGCFUNC FreeDetailedStringW(wchar_t str[]) = 0;
};
}
