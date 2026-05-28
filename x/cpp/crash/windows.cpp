// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
// clang-format off
// dbghelp.h must be included after windows.h.
#include <windows.h>
#include <dbghelp.h>
// clang-format on

#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <mutex>
#include <string>

#include "x/cpp/crash/crash.h"

namespace x::crash {
namespace {
/// @brief maximum symbol name length resolved per frame.
constexpr DWORD MAX_SYMBOL_NAME = 255;

/// @brief name printed in the crash banner. Empty until install sets it.
char program_name[256] = {};

/// @brief guards against concurrent and recursive crash handling. The driver is
/// multi-threaded, so a fault can hit several threads at once; without this, each would
/// dump its own trace and the output would interleave into an unreadable flood. Only
/// the first thread to arrive dumps; any other crashing thread blocks here instead.
std::atomic_flag crash_in_progress;

/// @brief lets the first crashing thread through; any later thread blocks forever so
/// the trace is written once.
void claim_crash() {
    if (crash_in_progress.test_and_set())
        while (true)
            Sleep(1000);
}

LONG WINAPI exception_filter(EXCEPTION_POINTERS *info) {
    claim_crash();
    std::fprintf(
        stderr,
        "*** %s crashed: exception code 0x%lx ***\nstack trace:\n",
        program_name,
        info->ExceptionRecord->ExceptionCode
    );
    std::fputs(capture_trace().c_str(), stderr);
    std::fflush(stderr);
    // Return EXCEPTION_CONTINUE_SEARCH so the OS default unhandled-exception path
    // (Windows Error Reporting, minidump, JIT debugger) still runs after the trace is
    // written, mirroring the POSIX handler's re-raise that preserves crash semantics.
    return EXCEPTION_CONTINUE_SEARCH;
}

[[noreturn]] void terminate_handler() {
    claim_crash();
    std::fprintf(stderr, "*** %s terminated: unhandled exception ***\n", program_name);
    if (const std::exception_ptr eptr = std::current_exception()) {
        try {
            std::rethrow_exception(eptr);
        } catch (const std::exception &e) {
            std::fprintf(stderr, "  what(): %s\n", e.what());
        } catch (...) { std::fprintf(stderr, "  (non-standard exception)\n"); }
    }
    std::fputs("stack trace:\n", stderr);
    std::fputs(capture_trace().c_str(), stderr);
    std::fflush(stderr);
    std::abort();
}
}

void install(const std::string &name) {
    std::snprintf(program_name, sizeof(program_name), "%s", name.c_str());
    // Reserve stack for the calling thread so a stack-overflow exception still leaves
    // room to run the handler instead of failing to report.
    ULONG guarantee = 64 * 1024;
    SetThreadStackGuarantee(&guarantee);
    SetUnhandledExceptionFilter(exception_filter);
    std::set_terminate(terminate_handler);
}

std::string capture_trace() {
    // DbgHelp uses a single global, single-threaded symbol session, so concurrent calls
    // race (one's SymCleanup tears down another's session). Serialize it; POSIX
    // backtrace is reentrant and needs no guard. The crash path is already
    // single-threaded via claim_crash, so this only matters for concurrent diagnostic
    // calls.
    static std::mutex sym_mu;
    const std::lock_guard lock(sym_mu);
    const HANDLE process = GetCurrentProcess();
    SymSetOptions(SYMOPT_UNDNAME | SYMOPT_DEFERRED_LOADS | SYMOPT_LOAD_LINES);
    SymInitialize(process, nullptr, TRUE);

    void *frames[MAX_FRAMES];
    const USHORT n = CaptureStackBackTrace(0, MAX_FRAMES, frames, nullptr);

    // SYMBOL_INFO is variable-length (the name trails the struct); allocate an aligned
    // buffer large enough for the struct plus the name, per the DbgHelp contract.
    ULONG64 buffer
        [(sizeof(SYMBOL_INFO) + MAX_SYMBOL_NAME + sizeof(ULONG64) - 1) /
         sizeof(ULONG64)];
    auto *symbol = reinterpret_cast<SYMBOL_INFO *>(buffer);
    symbol->SizeOfStruct = sizeof(SYMBOL_INFO);
    symbol->MaxNameLen = MAX_SYMBOL_NAME;

    std::string out;
    char line[512];
    for (USHORT i = 0; i < n; i++) {
        const auto addr = reinterpret_cast<DWORD64>(frames[i]);
        if (SymFromAddr(process, addr, nullptr, symbol))
            std::snprintf(
                line,
                sizeof(line),
                "  %u: %s [0x%llx]\n",
                i,
                symbol->Name,
                static_cast<unsigned long long>(addr)
            );
        else
            std::snprintf(
                line,
                sizeof(line),
                "  %u: [0x%llx]\n",
                i,
                static_cast<unsigned long long>(addr)
            );
        out += line;
    }
    SymCleanup(process);
    return out;
}
}
