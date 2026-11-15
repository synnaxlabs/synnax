// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/defer/defer.h"

#include "driver/ni/alias.h"
#include "driver/ni/errors/errors.h"

namespace driver::ni {
namespace {
const unsigned int SESSION_TIMEOUT_MS = 5000;
}

std::string trim_resource_name(const std::string &raw) {
    if (raw.size() <= 2) return raw;
    return raw.substr(1, raw.size() - 2);
}

std::pair<std::string, x::errors::Error>
find_alias(syscfg::SugaredAPI &syscfg, const std::string &resource_name) {
    NISysCfgSessionHandle session = nullptr;
    if (const auto err = syscfg.InitializeSession(
            nullptr,
            nullptr,
            nullptr,
            NISysCfgLocaleDefault,
            NISysCfgBoolTrue,
            SESSION_TIMEOUT_MS,
            nullptr,
            &session
        ))
        return {"", err};
    x::defer::defer close_session([&] { syscfg.CloseHandle(session); });
    NISysCfgFilterHandle filter = nullptr;
    if (const auto err = syscfg.CreateFilter(session, &filter)) return {"", err};
    x::defer::defer close_filter([&] { syscfg.CloseHandle(filter); });
    if (const auto err = syscfg.SetFilterProperty(
            filter,
            NISysCfgFilterPropertyIsDevice,
            NISysCfgBoolTrue
        ))
        return {"", err};
    if (const auto err = syscfg.SetFilterProperty(
            filter,
            NISysCfgFilterPropertyIsPresent,
            NISysCfgIsPresentTypePresent
        ))
        return {"", err};
    NISysCfgEnumResourceHandle resources = nullptr;
    const auto find_err = syscfg.FindHardware(
        session,
        NISysCfgFilterModeAll,
        filter,
        nullptr,
        &resources
    );
    if (find_err && !find_err.matches(errors::END_OF_ENUM)) return {"", find_err};
    // A device whose name does not read could be the one asked for, so its error
    // replaces NOT_FOUND when no other device matches.
    x::errors::Error read_err;
    if (!find_err) {
        x::defer::defer close_resources([&] { syscfg.CloseHandle(resources); });
        char buf[1024];
        NISysCfgResourceHandle resource = nullptr;
        while (true) {
            if (const auto err = syscfg.NextResource(session, resources, &resource)) {
                if (err.matches(errors::END_OF_ENUM)) break;
                return {"", err};
            }
            x::defer::defer close_resource([&] { syscfg.CloseHandle(resource); });
            if (const auto err = syscfg.GetResourceIndexedProperty(
                    resource,
                    NISysCfgIndexedPropertyExpertResourceName,
                    0,
                    buf
                )) {
                if (!read_err) read_err = err;
                continue;
            }
            if (trim_resource_name(buf) != resource_name) continue;
            if (const auto err = syscfg.GetResourceIndexedProperty(
                    resource,
                    NISysCfgIndexedPropertyExpertUserAlias,
                    0,
                    buf
                ))
                return {"", err};
            return {buf, x::errors::NIL};
        }
    }
    if (read_err) return {"", read_err};
    return {
        "",
        x::errors::Error(
            x::errors::NOT_FOUND,
            "no present NI device has the resource name " + resource_name
        )
    };
}
}
