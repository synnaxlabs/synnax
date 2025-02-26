#include "driver/ni/syscfg/sugared.h"

xerrors::Error SugaredSysCfg::process_error(NISysCfgStatus status) const {
    wchar_t *error_buf = nullptr;
    if (status == NISysCfg_OK) return xerrors::Error();
    const auto desc_status = this->syscfg->GetStatusDescriptionW(
        nullptr, status, &error_buf);
    if (desc_status != NISysCfg_OK || error_buf == nullptr)
        return xerrors::Error(
            "failed to retrieve error message for status code " +
            std::to_string(status));
    const auto str = std::wstring(error_buf);
    this->syscfg->FreeDetailedStringW(error_buf);
    return xerrors::Error(std::string(str.begin(), str.end()));
}

xerrors::Error SugaredSysCfg::InitializeSession(
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
        targetName, username, password, language,
        forcePropertyRefresh, connectTimeoutMsec,
        expertEnumHandle, sessionHandle
    );
    return process_error(status);
}

xerrors::Error SugaredSysCfg::CreateFilter(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterHandle *filterHandle
) {
    auto status = syscfg->CreateFilter(sessionHandle, filterHandle);
    return process_error(status);
}

xerrors::Error SugaredSysCfg::SetFilterProperty(
    NISysCfgFilterHandle filterHandle,
    NISysCfgFilterProperty propertyID,
    ...
) {
    va_list args;
    va_start(args, propertyID);
    auto status = syscfg->SetFilterProperty(filterHandle, propertyID, args);
    va_end(args);
    return process_error(status);
}

xerrors::Error SugaredSysCfg::CloseHandle(void *syscfgHandle) {
    auto status = syscfg->CloseHandle(syscfgHandle);
    return process_error(status);
}

xerrors::Error SugaredSysCfg::FindHardware(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgFilterMode filterMode,
    NISysCfgFilterHandle filterHandle,
    const char *expertNames,
    NISysCfgEnumResourceHandle *resourceEnumHandle
) {
    auto status = syscfg->FindHardware(
        sessionHandle, filterMode, filterHandle,
        expertNames, resourceEnumHandle
    );
    return process_error(status);
}

xerrors::Error SugaredSysCfg::NextResource(
    NISysCfgSessionHandle sessionHandle,
    NISysCfgEnumResourceHandle resourceEnumHandle,
    NISysCfgResourceHandle *resourceHandle
) {
    auto status = syscfg->NextResource(
        sessionHandle, resourceEnumHandle, resourceHandle
    );
    return process_error(status);
}

xerrors::Error SugaredSysCfg::GetResourceProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgResourceProperty propertyID,
    void *value
) {
    auto status = syscfg->GetResourceProperty(
        resourceHandle, propertyID, value
    );
    return process_error(status);
}

xerrors::Error SugaredSysCfg::GetResourceIndexedProperty(
    NISysCfgResourceHandle resourceHandle,
    NISysCfgIndexedProperty propertyID,
    unsigned int index,
    void *value
) {
    auto status = syscfg->GetResourceIndexedProperty(
        resourceHandle, propertyID, index, value
    );
    return process_error(status);
}
