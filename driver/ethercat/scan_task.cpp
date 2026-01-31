// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "glog/logging.h"

#include "driver/ethercat/scan_task.h"

extern "C" {
#include "soem/soem.h"
}

#include "driver/ethercat/master/master.h"

namespace ethercat {

std::shared_ptr<Master> create_master(
    const std::string &interface_name,
    const std::string &backend,
    unsigned int master_index
);
Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::Task task,
    ScanTaskConfig cfg,
    Factory *factory
):
    ctx(std::move(ctx)), task(std::move(task)), cfg(std::move(cfg)), factory(factory) {}

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
    const auto interfaces = this->enumerate_interfaces();
    VLOG(1) << SCAN_LOG_PREFIX << "scanning " << interfaces.size() << " interfaces";

    for (const auto &iface: interfaces) {
        std::vector<SlaveInfo> slaves;
        const bool is_active = this->factory != nullptr &&
                               this->factory->is_interface_active(iface.name);

        if (is_active) {
            VLOG(2) << SCAN_LOG_PREFIX << "using cached slaves for " << iface.name;
            slaves = this->factory->get_cached_slaves(iface.name);
        } else {
            VLOG(2) << SCAN_LOG_PREFIX << "probing " << iface.name;
            auto [probed_slaves, err] = this->probe_interface(iface.name);
            if (err) {
                VLOG(2) << SCAN_LOG_PREFIX << "probe failed for " << iface.name << ": "
                        << err.message();
                continue;
            }
            slaves = std::move(probed_slaves);
            if (!slaves.empty()) {
                LOG(INFO) << SCAN_LOG_PREFIX << "discovered " << slaves.size()
                          << " slaves on " << iface.name;
            }
        }

        if (slaves.empty()) continue;

        auto network_dev = this->create_network_device(iface, slaves);
        devices.push_back(std::move(network_dev));

        for (const auto &slave: slaves) {
            auto slave_dev = this->create_slave_device(slave, iface.name);
            devices.push_back(std::move(slave_dev));
        }
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

std::vector<InterfaceInfo> Scanner::enumerate_interfaces() {
    std::vector<InterfaceInfo> interfaces;
    ec_adaptert *adapter = ec_find_adapters();
    ec_adaptert *current = adapter;

    while (current != nullptr) {
        InterfaceInfo info;
        info.name = current->name;
        info.description = current->desc;
        interfaces.push_back(std::move(info));
        current = current->next;
    }

    ec_free_adapters(adapter);
    return interfaces;
}

std::pair<std::vector<SlaveInfo>, xerrors::Error>
Scanner::probe_interface(const std::string &interface) const {
    const auto master = create_master(interface, this->cfg.backend, 0);
    if (auto err = master->initialize()) return {{}, err};
    return {master->slaves(), xerrors::NIL};
}

synnax::Device Scanner::create_network_device(
    const InterfaceInfo &iface,
    const std::vector<SlaveInfo> &slaves
) {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const std::string key = this->generate_network_key(iface.name);

    nlohmann::json props;
    props["interface"] = iface.name;
    props["slave_count"] = slaves.size();

    const std::string status_msg = "Discovered " + std::to_string(slaves.size()) +
                                   " slaves";
    const std::string status_variant = status::variant::SUCCESS;

    synnax::Device dev;
    dev.key = key;
    dev.name = "EtherCAT Network " + iface.name;
    dev.make = DEVICE_MAKE;
    dev.model = NETWORK_DEVICE_MODEL;
    dev.location = iface.name;
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

synnax::Device Scanner::create_slave_device(
    const SlaveInfo &slave,
    const std::string &network_interface
) {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const std::string key = this->generate_slave_key(slave, network_interface);

    nlohmann::json props;
    props["vendor_id"] = slave.vendor_id;
    props["product_code"] = slave.product_code;
    props["revision"] = slave.revision;
    props["serial"] = slave.serial;
    props["name"] = slave.name;
    props["network"] = network_interface;
    props["position"] = slave.position;
    props["input_bits"] = slave.input_bits;
    props["output_bits"] = slave.output_bits;

    nlohmann::json input_pdos = nlohmann::json::array();
    for (const auto &pdo: slave.input_pdos) {
        input_pdos.push_back(
            {{"name", pdo.name},
             {"pdo_index", pdo.pdo_index},
             {"index", pdo.index},
             {"subindex", pdo.subindex},
             {"bit_length", pdo.bit_length},
             {"data_type", pdo.data_type.name()}}
        );
    }

    nlohmann::json output_pdos = nlohmann::json::array();
    for (const auto &pdo: slave.output_pdos) {
        output_pdos.push_back(
            {{"name", pdo.name},
             {"pdo_index", pdo.pdo_index},
             {"index", pdo.index},
             {"subindex", pdo.subindex},
             {"bit_length", pdo.bit_length},
             {"data_type", pdo.data_type.name()}}
        );
    }

    props["pdos"] = {{"inputs", input_pdos}, {"outputs", output_pdos}};

    std::string status_msg;
    std::string status_variant;
    if (slave.pdos_discovered) {
        if (slave.pdo_discovery_error.empty()) {
            status_msg = "Discovered (" + std::to_string(slave.input_pdos.size()) +
                         " inputs, " + std::to_string(slave.output_pdos.size()) +
                         " outputs)";
            status_variant = status::variant::SUCCESS;
        } else {
            status_msg = "Discovered (PDO enumeration: " + slave.pdo_discovery_error +
                         ")";
            status_variant = status::variant::WARNING;
        }
    } else {
        status_msg = "Discovered (no PDOs found)";
        status_variant = status::variant::WARNING;
    }

    synnax::Device dev;
    dev.key = key;
    dev.name = slave.name.empty() ? "EtherCAT Slave " + std::to_string(slave.position)
                                  : slave.name;
    dev.make = DEVICE_MAKE;
    dev.model = SLAVE_DEVICE_MODEL;
    dev.location = network_interface;
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

std::string Scanner::generate_network_key(const std::string &interface) {
    return "ethercat_" + interface;
}

std::string
Scanner::generate_slave_key(const SlaveInfo &slave, const std::string &interface) {
    if (slave.serial != 0) {
        return "ethercat_" + std::to_string(slave.vendor_id) + "_" +
               std::to_string(slave.product_code) + "_" + std::to_string(slave.serial);
    }
    return "ethercat_" + interface + "_" + std::to_string(slave.vendor_id) + "_" +
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
    auto [slaves, err] = this->probe_interface(args.interface);
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
