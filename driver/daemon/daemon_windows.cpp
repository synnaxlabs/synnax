// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <iostream>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "absl/log/log.h"
// clang-format off
// shlobj.h must be included after windows.h.
#include <windows.h>
#include <shlobj.h>
// clang-format on

#include "x/cpp/shutdown/shutdown.h"

#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
namespace {
const auto *const SERVICE_NAME = L"SynnaxDriver";
const auto *const DISPLAY_NAME = L"Synnax Driver";
const auto *const DESCRIPTION = L"Connects hardware to a Synnax Core.";
const fs::path LOG_PATH = "C:\\ProgramData\\synnax-driver\\logs\\driver.log";
/// @brief a log file past this size moves to driver.log.1 when the service starts.
constexpr std::uintmax_t MAX_LOG_SIZE = 10 * 1024 * 1024;
constexpr DWORD STOP_WAIT_HINT_MS = 30000;
constexpr auto STOP_TIMEOUT = std::chrono::seconds(30);

x::errors::Error windows_error(const std::string &action, const DWORD code) {
    char *text = nullptr;
    FormatMessageA(
        FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM |
            FORMAT_MESSAGE_IGNORE_INSERTS,
        nullptr,
        code,
        0,
        reinterpret_cast<LPSTR>(&text),
        0,
        nullptr
    );
    std::string msg = text == nullptr ? std::to_string(code) : text;
    LocalFree(text);
    while (!msg.empty() && (msg.back() == '\n' || msg.back() == '\r'))
        msg.pop_back();
    return x::errors::Error(action + ": " + msg);
}

x::errors::Error windows_error(const std::string &action) {
    return windows_error(action, GetLastError());
}

/// @brief a service manager handle that closes itself.
struct Handle {
    SC_HANDLE h = nullptr;
    ~Handle() {
        if (this->h != nullptr) CloseServiceHandle(this->h);
    }
};

x::errors::Error open_manager(Handle &manager, const DWORD access) {
    manager.h = OpenSCManagerW(nullptr, nullptr, access);
    if (manager.h == nullptr) {
        const auto code = GetLastError();
        if (code == ERROR_ACCESS_DENIED)
            return x::errors::Error(
                "failed to connect to the service manager. Run the command in an "
                "administrator terminal"
            );
        return windows_error("failed to connect to the service manager", code);
    }
    return x::errors::NIL;
}

/// @brief opens the Driver service. Sets service.h to nullptr and returns NIL when the
/// service is not installed.
x::errors::Error
open_service(const Handle &manager, Handle &service, const DWORD access) {
    service.h = OpenServiceW(manager.h, SERVICE_NAME, access);
    if (service.h != nullptr) return x::errors::NIL;
    const auto code = GetLastError();
    if (code == ERROR_SERVICE_DOES_NOT_EXIST) return x::errors::NIL;
    return windows_error("failed to open the Driver service", code);
}

const auto NOT_INSTALLED = x::errors::Error(
    "the Driver service is not installed. Run synnax-driver install"
);

/// @brief a process handle that closes itself.
struct Process {
    HANDLE h = nullptr;
    ~Process() {
        if (this->h != nullptr) CloseHandle(this->h);
    }
};

/// @brief stops the service and waits until its process exits. Returns NIL when the
/// service is not running.
x::errors::Error stop_and_wait(const Handle &service) {
    SERVICE_STATUS_PROCESS info{};
    DWORD size = 0;
    if (QueryServiceStatusEx(
            service.h,
            SC_STATUS_PROCESS_INFO,
            reinterpret_cast<LPBYTE>(&info),
            sizeof(info),
            &size
        ) == 0)
        return windows_error("failed to read the Driver service status");
    // The service reports SERVICE_STOPPED before its process exits and releases the
    // binary.
    Process process;
    if (info.dwProcessId != 0)
        process.h = OpenProcess(SYNCHRONIZE, FALSE, info.dwProcessId);
    SERVICE_STATUS status{};
    if (ControlService(service.h, SERVICE_CONTROL_STOP, &status) == 0) {
        const auto code = GetLastError();
        if (code == ERROR_SERVICE_NOT_ACTIVE) return x::errors::NIL;
        return windows_error("failed to stop the Driver service", code);
    }
    const auto deadline = std::chrono::steady_clock::now() + STOP_TIMEOUT;
    const auto timed_out = x::errors::Error(
        "timed out waiting for the Driver service to stop"
    );
    while (status.dwCurrentState != SERVICE_STOPPED) {
        if (std::chrono::steady_clock::now() > deadline) return timed_out;
        std::this_thread::sleep_for(std::chrono::milliseconds(250));
        if (QueryServiceStatus(service.h, &status) == 0)
            return windows_error("failed to read the Driver service status");
    }
    if (process.h == nullptr) return x::errors::NIL;
    const auto left = std::chrono::duration_cast<std::chrono::milliseconds>(
        deadline - std::chrono::steady_clock::now()
    );
    const DWORD wait = left.count() > 0 ? static_cast<DWORD>(left.count()) : 0;
    if (WaitForSingleObject(process.h, wait) != WAIT_OBJECT_0) return timed_out;
    return x::errors::NIL;
}

x::errors::Error start(const Handle &service) {
    if (StartServiceW(service.h, 0, nullptr) != 0) return x::errors::NIL;
    const auto code = GetLastError();
    if (code == ERROR_SERVICE_ALREADY_RUNNING) return x::errors::NIL;
    return windows_error("failed to start the Driver service", code);
}

/// @brief sets the description and makes the service manager restart the Driver after a
/// crash or a nonzero exit code.
x::errors::Error configure(const Handle &service) {
    SERVICE_DESCRIPTIONW description{const_cast<LPWSTR>(DESCRIPTION)};
    if (ChangeServiceConfig2W(service.h, SERVICE_CONFIG_DESCRIPTION, &description) == 0)
        return windows_error("failed to set the service description");
    SC_ACTION actions[] = {
        {SC_ACTION_RESTART, 5000},
        {SC_ACTION_RESTART, 30000},
        {SC_ACTION_RESTART, 60000},
    };
    SERVICE_FAILURE_ACTIONSW failure{};
    failure.dwResetPeriod = 86400;
    failure.cActions = 3;
    failure.lpsaActions = actions;
    if (ChangeServiceConfig2W(service.h, SERVICE_CONFIG_FAILURE_ACTIONS, &failure) == 0)
        return windows_error("failed to set the service recovery actions");
    SERVICE_FAILURE_ACTIONS_FLAG flag{TRUE};
    if (ChangeServiceConfig2W(service.h, SERVICE_CONFIG_FAILURE_ACTIONS_FLAG, &flag) ==
        0)
        return windows_error("failed to set the service recovery actions");
    return x::errors::NIL;
}

std::wstring executable_path() {
    std::vector<wchar_t> buf(MAX_PATH);
    while (true) {
        const auto n = GetModuleFileNameW(
            nullptr,
            buf.data(),
            static_cast<DWORD>(buf.size())
        );
        if (n < buf.size()) return {buf.data(), n};
        buf.resize(buf.size() * 2);
    }
}

/// @brief returns where the service binary goes. Users cannot create or change files
/// in Program Files, so only an administrator can replace the binary that LocalSystem
/// runs.
std::pair<fs::path, x::errors::Error> binary_path() {
    PWSTR dir = nullptr;
    const auto result = SHGetKnownFolderPath(FOLDERID_ProgramFiles, 0, nullptr, &dir);
    const fs::path program_files = SUCCEEDED(result) ? fs::path(dir) : fs::path();
    CoTaskMemFree(dir);
    if (FAILED(result))
        return {
            {},
            windows_error("failed to find Program Files", static_cast<DWORD>(result))
        };
    return {program_files / "Synnax Driver" / "synnax-driver.exe", x::errors::NIL};
}

/// @brief copies the running binary to path, unless it runs from path.
x::errors::Error install_binary(const fs::path &path) {
    LOG(INFO) << "Copying binary to " << path;
    const fs::path src = executable_path();
    std::error_code ec;
    fs::create_directories(path.parent_path(), ec);
    if (ec)
        return x::errors::Error("failed to create binary directory: " + ec.message());
    if (fs::exists(path, ec) && fs::equivalent(src, path, ec)) return x::errors::NIL;
    fs::path tmp_path = path;
    tmp_path += ".tmp";
    fs::copy_file(src, tmp_path, fs::copy_options::overwrite_existing, ec);
    if (!ec) fs::rename(tmp_path, path, ec);
    if (ec) {
        const auto msg = ec.message();
        fs::remove(tmp_path, ec);
        return x::errors::Error("failed to copy binary: " + msg);
    }
    return x::errors::NIL;
}

/// @brief the state of the service while the service manager runs it.
struct Service {
    const Config &config;
    SERVICE_STATUS_HANDLE handle = nullptr;
    std::mutex mu;
    SERVICE_STATUS status{SERVICE_WIN32_OWN_PROCESS, SERVICE_START_PENDING};
    int code = 1;

    void report(const DWORD state, const DWORD wait_hint = 0) {
        std::lock_guard lock(this->mu);
        this->status.dwCurrentState = state;
        this->status.dwControlsAccepted = state == SERVICE_RUNNING
                                            ? SERVICE_ACCEPT_STOP |
                                                  SERVICE_ACCEPT_SHUTDOWN
                                            : 0;
        this->status.dwWaitHint = wait_hint;
        this->status.dwCheckPoint = state == SERVICE_STOP_PENDING
                                      ? this->status.dwCheckPoint + 1
                                      : 0;
        if (state == SERVICE_STOPPED && this->code != 0) {
            this->status.dwWin32ExitCode = ERROR_SERVICE_SPECIFIC_ERROR;
            this->status.dwServiceSpecificExitCode = static_cast<DWORD>(this->code);
        }
        SetServiceStatus(this->handle, &this->status);
    }
};

// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): the service manager
// calls service_main with no context argument, so run hands it the service here.
Service *running = nullptr;
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

DWORD WINAPI handle_control(DWORD control, DWORD, LPVOID, LPVOID context) {
    auto *service = static_cast<Service *>(context);
    switch (control) {
        case SERVICE_CONTROL_STOP:
        case SERVICE_CONTROL_SHUTDOWN:
            service->report(SERVICE_STOP_PENDING, STOP_WAIT_HINT_MS);
            x::shutdown::signal_shutdown();
            return NO_ERROR;
        case SERVICE_CONTROL_INTERROGATE:
            return NO_ERROR;
        default:
            return ERROR_CALL_NOT_IMPLEMENTED;
    }
}

/// @brief sends stderr, and so the logs, to LOG_PATH. The service manager gives the
/// process no console.
void open_log() {
    std::error_code ec;
    fs::create_directories(LOG_PATH.parent_path(), ec);
    if (fs::exists(LOG_PATH, ec) && fs::file_size(LOG_PATH, ec) > MAX_LOG_SIZE) {
        auto rotated = LOG_PATH;
        rotated += ".1";
        fs::rename(LOG_PATH, rotated, ec);
    }
    if (_wfreopen(LOG_PATH.c_str(), L"a", stderr) != nullptr)
        setvbuf(stderr, nullptr, _IONBF, 0);
}

int call(const Config &config) {
    try {
        return config.callback();
    } catch (const std::exception &e) {
        LOG(ERROR) << "Application error: " << e.what();
        return 1;
    }
}

void WINAPI service_main(DWORD, LPWSTR *) {
    auto *service = running;
    const auto handle = RegisterServiceCtrlHandlerExW(
        SERVICE_NAME,
        handle_control,
        service
    );
    if (handle == nullptr) return;
    service->handle = handle;
    open_log();
    service->report(SERVICE_RUNNING);
    service->code = call(service->config);
    service->report(SERVICE_STOPPED);
}
}

int run(const Config &config) {
    Service service{.config = config};
    running = &service;
    SERVICE_TABLE_ENTRYW table[] = {
        {const_cast<LPWSTR>(SERVICE_NAME), service_main},
        {nullptr, nullptr},
    };
    if (StartServiceCtrlDispatcherW(table) != 0) return service.code;
    // The process was started by hand, not by the service manager.
    if (GetLastError() == ERROR_FAILED_SERVICE_CONTROLLER_CONNECT) return call(config);
    LOG(ERROR) << windows_error("failed to connect to the service manager");
    return 1;
}

bool installed() {
    Handle manager;
    if (open_manager(manager, SC_MANAGER_CONNECT)) return false;
    Handle service;
    if (open_service(manager, service, SERVICE_QUERY_STATUS)) return false;
    return service.h != nullptr;
}

x::errors::Error install_service() {
    LOG(INFO) << "Creating log directory at " << LOG_PATH.parent_path();
    std::error_code ec;
    fs::create_directories(LOG_PATH.parent_path(), ec);
    if (ec) return x::errors::Error("failed to create log directory: " + ec.message());
    Handle manager;
    if (auto err = open_manager(
            manager,
            SC_MANAGER_CONNECT | SC_MANAGER_CREATE_SERVICE
        ))
        return err;
    Handle service;
    if (auto err = open_service(manager, service, SERVICE_ALL_ACCESS)) return err;
    if (service.h != nullptr) {
        LOG(INFO) << "Existing service found, stopping and updating it";
        if (auto err = stop_and_wait(service)) return err;
    }
    const auto [binary, path_err] = binary_path();
    if (path_err) return path_err;
    if (auto err = install_binary(binary)) return err;
    const auto command = L"\"" + binary.wstring() +
                         L"\" internal-start --disable-stdin-stop";
    if (service.h != nullptr) {
        if (ChangeServiceConfigW(
                service.h,
                SERVICE_WIN32_OWN_PROCESS,
                SERVICE_AUTO_START,
                SERVICE_ERROR_NORMAL,
                command.c_str(),
                nullptr,
                nullptr,
                nullptr,
                nullptr,
                nullptr,
                DISPLAY_NAME
            ) == 0)
            return windows_error("failed to update the Driver service");
    } else {
        LOG(INFO) << "Creating service";
        service.h = CreateServiceW(
            manager.h,
            SERVICE_NAME,
            DISPLAY_NAME,
            SERVICE_ALL_ACCESS,
            SERVICE_WIN32_OWN_PROCESS,
            SERVICE_AUTO_START,
            SERVICE_ERROR_NORMAL,
            command.c_str(),
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr
        );
        if (service.h == nullptr)
            return windows_error("failed to create the Driver service");
    }
    return configure(service);
}

x::errors::Error uninstall_service() {
    Handle manager;
    if (auto err = open_manager(manager, SC_MANAGER_CONNECT)) return err;
    Handle service;
    if (auto err = open_service(
            manager,
            service,
            DELETE | SERVICE_STOP | SERVICE_QUERY_STATUS
        ))
        return err;
    if (service.h == nullptr) {
        LOG(WARNING) << "The Driver service is not installed";
        return x::errors::NIL;
    }
    LOG(INFO) << "Stopping and removing service";
    if (auto err = stop_and_wait(service)) return err;
    if (DeleteService(service.h) == 0)
        return windows_error("failed to remove the Driver service");
    return x::errors::NIL;
}

x::errors::Error start_service() {
    Handle manager;
    if (auto err = open_manager(manager, SC_MANAGER_CONNECT)) return err;
    Handle service;
    if (auto err = open_service(manager, service, SERVICE_START)) return err;
    if (service.h == nullptr) return NOT_INSTALLED;
    LOG(INFO) << "Starting service";
    return start(service);
}

x::errors::Error stop_service() {
    Handle manager;
    if (auto err = open_manager(manager, SC_MANAGER_CONNECT)) return err;
    Handle service;
    if (auto err = open_service(manager, service, SERVICE_STOP | SERVICE_QUERY_STATUS))
        return err;
    if (service.h == nullptr) return NOT_INSTALLED;
    LOG(INFO) << "Stopping service";
    return stop_and_wait(service);
}

x::errors::Error restart_service() {
    Handle manager;
    if (auto err = open_manager(manager, SC_MANAGER_CONNECT)) return err;
    Handle service;
    if (auto err = open_service(
            manager,
            service,
            SERVICE_START | SERVICE_STOP | SERVICE_QUERY_STATUS
        ))
        return err;
    if (service.h == nullptr) return NOT_INSTALLED;
    LOG(INFO) << "Restarting service";
    if (auto err = stop_and_wait(service)) return err;
    return start(service);
}

x::errors::Error view_logs() {
    std::error_code ec;
    if (!fs::exists(LOG_PATH, ec))
        return x::errors::Error(
            "no logs at " + LOG_PATH.string() + ". Start the Driver service first"
        );
    const auto cmd = "powershell -NoProfile -Command \"Get-Content -Path '" +
                     LOG_PATH.string() + "' -Tail 100 -Wait\"";
    // NOLINTNEXTLINE(bugprone-command-processor): built from constants.
    if (system(cmd.c_str()) != 0) return x::errors::Error("failed to view logs");
    return x::errors::NIL;
}

x::errors::Error status() {
    Handle manager;
    if (auto err = open_manager(manager, SC_MANAGER_CONNECT)) return err;
    Handle service;
    if (auto err = open_service(manager, service, SERVICE_QUERY_STATUS)) return err;
    if (service.h == nullptr) return NOT_INSTALLED;
    SERVICE_STATUS_PROCESS status{};
    DWORD size = 0;
    if (QueryServiceStatusEx(
            service.h,
            SC_STATUS_PROCESS_INFO,
            reinterpret_cast<LPBYTE>(&status),
            sizeof(status),
            &size
        ) == 0)
        return windows_error("failed to read the Driver service status");
    if (status.dwCurrentState != SERVICE_RUNNING)
        return x::errors::Error("Synnax Driver is not running");
    std::cout << "Synnax Driver is running (PID: " << status.dwProcessId << ")\n";
    return x::errors::NIL;
}

x::errors::Error load_env() {
    return x::errors::NIL;
}
}
