// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cmath>
#include <cstdint>
#include <format>
#include <limits>
#include <optional>
#include <string>
#include <string_view>
#include <type_traits>
#include <vector>

#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::slave {

/// @brief EtherCAT slave application layer states as defined in ETG.1000.
enum class State : uint8_t {
    UNKNOWN = 0,
    INIT = 1,
    PRE_OP = 2,
    SAFE_OP = 4,
    OP = 8,
    BOOT = 3
};

/// @brief converts a slave::State enum value to its string representation.
inline std::string to_string(const State state) {
    switch (state) {
        case State::INIT:
            return "INIT";
        case State::PRE_OP:
            return "PRE-OP";
        case State::SAFE_OP:
            return "SAFE-OP";
        case State::OP:
            return "OP";
        case State::BOOT:
            return "BOOT";
        default:
            return "UNKNOWN";
    }
}

/// @brief converts an EtherCAT AL (Application Layer) state register value to our
/// slave::State enum. Masks with 0x0F to strip error flag bits (SOEM sets bit 4 on
/// error). Works for both SOEM uint16 and IgH uint8 state values.
inline State from_al_state(const uint16_t al_state) {
    switch (al_state & 0x0F) {
        case 0x01:
            return State::INIT;
        case 0x02:
            return State::PRE_OP;
        case 0x03:
            return State::BOOT;
        case 0x04:
            return State::SAFE_OP;
        case 0x08:
            return State::OP;
        default:
            return State::UNKNOWN;
    }
}

/// @brief a CoE object the master writes to a slave in PRE-OP each time it moves the
/// slave to SAFE-OP, including recovery after a fault.
struct StartupParameter {
    /// @brief object dictionary index.
    uint16_t index = 0;
    /// @brief object dictionary sub-index.
    uint8_t sub_index = 0;
    /// @brief the type the slave expects, which sets the size of the write.
    x::telem::DataType data_type = x::telem::UINT8_T;
    /// @brief the value to write.
    double value = 0;
    /// @brief what the parameter sets, for display.
    std::string name;

    bool operator==(const StartupParameter &) const = default;

    /// @brief parses a startup parameter, adding a field error for an index, sub-index,
    /// data type, or value the slave cannot take.
    static StartupParameter parse(x::json::Parser &parser) {
        const auto errors_before = parser.errors->size();
        StartupParameter param;
        const auto index = parser.field<double>("index");
        const auto sub_index = parser.field<double>("sub_index");
        param.value = parser.field<double>("value");
        param.name = parser.field<std::string>("name", "");
        const auto data_type = parser.field<std::string>("data_type");
        if (parser.errors->size() > errors_before) return param;
        if (whole_in(index, std::numeric_limits<uint16_t>::max()))
            param.index = static_cast<uint16_t>(index);
        else
            parser.field_err("index", "must be a whole number from 0x0000 to 0xFFFF");
        if (whole_in(sub_index, std::numeric_limits<uint8_t>::max()))
            param.sub_index = static_cast<uint8_t>(sub_index);
        else
            parser.field_err("sub_index", "must be a whole number from 0 to 255");
        param.data_type = x::telem::DataType(data_type);
        std::string value_err;
        if (!visit(param.data_type, [&]<typename T>() {
                value_err = value_error<T>(param.value, data_type);
            }))
            parser.field_err("data_type", "unsupported data type " + data_type);
        else if (!value_err.empty())
            parser.field_err("value", value_err);
        return param;
    }

    /// @brief returns the object address and name, e.g. "0x8000:15 (Filter)".
    [[nodiscard]] std::string to_string() const {
        const auto address = telem::format_index_sub_index(
            this->index,
            this->sub_index
        );
        return this->name.empty() ? address : address + " (" + this->name + ")";
    }

    /// @brief returns the little-endian bytes of the value in data_type.
    [[nodiscard]] std::vector<uint8_t> bytes() const {
        const auto sample = this->data_type.cast(this->value);
        const auto *data = static_cast<const uint8_t *>(
            x::telem::cast_to_void_ptr(sample)
        );
        return {data, data + this->data_type.density()};
    }

private:
    /// @brief calls f with the C++ type of a numeric data type, returning false for any
    /// other data type.
    template<typename F>
    static bool visit(const x::telem::DataType &dt, const F &f) {
        if (dt == x::telem::INT8_T)
            f.template operator()<int8_t>();
        else if (dt == x::telem::INT16_T)
            f.template operator()<int16_t>();
        else if (dt == x::telem::INT32_T)
            f.template operator()<int32_t>();
        else if (dt == x::telem::INT64_T)
            f.template operator()<int64_t>();
        else if (dt == x::telem::UINT8_T)
            f.template operator()<uint8_t>();
        else if (dt == x::telem::UINT16_T)
            f.template operator()<uint16_t>();
        else if (dt == x::telem::UINT32_T)
            f.template operator()<uint32_t>();
        else if (dt == x::telem::UINT64_T)
            f.template operator()<uint64_t>();
        else if (dt == x::telem::FLOAT32_T)
            f.template operator()<float>();
        else if (dt == x::telem::FLOAT64_T)
            f.template operator()<double>();
        else
            return false;
        return true;
    }

    static bool whole_in(const double v, const double max) {
        return v == std::trunc(v) && v >= 0 && v <= max;
    }

    /// @brief returns why v is not a value of T, or an empty string when it is.
    template<typename T>
    static std::string value_error(const double v, const std::string &data_type) {
        if constexpr (std::is_floating_point_v<T>) {
            if (std::abs(v) <= static_cast<double>(std::numeric_limits<T>::max()))
                return "";
            return std::format("{} does not fit in {}", v, data_type);
        } else {
            // A double rounds whole numbers of magnitude 2^53 and above, so a larger
            // value may not be the one the user entered.
            constexpr int DIGITS = std::numeric_limits<double>::digits;
            constexpr int64_t EXACT = (int64_t{1} << DIGITS) - 1;
            constexpr auto MAX = static_cast<int64_t>(
                std::min<uint64_t>(EXACT, std::numeric_limits<T>::max())
            );
            constexpr auto MIN = std::max<int64_t>(
                -EXACT,
                std::numeric_limits<T>::min()
            );
            if (v == std::trunc(v) && v >= MIN && v <= MAX) return "";
            return std::format("must be a whole number from {} to {}", MIN, MAX);
        }
    }
};

/// @brief the settings the master applies to a slave when the bus starts.
struct Config {
    /// @brief whether the slave joins cyclic exchange.
    bool enabled = true;
    /// @brief the mode the user chose. With nullopt, the SOEM master runs the mode that
    /// Properties::catalog_op_mode gives, else the one that dc::select picks, and the
    /// IgH master sets up no DC sync.
    std::optional<dc::Choice> op_mode;
    /// @brief CoE objects written to the slave before each move to SAFE-OP, unless
    /// another program sets up the slave.
    std::vector<StartupParameter> startup_parameters;

    bool operator==(const Config &) const = default;

    /// @brief adds each parameter to the startup parameters, in place of one at the
    /// same index and sub-index.
    void add_startup_parameters(const std::vector<StartupParameter> &parameters) {
        for (const auto &param: parameters) {
            const auto it = std::ranges::find_if(
                this->startup_parameters,
                [&param](const StartupParameter &p) {
                    return p.index == param.index && p.sub_index == param.sub_index;
                }
            );
            if (it == this->startup_parameters.end())
                this->startup_parameters.push_back(param);
            else
                *it = param;
        }
    }
};

/// @brief static properties of an EtherCAT slave device stored in device.properties.
struct Properties {
    /// @brief network interface the slave is connected to.
    std::string network;
    /// @brief position of the slave on the bus, counting from 1.
    uint16_t position{};
    /// @brief EtherCAT vendor ID assigned by ETG.
    uint32_t vendor_id{};
    /// @brief product code identifying the slave type.
    uint32_t product_code{};
    /// @brief revision number for hardware/firmware versioning.
    uint32_t revision{};
    /// @brief serial number of the device (if available).
    uint32_t serial{};
    /// @brief human-readable name of the slave device.
    std::string name;
    /// @brief discovered input PDOs (TxPDO, slave→master).
    std::vector<pdo::Properties> input_pdos;
    /// @brief discovered output PDOs (RxPDO, master→slave).
    std::vector<pdo::Properties> output_pdos;
    /// @brief synchronization modes the slave lists in its SII, in SII order.
    std::vector<dc::OpMode> op_modes;
    /// @brief the settings the user chose. to_json leaves out the startup parameters,
    /// so a scan keeps them.
    Config config;
    /// @brief the settings that the ESI catalog describes for the slave at the last
    /// scan, as esi::read_settings returns them, or nullopt when it describes none.
    std::optional<nlohmann::json> settings;

    /// @brief returns the entry at the given address in one direction, or nullopt when
    /// the slave has no such entry. Padding never matches.
    [[nodiscard]] std::optional<pdo::Properties>
    find_pdo(const uint16_t index, const uint8_t sub_index, const bool is_input) const {
        for (const auto &pdo: is_input ? this->input_pdos : this->output_pdos)
            if (!pdo.is_padding() && pdo.index == index && pdo.sub_index == sub_index)
                return pdo;
        return std::nullopt;
    }

    /// @brief returns the sync mode that the catalog gives for the mode the card held
    /// at the scan, or nullopt when it gives none.
    [[nodiscard]] std::optional<dc::Choice> catalog_op_mode() const {
        if (!this->settings || !this->settings->contains("mode")) return std::nullopt;
        const auto &mode = this->settings->at("mode");
        if (!mode.contains("op_mode")) return std::nullopt;
        return dc::Choice::parse(x::json::Parser(mode.at("op_mode")));
    }

    /// @brief whether the card held, at the scan, a mode in which another program sets
    /// it up.
    [[nodiscard]] bool shared() const {
        if (!this->settings || !this->settings->contains("mode")) return false;
        return this->settings->at("mode").value("shared", false);
    }

    /// @brief parses slave properties from JSON.
    static Properties parse(x::json::Parser &parser) {
        Properties props;
        props.serial = parser.field<uint32_t>("serial");
        props.vendor_id = parser.field<uint32_t>("vendor_id");
        props.product_code = parser.field<uint32_t>("product_code");
        props.revision = parser.field<uint32_t>("revision");
        props.name = parser.field<std::string>("name");
        props.network = parser.field<std::string>("network", "");
        const auto errors_before = parser.errors->size();
        const auto position = parser.field<double>("position");
        if (position >= 1 && position <= std::numeric_limits<uint16_t>::max() &&
            position == std::trunc(position))
            props.position = static_cast<uint16_t>(position);
        else if (parser.errors->size() == errors_before)
            parser.field_err("position", "must be a whole number from 1 to 65535");
        props.config.enabled = parser.field<bool>("enabled");
        props.op_modes = parser.field<std::vector<dc::OpMode>>("op_modes", {});
        props.config.op_mode = parser.field<std::optional<dc::Choice>>("op_mode");
        auto &params = props.config.startup_parameters;
        if (parser.has("startup_parameters"))
            parser.iter("startup_parameters", [&params](x::json::Parser &param) {
                params.push_back(StartupParameter::parse(param));
            });
        if (parser.has("settings"))
            props.settings = parser.field<x::json::json>("settings");
        const auto pdos_parser = parser.child("pdos");
        if (!pdos_parser.error()) {
            pdos_parser.iter("inputs", [&props](x::json::Parser &pdo) {
                props.input_pdos.push_back(pdo::Properties::parse(pdo, true));
            });
            pdos_parser.iter("outputs", [&props](x::json::Parser &pdo) {
                props.output_pdos.push_back(pdo::Properties::parse(pdo, false));
            });
        }
        return props;
    }

    /// @brief serializes this slave's properties to JSON, leaving out padding entries.
    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json props;
        props["vendor_id"] = this->vendor_id;
        props["product_code"] = this->product_code;
        props["revision"] = this->revision;
        props["serial"] = this->serial;
        props["name"] = this->name;
        props["network"] = network;
        props["position"] = this->position;
        props["enabled"] = this->config.enabled;

        nlohmann::json op_modes = nlohmann::json::array();
        for (const auto &mode: this->op_modes)
            op_modes.push_back(mode.to_json());
        props["op_modes"] = op_modes;
        if (this->config.op_mode) props["op_mode"] = this->config.op_mode->to_json();

        nlohmann::json inputs = nlohmann::json::array();
        for (const auto &pdo: this->input_pdos)
            if (!pdo.is_padding()) inputs.push_back(pdo.to_json());

        nlohmann::json outputs = nlohmann::json::array();
        for (const auto &pdo: this->output_pdos)
            if (!pdo.is_padding()) outputs.push_back(pdo.to_json());

        props["pdos"] = {{"inputs", inputs}, {"outputs", outputs}};
        if (this->settings) props["settings"] = *this->settings;
        return props;
    }
};

/// @brief dynamic status information about an EtherCAT slave from discovery.
struct Status {
    /// @brief current application layer state of the slave.
    State state = State::UNKNOWN;
    /// @brief why PDO discovery failed, or empty when it succeeded.
    std::string pdo_discovery_error;
    /// @brief why the read of the settings failed, or empty when it succeeded.
    std::string settings_error;
    /// @brief why the read of the serial number got no answer, or empty when the slave
    /// answered. The serial number of a slave with one is not known.
    std::string serial_error;
};

/// @brief what to do about a slave that reports "Invalid DC SYNC configuration". A
/// message about one or more such slaves ends with it once.
inline constexpr std::string_view DC_SYNC_ADVICE =
    "To fix \"Invalid DC SYNC configuration\", right-click the device, select "
    "Configure, and pick another sync mode.";

/// @brief advice for a slave that reports an invalid input or output configuration.
inline constexpr std::string_view
    MODE_RESTART_ADVICE = "If the mode of the device changed, restart its power.";

/// @brief why the bus started without a slave that refused a state.
struct Refusal {
    /// @brief the refused state and the AL status text, such as "refused SAFE-OP:
    /// Inconsistent settings".
    std::string reason;
    /// @brief what the user can do about the refusal, such as DC_SYNC_ADVICE, or empty
    /// for none. Each message about the refusal ends with it.
    std::string_view advice;

    bool operator==(const Refusal &) const = default;
};

/// @brief the condition of a slave on a running bus.
struct Health {
    /// @brief position of the slave on the bus.
    uint16_t position = 0;
    /// @brief application layer state the slave reports.
    State state = State::UNKNOWN;
    /// @brief true when the slave set the error flag of its AL status, rejected a
    /// startup parameter, or the master could not restore its distributed clock.
    bool faulted = false;
    /// @brief true when the slave stopped responding.
    bool lost = false;
    /// @brief description of the slave's AL status code, empty when it reports none or
    /// the master cannot read it.
    std::string al_status;
    /// @brief why the master could not restore the slave's distributed clock, or empty.
    std::string dc_restore_error;
    /// @brief why the slave rejected a startup parameter, or empty.
    std::string startup_error;
    /// @brief why the bus started without the slave, or nullopt when the slave is on
    /// the bus.
    std::optional<Refusal> refusal;

    bool operator==(const Health &) const = default;

    /// @brief returns true when the slave is in OP with no fault.
    [[nodiscard]] bool operational() const {
        return this->state == State::OP && !this->faulted && !this->lost;
    }

    /// @brief returns a short description such as "SAFE-OP + ERROR: Sync manager
    /// watchdog".
    [[nodiscard]] std::string to_string() const {
        if (this->lost) return "not responding";
        std::string desc = slave::to_string(this->state);
        if (this->faulted) desc += " + ERROR";
        if (!this->al_status.empty()) desc += ": " + this->al_status;
        if (!this->dc_restore_error.empty())
            desc += ": failed to restore the distributed clock: " +
                    this->dc_restore_error;
        if (!this->startup_error.empty()) desc += ": " + this->startup_error;
        return desc;
    }
};

/// @brief combined result from slave discovery containing both static properties
/// and dynamic status.
struct DiscoveryResult {
    Properties properties;
    Status status;
};

/// @brief gives serial number 0 to each slave that has the vendor, product, and serial
/// number of a slave before it, so that a serial number names one slave. A slave with
/// serial number 0 has its position as its identity.
inline void clear_duplicate_serials(std::vector<DiscoveryResult> &slaves) {
    for (auto it = slaves.begin(); it != slaves.end(); ++it) {
        auto &props = it->properties;
        if (props.serial == 0) continue;
        if (std::any_of(slaves.begin(), it, [&props](const DiscoveryResult &s) {
                const auto &p = s.properties;
                return p.vendor_id == props.vendor_id &&
                       p.product_code == props.product_code && p.serial == props.serial;
            }))
            props.serial = 0;
    }
}

}
