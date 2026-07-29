// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

/// @brief crash implements process-wide handlers that dump a stack trace to stderr
/// when the process is about to die abnormally, so that field failures leave a
/// diagnosable trace in the driver logs instead of a silent exit.
namespace x::crash {
/// @brief maximum number of stack frames captured in a trace.
constexpr int MAX_FRAMES = 64;

/// @brief installs the process-wide crash handlers. Should be called once, as early
/// in process startup as possible.
///
/// Two classes of failure are covered:
///   - Fatal signals: SIGSEGV, SIGABRT, SIGILL, SIGFPE, and SIGBUS on POSIX;
///     unhandled structured exceptions (e.g. access violations) on Windows.
///   - Unhandled C++ exceptions, via std::set_terminate. The active exception's
///     what() message is included when available.
///
/// In both cases a stack trace is written to stderr (the same stream the Driver logs
/// to) and the process is then terminated with its normal crash semantics: on POSIX the
/// triggering signal is re-raised against the default handler so a core dump and
/// 128+signal exit code are still produced.
///
/// Trace quality depends on the available symbols: optimized, stripped release builds
/// yield addresses that must be resolved offline (addr2line on the unstripped binary,
/// or a matching .pdb on Windows). See driver/DEBUGGING.md.
///
/// The signal path is async-signal-safe; it allocates no memory and calls only
/// async-signal-safe functions. It runs on an alternate signal stack (POSIX) / with a
/// reserved stack guarantee (Windows) so that stack-overflow crashes can still be
/// traced. Does NOT install handlers for SIGINT or SIGTERM, which x::shutdown owns for
/// graceful shutdown.
///
/// @param program_name name printed in the crash banner. Truncated to 255 bytes.
void install(const std::string &program_name);

/// @brief captures a formatted, symbolized stack trace of the calling thread and
/// returns it, one frame per line. It uses the same capture and symbolization machinery
/// as the crash handlers (backtrace + demangling on POSIX, DbgHelp on Windows), so it
/// doubles as a way to exercise that machinery outside of a crash — both for voluntary
/// diagnostics (logging a trace on a suspicious but non-fatal path) and for testing.
///
/// Unlike the signal handler's trace, this allocates and is NOT async-signal-safe; do
/// not call it from a signal handler. Symbol name resolution depends on the available
/// symbols (see install): without symbols, frames are addresses only.
std::string capture_trace();
}
