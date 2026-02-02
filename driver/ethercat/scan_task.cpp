// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "glog/logging.h"

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/scan_task.h"

namespace ethercat {
namespace {
std::string get_pdo_error_guidance(const std::string &error) {
    if (error.find("CoE") != std::string::npos ||
        error.find("mailbox") != std::string::npos)
        return "Use manual channel configuration with explicit index/subindex values.";
    if (error.find("SII") != std::string::npos ||
        error.find("fallback") != std::string::npos)
        return "PDO order may be unreliable. Verify data correctness after configuration.";
    if (error.find("empty") != std::string::npos ||
        error.find("no PDO") != std::string::npos)
        return "Device reports no PDOs. Use manual channel configuration if needed.";
    return "Use manual channel configuration if automatic discovery is insufficient.";
}
}

Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::Task task,
    ScanTaskConfig cfg,
    std::shared_ptr<engine::Pool> pool
):
    ctx(std::move(ctx)),
    task(std::move(task)),
    cfg(std::move(cfg)),
    pool(std::move(pool)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX,
    };
}

xerrors::Error Scanner::start() {
    VLOG(1) << SCAN_LOG_PREFIX << "starting EtherCAT scanner";
    return xerrors::NIL;
}

xerrors::Error Scanner::stop() {
    VLOG(1) << SCAN_LOG_PREFIX << "stopping EtherCAT scanner";
    return xerrors::NIL;
}

std::pair<std::vector<synnax::Device>, xerrors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::Device> devices;
    if (this->pool == nullptr) return {devices, xerrors::NIL};

    const auto masters = this->pool->enumerate();
    VLOG(1) << SCAN_LOG_PREFIX << "scanning " << masters.size() << " masters";

    for (const auto &master_info: masters) {
        auto [slaves, err] = this->pool->discover_slaves(master_info.key);
        if (err) {
            VLOG(2) << SCAN_LOG_PREFIX << "discovery failed for " << master_info.key
                    << ": " << err.message();
            continue;
        }

        if (slaves.empty()) continue;

        for (const auto &slave: slaves)
            devices.push_back(
                this->create_slave_device(slave, master_info.key, scan_ctx)
            );
    }

    return {devices, xerrors::NIL};
}

bool Scanner::exec(
    task::Command &cmd,
    const synnax::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == TEST_INTERFACE_CMD_TYPE) {
        this->test_interface(cmd);
        return true;
    }
    return false;
}

synnax::Device Scanner::create_slave_device(
    const SlaveInfo &slave,
    const std::string &master_key,
    const common::ScannerContext &scan_ctx
) const {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const std::string key = this->generate_slave_key(slave, master_key);

    nlohmann::json props = get_existing_properties(key, scan_ctx);
    auto slave_props = slave.to_device_properties(master_key);
    for (auto &[k, v]: slave_props.items())
        props[k] = v;

    std::string status_msg;
    std::string status_variant;
    if (slave.pdos_discovered) {
        if (slave.pdo_discovery_error.empty()) {
            status_msg = "Discovered (" + std::to_string(slave.input_pdos.size()) +
                         " inputs, " + std::to_string(slave.output_pdos.size()) +
                         " outputs)";
            status_variant = status::variant::SUCCESS;
        } else {
            status_msg = "Discovered with warning: " + slave.pdo_discovery_error +
                         ". " + get_pdo_error_guidance(slave.pdo_discovery_error);
            status_variant = status::variant::WARNING;
        }
    } else {
        status_msg = "Discovered (no PDOs found). " + get_pdo_error_guidance("no PDOs");
        status_variant = status::variant::WARNING;
    }

    synnax::Device dev;
    dev.key = key;
    dev.name = slave.name.empty() ? "EtherCAT Slave " + std::to_string(slave.position)
                                  : slave.name;
    dev.make = DEVICE_MAKE;
    dev.model = SLAVE_DEVICE_MODEL;
    dev.location = master_key + ".Slot " + std::to_string(slave.position);
    dev.rack = rack_key;
    dev.properties = props.dump();
    dev.status = synnax::DeviceStatus{
        .key = dev.status_key(),
        .name = dev.name,
        .variant = status_variant,
        .message = status_msg,
        .time = telem::TimeStamp::now(),
        .details = {.rack = rack_key, .device = dev.key},
    };

    return dev;
}

nlohmann::json Scanner::get_existing_properties(
    const std::string &key,
    const common::ScannerContext &scan_ctx
) {
    if (scan_ctx.devices == nullptr) return nlohmann::json::object();
    const auto it = scan_ctx.devices->find(key);
    if (it == scan_ctx.devices->end()) return nlohmann::json::object();
    if (it->second.properties.empty()) return nlohmann::json::object();
    try {
        return nlohmann::json::parse(it->second.properties);
    } catch (const nlohmann::json::parse_error &) { return nlohmann::json::object(); }
}

std::string
Scanner::generate_slave_key(const SlaveInfo &slave, const std::string &master_key) {
    if (slave.serial != 0)
        return "ethercat_" + std::to_string(slave.vendor_id) + "_" +
               std::to_string(slave.product_code) + "_" + std::to_string(slave.serial);
    std::string safe_key = master_key;
    std::replace(safe_key.begin(), safe_key.end(), ':', '_');
    return "ethercat_" + safe_key + "_" + std::to_string(slave.vendor_id) + "_" +
           std::to_string(slave.product_code) + "_" + std::to_string(slave.position);
}

void Scanner::test_interface(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    TestInterfaceArgs args(parser);

    synnax::TaskStatus task_status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = status::variant::ERR,
        .details = synnax::TaskStatusDetails{
            .task = this->task.key,
            .cmd = cmd.key,
            .running = true,
        }
    };

    if (!parser.ok()) {
        LOG(WARNING) << SCAN_LOG_PREFIX << "test_interface: failed to parse command";
        task_status.message = "Failed to parse command";
        task_status.details.data = parser.error_json();
        this->ctx->set_status(task_status);
        return;
    }

    VLOG(1) << SCAN_LOG_PREFIX << "testing interface " << args.interface;
    auto [slaves, err] = this->pool->discover_slaves(args.interface);
    if (err) {
        VLOG(1) << SCAN_LOG_PREFIX << "test_interface failed for "
                << args.interface << ": " << err.message();
        task_status.message = "Failed to probe interface: " + err.message();
        this->ctx->set_status(task_status);
        return;
    }

    VLOG(1) << SCAN_LOG_PREFIX << "test_interface: found " << slaves.size()
            << " slaves on " << args.interface;
    task_status.variant = status::variant::SUCCESS;
    task_status.message = "Found " + std::to_string(slaves.size()) + " slaves on " +
                          args.interface;
    this->ctx->set_status(task_status);
}

}
