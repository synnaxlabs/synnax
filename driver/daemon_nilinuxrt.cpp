/// std.
#include <thread>
#include <mutex>
#include <condition_variable>
#include <filesystem>
#include <fstream>

/// external.
#include <sys/stat.h>
#include "glog/logging.h"

/// internal
#include "driver/daemon.h"

namespace fs = std::filesystem;

namespace daemond {

const std::string BINARY_INSTALL_DIR = "/usr/local/bin";
const std::string BINARY_NAME = "synnax-driver";
const std::string INIT_SCRIPT_PATH = "/etc/init.d/synnax-driver";

const char* INIT_SCRIPT_TEMPLATE = R"(#!/bin/sh
### BEGIN INIT INFO
# Provides:          synnax-driver
# Required-Start:    $network $local_fs
# Required-Stop:     $network $local_fs
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

# Load init functions
. /lib/lsb/init-functions

do_start() {
    log_daemon_msg "Starting $NAME"
    start-stop-daemon --start --background \
        --make-pidfile --pidfile $PIDFILE \
        --chuid $DAEMON_USER \
        --exec $DAEMON -- start
    log_end_msg $?
}

do_stop() {
    log_daemon_msg "Stopping $NAME"
    start-stop-daemon --stop --pidfile $PIDFILE --retry 30
    log_end_msg $?
    rm -f $PIDFILE
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
        status_of_proc -p $PIDFILE "$DAEMON" "$NAME"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
)";

freighter::Error create_system_user() {
    LOG(INFO) << "Creating system user";
    int result = system("id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax");
    if (result != 0) {
        return freighter::Error("Failed to create system user");
    }
    return freighter::NIL;
}

freighter::Error install_binary() {
    LOG(INFO) << "Moving binary to " << BINARY_INSTALL_DIR;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec) 
        return freighter::Error("Failed to get current executable path: " + ec.message());

    fs::create_directories(BINARY_INSTALL_DIR, ec);
    if (ec)
        return freighter::Error("Failed to create binary directory: " + ec.message());

    // Copy the binary
    const fs::path target_path = BINARY_INSTALL_DIR + "/" + BINARY_NAME;
    fs::copy_file(
        curr_bin_path,
        target_path,
        fs::copy_options::overwrite_existing,
        ec
    );
    if (ec)
        return freighter::Error("Failed to copy binary: " + ec.message());

    if (chmod(target_path.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0)
        return freighter::Error("Failed to set binary permissions");

    return freighter::NIL;
}

freighter::Error install_service() {
    if (auto err = create_system_user()) return err;
    if (auto err = install_binary()) return err;

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

    if (chmod(INIT_SCRIPT_PATH.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0)
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

void update_status(Status status, const std::string& message) {
    std::string status_str;
    switch (status) {
        case Status::INITIALIZING: status_str = "Initializing"; break;
        case Status::READY: status_str = "Ready"; break;
        case Status::RUNNING: status_str = "Running"; break;
        case Status::STOPPING: status_str = "Stopping"; break;
        case Status::ERROR: status_str = "Error"; break;
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

void run(const Config& config, int argc, char* argv[]) {
    update_status(Status::INITIALIZING);
    update_status(Status::READY);
    
    try {
        config.callback(argc, argv);
    } catch (const std::exception& e) {
        update_status(Status::ERROR, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    update_status(Status::STOPPING);
}

}  // namespace daemond 