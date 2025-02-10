#pragma once

#include "driver/ni/nisyscfg/nisyscfg.h"
#include "driver/ni/nisyscfg/nisyscfg_api.h"
#include "x/cpp/xlib/xlib.h"

class SysCfgProd : public SysCfg {
public:
    explicit SysCfgProd(std::unique_ptr<xlib::SharedLib> &lib_);

    static std::pair<std::shared_ptr<SysCfg>, xerrors::Error> load();

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

    NISYSCFGCFUNC CloseHandle(
        void *syscfgHandle
    ) override;

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
    } FunctionPointers;

    std::unique_ptr<xlib::SharedLib> lib;
    FunctionPointers function_pointers_;
};
