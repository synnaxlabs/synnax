// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/ni/nilibs/nisyscfg/nisyscfg.h"
#include "driver/ni/nilibs/nisyscfg/nisyscfg_errors.h"

class SysCfg {
public:
    virtual ~SysCfg() = default;

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

    virtual NISYSCFGCFUNC CloseHandle(
        void *syscfgHandle
    ) = 0;

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
};
