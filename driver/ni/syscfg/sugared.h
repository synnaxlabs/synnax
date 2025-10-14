// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xlib/xlib.h"

#include "driver/ni/syscfg/api.h"
#include "driver/ni/syscfg/nisyscfg.h"

namespace syscfg {
class SugaredAPI {
    std::shared_ptr<API> syscfg;

    xerrors::Error process_error(NISysCfgStatus status) const;

public:
    explicit SugaredAPI(std::shared_ptr<API> syscfg): syscfg(std::move(syscfg)) {}

    xerrors::Error InitializeSession(
        const char *targetName,
        const char *username,
        const char *password,
        NISysCfgLocale language,
        NISysCfgBool forcePropertyRefresh,
        unsigned int connectTimeoutMsec,
        NISysCfgEnumExpertHandle *expertEnumHandle,
        NISysCfgSessionHandle *sessionHandle
    );

    xerrors::Error CreateFilter(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterHandle *filterHandle
    );

    xerrors::Error SetFilterProperty(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        ...
    );

    xerrors::Error CloseHandle(void *syscfgHandle);

    xerrors::Error FindHardware(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterMode filterMode,
        NISysCfgFilterHandle filterHandle,
        const char *expertNames,
        NISysCfgEnumResourceHandle *resourceEnumHandle
    );

    xerrors::Error NextResource(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumResourceHandle resourceEnumHandle,
        NISysCfgResourceHandle *resourceHandle
    );

    xerrors::Error GetResourceProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        void *value
    );

    xerrors::Error GetResourceIndexedProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgIndexedProperty propertyID,
        unsigned int index,
        void *value
    );
};
}
