// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include <cstdlib>
#include <exception>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include "absl/log/log.h"
#include <mach-o/dyld.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>

#include "x/cpp/env/env.h"

#include "driver/daemon/common.h"
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
namespace {
const std::string LABEL = "com.synnaxlabs.driver";
const std::string TARGET = "system/" + LABEL;
const std::string PLIST_PATH = "/Library/LaunchDaemons/" + LABEL + ".plist";
const std::string LOG_PATH = "/Library/Logs/synnax-driver.log";
/// @brief only root can write the folders of the binary that the service runs.
const fs::path BINARY_PATH = "/Library/Application Support/synnax-driver/bin/"
                             "synnax-driver";

/// @brief launchd restarts the Driver when it exits with a nonzero code or crashes.
const std::string PLIST = R"(<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>)" + LABEL + R"(</string>
    <key>ProgramArguments</key>
    <array>
        <string>)" + BINARY_PATH.string() +
                          R"(</string>
        <string>internal-start</string>
        <string>--disable-stdin-stop</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>)" + LOG_PATH +
                          R"(</string>
    <key>StandardErrorPath</key>
    <string>)" + LOG_PATH +
                          R"(</string>
</dict>
</plist>
)";

std::pair<fs::path, x::errors::Error> executable_path() {
    uint32_t size = 0;
    _NSGetExecutablePath(nullptr, &size);
    std::vector<char> buf(size);
    if (_NSGetExecutablePath(buf.data(), &size) != 0)
        return {{}, x::errors::Error("failed to get current executable path")};
    std::error_code ec;
    auto path = fs::canonical(buf.data(), ec);
    if (ec)
        return {
            {},
            x::errors::Error("failed to get current executable path: " + ec.message())
        };
    return {path, x::errors::NIL};
}

// NOLINTBEGIN(bugprone-command-processor): each command is built from constants.
bool loaded() {
    return system(("launchctl print " + TARGET + " >/dev/null 2>&1").c_str()) == 0;
}

x::errors::Error launchctl(const std::string &args, const std::string &action) {
    if (system(("launchctl " + args).c_str()) != 0)
        return x::errors::Error("failed to " + action + " the Driver service");
    return x::errors::NIL;
}
// NOLINTEND(bugprone-command-processor)

/// @brief loads the service, which starts it, or starts it when launchd has it loaded.
x::errors::Error start() {
    if (loaded()) return launchctl("kickstart " + TARGET, "start");
    return launchctl("bootstrap system " + PLIST_PATH, "start");
}

const auto NOT_INSTALLED = x::errors::Error(
    "the Driver service is not installed. Run sudo synnax-driver install"
);
}

int run(const Config &config) {
    // launchd has no option to read an environment file.
    if (auto err = load_env()) {
        LOG(ERROR) << err;
        return 1;
    }
    try {
        return config.callback();
    } catch (const std::exception &e) {
        LOG(ERROR) << "Application error: " << e.what();
        return 1;
    }
}

bool installed() {
    return fs::exists(PLIST_PATH);
}

x::errors::Error load_env() {
    // Without the service, login saves the state of the user, which no file configures.
    if (!installed()) return x::errors::NIL;
    return x::env::load_file(ENV_FILE);
}

x::errors::Error install_service() {
    if (fs::exists(PLIST_PATH) && loaded()) {
        LOG(INFO) << "Existing service found, stopping it";
        if (auto err = launchctl("bootout " + TARGET, "stop")) return err;
    }
    if (auto err = create_env_file()) return err;
    const auto [binary, path_err] = executable_path();
    if (path_err) return path_err;
    if (auto err = install_binary(binary, BINARY_PATH, 0, 0)) return err;
    LOG(INFO) << "Creating service file at " << PLIST_PATH;
    const auto tmp_path = PLIST_PATH + ".tmp";
    std::ofstream file(tmp_path);
    if (!file) return x::errors::Error("failed to create service file");
    file << PLIST;
    file.close();
    // launchd refuses a file that someone other than root can write.
    if (chown(tmp_path.c_str(), 0, 0) != 0 ||
        chmod(tmp_path.c_str(), S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH) != 0 ||
        rename(tmp_path.c_str(), PLIST_PATH.c_str()) != 0) {
        fs::remove(tmp_path);
        return x::errors::Error("failed to write service file");
    }
    return x::errors::NIL;
}

x::errors::Error uninstall_service() {
    if (loaded()) {
        LOG(INFO) << "Stopping service";
        if (auto err = launchctl("bootout " + TARGET, "stop")) return err;
    }
    LOG(INFO) << "Removing service file at " << PLIST_PATH;
    fs::remove(PLIST_PATH);
    return x::errors::NIL;
}

x::errors::Error start_service() {
    if (!fs::exists(PLIST_PATH)) return NOT_INSTALLED;
    LOG(INFO) << "Starting service";
    return start();
}

x::errors::Error stop_service() {
    if (!fs::exists(PLIST_PATH)) return NOT_INSTALLED;
    LOG(INFO) << "Stopping service";
    if (!loaded()) return x::errors::NIL;
    return launchctl("bootout " + TARGET, "stop");
}

x::errors::Error restart_service() {
    if (!fs::exists(PLIST_PATH)) return NOT_INSTALLED;
    LOG(INFO) << "Restarting service";
    if (loaded()) return launchctl("kickstart -k " + TARGET, "restart");
    return launchctl("bootstrap system " + PLIST_PATH, "restart");
}

x::errors::Error view_logs() {
    // NOLINTNEXTLINE(bugprone-command-processor): a fixed string.
    const int result = system(("tail -n 100 -f " + LOG_PATH).c_str());
    // Ctrl+C ends tail with SIGINT, or the shell with exit code 130.
    const bool interrupted = (WIFSIGNALED(result) && WTERMSIG(result) == SIGINT) ||
                             (WIFEXITED(result) && WEXITSTATUS(result) == 130);
    if (result != 0 && !interrupted) return x::errors::Error("failed to view logs");
    return x::errors::NIL;
}

x::errors::Error status() {
    if (!fs::exists(PLIST_PATH)) return NOT_INSTALLED;
    // NOLINTNEXTLINE(bugprone-command-processor): a fixed string.
    if (system(("launchctl print " + TARGET).c_str()) != 0)
        return x::errors::Error("Service is not running");
    return x::errors::NIL;
}
}
