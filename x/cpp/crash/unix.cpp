// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <csignal>
#include <cstdlib>
#include <cstring>
#include <exception>
#include <sstream>
#include <string>

#include <cxxabi.h>
#include <execinfo.h>
#include <unistd.h>

#include "x/cpp/crash/crash.h"

namespace x::crash {
namespace {
/// @brief size of the alternate signal stack. Must be large enough to run the
/// trace-capturing handler when the normal stack is exhausted (the stack-overflow
/// case). 64 KiB is comfortably above MINSIGSTKSZ on every supported platform.
constexpr size_t ALT_STACK_SIZE = 64 * 1024;

/// @brief backing storage for the alternate signal stack. Static so it outlives
/// install() and remains valid when a signal fires.
alignas(16) char alt_stack[ALT_STACK_SIZE];

/// @brief name printed in the crash banner. A fixed buffer rather than a std::string
/// so that the signal handler can read it without touching the allocator. Empty until
/// install sets it.
char program_name[256] = {};

/// @brief guards against concurrent and recursive crash handling. The driver is
/// multi-threaded, so a fault (e.g. a use-after-free) can hit several threads at once;
/// without this, each would dump its own trace and the output would interleave into an
/// unreadable flood. Only the first thread to arrive dumps and terminates the process;
/// any other crashing thread blocks here instead.
std::atomic_flag crash_in_progress;

/// @brief lets the first crashing thread through; any later thread blocks forever (the
/// first thread takes the process down). This serializes crash handling so the trace is
/// written once and a faulting thread never returns to re-run the faulting instruction.
void claim_crash() {
    if (crash_in_progress.test_and_set())
        while (true)
            pause();
}

/// @brief writes a null-terminated string to fd. Async-signal-safe: uses only strlen
/// and write. The write result is intentionally unchecked, as there is no recovery
/// available mid-crash.
void write_str(const int fd, const char *s) {
    if (::write(fd, s, std::strlen(s)) < 0) return;
}

/// @brief returns a static, human-readable name for a fatal signal. The returned
/// pointer is a string literal, so reading it from a signal handler is safe.
const char *signal_name(const int sig) {
    switch (sig) {
        case SIGSEGV:
            return "SIGSEGV (segmentation fault)";
        case SIGABRT:
            return "SIGABRT (abort)";
        case SIGILL:
            return "SIGILL (illegal instruction)";
        case SIGFPE:
            return "SIGFPE (floating-point exception)";
        case SIGBUS:
            return "SIGBUS (bus error)";
        default:
            return "fatal signal";
    }
}

/// @brief captures and writes the current stack trace to fd. Async-signal-safe:
/// backtrace and backtrace_symbols_fd do not allocate. Symbol names are left mangled
/// because demangling requires the allocator.
void print_trace_safe(const int fd) {
    void *frames[MAX_FRAMES];
    const int n = backtrace(frames, MAX_FRAMES);
    backtrace_symbols_fd(frames, n, fd);
}

/// @brief extracts the mangled symbol token from a single backtrace_symbols line and
/// returns the line with that token replaced by its demangled form. Returns the line
/// unchanged when no symbol can be demangled. NOT async-signal-safe; only for the
/// terminate path, which runs in an ordinary (non-signal) context.
std::string demangle_line(const std::string &line) {
#ifdef __APPLE__
    // Format: "<idx>  <module>  <addr>  <mangled>  +  <offset>"
    std::istringstream iss(line);
    std::string idx, module, addr, mangled;
    if (!(iss >> idx >> module >> addr >> mangled)) return line;
#else
    // Format: "<module>(<mangled>+0x<offset>) [0x<addr>]"
    const auto lp = line.find('(');
    if (lp == std::string::npos) return line;
    const auto plus = line.find('+', lp);
    if (plus == std::string::npos || plus <= lp + 1) return line;
    const std::string mangled = line.substr(lp + 1, plus - lp - 1);
#endif
    int status = 0;
    char *demangled = abi::__cxa_demangle(mangled.c_str(), nullptr, nullptr, &status);
    if (status != 0 || demangled == nullptr) {
        std::free(demangled);
        return line;
    }
    std::string out = line;
    const auto pos = out.find(mangled);
    if (pos != std::string::npos) out.replace(pos, mangled.size(), demangled);
    std::free(demangled);
    return out;
}

void signal_handler(const int sig) {
    claim_crash();
    write_str(STDERR_FILENO, "*** ");
    write_str(STDERR_FILENO, program_name);
    write_str(STDERR_FILENO, " crashed: ");
    write_str(STDERR_FILENO, signal_name(sig));
    write_str(STDERR_FILENO, " ***\nstack trace:\n");
    print_trace_safe(STDERR_FILENO);
    // Restore the default disposition and re-raise so the process dies with normal
    // crash semantics (core dump, 128+signal exit). claim_crash has parked any other
    // crashing threads, so exactly one full trace is written before we go down.
    std::signal(sig, SIG_DFL);
    std::raise(sig);
}

[[noreturn]] void terminate_handler() {
    claim_crash();
    write_str(STDERR_FILENO, "*** ");
    write_str(STDERR_FILENO, program_name);
    write_str(STDERR_FILENO, " terminated: unhandled exception ***\n");
    if (const std::exception_ptr eptr = std::current_exception()) {
        try {
            std::rethrow_exception(eptr);
        } catch (const std::exception &e) {
            write_str(STDERR_FILENO, "  what(): ");
            write_str(STDERR_FILENO, e.what());
            write_str(STDERR_FILENO, "\n");
        } catch (...) { write_str(STDERR_FILENO, "  (non-standard exception)\n"); }
    }
    write_str(STDERR_FILENO, "stack trace:\n");
    const std::string trace = capture_trace();
    if (trace.empty())
        print_trace_safe(STDERR_FILENO);
    else
        write_str(STDERR_FILENO, trace.c_str());
    // SIGABRT is one of our handled signals; reset it so abort() dies cleanly instead
    // of re-entering signal_handler and printing a second, redundant banner.
    std::signal(SIGABRT, SIG_DFL);
    std::abort();
}

void install_signal(const int sig) {
    struct sigaction sa = {};
    sa.sa_handler = signal_handler;
    sigemptyset(&sa.sa_mask);
    // SA_ONSTACK runs the handler on the alternate stack so a stack-overflow crash can
    // still be traced. We deliberately do NOT set SA_RESETHAND: keeping the handler
    // installed means that in a multi-threaded crash the other faulting threads
    // re-enter it and park in claim_crash, rather than hitting the default action and
    // killing the process before the first thread finishes writing its trace. The first
    // thread restores the default disposition itself before re-raising.
    sa.sa_flags = SA_ONSTACK;
    sigaction(sig, &sa, nullptr);
}
}

void install(const std::string &name) {
    std::snprintf(program_name, sizeof(program_name), "%s", name.c_str());
    // Pre-warm the unwinder: glibc's first backtrace() lazily dlopens libgcc and may
    // allocate. Doing it here keeps the signal path allocation-free, avoiding a
    // deadlock if the crash is taken while the allocator lock is held (e.g. a fault in
    // malloc).
    void *warm[MAX_FRAMES];
    backtrace(warm, MAX_FRAMES);
    stack_t ss = {};
    ss.ss_sp = alt_stack;
    ss.ss_size = sizeof(alt_stack);
    ss.ss_flags = 0;
    sigaltstack(&ss, nullptr);
    install_signal(SIGSEGV);
    install_signal(SIGABRT);
    install_signal(SIGILL);
    install_signal(SIGFPE);
    install_signal(SIGBUS);
    std::set_terminate(terminate_handler);
}

std::string capture_trace() {
    void *frames[MAX_FRAMES];
    const int n = backtrace(frames, MAX_FRAMES);
    char **symbols = backtrace_symbols(frames, n);
    if (symbols == nullptr) return "";
    std::string out;
    for (int i = 0; i < n; i++) {
        out += demangle_line(symbols[i]);
        out += '\n';
    }
    std::free(symbols);
    return out;
}
}
