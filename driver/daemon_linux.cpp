#include "driver/daemon.h"
#include <systemd/sd-daemon.h>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <filesystem>
#include <fstream>
#include <sys/stat.h>
#include "glog/logging.h"

namespace fs = std::filesystem;
namespace daemon {

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

const char* SYSTEMD_SERVICE_TEMPLATE = R"([Unit]
Description=Synnax Driver Service
Documentation=https://docs.synnaxlabs.com/
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=notify
Environment=GLOG_logtostderr=1
Environment=GLOG_v=1
ExecStart=/usr/local/bin/synnax-driver start
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

freighter::Error create_system_user() {
    int result = system("id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax");
    if (result != 0) {
        return freighter::Error("Failed to create system user");
    }
    return {};
}

freighter::Error install_binary() {
    // Get the path to the current executable
    std::error_code ec;
    fs::path current_exe = fs::read_symlink("/proc/self/exe", ec);
    if (ec) {
        return freighter::Error("Failed to get current executable path: " + ec.message());
    }

    // Create target directory if it doesn't exist
    fs::create_directories("/usr/local/bin", ec);
    if (ec) {
        return freighter::Error("Failed to create binary directory: " + ec.message());
    }

    // Copy the binary
    fs::path target_path = "/usr/local/bin/synnax-driver";
    fs::copy_file(current_exe, target_path,
                  fs::copy_options::overwrite_existing, ec);
    if (ec) {
        return freighter::Error("Failed to copy binary: " + ec.message());
    }

    // Set permissions (755)
    if (chmod(target_path.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0) {
        return freighter::Error("Failed to set binary permissions");
    }

    return {};
}

void install_service() {
    // Create system user
    if (auto err = create_system_user()) {
        throw std::runtime_error(err.message());
    }

    // Install binary
    if (auto err = install_binary()) {
        throw std::runtime_error(err.message());
    }

    const char* service_path = "/etc/systemd/system/synnax-driver.service";

    // Create parent directories if they don't exist
    std::error_code ec;
    fs::create_directories(fs::path(service_path).parent_path(), ec);
    if (ec) {
        throw std::runtime_error("Failed to create service directory: " + ec.message());
    }

    // Write service file
    std::ofstream service_file(service_path);
    if (!service_file) {
        throw std::runtime_error("Failed to create service file");
    }
    service_file << SYSTEMD_SERVICE_TEMPLATE;
    service_file.close();

    // Set permissions (644)
    if (chmod(service_path, S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH) != 0) {
        throw std::runtime_error("Failed to set service file permissions");
    }

    // Reload systemd
    if (system("systemctl daemon-reload") != 0) {
        throw std::runtime_error("Failed to reload systemd");
    }
}

void uninstall_service() {
    // Stop and disable the service
    system("systemctl stop synnax-driver");
    system("systemctl disable synnax-driver");

    // Remove service file
    fs::remove("/etc/systemd/system/synnax-driver.service");

    // Reload systemd
    if (system("systemctl daemon-reload") != 0) {
        throw std::runtime_error("Failed to reload systemd");
    }

    // Note: We intentionally don't remove the binary or user
    // in case there are existing configurations or data we want to preserve
}

void update_status(Status status, const std::string& message) {
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
        case Status::ERROR:
            status_msg += "Error";
            break;
    }

    if (!message.empty()) {
        status_msg += ": " + message;
    }

    if (status == Status::READY) {
        status_msg += "\nREADY=1";
    } else if (status == Status::STOPPING) {
        status_msg += "\nSTOPPING=1";
    }

    sd_notify(0, status_msg.c_str());
}

void notify_watchdog() {
    sd_notify(0, "WATCHDOG=1");
}

void run(const Config& config, int argc, char* argv[]) {
    update_status(Status::INITIALIZING);

    // Start watchdog thread
    std::thread watchdog([&]() {
        while (!should_stop) {
            notify_watchdog();
            std::this_thread::sleep_for(std::chrono::seconds(config.watchdog_interval));
        }
    });

    update_status(Status::READY);
    
    // Run the main application logic
    try {
        config.callback(argc, argv);
    } catch (const std::exception& e) {
        update_status(Status::ERROR, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    // Cleanup
    update_status(Status::STOPPING);
    {
        std::lock_guard<std::mutex> lock(mtx);
        should_stop = true;
    }
    watchdog.join();
}

}  // namespace daemon
