// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <format>
#include <iterator>
#include <map>
#include <set>
#include <string>
#include <tuple>
#include <utility>

#include "absl/log/log.h"

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/scan_task.h"

namespace driver::ethercat {
namespace {
/// An interface that had no slaves is probed again this many scans later.
constexpr size_t EMPTY_INTERFACE_PROBE_INTERVAL = 6;
/// An interface with a slave whose PDO discovery failed is discovered again this many
/// scans later.
constexpr size_t PDO_REDISCOVERY_INTERVAL = 6;

std::string get_pdo_error_guidance(const std::string &error) {
    if (error.contains("SII") || error.contains("fallback"))
        return "PDO order may be unreliable. Verify data correctness after configuration.";
    return "Channels can use only the entries a scan discovers. The Driver tries again "
           "while no task uses the port.";
}

/// @brief sets the variant, message, description, and connection of status from the
/// condition of an enabled slave on a running bus.
void set_live_status(const slave::Health &health, synnax::device::Status &status) {
    status.details.connected = !health.lost;
    if (health.operational()) {
        status.variant = synnax::status::VARIANT_SUCCESS;
        status.message = "Operational";
        return;
    }
    if (health.refusal) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Failed to start";
        status.description = "It " + health.refusal->reason +
                             ". The other devices on the network started without it. "
                             "The Driver tries it again the next time the network "
                             "starts.";
        if (!health.refusal->advice.empty())
            status.description.append(" ").append(health.refusal->advice);
        return;
    }
    if (health.lost) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Not responding";
        status.description = "Check the power and cabling of the device.";
        return;
    }
    if (!health.dc_restore_error.empty()) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Failed to restore the distributed clock";
        status.description = "The device lost its clock settings, and the Driver "
                             "could not set them again: " +
                             health.dc_restore_error +
                             ". The Driver keeps the device out of OP and tries "
                             "again. If the error stays, check the power and cabling "
                             "of the device.";
        return;
    }
    const auto state = slave::to_string(health.state);
    if (health.faulted) {
        status.variant = synnax::status::VARIANT_ERROR;
        status.message = "Error in " + state;
        if (!health.startup_error.empty())
            status.description = errors::sentence(health.startup_error);
        else if (!health.al_status.empty())
            status.description = health.al_status;
        else
            status.description = "The device reports an error without an error "
                                 "code. With IgH, the kernel log (dmesg) shows it.";
        return;
    }
    status.variant = synnax::status::VARIANT_WARNING;
    status.message = "In " + state;
    status.description = "The Driver is returning the device to OP.";
}

/// @brief sets status to the error of saved slave properties that do not parse.
void set_invalid_properties_status(
    const x::errors::Error &err,
    synnax::device::Status &status
) {
    status.variant = synnax::status::VARIANT_ERROR;
    status.message = "Invalid device properties";
    status.description = common::config_err_description(err);
    status.details.connected = false;
}

/// @brief describes a PDO discovery error and what the user can do about it.
std::string pdo_error_description(const std::string &error) {
    return errors::sentence(error) + " " + get_pdo_error_guidance(error);
}

/// @brief whether the catalog of the slave marks an entry that only other software
/// reads, so a scan can show that a switch to Synnax took effect.
bool marks_entries(const slave::Properties &props) {
    if (!props.settings || !props.settings->contains("channels")) return false;
    for (const auto &channel: props.settings->at("channels"))
        for (const auto &entry: channel.at("entries"))
            if (entry.contains("only_read_by")) return true;
    return false;
}

/// @brief returns the program that alone reads an input entry of the slave, or an empty
/// string when Synnax can read each of them.
std::string only_reader(const slave::Properties &props) {
    if (!props.settings || !props.settings->contains("channels")) return "";
    for (const auto &channel: props.settings->at("channels"))
        for (const auto &entry: channel.at("entries"))
            if (entry.contains("only_read_by") &&
                props.find_pdo(
                    entry.at("index").get<uint16_t>(),
                    entry.at("sub_index").get<uint8_t>(),
                    true
                ))
                return entry.at("only_read_by").get<std::string>();
    return "";
}

/// @brief sets status to a warning when other software sets up the slave in a way that
/// keeps Synnax from using it, or when Synnax is on the wrong bus of the slave. Returns
/// whether it set one.
bool set_setup_status(const slave::Properties &props, synnax::device::Status &status) {
    const auto none = x::json::json::object();
    const auto &mode = props.settings && props.settings->contains("mode")
                         ? props.settings->at("mode")
                         : none;
    const auto bus = mode.value("bus", 0);
    auto reader = only_reader(props);
    const bool switched = !reader.empty() && mode.contains("switch") &&
                          mode.at("switch").empty();
    if (reader.empty()) reader = mode.value("only_read_by", "");
    if (switched) {
        status.message = "Restart the power of the device";
        if (bus == 2)
            status.description = "First, connect Synnax to EtherCAT bus 1 IN.";
    } else if (!reader.empty()) {
        status.message = "Set up for " + reader;
        status.description = "Synnax cannot read its data.";
        if (!mode.empty())
            status.description += " In Configure, switch the device to Synnax.";
    } else if (bus == 2 && !mode.value("shared", false)) {
        status.message = "Connect Synnax to EtherCAT bus 1 IN";
    } else
        return false;
    status.variant = synnax::status::VARIANT_WARNING;
    return true;
}

/// @brief adds a failed settings read to the description of status as a warning.
void add_settings_error(const std::string &error, synnax::device::Status &status) {
    if (error.empty()) return;
    if (!status.description.empty()) status.description += ' ';
    status.description += "Failed to read the settings: " + error +
                          ". The Driver tries again while no task uses the port.";
    status.variant = synnax::status::VARIANT_WARNING;
}

/// @brief warns on status when the card holds a mode whose sync mode its SII does not
/// list, and the user chose no sync mode.
void add_unlisted_op_mode(
    const slave::Properties &saved,
    synnax::device::Status &status
) {
    if (saved.config.op_mode || !saved.settings) return;
    const auto mode = saved.settings->find("mode");
    if (mode == saved.settings->end() || !mode->contains("unlisted_assign_activate"))
        return;
    const auto aa = mode->at("unlisted_assign_activate").get<uint16_t>();
    if (!status.description.empty()) status.description += ' ';
    status.description += std::format(
        "The mode of the module needs the sync mode with assign activate value "
        "{:#06x}, ",
        aa
    );
    status.description += saved.op_modes.empty()
                            ? "but the module lists no sync modes. The Driver uses SM "
                              "sync, and the module can refuse to start."
                            : "but the module does not list it. The Driver uses "
                              "another sync mode, and the module can refuse to start. "
                              "Choose a sync mode for the device.";
    status.variant = synnax::status::VARIANT_WARNING;
}

/// @brief whether the Core holds a device that a scan found on the network.
bool holds_devices_on(const common::ScannerContext &ctx, const std::string &network) {
    if (ctx.devices == nullptr) return false;
    return std::ranges::any_of(*ctx.devices, [&network](const auto &entry) {
        const auto &props = entry.second.properties;
        const auto it = props.find("network");
        return it != props.end() && it->second == network;
    });
}

/// @brief whether the master read the identity of the slave. A slave whose EEPROM read
/// failed, as while it restarts, has none.
bool has_identity(const slave::DiscoveryResult &slave) {
    return slave.properties.vendor_id != 0 || slave.properties.product_code != 0;
}

/// @brief whether the master could not read the serial number of the slave.
bool serial_unread(const slave::DiscoveryResult &slave) {
    return !slave.status.serial_error.empty();
}

/// @brief parses slave properties, returning x::errors::VALIDATION with each invalid
/// field when they do not parse.
std::pair<slave::Properties, x::errors::Error>
parse_properties(const x::json::json &json) {
    auto parser = x::json::Parser(json);
    auto props = slave::Properties::parse(parser);
    return {props, parser.error()};
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

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::device::Device> devices;

    std::set<std::string> failures;
    std::vector<std::pair<std::string, std::string>> blocked;
    std::set<std::string> kept;
    const auto masters = this->pool->enumerate();
    VLOG(1) << SCAN_LOG_PREFIX << "scanning " << masters.size() << " masters";

    for (const auto &master_info: masters) {
        const auto &key = master_info.key;
        if (const auto it = this->next_probes.find(key);
            it != this->next_probes.end() && scan_ctx.count < it->second.due) {
            if (!it->second.failure.empty())
                blocked.emplace_back(key, it->second.failure);
            continue;
        }
        const auto due = this->next_rediscoveries.find(key);
        const bool scheduled = due != this->next_rediscoveries.end();
        const bool rediscover = scheduled && scan_ctx.count >= due->second;
        auto [slaves, err] = rediscover ? this->pool->rediscover(key)
                                        : this->pool->discover(key);
        // One permission covers every master (CAP_NET_RAW for SOEM, one udev rule for
        // all IgH devices), so the other masters fail the same way.
        if (err.matches(errors::PERMISSION_DENIED)) return {{}, err};
        if (err.matches(errors::NO_SLAVES_FOUND)) {
            auto &probe = this->next_probes[key];
            probe.due = scan_ctx.count + EMPTY_INTERFACE_PROBE_INTERVAL;
            probe.failure = err.matches(errors::REPLIES_BLOCKED) ? err.data : "";
            if (!probe.failure.empty()) blocked.emplace_back(key, probe.failure);
            continue;
        }
        this->next_probes.erase(key);
        if (err) {
            failures.insert(err.data);
            continue;
        }
        const auto health = this->pool->health(key);
        const auto keys = slave_keys(slaves, key, scan_ctx, kept);
        bool rediscovery_needed = false;
        std::set<uint16_t> unidentified;
        for (size_t i = 0; i < slaves.size(); ++i) {
            const auto &slave = slaves[i];
            if (!has_identity(slave)) {
                LOG(WARNING) << SCAN_LOG_PREFIX << "slave " << slave.properties.position
                             << " on " << key << " has no identity, rediscovering";
                rediscovery_needed = true;
                unidentified.insert(slave.properties.position);
                continue;
            }
            if (serial_unread(slave)) {
                LOG(WARNING) << SCAN_LOG_PREFIX << "slave " << slave.properties.position
                             << " on " << key << " gave no serial number ("
                             << slave.status.serial_error << "), rediscovering";
                rediscovery_needed = true;
                if (keys[i].empty()) continue;
            }
            auto [dev, enabled] = this->create_slave_device(
                slave,
                keys[i],
                health,
                key,
                scan_ctx
            );
            rediscovery_needed |= enabled &&
                                  (!slave.status.pdo_discovery_error.empty() ||
                                   !slave.status.settings_error.empty());
            devices.push_back(std::move(dev));
        }
        if (unidentified.empty())
            this->unidentified.erase(key);
        else if (rediscover) {
            auto &[seen, reported] = this->unidentified[key];
            reported.clear();
            std::ranges::set_intersection(
                unidentified,
                seen,
                std::inserter(reported, reported.begin())
            );
            seen = std::move(unidentified);
        }
        if (const auto it = this->unidentified.find(key);
            it != this->unidentified.end() && !it->second.reported.empty()) {
            const auto [engine, acquire_err] = this->pool->acquire(key);
            if (acquire_err) return {{}, acquire_err};
            for (const auto position: it->second.reported)
                failures.insert(
                    "Slave " + std::to_string(position) + " on " + engine->name() +
                    " gives no identity. Restart its power."
                );
        }
        if (!rediscovery_needed)
            this->next_rediscoveries.erase(key);
        else if (!scheduled || rediscover)
            this->next_rediscoveries[key] = scan_ctx.count + PDO_REDISCOVERY_INTERVAL;
    }

    // A filter on an adapter with no devices can hide a bus only when no adapter shows
    // a device, or when the Core holds devices from the adapter.
    for (const auto &[key, failure]: blocked)
        if (devices.empty() || holds_devices_on(scan_ctx, key))
            failures.insert(failure);
    if (failures.empty()) return {devices, x::errors::NIL};
    std::string desc;
    for (const auto &failure: failures)
        desc += (desc.empty() ? "" : " ") + errors::sentence(failure);
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
    if (cmd.type == SET_STANDALONE_CMD_TYPE) {
        this->set_standalone(cmd);
        return true;
    }
    return false;
}

std::pair<synnax::device::Device, bool> Scanner::create_slave_device(
    const slave::DiscoveryResult &slave,
    const std::string &key,
    const std::vector<slave::Health> &health,
    const std::string &master_key,
    const common::ScannerContext &scan_ctx
) const {
    const auto &props = slave.properties;
    const auto &sts = slave.status;
    const auto rack_key = this->task.rack;

    nlohmann::json json_props = get_existing_properties(key, scan_ctx);
    auto slave_props = props.to_json();
    if (!sts.serial_error.empty()) slave_props.erase("serial");
    for (const auto &[k, v]: slave_props.items())
        if (k != "enabled" || !json_props.contains("enabled")) json_props[k] = v;

    const auto [saved, saved_err] = parse_properties(json_props);
    const bool is_enabled = saved.config.enabled;

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
        .name = dev.name,
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = rack_key, .connected = is_enabled},
        .owner = synnax::device::resource_id(dev.key),
    };
    const auto live = std::ranges::find(
        health,
        props.position,
        &slave::Health::position
    );
    if (saved_err)
        set_invalid_properties_status(saved_err, status);
    else if (!is_enabled) {
        status.message = "Device disabled";
        status.variant = synnax::status::VARIANT_DISABLED;
    } else if (live != health.end())
        set_live_status(*live, status);
    else if (set_setup_status(props, status)) {
        add_settings_error(sts.settings_error, status);
        add_unlisted_op_mode(saved, status);
    } else if (
        !sts.pdo_discovery_error.empty() && props.input_pdos.empty() &&
        props.output_pdos.empty()
    ) {
        status.message = "Failed to discover PDOs";
        status.description = pdo_error_description(sts.pdo_discovery_error);
        status.variant = synnax::status::VARIANT_WARNING;
    } else {
        const auto objects = [](const std::vector<pdo::Properties> &pdos) {
            return std::to_string(std::ranges::count_if(pdos, [](const auto &p) {
                return !p.is_padding();
            }));
        };
        status.message = "Discovered (" + objects(props.input_pdos) + " inputs, " +
                         objects(props.output_pdos) + " outputs)";
        status.variant = synnax::status::VARIANT_SUCCESS;
        // A fallback, such as the SII after a failed CoE read, gives PDOs and an error.
        if (!sts.pdo_discovery_error.empty()) {
            status.description = pdo_error_description(sts.pdo_discovery_error);
            status.variant = synnax::status::VARIANT_WARNING;
        }
        add_settings_error(sts.settings_error, status);
        add_unlisted_op_mode(saved, status);
    }
    dev.status = std::move(status);
    return {std::move(dev), is_enabled};
}

nlohmann::json Scanner::get_existing_properties(
    const std::string &key,
    const common::ScannerContext &scan_ctx
) {
    if (scan_ctx.devices == nullptr) return nlohmann::json::object();
    const auto it = scan_ctx.devices->find(key);
    if (it == scan_ctx.devices->end()) return nlohmann::json::object();
    if (it->second.properties.empty()) return nlohmann::json::object();
    return it->second.properties;
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

std::vector<std::string> Scanner::slave_keys(
    const std::vector<slave::DiscoveryResult> &slaves,
    const std::string &master_key,
    const common::ScannerContext &scan_ctx,
    std::set<std::string> &kept
) {
    std::map<std::string, slave::Properties> stored;
    if (scan_ctx.devices != nullptr)
        for (const auto &[key, dev]: *scan_ctx.devices) {
            if (dev.model != SLAVE_DEVICE_MODEL) continue;
            // A device with invalid properties keeps the fields that parse.
            stored.emplace(key, parse_properties(dev.properties).first);
        }
    const auto find = [&](const slave::DiscoveryResult &slave, const bool by_serial) {
        const auto &live = slave.properties;
        const auto it = std::ranges::find_if(stored, [&](const auto &s) {
            const auto &p = s.second;
            if (kept.contains(s.first) || p.vendor_id != live.vendor_id ||
                p.product_code != live.product_code)
                return false;
            const bool same_slot = p.network == master_key &&
                                   p.position == live.position;
            if (serial_unread(slave)) return !by_serial && same_slot;
            if (by_serial && live.serial != 0) return p.serial == live.serial;
            return p.serial == 0 && same_slot;
        });
        return it == stored.end() ? std::string() : it->first;
    };
    // A stored device with an identity that no slave took is the device of another
    // unit. A device with no identity is the device of the slave that its key names.
    const auto taken = [&](const std::string &key) {
        if (kept.contains(key)) return true;
        const auto it = stored.find(key);
        return it != stored.end() &&
               (it->second.vendor_id != 0 || it->second.product_code != 0);
    };
    std::vector<std::string> keys(slaves.size());
    // A device goes to a slave by its position only after each slave has taken its
    // own device.
    for (const bool by_serial: {true, false})
        for (size_t i = 0; i < slaves.size(); ++i) {
            if (!keys[i].empty() || !has_identity(slaves[i])) continue;
            keys[i] = find(slaves[i], by_serial);
            if (!keys[i].empty()) kept.insert(keys[i]);
        }
    for (size_t i = 0; i < slaves.size(); ++i) {
        if (!keys[i].empty() || !has_identity(slaves[i]) || serial_unread(slaves[i]))
            continue;
        const auto base = generate_slave_key(slaves[i].properties, master_key);
        auto key = base;
        for (int n = 2; taken(key); ++n)
            key = base + "_" + std::to_string(n);
        keys[i] = key;
        kept.insert(key);
    }
    return keys;
}

void Scanner::on_device_set(const synnax::device::Device &dev) {
    if (dev.model != SLAVE_DEVICE_MODEL) return;
    const auto [props, err] = parse_properties(dev.properties);
    const auto health = this->pool->health(props.network);
    const auto live = std::ranges::find(
        health,
        props.position,
        &slave::Health::position
    );
    const bool connected = props.config.enabled &&
                           (live == health.end() || !live->lost);
    synnax::device::Status status{
        .name = dev.name,
        .variant = props.config.enabled ? synnax::status::VARIANT_SUCCESS
                                        : synnax::status::VARIANT_DISABLED,
        .message = props.config.enabled ? "Device enabled" : "Device disabled",
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = dev.rack, .connected = connected},
        .owner = synnax::device::resource_id(dev.key),
    };
    if (err) set_invalid_properties_status(err, status);
    if (auto status_err = this->ctx->client->statuses.set(status); status_err)
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "failed to update device status: " << status_err;
}

void Scanner::test_interface(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    TestInterfaceArgs args(parser);

    synnax::task::Status task_status{
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details =
            synnax::task::StatusDetails{
                .running = true,
                .cmd = cmd.key,
                .config_hash = this->task.config_hash,
            },
        .owner = synnax::task::resource_id(this->task.key),
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
    auto [engine, err] = this->pool->acquire(args.interface);
    std::vector<slave::DiscoveryResult> slaves;
    if (!err) std::tie(slaves, err) = engine->discover();
    if (err) {
        VLOG(1) << SCAN_LOG_PREFIX << "test_interface failed for "
                << args.interface << ": " << err.message();
        task_status.message = "Failed to probe interface";
        task_status.description = errors::sentence(err.data);
        this->ctx->set_status(task_status);
        return;
    }

    VLOG(1) << SCAN_LOG_PREFIX << "test_interface: found " << slaves.size()
            << " slaves on " << args.interface;
    task_status.variant = synnax::status::VARIANT_SUCCESS;
    task_status.message = "Found " + std::to_string(slaves.size()) + " slaves on " +
                          engine->name();
    this->ctx->set_status(task_status);
}

void Scanner::set_standalone(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const SetStandaloneArgs args(parser);
    synnax::task::Status task_status{
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details =
            synnax::task::StatusDetails{
                .running = true,
                .cmd = cmd.key,
                .config_hash = this->task.config_hash,
            },
        .owner = synnax::task::resource_id(this->task.key),
    };
    const auto fail = [&](const std::string &description) {
        task_status.message = "Failed to switch the device";
        task_status.description = errors::sentence(description);
        this->ctx->set_status(task_status);
    };
    if (!parser.ok()) {
        task_status.details.data = parser.error_json();
        fail(common::config_err_description(parser.error()));
        return;
    }
    const auto [dev, dev_err] = this->ctx->client->devices.retrieve(args.device);
    if (dev_err) {
        fail(dev_err.message());
        return;
    }
    auto [saved, saved_err] = parse_properties(dev.properties);
    if (saved_err) {
        fail(common::config_err_description(saved_err));
        return;
    }
    const auto [engine, engine_err] = this->pool->acquire(saved.network);
    if (engine_err) {
        fail(engine_err.data);
        return;
    }
    saved.name = dev.name;
    std::vector<slave::StartupParameter> written;
    uint16_t position = 0;
    std::pair<uint16_t, uint8_t> mode_object;
    const auto err = engine->configure(
        saved,
        [&](const slave::DiscoveryResult &found)
            -> std::pair<std::vector<slave::StartupParameter>, x::errors::Error> {
            const auto &props = found.properties;
            position = props.position;
            if (!props.settings || !props.settings->contains("mode"))
                return {
                    {},
                    x::errors::Error(
                        errors::SLAVE_CONFIG_ERROR,
                        "Synnax does not know how to switch this device."
                    )
                };
            std::vector<slave::StartupParameter> parameters;
            x::json::Parser mode(props.settings->at("mode"));
            mode_object = {
                mode.field<uint16_t>("index"),
                mode.field<uint8_t>("sub_index"),
            };
            mode.iter("switch", [&parameters](x::json::Parser &param) {
                parameters.push_back(slave::StartupParameter::parse(param));
            });
            if (!mode.ok())
                return {
                    {},
                    x::errors::Error(
                        errors::SLAVE_CONFIG_ERROR,
                        common::config_err_description(mode.error())
                    )
                };
            written = parameters;
            return {std::move(parameters), x::errors::NIL};
        }
    );
    if (written.empty()) {
        if (err) {
            fail(err.data);
            return;
        }
        task_status.variant = synnax::status::VARIANT_SUCCESS;
        task_status.message = "Switched " + dev.name + " to Synnax only";
        task_status.description = std::format(
            "The device in slot {} already runs Synnax only.",
            position
        );
        this->ctx->set_status(task_status);
        return;
    }
    const auto sets_mode = [&mode_object](const slave::StartupParameter &p) {
        return std::pair{p.index, p.sub_index} == mode_object;
    };
    const auto mode = std::ranges::find_if(written, sets_mode);
    if (mode == written.end()) {
        fail(err ? err.data : "the switch does not set the mode of the device");
        return;
    }
    const auto [held, held_err] = engine->read(position, mode->index, mode->sub_index);
    if (held_err || static_cast<double>(held) != mode->value) {
        if (err)
            fail(err.data);
        else if (held_err)
            fail(held_err.data);
        else
            fail(std::format("the device holds mode {} after the switch", held));
        return;
    }
    // configure also returns the error of its scan after the write.
    if (err.matches(errors::STARTUP_PARAMETER_ERROR)) {
        task_status.variant = synnax::status::VARIANT_WARNING;
        task_status.message = "Switched " + dev.name +
                              " to Synnax until a power restart";
        task_status.description = std::format(
            "The device in slot {} did not save the switch, so it goes back to its "
            "saved mode when its power restarts. {}",
            position,
            errors::sentence(err.data)
        );
        this->ctx->set_status(task_status);
        return;
    }
    // The other parameters save the mode. What a save object reads after a save
    // differs by device, so the Driver only logs it.
    for (const auto &save: written)
        if (!sets_mode(save)) {
            const auto [flag, flag_err] = engine->read(
                position,
                save.index,
                save.sub_index
            );
            VLOG(1) << SCAN_LOG_PREFIX << dev.name << " reads "
                    << (flag_err ? flag_err.data : std::to_string(flag)) << " at "
                    << save.to_string() << " after the switch";
        }
    // configure discovered the slave again, with the entries of its new mode.
    const auto [slaves, discover_err] = engine->discover();
    const auto after = std::ranges::find(
        slaves,
        position,
        [](const slave::DiscoveryResult &s) { return s.properties.position; }
    );
    task_status.variant = synnax::status::VARIANT_SUCCESS;
    task_status.message = "Switched " + dev.name + " to Synnax only";
    const bool restart = err || discover_err || after == slaves.end() ||
                         !after->status.settings_error.empty() ||
                         !marks_entries(after->properties) ||
                         !only_reader(after->properties).empty();
    task_status.description = restart
                                ? std::format(
                                      "Restart the power of the device in slot {} to "
                                      "finish.",
                                      position
                                  )
                                : std::format(
                                      "The device in slot {} runs Synnax only now.",
                                      position
                                  );
    this->ctx->set_status(task_status);
}
}
