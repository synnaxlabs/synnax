// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <functional>

#include "driver/ethercat/slave/slave.h"

namespace ethercat::virtual_esc {

/// @name AL Register Addresses
/// Standard EtherCAT Application Layer register addresses (ETG.1000.4).
/// @{

/// AL Control register - write target state here.
constexpr uint16_t REG_AL_CONTROL = 0x0120;
/// AL Control register size in bytes.
constexpr size_t REG_AL_CONTROL_SIZE = 2;
/// AL Status register - read current state here.
constexpr uint16_t REG_AL_STATUS = 0x0130;
/// AL Status register size in bytes.
constexpr size_t REG_AL_STATUS_SIZE = 2;
/// AL Status Code register - error code when in ERROR state.
constexpr uint16_t REG_AL_STATUS_CODE = 0x0134;
/// AL Status Code register size in bytes.
constexpr size_t REG_AL_STATUS_CODE_SIZE = 2;

/// @}

/// @name DL Register Addresses
/// Data Link Layer register addresses.
/// @{

/// DL Control register.
constexpr uint16_t REG_DL_CONTROL = 0x0100;
/// DL Status register.
constexpr uint16_t REG_DL_STATUS = 0x0110;
/// Configured Station Address register.
constexpr uint16_t REG_STATION_ADDRESS = 0x0010;
/// Station Alias register.
constexpr uint16_t REG_STATION_ALIAS = 0x0012;

/// @}

/// @name EEPROM/SII Register Addresses
/// Slave Information Interface register addresses (ETG.1000.4).
/// @{

/// SII EEPROM Interface Control/Status register.
constexpr uint16_t REG_SII_CONTROL = 0x0500;
/// SII EEPROM Address register (4 bytes).
constexpr uint16_t REG_SII_ADDRESS = 0x0502;
/// SII EEPROM Data register (4 bytes, at 0x0508 per SOEM).
constexpr uint16_t REG_SII_DATA = 0x0508;

/// @}

/// @name Sync Manager Register Addresses
/// Sync Manager configuration base addresses.
/// @{

/// SM0 start address register.
constexpr uint16_t REG_SM0_START = 0x0800;
/// SM0 length register.
constexpr uint16_t REG_SM0_LENGTH = 0x0802;
/// SM0 control register.
constexpr uint16_t REG_SM0_CONTROL = 0x0804;
/// SM0 status register.
constexpr uint16_t REG_SM0_STATUS = 0x0805;
/// SM0 activate register.
constexpr uint16_t REG_SM0_ACTIVATE = 0x0806;
/// Sync Manager register block size.
constexpr size_t SM_REG_SIZE = 8;

/// @}

/// @name FMMU Register Addresses
/// Fieldbus Memory Management Unit base addresses.
/// @{

/// FMMU0 base address.
constexpr uint16_t REG_FMMU0 = 0x0600;
/// FMMU register block size.
constexpr size_t FMMU_REG_SIZE = 16;
/// Number of FMMUs.
constexpr size_t FMMU_COUNT = 8;

/// @}

/// @brief AL Status Codes (ETG.1000.6).
enum class ALStatusCode : uint16_t {
    NO_ERROR = 0x0000,
    UNSPECIFIED_ERROR = 0x0001,
    NO_MEMORY = 0x0002,
    INVALID_DEVICE_SETUP = 0x0003,
    SII_EEPROM_INFO_MISMATCH = 0x0006,
    FIRMWARE_UPDATE_FAILED = 0x0007,
    INVALID_STATE_CHANGE = 0x0011,
    UNKNOWN_REQUESTED_STATE = 0x0012,
    BOOTSTRAP_NOT_SUPPORTED = 0x0013,
    NO_VALID_FIRMWARE = 0x0014,
    INVALID_MAILBOX_CONFIG = 0x0016,
    INVALID_SM_CONFIG = 0x0017,
    NO_VALID_INPUTS = 0x0018,
    NO_VALID_OUTPUTS = 0x0019,
    SYNCHRONIZATION_ERROR = 0x001A,
    SM_WATCHDOG_TIMEOUT = 0x001B,
    INVALID_SM_TYPES = 0x001C,
    INVALID_OUTPUT_CONFIG = 0x001D,
    INVALID_INPUT_CONFIG = 0x001E,
    INVALID_DC_SYNC_CONFIG = 0x0030,
    INVALID_DC_LATCH_CONFIG = 0x0031,
    PLL_ERROR = 0x0032,
    DC_SYNC_IO_ERROR = 0x0033,
    DC_SYNC_TIMEOUT = 0x0034,
};

/// @brief EtherCAT slave state machine.
class StateMachine {
public:
    /// @brief Callback invoked on state transitions.
    using TransitionCallback = std::function<void(slave::State from, slave::State to)>;

    StateMachine(): state(slave::State::INIT), status_code(ALStatusCode::NO_ERROR) {}

    /// @brief Returns the current state.
    [[nodiscard]] slave::State current_state() const { return this->state; }

    /// @brief Returns the raw state value for AL Status register.
    [[nodiscard]] uint16_t al_status() const {
        return static_cast<uint16_t>(this->state);
    }

    /// @brief Returns the AL Status Code.
    [[nodiscard]] ALStatusCode al_status_code() const { return this->status_code; }

    /// @brief Sets the transition callback.
    void set_transition_callback(TransitionCallback cb) {
        this->on_transition = std::move(cb);
    }

    /// @brief Processes a write to the AL Control register.
    /// @return true if the state change was successful.
    bool request_state(uint16_t requested) {
        const auto target = static_cast<slave::State>(requested & 0x0F);
        if (!this->is_valid_transition(target)) {
            this->status_code = ALStatusCode::INVALID_STATE_CHANGE;
            return false;
        }
        this->state = target;
        this->status_code = ALStatusCode::NO_ERROR;
        return true;
    }

    /// @brief Sets the state directly (for error injection in tests).
    void set_state(slave::State s) { this->state = s; }

    /// @brief Sets the status code directly.
    void set_status_code(ALStatusCode code) { this->status_code = code; }

    /// @brief Returns true if in OPERATIONAL state.
    [[nodiscard]] bool is_operational() const { return this->state == slave::State::OP; }

private:
    slave::State state;
    ALStatusCode status_code;
    TransitionCallback on_transition;

    [[nodiscard]] bool is_valid_transition(slave::State target) const {
        switch (this->state) {
            case slave::State::INIT:
                return target == slave::State::INIT ||
                       target == slave::State::PRE_OP ||
                       target == slave::State::BOOT;
            case slave::State::PRE_OP:
                return target == slave::State::INIT ||
                       target == slave::State::PRE_OP ||
                       target == slave::State::SAFE_OP;
            case slave::State::SAFE_OP:
                return target == slave::State::INIT ||
                       target == slave::State::PRE_OP ||
                       target == slave::State::SAFE_OP ||
                       target == slave::State::OP;
            case slave::State::OP:
                return target == slave::State::INIT ||
                       target == slave::State::PRE_OP ||
                       target == slave::State::SAFE_OP ||
                       target == slave::State::OP;
            case slave::State::BOOT:
                return target == slave::State::INIT ||
                       target == slave::State::BOOT;
            default:
                return false;
        }
    }
};

}
