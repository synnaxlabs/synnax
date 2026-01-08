// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std.
#include <condition_variable>
#include <filesystem>
#include <fstream>
#include <mutex>
#include <thread>

#include "glog/logging.h"
#include <sys/stat.h>
#include <systemd/sd-daemon.h>

#include "x/cpp/thread/thread.h"

#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string BINARY_INSTALL_DIR = "/usr/local/bin";
const std::string BINARY_NAME = "synnax-driver";
const std::string SYSTEMD_SERVICE_PATH = "/etc/systemd/system/synnax-driver.service";

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

auto SYSTEMD_SERVICE_TEMPLATE = R"([Unit]
Description=Synnax Driver Service
Documentation=https://docs.synnaxlabs.com/reference/driver
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=notify
Environment=GLOG_logtostderr=1
Environment=GLOG_v=1
ExecStart=/usr/local/bin/synnax-driver internal-start
User=synnax
Group=synnax

# Watchdog configuration
WatchdogSec=30s

# State directory
StateDirectory=synnax
ConfigurationDirectory=synnax
CacheDirectory=synnax
LogsDirectory=synnax

# Logging
StandardOutput=journal
StandardError=journal

# Temporarily reduce security restrictions for debugging
#ProtectSystem=strict
#ProtectHome=true
#PrivateTmp=true
#PrivateDevices=true
#ProtectKernelTunables=true
#ProtectKernelModules=true
#ProtectControlGroups=true
#NoNewPrivileges=true
#RestrictNamespaces=true
#RestrictRealtime=true
#RestrictSUIDSGID=true
#MemoryDenyWriteExecute=true

# Resource limits
LimitNOFILE=65535
LimitCORE=infinity
TasksMax=4096

# Restart policy
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
)";

x::errors::Error create_system_user() {
    LOG(INFO) << "Creating system user";
    int result = system(
        "id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax"
    );
    if (result != 0) { return x::errors::Error("Failed to create system user"); }
    return {};
}

x::errors::Error install_binary() {
    LOG(INFO) << "Moving binary to " << BINARY_INSTALL_DIR;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec)
        return x::errors::Error("Failed to get current executable path: " + ec.message());

    fs::create_directories(BINARY_INSTALL_DIR, ec);
    if (ec) return x::errors::Error("Failed to create binary directory: " + ec.message());

    // Copy the binary
    const fs::path target_path = "/usr/local/bin/synnax-driver";
    fs::copy_file(curr_bin_path, target_path, fs::copy_options::overwrite_existing, ec);
    if (ec) return x::errors::Error("Failed to copy binary: " + ec.message());

    if (chmod(target_path.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) !=
        0)
        return x::errors::Error("Failed to set binary permissions");

    return x::errors::NIL;
}

x::errors::Error install_service() {
    // Check if service exists and is running
    LOG(INFO) << "Checking for existing service";
    if (fs::exists(SYSTEMD_SERVICE_PATH)) {
        LOG(INFO) << "Existing service found, stopping and removing it";
        if (int result = system("systemctl stop synnax-driver"); result != 0)
            LOG(WARNING) << "Failed to stop existing service (may not be running)";
        // Give it a moment to stop
        std::this_thread::sleep_for(std::chrono::seconds(2));
        // Uninstall the existing service
        if (auto err = uninstall_service()) return err;
    }

    if (auto err = create_system_user()) return err;
    if (auto err = install_binary()) return err;

    LOG(INFO) << "Creating service file at " << SYSTEMD_SERVICE_PATH;
    std::error_code ec;
    fs::create_directories(fs::path(SYSTEMD_SERVICE_PATH).parent_path(), ec);
    if (ec)
        return x::errors::Error("Failed to create service directory: " + ec.message());

    std::ofstream service_file(SYSTEMD_SERVICE_PATH.c_str());
    if (!service_file) return x::errors::Error("Failed to create service file");

    service_file << SYSTEMD_SERVICE_TEMPLATE;
    service_file.close();

    if (chmod(SYSTEMD_SERVICE_PATH.c_str(), S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH) != 0)
        return x::errors::Error("Failed to set service file permissions");

    LOG(INFO) << "Enabling and starting service";
    if (system("systemctl daemon-reload") != 0)
        return x::errors::Error("Failed to reload systemd");

    return x::errors::NIL;
}

x::errors::Error uninstall_service() {
    LOG(INFO) << "Stopping and disabling service";
    if (int result = system("systemctl stop synnax-driver"); result != 0)
        LOG(WARNING) << "Failed to stop service (may not be running)";
    if (int result = system("systemctl disable synnax-driver"); result != 0)
        LOG(WARNING) << "Failed to disable service (may not be enabled)";

    fs::remove(SYSTEMD_SERVICE_PATH);

    if (system("systemctl daemon-reload") != 0)
        return x::errors::Error("Failed to reload systemd");

    // Note: We intentionally don't remove the binary or user
    // in case there are existing configurations or data we want to preserve
    return x::errors::NIL;
}

void update_status(Status status, const std::string &message) {
    std::string status_msg = "STATUS=";
    switch (status) {
        case Status::INITIALIZING:
            status_msg += "Initializing";
            break;
        case Status::READY:
            status_msg += "Ready";
            break;
        case Status::RUNNING:
            status_msg += "Running";
            break;
        case Status::STOPPING:
            status_msg += "Stopping";
            break;
        case Status::ERROR_:
            status_msg += "Error";
            break;
    }

    if (!message.empty()) status_msg += ": " + message;

    if (status == Status::READY)
        status_msg += "\nREADY=1";
    else if (status == Status::STOPPING)
        status_msg += "\nSTOPPING=1";
    sd_notify(0, status_msg.c_str());
}

void notify_watchdog() {
    sd_notify(0, "WATCHDOG=1");
}

void run(const Config &config, int argc, char *argv[]) {
    update_status(Status::INITIALIZING, "Starting daemon");

    // Start watchdog thread
    std::thread watchdog([&]() {
        x::thread::set_name("watchdog");
        while (!should_stop) {
            notify_watchdog();
            std::this_thread::sleep_for(std::chrono::seconds(config.watchdog_interval));
        }
    });

    update_status(Status::READY, "Daemon ready");

    // Run the main application logic
    try {
        config.callback(argc, argv);
    } catch (const std::exception &e) {
        update_status(Status::ERROR_, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    // Cleanup
    update_status(Status::STOPPING, "Stopping daemon");
    {
        std::lock_guard<std::mutex> lock(mtx);
        should_stop = true;
    }
    watchdog.join();
}

x::errors::Error start_service() {
    LOG(INFO) << "Starting service";
    if (system("systemctl start synnax-driver") != 0)
        return x::errors::Error("Failed to start service");
    return x::errors::NIL;
}

x::errors::Error stop_service() {
    LOG(INFO) << "Stopping service";
    if (system("systemctl stop synnax-driver") != 0)
        return x::errors::Error("Failed to stop service");
    return x::errors::NIL;
}

x::errors::Error restart_service() {
    LOG(INFO) << "Restarting service";
    if (system("systemctl restart synnax-driver") != 0)
        return x::errors::Error("Failed to restart service");
    return x::errors::NIL;
}

std::string get_log_file_path() {
    // For systemd, logs are in the journal
    return "";
}

x::errors::Error view_logs() {
    // For systemd, we use journalctl
    int result = system("journalctl -fu synnax-driver");
    // Exit code 130 indicates Ctrl+C termination
    if (result != 0 && WEXITSTATUS(result) != 130)
        return x::errors::Error("Failed to view logs");
    return x::errors::NIL;
}

x::errors::Error status() {
    LOG(INFO) << "Checking service status";
    int result = system("systemctl status synnax-driver");
    if (result != 0) return x::errors::Error("Service is not running");
    return x::errors::NIL;
}
}
