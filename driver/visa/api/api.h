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
#include "driver/visa/api/types.h"

namespace visa_api {

using namespace visa_types;

// Platform-specific library names
#ifdef _WIN32
const std::string VISA_LIBRARY_NAME = "visa64.dll";
#elif __APPLE__
const std::string VISA_LIBRARY_NAME = "/Library/Frameworks/VISA.framework/VISA";
#else
const std::string VISA_LIBRARY_NAME = "libvisa.so";
#endif

const auto LOAD_ERROR = xerrors::Error(
    xlib::LOAD_ERROR,
    "failed to load VISA shared libraries. Are they installed? Expected to find them at " +
        VISA_LIBRARY_NAME +
        ". Install NI-VISA or another VISA implementation from https://www.ni.com/en-us/support/downloads/drivers/download.ni-visa.html"
);

/// @brief API wrapper on top of VISA functions that the Synnax driver requires.
/// Uses dynamic loading to gracefully handle when VISA is not installed.
class API {
    /// @brief Function pointer types for VISA functions.
    /// These match the VISA function signatures.
    using viOpenDefaultRM_t = ViStatus (*)(ViSession *);
    using viClose_t = ViStatus (*)(ViSession);
    using viOpen_t = ViStatus (*)(ViSession, ViRsrc, ViUInt32, ViUInt32, ViSession *);
    using viRead_t = ViStatus (*)(ViSession, ViBuf, ViUInt32, ViUInt32 *);
    using viWrite_t = ViStatus (*)(ViSession, ViBuf, ViUInt32, ViUInt32 *);
    using viFindRsrc_t = ViStatus (*)(ViSession, ViString, ViFindList *, ViUInt32 *, ViChar *);
    using viFindNext_t = ViStatus (*)(ViFindList, ViChar *);
    using viStatusDesc_t = ViStatus (*)(ViSession, ViStatus, ViChar *);
    using viSetAttribute_t = ViStatus (*)(ViSession, ViUInt32, ViUInt32);
    using viGetAttribute_t = ViStatus (*)(ViSession, ViUInt32, void *);

    /// @brief Function pointers to the VISA functions.
    struct FunctionPointers {
        viOpenDefaultRM_t OpenDefaultRM;
        viClose_t Close;
        viOpen_t Open;
        viRead_t Read;
        viWrite_t Write;
        viFindRsrc_t FindRsrc;
        viFindNext_t FindNext;
        viStatusDesc_t StatusDesc;
        viSetAttribute_t SetAttribute;
        viGetAttribute_t GetAttribute;
    };

    /// @brief Shared library handle.
    std::unique_ptr<xlib::SharedLib> lib;
    FunctionPointers func_ptrs;

public:
    explicit API(std::unique_ptr<xlib::SharedLib> lib_): lib(std::move(lib_)) {
        memset(&func_ptrs, 0, sizeof(func_ptrs));

        func_ptrs.OpenDefaultRM = reinterpret_cast<viOpenDefaultRM_t>(
            const_cast<void *>(lib->get_func_ptr("viOpenDefaultRM"))
        );
        func_ptrs.Close = reinterpret_cast<viClose_t>(
            const_cast<void *>(lib->get_func_ptr("viClose"))
        );
        func_ptrs.Open = reinterpret_cast<viOpen_t>(
            const_cast<void *>(lib->get_func_ptr("viOpen"))
        );
        func_ptrs.Read = reinterpret_cast<viRead_t>(
            const_cast<void *>(lib->get_func_ptr("viRead"))
        );
        func_ptrs.Write = reinterpret_cast<viWrite_t>(
            const_cast<void *>(lib->get_func_ptr("viWrite"))
        );
        func_ptrs.FindRsrc = reinterpret_cast<viFindRsrc_t>(
            const_cast<void *>(lib->get_func_ptr("viFindRsrc"))
        );
        func_ptrs.FindNext = reinterpret_cast<viFindNext_t>(
            const_cast<void *>(lib->get_func_ptr("viFindNext"))
        );
        func_ptrs.StatusDesc = reinterpret_cast<viStatusDesc_t>(
            const_cast<void *>(lib->get_func_ptr("viStatusDesc"))
        );
        func_ptrs.SetAttribute = reinterpret_cast<viSetAttribute_t>(
            const_cast<void *>(lib->get_func_ptr("viSetAttribute"))
        );
        func_ptrs.GetAttribute = reinterpret_cast<viGetAttribute_t>(
            const_cast<void *>(lib->get_func_ptr("viGetAttribute"))
        );
    }

    /// @brief Attempts to load the VISA shared library.
    /// @return A shared pointer to the API wrapper and an error if loading failed.
    static std::pair<std::shared_ptr<API>, xerrors::Error> load() {
        auto lib = std::make_unique<xlib::SharedLib>(VISA_LIBRARY_NAME);
        if (!lib->load()) return {nullptr, LOAD_ERROR};
        return {std::make_shared<API>(std::move(lib)), xerrors::NIL};
    }

    /// @brief Opens the default resource manager.
    [[nodiscard]] ViStatus open_default_rm(ViSession *rm) const {
        if (func_ptrs.OpenDefaultRM == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.OpenDefaultRM(rm);
    }

    /// @brief Closes a session.
    [[nodiscard]] ViStatus close(ViSession session) const {
        if (func_ptrs.Close == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.Close(session);
    }

    /// @brief Opens a session to a resource.
    [[nodiscard]] ViStatus open(
        ViSession rm,
        ViRsrc resource_name,
        ViUInt32 access_mode,
        ViUInt32 timeout,
        ViSession *session
    ) const {
        if (func_ptrs.Open == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.Open(rm, resource_name, access_mode, timeout, session);
    }

    /// @brief Reads data from a device.
    [[nodiscard]] ViStatus read(
        ViSession session,
        ViBuf buffer,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) const {
        if (func_ptrs.Read == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.Read(session, buffer, count, ret_count);
    }

    /// @brief Writes data to a device.
    [[nodiscard]] ViStatus write(
        ViSession session,
        ViBuf buffer,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) const {
        if (func_ptrs.Write == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.Write(session, buffer, count, ret_count);
    }

    /// @brief Finds resources matching a pattern.
    [[nodiscard]] ViStatus find_rsrc(
        ViSession rm,
        ViString expr,
        ViFindList *find_list,
        ViUInt32 *ret_count,
        ViChar *desc
    ) const {
        if (func_ptrs.FindRsrc == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.FindRsrc(rm, expr, find_list, ret_count, desc);
    }

    /// @brief Finds the next resource in a find list.
    [[nodiscard]] ViStatus find_next(ViFindList find_list, ViChar *desc) const {
        if (func_ptrs.FindNext == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.FindNext(find_list, desc);
    }

    /// @brief Gets a description for a status code.
    [[nodiscard]] ViStatus status_desc(
        ViSession session,
        ViStatus status,
        ViChar *desc
    ) const {
        if (func_ptrs.StatusDesc == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.StatusDesc(session, status, desc);
    }

    /// @brief Sets an attribute on a session.
    [[nodiscard]] ViStatus set_attribute(
        ViSession session,
        ViUInt32 attribute,
        ViUInt32 value
    ) const {
        if (func_ptrs.SetAttribute == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.SetAttribute(session, attribute, value);
    }

    /// @brief Gets an attribute from a session.
    [[nodiscard]] ViStatus get_attribute(
        ViSession session,
        ViUInt32 attribute,
        void *value
    ) const {
        if (func_ptrs.GetAttribute == nullptr) return VI_ERROR_NSUP_OPER;
        return func_ptrs.GetAttribute(session, attribute, value);
    }
};

}