// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// @brief NI Linux Real-Time does not support systemd, so we use a traditional init
/// script instead.

/// std.
#include <thread>
#include <condition_variable>
#include <filesystem>
#include <fstream>

/// external.
#include <sys/stat.h>
#include "glog/logging.h"

/// internal
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace daemond {
const std::string BINARY_INSTALL_DIR = "/usr/local/bin";
const std::string BINARY_NAME = "synnax-driver";
const std::string INIT_SCRIPT_PATH = "/etc/init.d/synnax-driver";

auto INIT_SCRIPT_TEMPLATE = R"###(#!/bin/sh
### BEGIN INIT INFO
# Provides:          synnax-driver
# Required-Start:    $network $local_fs $ni_rseries
# Required-Stop:     $network $local_fs $ni_rseries
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Synnax Driver Service
# Description:       Synnax Driver Service for data acquisition and control
### END INIT INFO

NAME="synnax-driver"
DAEMON="/usr/local/bin/$NAME"
DAEMON_USER="synnax"
PIDFILE="/var/run/$NAME.pid"
LOGFILE="/var/log/$NAME.log"

# Exit if executable not installed
[ -x "$DAEMON" ] || exit 0

log_message() {
    echo "$1" | tee -a $LOGFILE
}

do_start() {
    log_message "Starting $NAME at $(date)"
    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_message "$NAME is already running (PID: $PID)"
            return 1
        else
            rm -f "$PIDFILE"
        fi
    fi

    # Add debug logging
    log_message "Starting daemon with command: $DAEMON internal-start"
    log_message "Current working directory: $(pwd)"
    log_message "Running as user: $(whoami)"

    # Try starting with explicit working directory
    cd /
    start-stop-daemon --start --background \
        --make-pidfile --pidfile $PIDFILE \
        --chuid $DAEMON_USER \
        --startas /bin/bash -- -c "exec $DAEMON internal-start >> $LOGFILE 2>&1"

    RETVAL=$?
    if [ $RETVAL -eq 0 ]; then
        log_message "$NAME started successfully"
        # Add 5 second wait and status check
        sleep 5
        if kill -0 $(cat $PIDFILE) 2>/dev/null; then
            log_message "Process verified running after 5 seconds"
        else
            log_message "Process failed to stay running"
            return 1
        fi
    else
        log_message "Failed to start $NAME"
    fi
    return $RETVAL
}

do_stop() {
    log_message "Stopping $NAME at $(date)"
    start-stop-daemon --stop --pidfile $PIDFILE --retry 30
    RETVAL=$?
    if [ $RETVAL -eq 0 ]; then
        rm -f $PIDFILE
        log_message "$NAME stopped successfully"
    else
        log_message "Failed to stop $NAME"
    fi
    return $RETVAL
}

do_status() {
    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_message "$NAME is running (PID: $PID)"
            return 0
        else
            log_message "$NAME is not running (stale PID file)"
            return 1
        fi
    else
        log_message "$NAME is not running"
        return 3
    fi
}

case "$1" in
    start)
        do_start
        ;;
    stop)
        do_stop
        ;;
    restart)
        do_stop
        do_start
        ;;
    status)
        do_status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
)###";

freighter::Error create_system_user() {
    LOG(INFO) << "Creating system user";
    const int result = system(
        "id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax");
    if (result != 0)
        return freighter::Error("Failed to create system user");
    return freighter::NIL;
}

freighter::Error install_binary() {
    LOG(INFO) << "Moving binary to " << BINARY_INSTALL_DIR;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec)
        return freighter::Error(
            "Failed to get current executable path: " + ec.message());

    fs::create_directories(BINARY_INSTALL_DIR, ec);
    if (ec)
        return freighter::Error("Failed to create binary directory: " + ec.message());

    const fs::path target_path = BINARY_INSTALL_DIR + "/" + BINARY_NAME;
    fs::copy_file(
        curr_bin_path,
        target_path,
        fs::copy_options::overwrite_existing,
        ec
    );
    if (ec)
        return freighter::Error("Failed to copy binary: " + ec.message());

    if (chmod(target_path.c_str(),
              S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0)
        return freighter::Error("Failed to set binary permissions");

    return freighter::NIL;
}

freighter::Error install_service() {
    // Check if service exists and is running
    LOG(INFO) << "Checking for existing service";
    if (fs::exists(INIT_SCRIPT_PATH)) {
        LOG(INFO) << "Existing service found, stopping and removing it";
        system("/etc/init.d/synnax-driver stop");
        // Give it a moment to stop
        std::this_thread::sleep_for(std::chrono::seconds(2));
        // Uninstall the existing service
        if (auto err = uninstall_service()) return err;
    }

    if (auto err = create_system_user()) return err;
    if (auto err = install_binary()) return err;

    // Create log file with proper permissions
    LOG(INFO) << "Creating log file";
    std::ofstream log_file("/var/log/synnax-driver.log");
    if (!log_file)
        return freighter::Error("Failed to create log file");
    log_file.close();

    // Set permissions so both root and synnax user can write to it
    if (chmod("/var/log/synnax-driver.log", 0666) != 0)
        return freighter::Error("Failed to set log file permissions");

    if (system("chown synnax:synnax /var/log/synnax-driver.log") != 0)
        return freighter::Error("Failed to set log file ownership");

    LOG(INFO) << "Creating init script at " << INIT_SCRIPT_PATH;
    std::error_code ec;
    fs::create_directories(fs::path(INIT_SCRIPT_PATH).parent_path(), ec);
    if (ec)
        return freighter::Error("Failed to create init.d directory: " + ec.message());

    std::ofstream init_file(INIT_SCRIPT_PATH);
    if (!init_file)
        return freighter::Error("Failed to create init script");

    init_file << INIT_SCRIPT_TEMPLATE;
    init_file.close();

    if (chmod(INIT_SCRIPT_PATH.c_str(),
              S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0)
        return freighter::Error("Failed to set init script permissions");

    LOG(INFO) << "Configuring service runlevels";
    if (system("update-rc.d synnax-driver defaults") != 0)
        return freighter::Error("Failed to configure service runlevels");

    return freighter::NIL;
}

freighter::Error uninstall_service() {
    LOG(INFO) << "Removing service";
    system("update-rc.d -f synnax-driver remove");
    fs::remove(INIT_SCRIPT_PATH);

    // Note: We intentionally don't remove the binary or user
    // in case there are existing configurations or data we want to preserve
    return freighter::NIL;
}

void update_status(Status status, const std::string &message) {
    std::string status_str;
    switch (status) {
        case Status::INITIALIZING: status_str = "Initializing";
            break;
        case Status::READY: status_str = "Ready";
            break;
        case Status::RUNNING: status_str = "Running";
            break;
        case Status::STOPPING: status_str = "Stopping";
            break;
        case Status::ERROR: status_str = "Error";
            break;
    }

    if (!message.empty()) {
        LOG(INFO) << "[daemon] Status: " << status_str << " - " << message;
    } else {
        LOG(INFO) << "[daemon] Status: " << status_str;
    }
}

void notify_watchdog() {
    // No-op for NILinuxRT as it doesn't have native watchdog support
}

void run(const Config &config, int argc, char *argv[]) {
    // Initialize logging
    google::SetLogDestination(google::INFO, "/var/log/synnax-driver");

    update_status(Status::INITIALIZING, "Starting daemon");
    update_status(Status::READY, "Daemon ready");

    try {
        config.callback(argc, argv);
    } catch (const std::exception &e) {
        update_status(Status::ERROR, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    update_status(Status::STOPPING, "Stopping daemon");
}

freighter::Error start_service() {
    LOG(INFO) << "Starting service";
    if (system("/etc/init.d/synnax-driver start") != 0)
        return freighter::Error("Failed to start service");
    return freighter::NIL;
}

freighter::Error stop_service() {
    LOG(INFO) << "Stopping service";
    if (system("/etc/init.d/synnax-driver stop") != 0)
        return freighter::Error("Failed to stop service");
    return freighter::NIL;
}

freighter::Error restart_service() {
    LOG(INFO) << "Restarting service";
    if (system("/etc/init.d/synnax-driver restart") != 0)
        return freighter::Error("Failed to restart service");
    return freighter::NIL;
}

std::string get_log_file_path() {
    return "/var/log/synnax-driver.log";
}

freighter::Error view_logs() {
    int result = system("tail -f /var/log/synnax-driver.log");
    // Exit code 130 indicates Ctrl+C termination
    if (result != 0 && WEXITSTATUS(result) != 130)
        return freighter::Error("Failed to view logs");
    return freighter::NIL;
}

freighter::Error status() {
    LOG(INFO) << "Checking service status";
    int result = system("/etc/init.d/synnax-driver status");
    if (result != 0)
        return freighter::Error("Service is not running");
    return freighter::NIL;
}
} // namespace daemond
