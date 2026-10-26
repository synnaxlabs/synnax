// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cctype>
#include <ranges>
#include <set>
#include <string>
#include <utility>

#include "absl/log/log.h"

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/scan_task.h"

namespace driver::ethercat {
namespace {
/// An interface that had no slaves is probed again this many scans later.
constexpr size_t EMPTY_INTERFACE_PROBE_INTERVAL = 6;

std::string get_pdo_error_guidance(const std::string &error) {
    if (error.contains("CoE") || error.contains("mailbox"))
        return "Check the slave's CoE mailbox and scan again. Channels can use only "
               "the entries a scan discovers.";
    if (error.contains("SII") || error.contains("fallback"))
        return "PDO order may be unreliable. Verify data correctness after configuration.";
    return "Channels can use only the entries a scan discovers. Scan again if entries "
           "are missing.";
}

/// @brief sets the variant, message, and description of status from the condition of an
/// enabled slave on a running bus.
void set_live_status(const slave::Health &health, synnax::device::Status &status) {
    if (health.operational()) {
        status.variant = synnax::status::VARIANT_SUCCESS;
        status.message = "Operational";
        return;
    }
    if (health.lost) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Not responding";
        status.description = "Check the power and cabling of the device.";
        return;
    }
    const auto state = slave::slave_state_to_string(health.state);
    if (health.faulted) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Error in " + state;
        status.description = health.al_status.empty()
                               ? "The device reports an error without an error "
                                 "code. With IgH, the kernel log (dmesg) shows it."
                               : health.al_status;
        return;
    }
    status.variant = synnax::status::VARIANT_WARNING;
    status.message = "In " + state;
    status.description = "The Driver is returning the device to OP.";
}

/// @brief describes a PDO discovery error and what the user can do about it.
std::string pdo_error_description(const std::string &error) {
    auto sentence = error;
    sentence[0] = static_cast<char>(
        std::toupper(static_cast<unsigned char>(sentence[0]))
    );
    return sentence + ". " + get_pdo_error_guidance(error);
}
}

Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    ScanTaskConfig cfg,
    std::shared_ptr<engine::Pool> pool
):
    ctx(std::move(ctx)), task(std::move(task)), cfg(cfg), pool(std::move(pool)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX,
    };
}

x::errors::Error Scanner::start() {
    VLOG(1) << SCAN_LOG_PREFIX << "starting EtherCAT scanner";
    return x::errors::NIL;
}

x::errors::Error Scanner::stop() {
    VLOG(1) << SCAN_LOG_PREFIX << "stopping EtherCAT scanner";
    return x::errors::NIL;
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::device::Device> devices;
    if (this->pool == nullptr) return {devices, x::errors::NIL};

    if (scan_ctx.count == 0 && scan_ctx.devices != nullptr) {
        for (const auto &dev: *scan_ctx.devices | std::views::values) {
            auto props_parser = x::json::Parser(dev.properties);
            auto props = slave::Properties::parse(props_parser);
            if (props_parser.error() || props.enabled) continue;

            auto [engine, err] = this->pool->acquire(props.network);
            if (err) continue;

            LOG(INFO) << SCAN_LOG_PREFIX << "initializing slave " << props.position
                      << " on " << props.network << " as disabled";
            engine->set_slave_enabled(props.position, props.enabled);
        }
    }

    std::set<std::string> failures;
    const auto masters = this->pool->enumerate();
    VLOG(1) << SCAN_LOG_PREFIX << "scanning " << masters.size() << " masters";

    for (const auto &master_info: masters) {
        const auto &key = master_info.key;
        if (const auto it = this->next_probes.find(key);
            it != this->next_probes.end() && scan_ctx.count < it->second)
            continue;
        auto [slaves, err] = this->pool->discover_slaves(key);
        // One permission covers every master (CAP_NET_RAW for SOEM, one udev rule for
        // all IgH devices), so the other masters fail the same way.
        if (err.matches(errors::PERMISSION_DENIED)) return {{}, err};
        if (err.matches(errors::NO_SLAVES_FOUND)) {
            this->next_probes[key] = scan_ctx.count + EMPTY_INTERFACE_PROBE_INTERVAL;
            continue;
        }
        this->next_probes.erase(key);
        if (err) {
            failures.insert(err.data);
            continue;
        }
        const auto health = this->pool->health(key);
        for (const auto &slave: slaves)
            devices.push_back(this->create_slave_device(slave, health, key, scan_ctx));
    }

    if (failures.empty()) return {devices, x::errors::NIL};
    std::string desc;
    for (const auto &failure: failures)
        desc += (desc.empty() ? "" : "; ") + failure;
    return {devices, x::errors::Error(common::PARTIAL_SCAN_ERROR, desc)};
}

bool Scanner::exec(
    synnax::task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == TEST_INTERFACE_CMD_TYPE) {
        this->test_interface(cmd);
        return true;
    }
    return false;
}

synnax::device::Device Scanner::create_slave_device(
    const slave::DiscoveryResult &slave,
    const std::vector<slave::Health> &health,
    const std::string &master_key,
    const common::ScannerContext &scan_ctx
) const {
    const auto &props = slave.properties;
    const auto &sts = slave.status;
    const auto rack_key = this->task.rack;
    const std::string key = this->generate_slave_key(props, master_key);

    nlohmann::json json_props = get_existing_properties(key, scan_ctx);
    auto slave_props = props.to_json();
    for (const auto &[k, v]: slave_props.items())
        if (k != "enabled" || !json_props.contains("enabled")) json_props[k] = v;

    const bool is_enabled = json_props["enabled"].get<bool>();

    synnax::device::Device dev;
    dev.key = key;
    dev.name = props.name.empty() ? "EtherCAT Slave " + std::to_string(props.position)
                                  : props.name;
    dev.make = DEVICE_MAKE;
    dev.model = SLAVE_DEVICE_MODEL;
    dev.location = master_key + ".Slot " + std::to_string(props.position);
    dev.rack = rack_key;
    dev.properties = json_props;
    synnax::device::Status status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = rack_key, .device = dev.key},
    };
    const auto live = std::ranges::find(
        health,
        props.position,
        &slave::Health::position
    );
    if (!is_enabled) {
        status.message = "Device disabled";
        status.variant = synnax::status::VARIANT_DISABLED;
    } else if (live != health.end())
        set_live_status(*live, status);
    else if (sts.pdos_discovered && sts.pdo_discovery_error.empty()) {
        const auto objects = [](const std::vector<pdo::Properties> &pdos) {
            return std::to_string(std::ranges::count_if(pdos, [](const auto &p) {
                return !p.is_padding();
            }));
        };
        status.message = "Discovered (" + objects(props.input_pdos) + " inputs, " +
                         objects(props.output_pdos) + " outputs)";
        status.variant = synnax::status::VARIANT_SUCCESS;
    } else {
        status.message = "Failed to discover PDOs";
        status.description = pdo_error_description(
            sts.pdo_discovery_error.empty() ? "no PDOs found" : sts.pdo_discovery_error
        );
        status.variant = synnax::status::VARIANT_WARNING;
    }
    dev.status = std::move(status);
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
        return it->second.properties;
    } catch (const nlohmann::json::parse_error &) { return nlohmann::json::object(); }
}

std::string Scanner::generate_slave_key(
    const slave::Properties &slave,
    const std::string &master_key
) {
    if (slave.serial != 0)
        return "ethercat_" + std::to_string(slave.vendor_id) + "_" +
               std::to_string(slave.product_code) + "_" + std::to_string(slave.serial);
    std::string safe_key = master_key;
    std::ranges::replace(safe_key, ':', '_');
    return "ethercat_" + safe_key + "_" + std::to_string(slave.vendor_id) + "_" +
           std::to_string(slave.product_code) + "_" + std::to_string(slave.position);
}

void Scanner::on_device_set(const synnax::device::Device &dev) {
    auto props_parser = x::json::Parser(dev.properties);
    auto props = slave::Properties::parse(props_parser);
    if (props_parser.error()) return;

    auto [engine, err] = this->pool->acquire(props.network);
    if (err) return;

    LOG(INFO) << SCAN_LOG_PREFIX << "setting slave " << props.position << " on "
              << props.network << " enabled=" << (props.enabled ? "true" : "false");
    engine->set_slave_enabled(props.position, props.enabled);

    synnax::device::Status status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = props.enabled ? synnax::status::VARIANT_SUCCESS
                                 : synnax::status::VARIANT_DISABLED,
        .message = props.enabled ? "Device enabled" : "Device disabled",
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = dev.rack, .device = dev.key},
    };
    if (auto status_err = this->ctx->client->statuses.set(status); status_err)
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "failed to update device status: " << status_err;
}

void Scanner::test_interface(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    TestInterfaceArgs args(parser);

    synnax::task::Status task_status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = this->task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = this->task.config_hash,
        }
    };

    if (!parser.ok()) {
        LOG(WARNING) << SCAN_LOG_PREFIX << "test_interface: failed to parse command";
        task_status.message = "Failed to parse test command";
        task_status.description = common::config_err_description(parser.error());
        task_status.details.data = parser.error_json();
        this->ctx->set_status(task_status);
        return;
    }

    VLOG(1) << SCAN_LOG_PREFIX << "testing interface " << args.interface;
    auto [slaves, err] = this->pool->discover_slaves(args.interface);
    if (err) {
        VLOG(1) << SCAN_LOG_PREFIX << "test_interface failed for "
                << args.interface << ": " << err.message();
        task_status.message = "Failed to probe interface";
        task_status.description = err.data;
        this->ctx->set_status(task_status);
        return;
    }

    VLOG(1) << SCAN_LOG_PREFIX << "test_interface: found " << slaves.size()
            << " slaves on " << args.interface;
    task_status.variant = synnax::status::VARIANT_SUCCESS;
    task_status.message = "Found " + std::to_string(slaves.size()) + " slaves on " +
                          args.interface;
    this->ctx->set_status(task_status);
}

}
