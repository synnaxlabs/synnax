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
#include "x/cpp/xos/xos.h"

#include "driver/labjack/ljm/LabJackM.h"

namespace ljm {
#ifdef __APPLE__
const std::string LJM_LIBRARY_NAME = "/usr/local/lib/libLabJackM.dylib";
#else
const std::string LJM_LIBRARY_NAME = "LabjackM.dll";
#endif

const auto LOAD_ERROR = xerrors::Error(
    xlib::LOAD_ERROR,
    "failed load LJM shared libraries. Are they installed? Expected to find them "
    "at " +
        LJM_LIBRARY_NAME
);

/// @brief API wrapped on top of LJM functions that the Synnax driver requires.
class API {
    /// @brief Function pointers to the LJM functions.
    struct FunctionPointers {
        decltype(&LJM_eStreamRead) eStreamRead;
        decltype(&LJM_eStreamStop) eStreamStop;
        decltype(&LJM_eWriteAddress) eWriteAddress;
        decltype(&LJM_eWriteAddresses) eWriteAddresses;
        decltype(&LJM_StartInterval) StartInterval;
        decltype(&LJM_CleanInterval) CleanInterval;
        decltype(&LJM_eWriteName) eWriteName;
        decltype(&LJM_NamesToAddresses) NamesToAddresses;
        decltype(&LJM_ErrorToString) ErrorToString;
        decltype(&LJM_eWriteNames) eWriteNames;
        decltype(&LJM_ListAll) ListAll;
        decltype(&LJM_Open) Open;
        decltype(&LJM_Close) Close;
        decltype(&LJM_eReadNames) eReadNames;
        decltype(&LJM_eReadName) eReadName;
        decltype(&LJM_WaitForNextInterval) WaitForNextInterval;
        decltype(&LJM_eStreamStart) eStreamStart;
    };

    /// @brief Shared library handle.
    std::unique_ptr<xlib::SharedLib> lib;
    FunctionPointers func_ptrs;

public:
    explicit API(std::unique_ptr<xlib::SharedLib> lib_): lib(std::move(lib_)) {
        memset(&func_ptrs, 0, sizeof(func_ptrs));
        func_ptrs.eStreamRead = reinterpret_cast<decltype(&LJM_eStreamRead)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamRead"))
        );
        func_ptrs.eStreamStop = reinterpret_cast<decltype(&LJM_eStreamStop)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamStop"))
        );
        func_ptrs.eWriteAddress = reinterpret_cast<decltype(&LJM_eWriteAddress)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteAddress"))
        );
        func_ptrs.eWriteAddresses = reinterpret_cast<decltype(&LJM_eWriteAddresses)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteAddresses"))
        );
        func_ptrs.StartInterval = reinterpret_cast<decltype(&LJM_StartInterval)>(
            const_cast<void *>(lib->get_func_ptr("LJM_StartInterval"))
        );
        func_ptrs.CleanInterval = reinterpret_cast<decltype(&LJM_CleanInterval)>(
            const_cast<void *>(lib->get_func_ptr("LJM_CleanInterval"))
        );
        func_ptrs.eWriteName = reinterpret_cast<decltype(&LJM_eWriteName)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteName"))
        );
        func_ptrs.NamesToAddresses = reinterpret_cast<decltype(&LJM_NamesToAddresses)>(
            const_cast<void *>(lib->get_func_ptr("LJM_NamesToAddresses"))
        );
        func_ptrs.ErrorToString = reinterpret_cast<decltype(&LJM_ErrorToString)>(
            const_cast<void *>(lib->get_func_ptr("LJM_ErrorToString"))
        );
        func_ptrs.eWriteNames = reinterpret_cast<decltype(&LJM_eWriteNames)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteNames"))
        );
        func_ptrs.eWriteName = reinterpret_cast<decltype(&LJM_eWriteName)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteName"))
        );
        func_ptrs.ListAll = reinterpret_cast<decltype(&LJM_ListAll)>(
            const_cast<void *>(lib->get_func_ptr("LJM_ListAll"))
        );
        func_ptrs.Open = reinterpret_cast<decltype(&LJM_Open)>(
            const_cast<void *>(lib->get_func_ptr("LJM_Open"))
        );
        func_ptrs.Close = reinterpret_cast<decltype(&LJM_Close)>(
            const_cast<void *>(lib->get_func_ptr("LJM_Close"))
        );
        func_ptrs.eReadNames = reinterpret_cast<decltype(&LJM_eReadNames)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eReadNames"))
        );
        func_ptrs
            .WaitForNextInterval = reinterpret_cast<decltype(&LJM_WaitForNextInterval)>(
            const_cast<void *>(lib->get_func_ptr("LJM_WaitForNextInterval"))
        );
        func_ptrs.eReadName = reinterpret_cast<decltype(&LJM_eReadName)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eReadName"))
        );
        func_ptrs.eStreamStart = reinterpret_cast<decltype(&LJM_eStreamStart)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamStart"))
        );
    }

    static std::pair<std::shared_ptr<API>, xerrors::Error> load() {
        auto lib = std::make_unique<xlib::SharedLib>(LJM_LIBRARY_NAME);
        if (!lib->load()) return {nullptr, LOAD_ERROR};
        return {std::make_shared<API>(std::move(lib)), xerrors::NIL};
    }

    [[nodiscard]] LJM_ERROR_RETURN e_stream_read(
        const int dev_handle,
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog
    ) const {
        return this->func_ptrs
            .eStreamRead(dev_handle, data, dev_scan_backlog, ljm_scan_backlog);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_stream_stop(const int dev_handle) const {
        return this->func_ptrs.eStreamStop(dev_handle);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_write_addr(
        const int dev_handle,
        const int addr,
        const int type,
        const double value
    ) const {
        return this->func_ptrs.eWriteAddress(dev_handle, addr, type, value);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_write_addrs(
        const int dev_handle,
        const int num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addr
    ) const {
        return this->func_ptrs
            .eWriteAddresses(dev_handle, num_frames, addrs, types, values, error_addr);
    }

    [[nodiscard]] LJM_ERROR_RETURN
    start_interval(const int interval_handle, const int microseconds) const {
        return this->func_ptrs.StartInterval(interval_handle, microseconds);
    }

    [[nodiscard]] LJM_ERROR_RETURN clean_interval(const int interval_handle) const {
        return this->func_ptrs.CleanInterval(interval_handle);
    }

    [[nodiscard]] LJM_ERROR_RETURN
    e_write_name(const int dev_handle, const char *name, const double value) const {
        return this->func_ptrs.eWriteName(dev_handle, name, value);
    }

    [[nodiscard]] LJM_ERROR_RETURN names_to_addrs(
        const int num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const {
        return this->func_ptrs.NamesToAddresses(num_frames, names, addrs, types);
    }

    LJM_VOID_RETURN err_to_string(const int err_code, char *err_string) const {
        return this->func_ptrs.ErrorToString(err_code, err_string);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_write_names(
        const int dev_handle,
        const int num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const {
        return this->func_ptrs
            .eWriteNames(dev_handle, num_frames, names, values, err_addr);
    }

    [[nodiscard]] LJM_ERROR_RETURN list_all(
        const int dev_type,
        const int conn_type,
        int *num_found,
        int *dev_types,
        int *conn_types,
        int *serial_numbers,
        int *a_ip_addrs
    ) const {
        return this->func_ptrs.ListAll(
            dev_type,
            conn_type,
            num_found,
            dev_types,
            conn_types,
            serial_numbers,
            a_ip_addrs
        );
    }

    [[nodiscard]] LJM_ERROR_RETURN open(
        const int dev_type,
        const int conn_type,
        const char *id,
        int *dev_handle
    ) const {
        return func_ptrs.Open(dev_type, conn_type, id, dev_handle);
    }

    [[nodiscard]] LJM_ERROR_RETURN close(const int dev_handle) const {
        return func_ptrs.Close(dev_handle);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_read_names(
        const int dev_handle,
        const int num_frames,
        const char **a_names,
        double *a_values,
        int *err_addr
    ) const {
        return this->func_ptrs
            .eReadNames(dev_handle, num_frames, a_names, a_values, err_addr);
    }

    [[nodiscard]] LJM_ERROR_RETURN
    e_read_name(const int dev_handle, const char *name, double *value) const {
        return this->func_ptrs.eReadName(dev_handle, name, value);
    }

    [[nodiscard]] LJM_ERROR_RETURN
    wait_for_next_interval(const int interval_handle, int *skipped_intervals) const {
        return this->func_ptrs.WaitForNextInterval(interval_handle, skipped_intervals);
    }

    [[nodiscard]] LJM_ERROR_RETURN e_stream_start(
        const int dev_handle,
        const int scans_per_read,
        const int num_addrs,
        const int *scan_list,
        double *scan_rate
    ) const {
        return this->func_ptrs
            .eStreamStart(dev_handle, scans_per_read, num_addrs, scan_list, scan_rate);
    }
};
}
