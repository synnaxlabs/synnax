// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 3/2/25.
//

#pragma once

#include "driver/labjack/ljm/LabJackM.h"
#include "x/cpp/xlib/xlib.h"
#include "x/cpp/xos/xos.h"

namespace ljm {
const auto LOAD_ERROR = xerrors::Error(
    xlib::LOAD_ERROR,
    "failed load LJM shared libraries. Are they installed?"
);


class API {
    struct FunctionPointers {
        decltype(&LJM_eStreamRead) eStreamRead;
        decltype(&LJM_eStreamStop) eStreamStop;
        decltype(&LJM_eWriteAddress) eWriteAddress;
        decltype(&LJM_eWriteAddresses) eWriteAddresses;
        decltype(&LJM_StartInterval) StartInterval;
        decltype(&LJM_eWriteName) eWriteName;
        decltype(&LJM_NamesToAddresses) NamesToAddresses;
        decltype(&LJM_ErrorToString) ErrorToString;
        decltype(&LJM_eWriteNames) eWriteNames;
        decltype(&LJM_ListAll) ListAll;
        decltype(&LJM_Open) Open;
        decltype(&LJM_Close) Close;
        decltype(&LJM_eReadNames) eReadNames;
        decltype(&LJM_WaitForNextInterval) WaitForNextInterval;
        decltype(&LJM_eStreamStart) eStreamStart;
    };

    std::unique_ptr<xlib::SharedLib> lib;
    FunctionPointers function_pointers_;


public:
    explicit API(std::unique_ptr<xlib::SharedLib> lib_) : lib(std::move(lib_)) {
        memset(&function_pointers_, 0, sizeof(function_pointers_));
        function_pointers_.eStreamRead = reinterpret_cast<decltype(&LJM_eStreamRead)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamRead")));
        function_pointers_.eStreamStop = reinterpret_cast<decltype(&LJM_eStreamStop)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamStop")));
        function_pointers_.eWriteAddress = reinterpret_cast<decltype(&LJM_eWriteAddress
        )>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteAddress")));
        function_pointers_.eWriteAddresses = reinterpret_cast<decltype(&
            LJM_eWriteAddresses)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteAddresses")));
        function_pointers_.StartInterval = reinterpret_cast<decltype(&LJM_StartInterval
        )>(
            const_cast<void *>(lib->get_func_ptr("LJM_StartInterval")));
        function_pointers_.eWriteName = reinterpret_cast<decltype(&LJM_eWriteName)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteName")));
        function_pointers_.NamesToAddresses = reinterpret_cast<decltype(&
            LJM_NamesToAddresses)>(
            const_cast<void *>(lib->get_func_ptr("LJM_NamesToAddresses")));
        function_pointers_.ErrorToString = reinterpret_cast<decltype(&LJM_ErrorToString
        )>(
            const_cast<void *>(lib->get_func_ptr("LJM_ErrorToString")));
        function_pointers_.eWriteNames = reinterpret_cast<decltype(&LJM_eWriteNames)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eWriteNames")));
        function_pointers_.ListAll = reinterpret_cast<decltype(&LJM_ListAll)>(
            const_cast<void *>(lib->get_func_ptr("LJM_ListAll")));
        function_pointers_.Open = reinterpret_cast<decltype(&LJM_Open)>(
            const_cast<void *>(lib->get_func_ptr("LJM_Open")));
        function_pointers_.Close = reinterpret_cast<decltype(&LJM_Close)>(
            const_cast<void *>(lib->get_func_ptr("LJM_Close")));
        function_pointers_.eReadNames = reinterpret_cast<decltype(&LJM_eReadNames)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eReadNames")));
        function_pointers_.WaitForNextInterval = reinterpret_cast<decltype(&
            LJM_WaitForNextInterval)>(
            const_cast<void *>(lib->get_func_ptr("LJM_WaitForNextInterval")));
        function_pointers_.eStreamStart = reinterpret_cast<decltype(&LJM_eStreamStart)>(
            const_cast<void *>(lib->get_func_ptr("LJM_eStreamStart")));
    }


    static std::pair<std::shared_ptr<API>, xerrors::Error> load() {
        if (xos::get() != xos::WINDOWS) return {nullptr, xerrors::NIL};
        auto lib = std::make_unique<xlib::SharedLib>("LabjackM.dll");
        if (!lib->load()) return {nullptr, LOAD_ERROR};
        return {std::make_shared<API>(std::move(lib)), xerrors::NIL};
    }

    LJM_ERROR_RETURN eStreamRead(
        int Handle,
        double *aData,
        int *DeviceScanBacklog,
        int *LJMScanBacklog
    ) {
        return function_pointers_.eStreamRead(Handle, aData, DeviceScanBacklog,
                                              LJMScanBacklog);
    }

    LJM_ERROR_RETURN eStreamStop(int Handle) {
        return function_pointers_.eStreamStop(Handle);
    }

    LJM_ERROR_RETURN eWriteAddress(int Handle, int Address, int Type, double Value) {
        return function_pointers_.eWriteAddress(Handle, Address, Type, Value);
    }

    LJM_ERROR_RETURN eWriteAddresses(
        int Handle,
        int NumFrames,
        const int *aAddresses,
        const int *aTypes,
        const double *aValues,
        int *ErrorAddress
    ) {
        return function_pointers_.eWriteAddresses(Handle, NumFrames, aAddresses, aTypes,
                                                  aValues, ErrorAddress);
    }

    LJM_ERROR_RETURN StartInterval(int IntervalHandle, int Microseconds) {
        return function_pointers_.StartInterval(IntervalHandle, Microseconds);
    }

    LJM_ERROR_RETURN eWriteName(int Handle, const char *Name, double Value) {
        return function_pointers_.eWriteName(Handle, Name, Value);
    }

    LJM_ERROR_RETURN NamesToAddresses(
        int NumFrames,
        const char **aNames,
        int *aAddresses,
        int *aTypes
    ) {
        return function_pointers_.NamesToAddresses(NumFrames, aNames, aAddresses,
                                                   aTypes);
    }

    LJM_VOID_RETURN ErrorToString(int ErrorCode, char *ErrorString) {
        return function_pointers_.ErrorToString(ErrorCode, ErrorString);
    }

    LJM_ERROR_RETURN eWriteNames(
        int Handle,
        int NumFrames,
        const char **aNames,
        const double *aValues,
        int *ErrorAddress
    ) {
        return function_pointers_.eWriteNames(Handle, NumFrames, aNames, aValues,
                                              ErrorAddress);
    }

    LJM_ERROR_RETURN ListAll(
        int DeviceType,
        int ConnectionType,
        int *NumFound,
        int *aDeviceTypes,
        int *aConnectionTypes,
        int *aSerialNumbers,
        int *aIPAddresses
    ) {
        return function_pointers_.ListAll(DeviceType, ConnectionType, NumFound,
                                          aDeviceTypes, aConnectionTypes,
                                          aSerialNumbers, aIPAddresses);
    }

    LJM_ERROR_RETURN Open(
        int DeviceType,
        int ConnectionType,
        const char *Identifier,
        int *Handle
    ) {
        return function_pointers_.Open(DeviceType, ConnectionType, Identifier, Handle);
    }

    LJM_ERROR_RETURN Close(int Handle) {
        return function_pointers_.Close(Handle);
    }

    LJM_ERROR_RETURN eReadNames(
        int Handle,
        int NumFrames,
        const char **aNames,
        double *aValues,
        int *ErrorAddress
    ) {
        return function_pointers_.eReadNames(Handle, NumFrames, aNames, aValues,
                                             ErrorAddress);
    }

    LJM_ERROR_RETURN WaitForNextInterval(int IntervalHandle, int *SkippedIntervals) {
        return function_pointers_.WaitForNextInterval(IntervalHandle, SkippedIntervals);
    }

    LJM_ERROR_RETURN eStreamStart(
        int Handle,
        int ScansPerRead,
        int NumAddresses,
        const int *aScanList,
        double *ScanRate
    ) {
        return function_pointers_.eStreamStart(Handle, ScansPerRead, NumAddresses,
                                               aScanList, ScanRate);
    }
};
}
