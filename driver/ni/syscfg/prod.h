// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/ni/syscfg/api.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/nisyscfg_wide.h"
#include "x/cpp/xlib/xlib.h"

namespace syscfg {
class ProdAPI final : public API {
public:
    explicit ProdAPI(std::unique_ptr<xlib::SharedLib> &lib_);

    static std::pair<std::shared_ptr<API>, xerrors::Error> load();

    NISYSCFGCFUNC InitializeSession(
        const char *targetName,
        const char *username,
        const char *password,
        NISysCfgLocale language,
        NISysCfgBool forcePropertyRefresh,
        unsigned int connectTimeoutMsec,
        NISysCfgEnumExpertHandle *expertEnumHandle,
        NISysCfgSessionHandle *sessionHandle
    ) override;

    NISYSCFGCFUNC CreateFilter(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterHandle *filterHandle
    ) override;

    NISYSCFGCDECL SetFilterProperty(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        ...
    ) override;

    NISYSCFGCFUNC SetFilterPropertyV(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        va_list args
    ) override;

    NISYSCFGCFUNC CloseHandle(void *syscfgHandle) override;

    NISYSCFGCFUNC FindHardware(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterMode filterMode,
        NISysCfgFilterHandle filterHandle,
        const char *expertNames,
        NISysCfgEnumResourceHandle *resourceEnumHandle
    ) override;

    NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumResourceHandle resourceEnumHandle,
        NISysCfgResourceHandle *resourceHandle
    ) override;

    NISYSCFGCFUNC GetResourceProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        void *value
    ) override;

    NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgIndexedProperty propertyID,
        unsigned int index,
        void *value
    ) override;

    NISysCfgStatus GetStatusDescriptionW(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgStatus status,
        wchar_t **detailedDescription
    ) override;

    NISysCfgStatus FreeDetailedStringW(wchar_t str[]) override;

private:
    // Function pointer typedefs
    using InitializeSessionPtr = decltype(&NISysCfgInitializeSession);
    using CreateFilterPtr = decltype(&NISysCfgCreateFilter);
    using SetFilterPropertyVPtr = decltype(&NISysCfgSetFilterPropertyV);
    using CloseHandlePtr = decltype(&NISysCfgCloseHandle);
    using FindHardwarePtr = decltype(&NISysCfgFindHardware);
    using NextResourcePtr = decltype(&NISysCfgNextResource);
    using GetResourcePropertyPtr = decltype(&NISysCfgGetResourceProperty);
    using GetResourceIndexedPropertyPtr = decltype(&NISysCfgGetResourceIndexedProperty);
    using GetStatusDescriptionWPtr = decltype(&NISysCfgGetStatusDescriptionW);
    using FreeDetailedStringWPtr = decltype(&NISysCfgFreeDetailedStringW);

    // Function pointers struct
    typedef struct FunctionPointers {
        InitializeSessionPtr InitializeSession;
        CreateFilterPtr CreateFilter;
        SetFilterPropertyVPtr SetFilterPropertyV;
        CloseHandlePtr CloseHandle;
        FindHardwarePtr FindHardware;
        NextResourcePtr NextResource;
        GetResourcePropertyPtr GetResourceProperty;
        GetResourceIndexedPropertyPtr GetResourceIndexedProperty;
        GetStatusDescriptionWPtr GetStatusDescriptionW;
        FreeDetailedStringWPtr FreeDetailedStringW;
    } FunctionPointers;

    std::unique_ptr<xlib::SharedLib> lib;
    FunctionPointers function_pointers_;
};
}
