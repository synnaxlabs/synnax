// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include "glog/logging.h"
#include <windows.h>

#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {
namespace {
using AvSetMmThreadCharacteristicsWFn = HANDLE(WINAPI *)(LPCWSTR, LPDWORD);
using AvSetMmThreadPriorityFn = BOOL(WINAPI *)(HANDLE, int);
using AvRevertMmThreadCharacteristicsFn = BOOL(WINAPI *)(HANDLE);

constexpr int AVRT_PRIORITY_CRITICAL = 2;

struct MMCSSFunctions {
    HMODULE module = nullptr;
    AvSetMmThreadCharacteristicsWFn set_characteristics = nullptr;
    AvSetMmThreadPriorityFn set_priority = nullptr;
    AvRevertMmThreadCharacteristicsFn revert = nullptr;

    bool available() const {
        return this->module && this->set_characteristics && this->set_priority;
    }
};

MMCSSFunctions load_mmcss() {
    MMCSSFunctions funcs;
    funcs.module = LoadLibraryW(L"Avrt.dll");
    if (!funcs.module) return funcs;
    // Suppress C4191: GetProcAddress returns FARPROC which requires cast to specific
    // function pointer types. This is the standard pattern for runtime DLL loading.
#pragma warning(push)
#pragma warning(disable : 4191)
    funcs.set_characteristics = reinterpret_cast<AvSetMmThreadCharacteristicsWFn>(
        GetProcAddress(funcs.module, "AvSetMmThreadCharacteristicsW")
    );
    funcs.set_priority = reinterpret_cast<AvSetMmThreadPriorityFn>(
        GetProcAddress(funcs.module, "AvSetMmThreadPriority")
    );
    funcs.revert = reinterpret_cast<AvRevertMmThreadCharacteristicsFn>(
        GetProcAddress(funcs.module, "AvRevertMmThreadCharacteristics")
    );
#pragma warning(pop)
    return funcs;
}

const MMCSSFunctions &get_mmcss() {
    static MMCSSFunctions funcs = load_mmcss();
    return funcs;
}

thread_local HANDLE mmcss_task_handle = nullptr;

bool apply_mmcss() {
    const auto &funcs = get_mmcss();
    if (!funcs.available()) {
        LOG(WARNING) << "[xthread] MMCSS not available (Avrt.dll not found)";
        return false;
    }
    if (mmcss_task_handle) {
        funcs.revert(mmcss_task_handle);
        mmcss_task_handle = nullptr;
    }
    DWORD task_index = 0;
    mmcss_task_handle = funcs.set_characteristics(L"Pro Audio", &task_index);
    if (!mmcss_task_handle) {
        LOG(WARNING) << "[xthread] Failed to set MMCSS Pro Audio: " << GetLastError();
        return false;
    }
    if (!funcs.set_priority(mmcss_task_handle, AVRT_PRIORITY_CRITICAL)) {
        LOG(WARNING) << "[xthread] Failed to set MMCSS priority: " << GetLastError();
        return false;
    }
    VLOG(1) << "[xthread] Set MMCSS Pro Audio with critical priority";
    return true;
}

void apply_thread_priority(int priority) {
    int win_priority;
    if (priority >= 90)
        win_priority = THREAD_PRIORITY_TIME_CRITICAL;
    else if (priority >= 70)
        win_priority = THREAD_PRIORITY_HIGHEST;
    else if (priority >= 50)
        win_priority = THREAD_PRIORITY_ABOVE_NORMAL;
    else
        win_priority = THREAD_PRIORITY_NORMAL;
    if (!SetThreadPriority(GetCurrentThread(), win_priority))
        LOG(WARNING) << "[xthread] Failed to set thread priority: " << GetLastError();
    else
        VLOG(1) << "[xthread] Set thread priority to " << win_priority;
}
}

Capabilities get_capabilities() {
    static Capabilities caps = [] {
        Capabilities c;
        c.priority_scheduling = {true, true};
        c.mmcss = {true, get_mmcss().available()};
        c.cpu_affinity = {true, true};
        return c;
    }();
    return caps;
}

std::string Capabilities::permissions_guidance() const {
    std::string guidance;
    if (this->mmcss.missing_permissions()) {
        guidance += "MMCSS not available. Ensure Windows Multimedia Class Scheduler ";
        guidance += "service is running (Avrt.dll).";
    }
    return guidance;
}

bool has_support() {
    return get_capabilities().any();
}

errors::Error apply_config(const Config &cfg) {
    if (cfg.enabled) {
        bool used_mmcss = false;
        if (cfg.use_mmcss) used_mmcss = apply_mmcss();
        if (!used_mmcss) apply_thread_priority(cfg.priority);
    }

    if (cfg.cpu_affinity >= 0) {
        const DWORD_PTR mask = static_cast<DWORD_PTR>(1) << cfg.cpu_affinity;
        if (!SetThreadAffinityMask(GetCurrentThread(), mask))
            LOG(WARNING) << "[xthread] Failed to set CPU affinity to "
                         << cfg.cpu_affinity << ": " << GetLastError();
        else
            VLOG(1) << "[xthread] Set CPU affinity to core " << cfg.cpu_affinity;
    }

    if (cfg.lock_memory)
        LOG(WARNING) << "[xthread] Memory locking on Windows requires VirtualLock API";

    return errors::NIL;
}
}
