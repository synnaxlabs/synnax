// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <chrono>
#include <format>
#include <functional>
#include <iterator>
#include <string_view>
#include <thread>
#include <tuple>

#include "absl/log/log.h"

#include "x/cpp/binary/binary.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/sii.h"
#include "driver/ethercat/soem/util.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// Timeout for cyclic process data exchange in microseconds.
const int PROCESSDATA_TIMEOUT = static_cast<int>(x::telem::MILLISECOND.microseconds());
/// Time between state reads while slaves move to a requested state.
const auto STATE_POLL_INTERVAL = x::telem::MILLISECOND;
/// AL status codes of faults that a new start can clear. The start takes every slave
/// back through INIT with an acknowledgment.
constexpr std::array<uint16_t, 16> TRANSIENT_FAULTS = {
    0x0018, // No valid inputs available
    0x0019, // No valid outputs
    0x001A, // Synchronization error
    0x001B, // Sync manager watchdog
    0x0021, // Slave needs INIT
    0x0022, // Slave needs PREOP
    0x0023, // Slave needs SAFEOP
    0x002A, // Background watchdog
    0x002B, // No valid inputs and outputs
    0x002C, // Fatal sync error
    0x002D, // No sync error
    0x0032, // PLL error
    0x0033, // DC sync IO error
    0x0034, // DC sync timeout error
    0x0052, // External hardware not ready
    0x0060, // Slave restarted locally
};
/// Timeout for one recovery step on a slave in microseconds.
const int RECOVERY_TIMEOUT = static_cast<int>(
    (x::telem::MILLISECOND * 500).microseconds()
);
/// Timeout for one SDO (Service Data Object) read or write in microseconds.
const int SDO_TIMEOUT = static_cast<int>((x::telem::MILLISECOND * 700).microseconds());
/// The object in which a modular slave lists the identities of the modules it detects.
constexpr uint16_t DETECTED_MODULES_INDEX = 0xF050;
/// The CoE identity object and the sub-index that holds the serial number.
constexpr uint16_t IDENTITY_INDEX = 0x1018;
constexpr uint8_t SERIAL_SUB_INDEX = 4;
/// The largest object a complete access read returns: sub-index 0 as 16 bits, then 255
/// entries of four bytes.
constexpr size_t MAX_OBJECT_SIZE = 2 + (255 * 4);

/// Time from the read of a slave's DC system time to its first SYNC0 event, so the
/// slave takes the start time before the event passes.
const auto DC_START_DELAY = x::telem::MILLISECOND * 100;
/// The largest system time difference (0x092C) of a settled slave clock.
const auto DC_SETTLED_DIFF = x::telem::MICROSECOND * 10;
/// Time between reads of the system time differences while the clocks settle.
const auto DC_SETTLE_POLL_INTERVAL = x::telem::MILLISECOND * 10;
namespace {
/// @brief converts a span to the microseconds that SOEM timeouts take.
int micros(const x::telem::TimeSpan span) {
    return static_cast<int>(span.microseconds());
}

/// @brief returns what the user can do about a slave that refuses a state with the AL
/// status code, or an empty view for no advice.
std::string_view refusal_advice(const uint16_t code) {
    switch (code) {
        case 0x001D: // Invalid output configuration
        case 0x001E: // Invalid input configuration
            return slave::MODE_RESTART_ADVICE;
        case 0x0030: // Invalid DC SYNC configuration
            return slave::DC_SYNC_ADVICE;
        default:
            return {};
    }
}

/// @brief describes what a slave did with a datagram that failed with the working
/// counter wkc: below 0 means no frame came back.
std::string_view response(const int wkc) {
    return wkc < 0 ? "did not reply to" : "refused";
}

}

/// SOEM group 0 addresses every slave, so enabled and disabled slaves get their own.
constexpr uint8_t ENABLED_GROUP = 1;
constexpr uint8_t DISABLED_GROUP = 2;
// SOEM indexes its group list by the group of each slave.
static_assert(DISABLED_GROUP < EC_MAXGROUP);
/// IOMap buffer size for process data image. This is local memory only - actual wire
/// traffic is determined by slave PDO mappings. 128KB handles networks with hundreds
/// of slaves or slaves with large PDO mappings (e.g., multi-axis drives).
constexpr size_t DEFAULT_IOMAP_SIZE = 131072;

namespace {
/// The SOEM packet error codes of an SDO transfer.
constexpr uint16_t UNEXPECTED_FRAME = 1;
constexpr uint16_t CONTAINER_TOO_SMALL = 3;

/// @brief empties the SOEM error list, returning why the latest SDO transfer of the
/// object in it failed. With no error for the object, the transfer got no reply.
errors::SDOFailure pop_sdo_failure(
    API &api,
    const uint16_t slave,
    const uint16_t index,
    const uint8_t sub_index
) {
    using Kind = errors::SDOFailure::Kind;
    errors::SDOFailure failure;
    ec_errort err{};
    while (api.poperror(err)) {
        if (err.Slave != slave || err.Index != index || err.SubIdx != sub_index)
            continue;
        if (err.Etype == EC_ERR_TYPE_SDO_ERROR)
            failure = {
                .kind = Kind::ABORTED,
                .abort_code = static_cast<uint32_t>(err.AbortCode),
            };
        else if (
            err.Etype == EC_ERR_TYPE_PACKET_ERROR && err.ErrorCode == UNEXPECTED_FRAME
        )
            failure = {.kind = Kind::UNEXPECTED_REPLY};
        else if (
            err.Etype == EC_ERR_TYPE_PACKET_ERROR &&
            err.ErrorCode == CONTAINER_TOO_SMALL
        )
            failure = {.kind = Kind::REPLY_TOO_LARGE};
    }
    return failure;
}

/// @brief returns errors::REPLIES_BLOCKED when a filter on the adapter of iface blocks
/// the replies, else errors::NO_SLAVES_FOUND. The message calls the adapter name.
x::errors::Error
no_slaves_error(API &api, const std::string &iface, const std::string &name) {
    const auto found = "No EtherCAT slaves found on " + name + ". ";
    if (!api.dewesoft_filter_bound(iface.c_str()))
        return x::errors::Error(
            errors::NO_SLAVES_FOUND,
            found + "Check the cable and the power of the first device."
        );
    return x::errors::Error(
        errors::REPLIES_BLOCKED,
        found + std::format(
                    "The DewesoftX \"EthercatDAQ Filter\" on this adapter blocks "
                    "EtherCAT replies. Clear it in the properties of \"{}\" "
                    "(Settings > Network & internet > Advanced network settings > "
                    "More network adapter options, then right-click the adapter > "
                    "Properties).",
                    name
                )
    );
}
}

std::string Master::name() const {
    return this->api->adapter_name(this->iface_name.c_str()).value_or(this->iface_name);
}

Master::Master(std::unique_ptr<API> api, std::string interface_name, Config cfg):
    iface_name(std::move(interface_name)),
    api(std::move(api)),
    cfg(std::move(cfg)),
    iomap(DEFAULT_IOMAP_SIZE, 0) {}

// NOLINTNEXTLINE(bugprone-exception-escape): its strings throw only past max_size.
Master::~Master() {
    if (this->session || this->initialized) this->deactivate();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Master::scan() {
    if (!this->initialized) {
        if (auto err = this->api->check_socket(this->iface_name.c_str()))
            return {{}, err};
        if (this->api->init(this->iface_name.c_str()) <= 0) {
            this->api->close();
            return {
                {},
                x::errors::Error(
                    errors::MASTER_INIT_ERROR,
                    "Cannot open " + this->name() +
                        " for raw Ethernet access. Run the Driver as root (or with "
                        "CAP_NET_RAW on Linux), and install Npcap on Windows."
                )
            };
        }
        if (this->api->config_init() <= 0) {
            this->api->close();
            return {{}, no_slaves_error(*this->api, this->iface_name, this->name())};
        }
        // config_init requests PRE-OP without waiting for it, and SOEM sends no mailbox
        // request to a slave whose last read state is below PRE-OP.
        this->api->statecheck(0, EC_STATE_PRE_OP, micros(this->cfg.pre_op_timeout));
        this->api->readstate();
        this->populate_slaves();
        this->initialized = true;
    }
    return {this->slave_list, x::errors::NIL};
}

std::pair<size_t, x::errors::Error> Master::count_slaves() {
    if (!this->initialized)
        return {
            0,
            x::errors::Error(errors::MASTER_INIT_ERROR, "master not initialized")
        };
    const int count = this->api->count_bus_slaves();
    return {count > 0 ? static_cast<size_t>(count) : 0, x::errors::NIL};
}

x::errors::Error Master::write_parameters(
    const uint16_t pos,
    const std::vector<slave::StartupParameter> &parameters
) {
    for (const auto &param: parameters) {
        const auto data = param.bytes();
        if (this->api->SDOwrite(
                pos,
                param.index,
                param.sub_index,
                false,
                static_cast<int>(data.size()),
                data.data(),
                SDO_TIMEOUT
            ) > 0)
            continue;
        const auto
            failure = pop_sdo_failure(*this->api, pos, param.index, param.sub_index);
        if (failure.kind == errors::SDOFailure::Kind::ABORTED)
            return x::errors::Error(
                errors::startup_parameter_error(failure.abort_code),
                "rejected startup parameter " + param.to_string() + ": " +
                    failure.reason()
            );
        return x::errors::Error(
            errors::STARTUP_PARAMETER_ERROR,
            "failed to write startup parameter " + param.to_string() + ": " +
                failure.reason()
        );
    }
    return x::errors::NIL;
}

bool Master::write_startup_parameters(
    const uint16_t pos,
    const std::vector<slave::StartupParameter> &parameters
) {
    const auto err = this->write_parameters(pos, parameters);
    if (err)
        this->session.value().startup_failures[pos] = err;
    else
        this->session.value().startup_failures.erase(pos);
    return !err;
}

master::Activation Master::activate(
    const master::Plan &plan,
    const std::function<void(const master::Image &)> &start_cycle
) {
    if (!this->initialized)
        return {
            .err = x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized")
        };
    if (this->session)
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "master already activated"
            )
        };
    this->session.emplace().plan = plan;
    return this->start(start_cycle);
}

slave::Config Master::config(const uint16_t pos) const {
    const auto &slaves = this->session.value().plan.slaves;
    const auto it = slaves.find(pos);
    return it == slaves.end() ? slave::Config{} : it->second;
}

master::Activation
Master::start(const std::function<void(const master::Image &)> &start_cycle) {
    auto &session = this->session.value();
    // A slave whose EEPROM read failed has no identity and no SII layout to map.
    std::map<uint16_t, slave::Refusal> unidentified;
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        const auto pos = static_cast<uint16_t>(i);
        const auto &rec = this->record(pos);
        if (!this->config(pos).enabled || rec.eep_man != 0 || rec.eep_id != 0) continue;
        unidentified[pos] = {
            .reason = "gave no identity when the Driver read its EEPROM"
        };
        session.plan.slaves[pos].enabled = false;
    }
    const auto refuse_unidentified = [&] {
        std::string msg;
        for (const auto &[pos, refusal]: unidentified)
            msg += std::format(
                "{}slave {} ({}) {}",
                msg.empty() ? "" : "; ",
                pos,
                this->slave_name(pos),
                refusal.reason
            );
        return master::Activation{
            .err = x::errors::Error(errors::ACTIVATION_ERROR, msg),
            .refusals = std::move(unidentified),
        };
    };
    auto [dc_setups, dc_err] = this->resolve_dc();
    if (dc_err) return {.err = dc_err};
    session.dc_setups = std::move(dc_setups);

    std::vector<uint16_t> parameterized;
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        const auto pos = static_cast<uint16_t>(i);
        auto config = this->config(pos);
        this->record(pos).group = config.enabled ? ENABLED_GROUP : DISABLED_GROUP;
        if (!config.enabled || config.startup_parameters.empty()) {
            this->api->set_PO2SOconfig(pos, nullptr);
            continue;
        }
        parameterized.push_back(pos);
        this->api->set_PO2SOconfig(
            pos,
            [this, params = std::move(config.startup_parameters)](uint16_t p) {
                this->write_startup_parameters(p, params);
            }
        );
    }

    if (const auto err = this->clear_dc_activation()) return {.err = err};

    // The master requests SAFE-OP itself, once the bus cycles and SYNC0 runs.
    this->api->context.manualstatechange = 1;
    // SOEM keeps the SII length of a sync manager whose PDO assignment is empty or that
    // 0x1C00 does not list, and turns it on. A slave can refuse SAFE-OP for that.
    // config_map_group then gives a sync manager a length only from its PDOs.
    for (int i = 1; i <= this->api->context.slavecount; ++i)
        if (this->record(i).group == ENABLED_GROUP)
            for (int sm = 2; sm < EC_MAXSM; ++sm)
                this->record(i).SM[sm].SMlength = 0;
    const int iomap_size = this->api->config_map_group(
        this->iomap.data(),
        ENABLED_GROUP
    );
    this->api->context.manualstatechange = 0;

    if (iomap_size <= 0) {
        if (!unidentified.empty()) return refuse_unidentified();
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to configure PDO mapping"
            )
        };
    }

    std::string failures;
    // One failure with no abort code is enough for a retry to help.
    auto type = errors::STARTUP_PARAMETER_REJECTED;
    for (const auto &[pos, failure]: session.startup_failures) {
        failures += (failures.empty() ? "" : "; ") + std::string("slave ") +
                    std::to_string(pos) + " (" + this->slave_name(pos) + ") " +
                    failure.data;
        if (!failure.matches(errors::STARTUP_PARAMETER_REJECTED))
            type = errors::STARTUP_PARAMETER_ERROR;
    }
    if (!failures.empty()) return {.err = x::errors::Error(type, failures)};

    // A startup parameter can change the slave's PDO assignment.
    for (const auto pos: parameterized) {
        auto result = this->discover_slave(pos);
        auto &scanned = this->slave_list[pos - 1];
        result.properties.serial = scanned.properties.serial;
        result.status.serial_error = std::move(scanned.status.serial_error);
        scanned = std::move(result);
    }

    this->api->configdc();
    // configdc makes only the frames of the reference slave's group carry the DC time.
    auto &group = this->api->context.grouplist[ENABLED_GROUP];
    group.hasdc = this->record(0).hasdc;
    group.DCnext = this->record(0).DCnext;
    session.has_dc = group.hasdc != 0;
    session.input_sz = group.Ibytes;
    session.output_sz = group.Obytes;
    session.expected_wkc = (group.outputsWKC * 2) + group.inputsWKC;
    // A slave can stop its process data task for good when the master reads its inputs
    // while it moves to SAFE-OP.
    session.process_data_held.store(true, std::memory_order_release);
    start_cycle(this->image());
    // The caller leaves out a refused slave only when the image maps another one.
    if (!unidentified.empty()) return refuse_unidentified();
    if (auto err = this->start_dc()) return {.err = err};

    const int timeout = micros(this->cfg.state_change_timeout);
    if (auto safe_op = this->request_state(EC_STATE_SAFE_OP, timeout); safe_op.err)
        return safe_op;
    session.process_data_held.store(false, std::memory_order_release);
    // A slave with outputs refuses OP with a sync manager watchdog error until a frame
    // passes it in SAFE-OP.
    if (!this->wait_for_frame(
            this->cfg.clock.steady() + std::chrono::microseconds(timeout),
            [] { return false; }
        ))
        return {
            .err = x::errors::Error(
                errors::STATE_CHANGE_ERROR,
                "failed to reach OP: no process data passed the slaves in SAFE-OP"
            )
        };
    return this->request_state(EC_STATE_OPERATIONAL, timeout);
}

void Master::deactivate() {
    if (this->session) {
        this->request_state(EC_STATE_INIT, micros(this->cfg.init_timeout));
        // INIT leaves the DC activation as it is.
        if (const auto err = this->clear_dc_activation())
            LOG(WARNING) << "[ethercat] " << err.data;
        this->session.reset();
    }
    if (this->initialized) {
        this->api->close();
        this->initialized = false;
        this->slave_list.clear();
    }
}

x::errors::Error Master::receive() {
    if (!this->session)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");
    auto &session = *this->session;

    session.shortfall.reset();
    if (session.held_err) return session.held_err;
    const int wkc = this->api->receive_processdata(ENABLED_GROUP, PROCESSDATA_TIMEOUT);

    if (wkc < 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data receive failed");
    this->frames.fetch_add(1, std::memory_order_release);

    if (wkc != session.expected_wkc) {
        if (wkc < session.expected_wkc)
            session.shortfall = static_cast<uint32_t>(session.expected_wkc - wkc);
        return x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " +
                std::to_string(session.expected_wkc) + ", got " + std::to_string(wkc)
        );
    }

    // SOEM takes the DC time without the working counter of its datagram. A lost
    // reference clock gives a frozen time, and a reset one goes back. The frame read
    // the time as it went out, which can be a bus period before this receive.
    if (const int64_t dc = this->api->context.DCtime;
        session.has_dc && dc > session.last_dc_time) {
        session.last_dc_time = dc;
        const x::telem::TimeSpan sent_at(session.sent_at.time_since_epoch());
        session.host_dc_offset.store(
            dc - sent_at.nanoseconds(),
            std::memory_order_relaxed
        );
    }
    return x::errors::NIL;
}

x::errors::Error Master::send() {
    if (!this->session)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");
    auto &session = *this->session;

    session.held_err = x::errors::NIL;
    if (session.process_data_held.load(std::memory_order_acquire)) {
        if (session.has_dc &&
            this->api->sync_dc(ENABLED_GROUP, PROCESSDATA_TIMEOUT) <= 0) {
            session.held_err = x::errors::Error(
                errors::CYCLIC_ERROR,
                "reference clock frame failed"
            );
            return session.held_err;
        }
        if (session.has_dc) this->frames.fetch_add(1, std::memory_order_release);
        session.held_err = x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "no process data passes the slaves until they reach SAFE-OP"
        );
        return x::errors::NIL;
    }
    const auto sent_at = this->cfg.clock.steady();
    const int result = this->api->send_processdata(ENABLED_GROUP);

    if (result <= 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data send failed");
    session.sent_at = sent_at;
    return x::errors::NIL;
}

std::optional<uint32_t> Master::shortfall() const {
    if (!this->session) return std::nullopt;
    return this->session->shortfall;
}

std::optional<x::telem::TimeStamp> Master::dc_time() const {
    if (!this->session || !this->session.value().has_dc) return std::nullopt;
    return master::DC_EPOCH + x::telem::TimeSpan(this->api->context.DCtime);
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &breaker) {
    std::lock_guard lock(this->mailbox_mu);
    this->api->readstate();
    std::vector<slave::Health> health;
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        if (this->record(i).group != ENABLED_GROUP) continue;
        this->update_lost(i);
        const uint16_t state = this->record(i).state;
        auto &h = health.emplace_back(
            slave::Health{
                .position = static_cast<uint16_t>(i),
                .state = slave::from_al_state(state),
                .faulted = (state & EC_STATE_ERROR) != 0,
                .lost = this->record(i).islost != 0,
            }
        );
        if (this->al_error(i))
            h.al_status = ec_ALstatuscode2string(this->record(i).ALstatuscode);
        const auto &session = this->session.value();
        if (const auto it = session.dc_restore_errors.find(i);
            it != session.dc_restore_errors.end()) {
            h.faulted = true;
            h.dc_restore_error = it->second;
        }
        if (const auto it = session.startup_failures.find(i);
            it != session.startup_failures.end()) {
            h.faulted = true;
            h.startup_error = it->second.data;
        }
        if (breaker.running()) this->recover(i, breaker);
    }
    return health;
}

x::errors::Error Master::write(
    const uint16_t pos,
    const std::vector<slave::StartupParameter> &parameters
) {
    std::lock_guard lock(this->mailbox_mu);
    if (const auto err = this->write_parameters(pos, parameters))
        return x::errors::Error(
            err,
            "Slave " + std::to_string(pos) + " (" + this->slave_name(pos) + ") " +
                err.data + "."
        );
    if (this->session)
        this->session->plan.slaves[pos].add_startup_parameters(parameters);
    return x::errors::NIL;
}

std::pair<uint32_t, x::errors::Error>
Master::read(const uint16_t pos, const uint16_t index, const uint8_t sub_index) {
    std::lock_guard lock(this->mailbox_mu);
    return this->read_sdo(pos, index, sub_index);
}

void Master::update_lost(const uint16_t pos) {
    if (this->record(pos).state == EC_STATE_NONE && !this->record(pos).islost)
        this->api->statecheck(pos, EC_STATE_OPERATIONAL, EC_TIMEOUTRET);
    this->record(pos).islost = this->record(pos).state == EC_STATE_NONE ? 1 : 0;
}

void Master::recover(const uint16_t pos, const x::breaker::Breaker &breaker) {
    if (this->session.value().startup_failures.contains(pos)) return;
    const uint16_t state = this->record(pos).state;
    if (state == EC_STATE_SAFE_OP + EC_STATE_ERROR) {
        this->record(pos).state = EC_STATE_SAFE_OP + EC_STATE_ACK;
        this->api->writestate(pos);
    } else if (state == EC_STATE_SAFE_OP) {
        // From PRE-OP, the next check restores the clock again.
        this->record(pos).state = this->restore_failed(pos) ? EC_STATE_PRE_OP
                                                            : EC_STATE_OPERATIONAL;
        this->api->writestate(pos);
    } else if (state > EC_STATE_NONE && state != EC_STATE_OPERATIONAL) {
        auto params = this->config(pos).startup_parameters;
        const bool has_dc = this->record(pos).hasdc != 0;
        if (has_dc || !params.empty())
            this->api->set_PO2SOconfig(
                pos,
                [this, pos, &breaker, params = std::move(params), has_dc](uint16_t) {
                    if (!this->write_startup_parameters(pos, params) || !has_dc) return;
                    const auto err = this->restore_dc(pos, breaker);
                    if (err)
                        this->session.value().dc_restore_errors[pos] = err.data;
                    else
                        this->session.value().dc_restore_errors.erase(pos);
                }
            );
        this->api->reconfig_slave(pos, RECOVERY_TIMEOUT);
        this->api->set_PO2SOconfig(pos, nullptr);
        if (this->session.value().startup_failures.contains(pos)) {
            this->record(pos).state = EC_STATE_PRE_OP;
            this->api->writestate(pos);
        }
    }
    if (this->record(pos).islost && this->api->recover_slave(pos, RECOVERY_TIMEOUT) > 0)
        this->record(pos).islost = 0;
}

void Master::populate_slaves() {
    this->slave_list.clear();
    this->slave_list.reserve(this->api->context.slavecount);

    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        auto slave = this->discover_slave(i);
        this->read_serial(slave);
        this->read_settings(slave);
        this->slave_list.push_back(std::move(slave));
    }
    slave::clear_duplicate_serials(this->slave_list);
}

void Master::read_serial(slave::DiscoveryResult &slave) {
    const auto pos = slave.properties.position;
    if ((this->record(pos).mbx_proto & ECT_MBXPROT_COE) == 0) return;
    ec_errort err{};
    // Only the errors of this read can show that the slave answered it.
    while (this->api->poperror(err)) {}
    uint32_t serial = 0;
    int size = sizeof(serial);
    if (this->api->SDOread(
            pos,
            IDENTITY_INDEX,
            SERIAL_SUB_INDEX,
            FALSE,
            &size,
            &serial,
            SDO_TIMEOUT
        ) > 0) {
        if (serial != 0) slave.properties.serial = serial;
        return;
    }
    // A slave that answers the read with an error, such as an SDO abort or a mailbox
    // error, has no CoE serial number. An emergency is no answer.
    bool answered = false;
    while (this->api->poperror(err))
        answered |= err.Slave == pos && err.Etype != EC_ERR_TYPE_EMERGENCY;
    if (!answered)
        slave.status.serial_error = coe::read_error(IDENTITY_INDEX, SERIAL_SUB_INDEX, 0)
                                        .data;
}

void Master::read_settings(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    const auto pos = props.position;
    if ((this->record(pos).mbx_proto & ECT_MBXPROT_COE) == 0) return;
    auto [settings, err] = esi::read_settings(
        props.vendor_id,
        props.product_code,
        props.revision,
        props.op_modes,
        [this, pos](const uint16_t index, const uint8_t sub_index) {
            return this->read_sdo(pos, index, sub_index);
        }
    );
    slave.status.settings_error = err.data;
    props.settings = std::move(settings);
}

slave::DiscoveryResult Master::discover_slave(const uint16_t pos) {
    slave::DiscoveryResult result{};
    result.properties.network = this->iface_name;
    result.properties.position = pos;
    result.properties.vendor_id = this->record(pos).eep_man;
    result.properties.product_code = this->record(pos).eep_id;
    result.properties.revision = this->record(pos).eep_rev;
    result.properties.serial = this->record(pos).eep_ser;
    result.properties.name = this->slave_name(pos);
    result.status.state = slave::from_al_state(this->record(pos).state);
    Description unshared;
    const auto model = this->read_model(pos);
    auto &description = model ? this->descriptions[*model] : unshared;
    this->discover_slave_pdos(result, description);
    result.properties.op_modes = this->read_op_modes(pos, description);
    return result;
}

master::Image Master::image() {
    const auto &session = this->session.value();
    master::Image image{
        .inputs = {this->iomap.data() + session.output_sz, session.input_sz},
        .outputs = {this->iomap.data(), session.output_sz},
    };
    for (const auto &[pos, regs]: session.dc_setups)
        if (regs.sync0_cycle > 0 && (regs.assign_activate & dc::PDI_SYNC_UNIT) == 0)
            image.dc_sync.push_back(regs);
    image.slaves = this->slave_list;
    const auto *base = this->iomap.data();
    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        if (this->record(props.position).group != ENABLED_GROUP) continue;
        const auto *outputs = this->record(props.position).outputs;
        const auto *inputs = this->record(props.position).inputs;
        const size_t output_offset = outputs != nullptr
                                       ? static_cast<size_t>(outputs - base)
                                       : 0;
        const size_t input_offset = inputs != nullptr
                                      ? static_cast<size_t>(inputs - base) -
                                            session.output_sz
                                      : 0;
        const auto input_bits = pdo::total_bits(props.input_pdos);
        const auto output_bits = pdo::total_bits(props.output_pdos);
        const auto mapped_input_bits = this->record(props.position).Ibits;
        const auto mapped_output_bits = this->record(props.position).Obits;
        // A frame adds 2 for each slave that takes its outputs and 1 for each slave
        // that gives its inputs.
        if (const master::WorkingCounterShare share{
                .outputs = mapped_output_bits > 0 ? 2U : 0U,
                .inputs = mapped_input_bits > 0 ? 1U : 0U,
            };
            share != master::WorkingCounterShare{})
            image.working_counter_shares[props.position] = share;
        if (input_bits != mapped_input_bits || output_bits != mapped_output_bits) {
            const auto err = x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                std::format(
                    "Slave {} ({}) maps {} input and {} output bits, but its "
                    "discovered PDO entries hold {} and {}.",
                    props.position,
                    props.name,
                    mapped_input_bits,
                    mapped_output_bits,
                    input_bits,
                    output_bits
                )
            );
            LOG(WARNING) << err.message();
            for (const auto *pdos: {&props.input_pdos, &props.output_pdos})
                for (const auto &pdo: *pdos)
                    image.errors.emplace(
                        pdo::Key{
                            props.position,
                            pdo.index,
                            pdo.sub_index,
                            pdo.is_input
                        },
                        err
                    );
            continue;
        }
        pdo::compute_offsets(
            image.offsets,
            props.position,
            props.input_pdos,
            true,
            input_offset,
            this->record(props.position).Istartbit
        );
        pdo::compute_offsets(
            image.offsets,
            props.position,
            props.output_pdos,
            false,
            output_offset,
            this->record(props.position).Ostartbit
        );
    }
    return image;
}

std::pair<uint32_t, x::errors::Error>
Master::read_sdo(const uint16_t pos, const uint16_t index, const uint8_t sub_index) {
    uint32_t value = 0;
    int size = sizeof(value);
    if (this->api->SDOread(pos, index, sub_index, FALSE, &size, &value, SDO_TIMEOUT) >
        0)
        return {value, x::errors::NIL};
    return {
        0,
        coe::read_error(
            index,
            sub_index,
            pop_sdo_failure(*this->api, pos, index, sub_index)
        )
    };
}

std::optional<std::vector<uint8_t>>
Master::read_object(const uint16_t pos, const uint16_t index) {
    std::vector<uint8_t> bytes(MAX_OBJECT_SIZE);
    int size = static_cast<int>(bytes.size());
    if (this->api->SDOread(pos, index, 0, TRUE, &size, bytes.data(), SDO_TIMEOUT) <=
        0) {
        pop_sdo_failure(*this->api, pos, index, 0);
        return std::nullopt;
    }
    bytes.resize(static_cast<size_t>(size));
    return bytes;
}

std::optional<Master::Model> Master::read_model(const uint16_t pos) {
    const auto &rec = this->record(pos);
    Model model{rec.eep_man, rec.eep_id, rec.eep_rev, {}};
    if ((rec.mbx_proto & ECT_MBXPROT_COE) == 0) return model;
    const auto [count, err] = this->read_sdo(pos, DETECTED_MODULES_INDEX, 0);
    if (err.matches(coe::OBJECT_MISSING)) return model;
    if (err) return std::nullopt;
    auto &modules = std::get<3>(model);
    for (uint32_t i = 1; i <= count; ++i) {
        const auto [module, module_err] = this->read_sdo(
            pos,
            DETECTED_MODULES_INDEX,
            static_cast<uint8_t>(i)
        );
        if (module_err) return std::nullopt;
        modules.push_back(module);
    }
    return model;
}

void Master::discover_slave_pdos(
    slave::DiscoveryResult &slave,
    Description &description
) {
    auto &props = slave.properties;
    const auto pos = props.position;
    x::errors::Error layout_err;
    if ((this->record(pos).mbx_proto & ECT_MBXPROT_COE) != 0) {
        coe::Read read = [this, pos](const uint16_t index, const uint8_t sub_index) {
            return this->read_sdo(pos, index, sub_index);
        };
        if ((this->record(pos).CoEdetails & ECT_COEDET_SDOCA) != 0)
            read = coe::with_complete_access(
                std::move(read),
                [this, pos](const uint16_t index) {
                    return this->read_object(pos, index);
                }
            );
        auto [syncs, err] = coe::read_layout(read, EC_MAXSM, coe::OnError::SKIP);
        layout_err = err;
        for (auto &sync: syncs)
            for (auto &pdo: sync.pdos) {
                pdo.name = this->read_object_name(pos, pdo.index, description);
                for (auto &entry: pdo.entries)
                    if (entry.index != 0)
                        std::tie(entry.name, entry.type) = this->read_object_entry(
                            pos,
                            entry.index,
                            entry.sub_index,
                            description
                        );
                auto &pdos = sync.is_input ? props.input_pdos : props.output_pdos;
                pdo::append(pdos, sync.is_input, pdo);
            }
        const auto lists_fixed_pdos = [&](const pdo::Sync &sync) {
            const auto on_sync = [&](const uint8_t sm) { return sm == sync.index; };
            return sync.fixed &&
                   !this->read_sii_pdos(pos, sync.is_input, on_sync).empty();
        };
        const bool assigned = !props.input_pdos.empty() || !props.output_pdos.empty();
        const auto fixed = assigned ? std::ranges::find_if(syncs, lists_fixed_pdos)
                                    : syncs.end();
        if (fixed != syncs.end()) {
            props.input_pdos.clear();
            props.output_pdos.clear();
            slave.status.pdo_discovery_error = std::format(
                "SOEM cannot map this slave because sync manager {} has fixed PDOs and "
                "the other sync managers have assigned PDOs. Use IgH for this slave",
                fixed->index
            );
            // No esi::fill: SOEM would still leave the fixed sync manager unmapped.
            return;
        }
    }
    const bool read_coe = !props.input_pdos.empty() || !props.output_pdos.empty();
    // SOEM maps a slave from its SII when CoE gives no process data.
    if (!read_coe) this->discover_pdos_sii(slave);
    const bool found = !props.input_pdos.empty() || !props.output_pdos.empty();
    esi::fill(slave);
    // The catalog layout does not clear a failed read, so the scanner warns and
    // discovers the slave again.
    if (layout_err) {
        auto &msg = slave.status.pdo_discovery_error;
        msg = "failed to read the PDO layout: " + layout_err.data;
        if (read_coe)
            msg += ", so the Driver uses the PDOs it read";
        else if (found)
            msg += ", so the Driver uses the PDOs in its SII";
    }
}

std::vector<uint8_t>
Master::read_sii_category(const uint16_t pos, const uint16_t category) {
    const uint16_t start = this->api->siifind(pos, category);
    if (start == 0) return {};
    const auto byte = [&](const size_t offset) {
        return this->api->siigetbyte(pos, static_cast<uint16_t>(start + offset));
    };
    std::vector<uint8_t> data(static_cast<size_t>(byte(0) | byte(1) << 8) * 2);
    for (size_t i = 0; i < data.size(); ++i)
        data[i] = byte(2 + i);
    return data;
}

sii::Strings Master::sii_strings(const uint16_t pos) {
    return [this, pos](const uint8_t index) {
        if (index == 0) return std::string{};
        char str[EC_MAXNAME + 1] = {0};
        this->api->siistring(str, pos, index);
        return std::string(str);
    };
}

std::vector<pdo::PDO> Master::read_sii_pdos(
    const uint16_t pos,
    const bool is_input,
    const std::function<bool(uint8_t sync)> &keep
) {
    const auto data = this->read_sii_category(
        pos,
        is_input ? ECT_SII_PDO : ECT_SII_PDO + 1
    );
    auto pdos = sii::parse_pdos(data, keep, this->sii_strings(pos));
    this->api->eeprom2pdi(pos);
    return pdos;
}

void Master::discover_pdos_sii(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    const auto pos = props.position;
    const auto below_max = [](const uint8_t sync) { return sync < EC_MAXSM; };
    for (const bool is_input: {true, false}) {
        auto &pdos = is_input ? props.input_pdos : props.output_pdos;
        for (const auto &pdo: this->read_sii_pdos(pos, is_input, below_max))
            pdo::append(pdos, is_input, pdo);
    }
    if (props.input_pdos.empty() && props.output_pdos.empty())
        slave.status.pdo_discovery_error = "no PDOs found";
}

/// SII category that lists the synchronization modes of a slave.
constexpr uint16_t SII_DC_CATEGORY = 60;

std::vector<dc::OpMode>
Master::read_op_modes(const uint16_t pos, Description &description) {
    auto &known = description.op_modes;
    if (!known.empty()) return known;
    auto modes = sii::parse_op_modes(
        this->read_sii_category(pos, SII_DC_CATEGORY),
        this->sii_strings(pos)
    );
    this->api->eeprom2pdi(pos);
    known = modes;
    return modes;
}

std::optional<std::string> Master::read_object_name(
    const uint16_t slave_pos,
    const uint16_t index,
    Description &description
) {
    auto &names = description.object_names;
    if (const auto it = names.find(index); it != names.end()) return it->second;
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;
    if (this->api->readODdescription(0, &od_list) > 0 && od_list.Name[0][0] != '\0')
        return names[index] = std::string(od_list.Name[0]);
    return std::nullopt;
}

std::pair<std::string, telem::DataType> Master::read_object_entry(
    const uint16_t slave_pos,
    const uint16_t index,
    const uint8_t sub_index,
    Description &description
) {
    auto &entries = description.entries;
    if (const auto it = entries.find({index, sub_index}); it != entries.end())
        return it->second;
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;

    ec_OElistt oe_list{};
    if (this->api->readOEsingle(0, sub_index, &od_list, &oe_list) <= 0 ||
        oe_list.Entries == 0)
        return {"", telem::DataType::EC_UNKNOWN};
    const std::pair entry{
        std::string(oe_list.Name[sub_index]),
        static_cast<telem::DataType>(oe_list.DataType[sub_index])
    };
    entries[{index, sub_index}] = entry;
    return entry;
}

template<std::integral T>
x::errors::Error
Master::write_dc_register(const uint16_t pos, const uint16_t address, const T value) {
    const auto bytes = x::binary::to_little_endian(value);
    const int wkc = this->api->write_register(
        pos,
        address,
        {bytes.begin(), bytes.end()},
        EC_TIMEOUTRET3
    );
    if (wkc == 1) return x::errors::NIL;
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        std::format(
            "slave {} ({}) {} a write to DC register 0x{:04X}",
            pos,
            this->slave_name(pos),
            response(wkc),
            address
        )
    );
}

template<std::integral T>
std::pair<T, x::errors::Error>
Master::read_dc_register(const uint16_t pos, const uint16_t address) {
    std::array<uint8_t, sizeof(T)> bytes{};
    const int wkc = this->api->read_register(pos, address, bytes, EC_TIMEOUTRET3);
    if (wkc == 1) return {x::binary::from_little_endian<T>(bytes), x::errors::NIL};
    return {
        0,
        x::errors::Error(
            errors::ACTIVATION_ERROR,
            std::format(
                "slave {} ({}) {} a read of DC register 0x{:04X}",
                pos,
                this->slave_name(pos),
                response(wkc),
                address
            )
        )
    };
}

x::errors::Error Master::clear_sync(const uint16_t pos) {
    // 0x0980 goes first: while it gives the sync unit to the PDI, the slave refuses
    // writes to 0x0981.
    for (const uint16_t address: {ECT_REG_DCCUC, ECT_REG_DCSYNCACT})
        if (auto err = this->write_dc_register<uint8_t>(pos, address, 0)) return err;
    return x::errors::NIL;
}

x::errors::Error Master::clear_dc_activation() {
    std::string failed;
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        if (this->record(i).group != ENABLED_GROUP || !this->record(i).hasdc) continue;
        if (const auto err = this->clear_sync(i)) {
            if (!failed.empty()) failed += "; ";
            failed += err.data;
        }
    }
    if (failed.empty()) return x::errors::NIL;
    return x::errors::Error(errors::ACTIVATION_ERROR, failed);
}

std::pair<std::vector<Master::DCSetup>, x::errors::Error> Master::resolve_dc() const {
    const auto period = this->session.value().plan.period;
    std::vector<DCSetup> setups;
    std::vector<x::telem::TimeSpan> cycles;
    std::string events;
    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        const auto config = this->config(props.position);
        if (!config.enabled) continue;
        const auto choice = config.op_mode ? config.op_mode : props.catalog_op_mode();
        const auto [mode, select_err] = dc::select(props.op_modes, choice);
        if (select_err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "slave {} ({}) cannot use its chosen mode: {}",
                        props.position,
                        props.name,
                        select_err.data
                    )
                )
            };
        if (!mode) continue;
        auto [regs, err] = dc::resolve(
            *mode,
            period,
            choice ? choice->cycles : dc::Cycles{}
        );
        if (err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "slave {} ({}) cannot run its DC mode at a bus cycle of {}: {}",
                        props.position,
                        props.name,
                        period.to_string(),
                        err.data
                    )
                )
            };
        setups.push_back({.position = props.position, .registers = regs});
        if (!regs.marks_data() || (regs.assign_activate & dc::PDI_SYNC_UNIT) != 0)
            continue;
        cycles.emplace_back(regs.sync0_cycle);
        if (!events.empty()) events += ", ";
        events += std::format(
            "slave {} ({}) every {}",
            props.position,
            props.name,
            cycles.back().to_string()
        );
    }
    if (!cycles.empty() && dc::grid(period, cycles) <
                               static_cast<double>(dc::MIN_EVENT_CYCLE.nanoseconds()))
        return {
            {},
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                std::format(
                    "SYNC0 events do not line up with the bus cycle of {}: {}. Change "
                    "the bus rate or the SYNC0 cycles so that a span of {} or more "
                    "divides the bus cycle and each SYNC0 cycle.",
                    period.to_string(),
                    events,
                    dc::MIN_EVENT_CYCLE.to_string()
                )
            )
        };
    return {setups, x::errors::NIL};
}

x::errors::Error Master::wait_for_dc_settle(
    const std::vector<uint16_t> &positions,
    const std::function<bool()> &stopped
) {
    const auto deadline = this->cfg.clock.steady() +
                          this->cfg.dc_settle_timeout.chrono();
    const x::errors::Error stop_err(
        errors::ACTIVATION_ERROR,
        "the bus stopped before the clocks settled"
    );
    // 0x092C keeps its old value until a frame carries the reference clock to the
    // slave.
    if (!this->wait_for_frame(deadline, stopped)) {
        if (stopped()) return stop_err;
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no frame carried the reference clock to the slaves within " +
                this->cfg.dc_settle_timeout.to_string()
        );
    }
    while (true) {
        uint16_t worst_pos = 0;
        x::telem::TimeSpan worst(0);
        for (const auto pos: positions) {
            const auto [raw, err] = this->read_dc_register<uint32_t>(
                pos,
                ECT_REG_DCSYSDIFF
            );
            if (err) return err;
            // Bit 31 is the sign. The rest is the difference in nanoseconds.
            const x::telem::TimeSpan diff(raw & 0x7FFFFFFF);
            if (diff > worst) {
                worst = diff;
                worst_pos = pos;
            }
        }
        if (worst <= DC_SETTLED_DIFF) return x::errors::NIL;
        if (stopped()) return stop_err;
        if (this->cfg.clock.steady() >= deadline)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                std::format(
                    "the clock of slave {} ({}) is {} off the reference clock after "
                    "{}. It must be within {} before SYNC0 starts.",
                    worst_pos,
                    this->slave_name(worst_pos),
                    worst.to_string(),
                    this->cfg.dc_settle_timeout.to_string(),
                    DC_SETTLED_DIFF.to_string()
                )
            );
        this->cfg.clock.sleep(DC_SETTLE_POLL_INTERVAL);
    }
}

x::errors::Error Master::start_dc() {
    const auto &setups = this->session.value().dc_setups;
    if (setups.empty()) return x::errors::NIL;
    std::vector<uint16_t> positions;
    positions.reserve(setups.size());
    for (const auto &setup: setups)
        positions.push_back(setup.position);
    if (auto err = this->wait_for_dc_settle(positions, [] { return false; }))
        return err;
    for (const auto &setup: setups)
        if (auto err = this->start_sync(setup)) return err;
    return x::errors::NIL;
}

x::errors::Error Master::start_sync(const DCSetup &setup) {
    const auto &[pos, regs] = setup;
    const auto cuc = static_cast<uint8_t>(regs.assign_activate & 0xFF);
    const auto act = static_cast<uint8_t>(regs.assign_activate >> 8);
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCUC, cuc)) return err;
    // While 0x0980 gives the sync unit to the PDI, the PDI sets it up.
    if ((regs.assign_activate & dc::PDI_SYNC_UNIT) != 0) return x::errors::NIL;
    const auto [time, time_err] = this->read_dc_register<int64_t>(
        pos,
        ECT_REG_DCSYSTIME
    );
    if (time_err) return time_err;
    const int64_t cycle = regs.sync0_cycle;
    const int64_t shift = regs.sync0_shift;
    const int64_t ahead = time + DC_START_DELAY.nanoseconds();
    // SYNC0 events fall on whole cycles of DC time plus the shift, the grid the cycle
    // lock uses. The first one is at least DC_START_DELAY away.
    int64_t start = ahead;
    if (cycle > 0) start = ((ahead - shift) / cycle + 1) * cycle + shift;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCYCLE0, regs.sync0_cycle))
        return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCYCLE1, regs.sync1_cycle))
        return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCSTART0, start)) return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCSYNCACT, act)) return err;
    LOG(INFO) << std::format(
        "[ethercat] slave {} ({}) runs DC sync: AssignActivate 0x{:04X}, SYNC0 "
        "cycle {} ns, SYNC1 register {} ns, first SYNC0 at {} ns",
        pos,
        this->slave_name(pos),
        regs.assign_activate,
        regs.sync0_cycle,
        regs.sync1_cycle,
        start
    );
    return x::errors::NIL;
}

x::errors::Error Master::align_clock(const uint16_t pos) {
    const x::telem::TimeSpan latched_at(this->cfg.clock.steady().time_since_epoch());
    // A resend latches the times again after latched_at, so the latch gets one try.
    const int latched = this->api->latch_receive_times(EC_TIMEOUTRET);
    if (latched < 0)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "the frame that latches the receive times got no reply"
        );
    if (latched == 0)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no slave latched its receive time"
        );
    const auto [local, local_err] = this->read_dc_register<int64_t>(pos, ECT_REG_DCSOF);
    if (local_err) return local_err;
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        const auto other = static_cast<uint16_t>(i);
        // A slave in SAFE-OP or OP kept its clock since activation or a restore that
        // worked. A failed restore leaves its slave in SAFE-OP until the next check.
        if (other == pos || this->record(other).group != ENABLED_GROUP ||
            !this->record(other).hasdc || this->record(other).islost ||
            (this->record(other).state & 0x0F) < EC_STATE_SAFE_OP ||
            this->restore_failed(other))
            continue;
        const auto [other_local, other_local_err] = this->read_dc_register<int64_t>(
            other,
            ECT_REG_DCSOF
        );
        if (other_local_err) return other_local_err;
        const auto [other_offset, other_offset_err] = this->read_dc_register<int64_t>(
            other,
            ECT_REG_DCSYSOFFSET
        );
        if (other_offset_err) return other_offset_err;
        // Each slave latched its propagation delay after the reference clock.
        const int64_t system = other_local + other_offset + this->record(pos).pdelay -
                               this->record(other).pdelay;
        return this->write_dc_register(pos, ECT_REG_DCSYSOFFSET, system - local);
    }
    const int64_t offset = this->session.value().host_dc_offset.load(
        std::memory_order_relaxed
    );
    if (offset == NO_DC_FRAME)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no frame has carried the DC time to set the clock from"
        );
    return this->write_dc_register(
        pos,
        ECT_REG_DCSYSOFFSET,
        latched_at.nanoseconds() + offset - local
    );
}

x::errors::Error
Master::restore_dc(const uint16_t pos, const x::breaker::Breaker &breaker) {
    if (auto err = this->clear_sync(pos)) return err;
    if (auto err = this->write_dc_register(
            pos,
            ECT_REG_DCSYSDELAY,
            this->record(pos).pdelay
        ))
        return err;
    if (auto err = this->align_clock(pos)) return err;
    if (auto err = this->wait_for_dc_settle({pos}, [&breaker] {
            return !breaker.running();
        }))
        return err;
    const auto &setups = this->session.value().dc_setups;
    const auto setup = std::ranges::find(setups, pos, &DCSetup::position);
    if (setup == setups.end()) return x::errors::NIL;
    return this->start_sync(*setup);
}

master::Activation Master::request_state(const uint16_t state, const int timeout) {
    for (int i = 1; i <= this->api->context.slavecount; ++i) {
        if (this->record(i).group != ENABLED_GROUP) continue;
        this->record(i).state = state;
        this->api->writestate(i);
    }
    const auto deadline = this->cfg.clock.steady() + std::chrono::microseconds(timeout);
    while (true) {
        this->api->readstate();
        std::vector<uint16_t> pending;
        // Like ecx_statecheck, a slave in the state counts even with its error flag
        // set: a request without the ACK bit leaves the flag as it is.
        for (int i = 1; i <= this->api->context.slavecount; ++i)
            if (this->record(i).group == ENABLED_GROUP &&
                (this->record(i).state & ~EC_STATE_ERROR) != state)
                pending.push_back(i);
        if (pending.empty()) return {};
        std::vector<uint16_t> refused;
        std::ranges::copy_if(pending, std::back_inserter(refused), [&](auto pos) {
            return this->refuses(pos, state);
        });
        if (!refused.empty()) return this->state_error(refused, state);
        if (this->cfg.clock.steady() >= deadline)
            return this->state_error(pending, state);
        this->cfg.clock.sleep(STATE_POLL_INTERVAL);
    }
}

bool Master::al_error(const uint16_t pos) const {
    return (this->record(pos).state & EC_STATE_ERROR) != 0 &&
           this->record(pos).ALstatuscode != 0;
}

bool Master::restore_failed(const uint16_t pos) const {
    return this->session.value().dc_restore_errors.contains(pos);
}

bool Master::wait_for_frame(
    const std::chrono::steady_clock::time_point deadline,
    const std::function<bool()> &stopped
) const {
    // The next frame to come back can have left before the call.
    const auto target = this->frames.load(std::memory_order_acquire) + 2;
    while (this->frames.load(std::memory_order_acquire) < target) {
        if (this->cfg.clock.steady() >= deadline || stopped()) return false;
        this->cfg.clock.sleep(STATE_POLL_INTERVAL);
    }
    return true;
}

bool Master::refuses(const uint16_t pos, const uint16_t state) const {
    return this->al_error(pos) && (this->record(pos).state & ~EC_STATE_ERROR) < state;
}

master::Activation
Master::state_error(const std::vector<uint16_t> &failed, const uint16_t state) {
    const auto target = slave::to_string(slave::from_al_state(state));
    std::string msg;
    std::map<uint16_t, slave::Refusal> refusing;
    for (const auto pos: failed) {
        if (!msg.empty()) msg += "; ";
        msg += "slave " + std::to_string(pos) + " (" + this->slave_name(pos) +
               ") is in " +
               slave::to_string(slave::from_al_state(this->record(pos).state));
        if (!this->al_error(pos)) continue;
        const auto code = this->record(pos).ALstatuscode;
        const std::string text = ec_ALstatuscode2string(code);
        if (this->refuses(pos, state) &&
            std::ranges::find(TRANSIENT_FAULTS, code) == TRANSIENT_FAULTS.end())
            refusing[pos] = {
                .reason = "refused " + target + ": " + text,
                .advice = refusal_advice(code),
                .code = code,
            };
        msg += ": " + text;
    }
    return {
        .err = x::errors::Error(
            refusing.empty() ? errors::STATE_CHANGE_ERROR
                             : errors::STATE_CHANGE_REFUSED,
            "failed to reach " + target + ": " + msg
        ),
        .refusals = std::move(refusing),
    };
}

std::vector<master::Info> Manager::enumerate() {
    std::vector<master::Info> masters;
    ec_adaptert *adapter = ec_find_adapters();
    ec_adaptert *current = adapter;

    while (current != nullptr) {
        if (is_wired_interface(current->name, current->desc)) {
            master::Info info;
            info.key = current->name;
            info.description = current->desc;
            masters.push_back(std::move(info));
        } else {
            VLOG(1) << "[ethercat] skipping interface that cannot carry EtherCAT: "
                    << current->name;
        }
        current = current->next;
    }

    ec_free_adapters(adapter);
    return masters;
}

std::pair<std::shared_ptr<master::Master>, x::errors::Error>
Manager::create(const std::string &key) {
    if (key.empty())
        return {
            nullptr,
            x::errors::Error(errors::MASTER_INIT_ERROR, "empty interface name")
        };
    if (key.size() >= 4 && key.starts_with("igh:"))
        return {
            nullptr,
            x::errors::Error(
                errors::MASTER_INIT_ERROR,
                "invalid SOEM interface '" + key + "': IgH-style keys not supported"
            )
        };
    return {std::make_shared<Master>(std::make_unique<ProdAPI>(), key), x::errors::NIL};
}

}
