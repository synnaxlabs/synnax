// Copyright 2026 Synnax Labs, Inc.
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
#include <condition_variable>
#include <filesystem>
#include <fstream>
#include <thread>

#include "glog/logging.h"
#include <signal.h>
#include <sys/stat.h>

#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string BINARY_INSTALL_DIR = "/usr/local/bin";
const std::string BINARY_NAME = "synnax-driver";
const std::string INIT_SCRIPT_PATH = "/etc/init.d/synnax-driver";
const std::string DRIVER_PID_FILE = "/var/run/synnax-driver/synnax-driver.pid";

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
PRETTY_NAME="Synnax Driver"
DAEMON="/usr/local/bin/$NAME"
DAEMON_USER="synnax"
PIDFILE="(pid_file)"
LOGFILE="/var/log/$NAME.log"
START_CMD="start -s --disable-stdin-stop"
HEALTH_CHECK_DELAY_SECONDS=2

# Store additional arguments passed to start command
ADDITIONAL_ARGS=""
if [ "$#" -gt 1 ]; then
    shift  # Remove the first argument (which is 'start')
    ADDITIONAL_ARGS="$@"
fi

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Exit if executable not installed. This is an impossible condition.
[ -x "$DAEMON" ] || exit 0

log_message() {
    # First argument is the message
    # Second argument (optional) is the color
    COLOR=${2:-$BLUE}
    echo -e "${COLOR}$1${NC}" | tee -a $LOGFILE
}

VERSIONED_NAME=$($DAEMON version)

do_start() {
    # Check if the driver is already running using the official PID.
    log_message "Starting $VERSIONED_NAME at $(date)" "$BLUE"
    log_message "PID file location: $PIDFILE" "$BLUE"

    # Ensure PID directory exists with correct permissions
    PID_DIR=$(dirname "$PIDFILE")
    if [ ! -d "$PID_DIR" ]; then
        mkdir -p "$PID_DIR"
        chmod 777 "$PID_DIR"
    fi

    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_message "$PRETTY_NAME is already running (PID: $PID)" "$YELLOW"
            return 1
        else
            rm -f "$PIDFILE"
        fi
    fi

    # Add debug logging
    log_message "Starting daemon with command: $DAEMON $START_CMD $ADDITIONAL_ARGS" "$BLUE"
    log_message "Running as user: $(whoami)" "$BLUE"

    # Use start-stop-daemon to properly manage the PID file
    start-stop-daemon --start --background \
        --make-pidfile --pidfile $PIDFILE \
        --startas /bin/bash -- -c "exec $DAEMON $START_CMD $ADDITIONAL_ARGS >> $LOGFILE 2>&1"

    # Wait for health check period
    sleep $HEALTH_CHECK_DELAY_SECONDS

    # Check if process is running
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        log_message "Process started successfully" "$GREEN"
        return 0
    else
        log_message "Process failed to start" "$RED"
        return 1
    fi
}

do_stop() {
    log_message "Stopping $VERSIONED_NAME at $(date)" "$BLUE"
    if [ ! -f "$PIDFILE" ]; then
        log_message "$PRETTY_NAME is not currently running" "$YELLOW"
        return 0
    fi

    PID=$(cat "$PIDFILE")
    if ! kill -0 "$PID" 2>/dev/null; then
        log_message "Removing stale PID file" "$YELLOW"
        rm -f "$PIDFILE"
        return 0
    fi
    log_message "Stopping $PRETTY_NAME with PID $PID" "$BLUE"

    start-stop-daemon --stop --pidfile $PIDFILE --retry 30
    RETVAL=$?
    if [ $RETVAL -eq 0 ]; then
        rm -f $PIDFILE
        log_message "$PRETTY_NAME stopped successfully" "$GREEN"
    else
        log_message "Failed to stop $PRETTY_NAME" "$RED"
    fi
    return $RETVAL
}

do_status() {
    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_message "$PRETTY_NAME is running (PID: $PID)" "$GREEN"
            return 0
        else
            log_message "$PRETTY_NAME is not running (stale PID file)" "$RED"
            return 1
        fi
    else
        log_message "$PRETTY_NAME is not running" "$RED"
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
        echo -e "${RED}Usage: $0 {start|stop|restart|status}${NC}"
        exit 1
        ;;
esac

exit 0
)###";

x::errors::Error create_system_user() {
    LOG(INFO) << "creating system user";
    const int result = system(
        "id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax"
    );
    if (result != 0) return x::errors::Error("failed to create system user");
    return x::errors::NIL;
}

x::errors::Error install_binary() {
    LOG(INFO) << "moving binary to " << BINARY_INSTALL_DIR;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec)
        return x::errors::Error(
            "failed to get current executable path: " + ec.message()
        );

    fs::create_directories(BINARY_INSTALL_DIR, ec);
    if (ec)
        return x::errors::Error("failed to create binary directory: " + ec.message());

    const fs::path target_path = BINARY_INSTALL_DIR + "/" + BINARY_NAME;

    if (fs::exists(target_path)) {
        fs::remove(target_path, ec);
        if (ec)
            return x::errors::Error(
                "failed to remove existing binary: " + ec.message()
            );
    }

    fs::copy_file(curr_bin_path, target_path, fs::copy_options::overwrite_existing, ec);
    if (ec) return x::errors::Error("failed to copy binary: " + ec.message());

    if (chmod(target_path.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) !=
        0)
        return x::errors::Error("failed to set binary permissions");

    return x::errors::NIL;
}

x::errors::Error setup_pid_file() {
    LOG(INFO) << "Setting up dedicated PID directory and file";
    const std::string pid_dir = "/var/run/synnax-driver";
    const std::string pid_file = pid_dir + "/synnax-driver.pid";

    // Check if directory exists with correct permissions
    std::error_code ec;
    if (!fs::exists(pid_dir)) {
        // Setup PID directory
        fs::create_directories(pid_dir, ec);
        if (ec)
            return x::errors::Error(
                "failed to create pid directory. try running with sudo: " + ec.message()
            );
        LOG(INFO) << "PID directory created";

        if (chmod(pid_dir.c_str(), 0755) != 0)
            return x::errors::Error("failed to set PID directory permissions");
        LOG(INFO) << "PID directory permissions set";

        std::string chown_dir_cmd = "chown synnax:synnax " + pid_dir;
        if (system(chown_dir_cmd.c_str()) != 0)
            return x::errors::Error("failed to change owner of PID directory");
        LOG(INFO) << "PID directory ownership changed";
    } else {
        LOG(INFO) << "PID directory already exists";
    }

    // Check if PID file exists
    if (!fs::exists(pid_file)) {
        // Setup PID file
        std::ofstream pid_file_stream(pid_file);
        if (!pid_file_stream)
            return x::errors::Error("failed to create PID file: " + pid_file);
        pid_file_stream.close();
        LOG(INFO) << "PID file created";

        if (chmod(pid_file.c_str(), 0666) != 0)
            return x::errors::Error("failed to set PID file permissions");
        LOG(INFO) << "PID file permissions set";

        std::string chown_file_cmd = "chown synnax:synnax " + pid_file;
        if (system(chown_file_cmd.c_str()) != 0)
            return x::errors::Error("failed to change owner of PID file");
        LOG(INFO) << "PID file ownership changed";
    } else {
        LOG(INFO) << "PID file already exists";
    }

    return x::errors::NIL;
}

x::errors::Error install_service() {
    // Check if service exists and is running
    LOG(INFO) << "checking for existing service";
    if (fs::exists(INIT_SCRIPT_PATH)) {
        LOG(INFO) << "existing service found, stopping and removing it";
        if (int result = system("/etc/init.d/synnax-driver stop"); result != 0)
            LOG(WARNING) << "failed to stop existing service (may not be running)";
        // Give it a moment to stop
        std::this_thread::sleep_for(std::chrono::seconds(2));
        // Uninstall the existing service
        if (auto err = uninstall_service()) return err;
    }

    if (auto err = create_system_user()) return err;
    if (auto err = install_binary()) return err;

    if (auto err = setup_pid_file()) {
        LOG(ERROR) << "failed to setup PID file: " << err.message();
        return err;
    }

    // Create log file with proper permissions
    LOG(INFO) << "Creating log file";
    std::ofstream log_file("/var/log/synnax-driver.log");
    if (!log_file) return x::errors::Error("failed to create log file");
    log_file.close();

    if (chmod("/var/log/synnax-driver.log", 0666) != 0)
        return x::errors::Error("failed to set log file permissions");

    if (system("chown synnax:synnax /var/log/synnax-driver.log") != 0)
        return x::errors::Error("failed to set log file ownership");

    // Update the init script template to reference the new PID file location.
    // We expect the template to contain the marker "(pid_file)" and we replace it.
    std::string init_script = INIT_SCRIPT_TEMPLATE;
    const std::string old_pid_reference = "(pid_file)"; // Marker in the template.
    const std::string new_pid_reference = DRIVER_PID_FILE;
    auto pos = init_script.find(old_pid_reference);
    if (pos != std::string::npos) {
        init_script.replace(pos, old_pid_reference.length(), new_pid_reference);
    }

    LOG(INFO) << "Creating init script at " << INIT_SCRIPT_PATH;
    std::error_code ec;
    fs::create_directories(fs::path(INIT_SCRIPT_PATH).parent_path(), ec);
    if (ec)
        return x::errors::Error("failed to create init.d directory: " + ec.message());

    std::ofstream init_file(INIT_SCRIPT_PATH);
    if (!init_file) return x::errors::Error("failed to create init script");

    init_file << init_script;
    init_file.close();

    if (chmod(
            INIT_SCRIPT_PATH.c_str(),
            S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH
        ) != 0)
        return x::errors::Error("failed to set init script permissions");

    LOG(INFO) << "Configuring service runlevels";
    if (system("update-rc.d synnax-driver defaults") != 0)
        return x::errors::Error("failed to configure service runlevels");

    return x::errors::NIL;
}

x::errors::Error uninstall_service() {
    LOG(INFO) << "Removing service";
    if (int result = system("update-rc.d -f synnax-driver remove"); result != 0)
        LOG(
            WARNING
        ) << "failed to remove service from runlevels (may not be installed)";
    fs::remove(INIT_SCRIPT_PATH);

    // Note: We intentionally don't remove the binary or user
    // in case there are existing configurations or data we want to preserve
    return x::errors::NIL;
}

void update_status(Status status, const std::string &message) {
    std::string status_str;
    switch (status) {
        case Status::INITIALIZING:
            status_str = "Initializing";
            break;
        case Status::READY:
            status_str = "Ready";
            break;
        case Status::RUNNING:
            status_str = "Running";
            break;
        case Status::STOPPING:
            status_str = "Stopping";
            break;
        case Status::ERROR_:
            status_str = "Error";
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
    google::SetLogDestination(google::INFO, "/var/log/synnax-driver");
    update_status(Status::INITIALIZING, "Starting daemon");
    update_status(Status::READY, "Daemon ready");
    try {
        config.callback(argc, argv);
    } catch (const std::exception &e) {
        update_status(Status::ERROR_, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }
    update_status(Status::STOPPING, "Stopping daemon");
}

x::errors::Error check_stranded_processes() {
    // Get current process PID, so we don't kill ourselves.
    pid_t current_pid = getpid();
    // Use pgrep to find all synnax-driver processes
    FILE *pipe = popen("pgrep -x synnax-driver", "r");
    if (!pipe) return x::errors::Error("failed to execute pgrep command");

    std::vector<pid_t> pids;
    char buffer[128];
    while (fgets(buffer, sizeof(buffer), pipe)) {
        pid_t pid = std::stoi(buffer);
        // Don't include current process in the list
        if (pid != current_pid) { pids.push_back(pid); }
    }
    pclose(pipe);

    // Get the "official" PID from the PID file using the new PID location.
    pid_t official_pid = 0;
    std::ifstream pid_file(DRIVER_PID_FILE);
    if (pid_file) {
        pid_file >> official_pid;
        pid_file.close();
    }

    // Kill stranded processes
    bool found_stranded = false;
    for (pid_t pid: pids) {
        if (pid != official_pid && pid != current_pid) {
            LOG(WARNING) << "found stranded driver process with PID: " << pid;
            found_stranded = true;
            if (kill(pid, SIGTERM) != 0) {
                LOG(WARNING) << "failed to terminate process " << pid << ": "
                             << strerror(errno) << ", killing instead.";
                if (kill(pid, SIGKILL) != 0)
                    LOG(ERROR)
                        << "failed to kill process " << pid << ": " << strerror(errno);
            }
        }
    }

    if (found_stranded) LOG(INFO) << "cleaned up stranded processes";

    return x::errors::NIL;
}

x::errors::Error start_service() {
    LOG(INFO) << "starting service";
    if (auto err = check_stranded_processes()) return err;
    if (auto err = setup_pid_file()) return err;
    if (system("/etc/init.d/synnax-driver start") != 0)
        return x::errors::Error("failed to start service");
    return x::errors::NIL;
}

x::errors::Error stop_service() {
    LOG(INFO) << "stopping service";
    // Check for stranded processes before stopping
    if (auto err = check_stranded_processes()) return err;
    // Check if service is running first using the new PID file path
    if (!fs::exists(DRIVER_PID_FILE)) {
        LOG(INFO) << "service is not currently running";
        return x::errors::NIL;
    }

    if (system("/etc/init.d/synnax-driver stop") != 0) {
        return x::errors::Error("failed to stop service");
    }
    return x::errors::NIL;
}

x::errors::Error restart_service() {
    LOG(INFO) << "restarting service";
    // Check for stranded processes before restart
    if (auto err = check_stranded_processes()) return err;
    if (system("/etc/init.d/synnax-driver restart") != 0)
        return x::errors::Error("failed to restart service");
    return x::errors::NIL;
}

std::string get_log_file_path() {
    return "/var/log/synnax-driver.log";
}

x::errors::Error view_logs() {
    int result = system("tail -f /var/log/synnax-driver.log");
    if (result < 0) return x::errors::Error("Failed to execute tail command");
    const int exit_status = WEXITSTATUS(result);
    const bool was_interrupted = WIFSIGNALED(result) && WTERMSIG(result) == SIGINT;
    if (!was_interrupted && exit_status != 0)
        return x::errors::Error("Failed to view logs");
    return x::errors::NIL;
}

x::errors::Error status() {
    LOG(INFO) << "Checking service status";
    const int result = system("/etc/init.d/synnax-driver status");
    if (result != 0) return x::errors::Error("Service is not running");
    return x::errors::NIL;
}
}
