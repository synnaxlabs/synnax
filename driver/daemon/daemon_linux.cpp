// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <condition_variable>
#include <filesystem>
#include <fstream>
#include <mutex>
#include <thread>

#include "absl/log/log.h"
#include <pwd.h>
#include <sys/stat.h>
#include <systemd/sd-daemon.h>
#include <unistd.h>

#include "x/cpp/env/env.h"
#include "x/cpp/thread/thread.h"

#include "driver/daemon/common.h"
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string SYSTEMD_SERVICE_PATH = "/etc/systemd/system/synnax-driver.service";

/// @brief the service user owns the folder of the binary, so the Driver can replace
/// itself.
const fs::path BINARY_DIR = "/var/lib/synnax-driver/bin";
const fs::path BINARY_PATH = BINARY_DIR / "synnax-driver";
const fs::path LINK_PATH = "/usr/local/bin/synnax-driver";

const std::string SYSTEMD_SERVICE = R"([Unit]
Description=Synnax Driver Service
Documentation=https://docs.synnaxlabs.com/reference/enterprise/driver/install
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=notify
ExecStart=)" + BINARY_PATH.string() +
                                    R"( internal-start --disable-stdin-stop
User=synnax
Group=synnax
EnvironmentFile=-/etc/synnax/driver.env

# Raw Ethernet access for EtherCAT
AmbientCapabilities=CAP_NET_RAW

# Watchdog configuration
WatchdogSec=30s

# State directory
StateDirectory=synnax synnax-driver/bin
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

# Resource limits. RTPRIO and MEMLOCK let Arc tasks run on a real-time thread.
LimitRTPRIO=99
LimitMEMLOCK=infinity
LimitNOFILE=65535
LimitCORE=infinity
TasksMax=4096

# Restart policy
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
)";

const std::string
    LATENCY_RULE_PATH = "/etc/udev/rules.d/99-synnax-driver-latency.rules";

/// @brief lets the synnax group hold CPU cores out of deep idle states, which Arc
/// tasks at high performance do through /dev/cpu_dma_latency.
const std::string
    LATENCY_RULE = "KERNEL==\"cpu_dma_latency\", GROUP=\"synnax\", MODE=\"0660\"\n";
const std::string LATENCY_MATCH = "--name-match=cpu_dma_latency";

const std::string
    ETHERCAT_RULE_PATH = "/etc/udev/rules.d/99-synnax-driver-ethercat.rules";

/// @brief lets the synnax group open IgH EtherCAT masters through /dev/EtherCAT*.
const std::string
    ETHERCAT_RULE = "KERNEL==\"EtherCAT[0-9]*\", GROUP=\"synnax\", MODE=\"0660\"\n";
const std::string ETHERCAT_MATCH = "--subsystem-match=EtherCAT";

// NOLINTBEGIN(bugprone-command-processor): each command is built from constants.
/// @brief reloads the udev rules and applies them to the existing devices that match
/// the udevadm trigger filter in match.
x::errors::Error apply_udev_rules(const std::string &match) {
    if (system("udevadm control --reload") != 0)
        return x::errors::Error("Failed to reload udev rules");
    if (system(("udevadm trigger " + match).c_str()) != 0)
        return x::errors::Error("Failed to apply udev rules");
    return x::errors::NIL;
}

x::errors::Error install_udev_rule(
    const std::string &path,
    const std::string &rule,
    const std::string &match
) {
    LOG(INFO) << "Creating udev rule at " << path;
    std::ofstream rule_file(path.c_str());
    if (!rule_file) return x::errors::Error("Failed to create udev rule");
    rule_file << rule;
    rule_file.close();
    return apply_udev_rules(match);
}

/// @brief installs the running binary in BINARY_DIR, owned by the service user, and
/// points LINK_PATH at it.
x::errors::Error install_service_binary() {
    const passwd *user = getpwnam("synnax");
    if (user == nullptr) return x::errors::Error("Failed to find the synnax user");
    std::error_code ec;
    fs::create_directories(BINARY_DIR, ec);
    if (ec)
        return x::errors::Error("Failed to create binary directory: " + ec.message());
    if (chown(BINARY_DIR.c_str(), user->pw_uid, user->pw_gid) != 0 ||
        chmod(BINARY_DIR.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0)
        return x::errors::Error("Failed to give binary directory to the synnax user");
    if (auto err = install_binary(
            "/proc/self/exe",
            BINARY_PATH,
            user->pw_uid,
            user->pw_gid
        ))
        return err;
    LOG(INFO) << "Linking " << LINK_PATH << " to " << BINARY_PATH;
    fs::path tmp_path = LINK_PATH;
    tmp_path += "." + std::to_string(getpid()) + ".tmp";
    fs::create_directories(LINK_PATH.parent_path(), ec);
    fs::remove(tmp_path, ec);
    fs::create_symlink(BINARY_PATH, tmp_path, ec);
    if (!ec) fs::rename(tmp_path, LINK_PATH, ec);
    if (ec) {
        const auto msg = ec.message();
        fs::remove(tmp_path, ec);
        return x::errors::Error("Failed to link binary: " + msg);
    }
    return x::errors::NIL;
}

bool installed() {
    return fs::exists(SYSTEMD_SERVICE_PATH);
}

x::errors::Error load_env() {
    return x::env::load_file(ENV_FILE);
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
    if (auto err = create_env_file()) return err;
    if (auto err = install_service_binary()) return err;
    if (auto err = install_udev_rule(LATENCY_RULE_PATH, LATENCY_RULE, LATENCY_MATCH))
        LOG(WARNING) << "Arc tasks at high performance cannot keep CPU cores awake: "
                     << err;
    if (auto err = install_udev_rule(ETHERCAT_RULE_PATH, ETHERCAT_RULE, ETHERCAT_MATCH))
        LOG(WARNING) << "EtherCAT tasks cannot use an IgH master: " << err;

    LOG(INFO) << "Creating service file at " << SYSTEMD_SERVICE_PATH;
    std::error_code ec;
    fs::create_directories(fs::path(SYSTEMD_SERVICE_PATH).parent_path(), ec);
    if (ec)
        return x::errors::Error("Failed to create service directory: " + ec.message());

    std::ofstream service_file(SYSTEMD_SERVICE_PATH.c_str());
    if (!service_file) return x::errors::Error("Failed to create service file");

    service_file << SYSTEMD_SERVICE;
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
    if (fs::remove(LATENCY_RULE_PATH)) {
        if (auto err = apply_udev_rules(LATENCY_MATCH))
            LOG(WARNING) << "Failed to reload udev rules: " << err;
    }
    if (fs::remove(ETHERCAT_RULE_PATH)) {
        if (auto err = apply_udev_rules(ETHERCAT_MATCH))
            LOG(WARNING) << "Failed to reload udev rules: " << err;
    }

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

int run(const Config &config) {
    std::mutex mtx;
    std::condition_variable cv;
    bool should_stop = false;
    update_status(Status::INITIALIZING, "Starting daemon");

    std::thread watchdog([&] {
        x::thread::set_name("watchdog");
        std::unique_lock<std::mutex> lock(mtx);
        while (!should_stop) {
            notify_watchdog();
            cv.wait_for(lock, std::chrono::seconds(config.watchdog_interval), [&] {
                return should_stop;
            });
        }
    });

    update_status(Status::READY, "Daemon ready");

    bool failed = false;
    int code = 1;
    try {
        code = config.callback();
    } catch (const std::exception &e) {
        failed = true;
        update_status(Status::ERROR_, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    // Cleanup. On failure, send STOPPING=1 alone so the error stays visible as the
    // unit's STATUS instead of being overwritten with "Stopping".
    if (failed)
        sd_notify(0, "STOPPING=1");
    else
        update_status(Status::STOPPING, "Stopping daemon");
    {
        std::lock_guard<std::mutex> lock(mtx);
        should_stop = true;
    }
    cv.notify_all();
    watchdog.join();
    return code;
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

// NOLINTEND(bugprone-command-processor)
}
