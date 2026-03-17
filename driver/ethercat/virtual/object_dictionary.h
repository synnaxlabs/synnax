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
#include <cstring>
#include <optional>
#include <string>
#include <unordered_map>
#include <variant>
#include <vector>

namespace ethercat::virtual_esc {

/// @name Standard Object Dictionary Indices
/// @{

/// Device Type object.
constexpr uint16_t OD_DEVICE_TYPE = 0x1000;
/// Device Name object.
constexpr uint16_t OD_DEVICE_NAME = 0x1008;
/// Hardware Version object.
constexpr uint16_t OD_HW_VERSION = 0x1009;
/// Software Version object.
constexpr uint16_t OD_SW_VERSION = 0x100A;
/// Identity Object.
constexpr uint16_t OD_IDENTITY = 0x1018;
/// Sync Manager Communication Type.
constexpr uint16_t OD_SM_COM_TYPE = 0x1C00;
/// RxPDO Assign.
constexpr uint16_t OD_RXPDO_ASSIGN = 0x1C12;
/// TxPDO Assign.
constexpr uint16_t OD_TXPDO_ASSIGN = 0x1C13;
/// First RxPDO Mapping.
constexpr uint16_t OD_RXPDO_MAPPING = 0x1600;
/// First TxPDO Mapping.
constexpr uint16_t OD_TXPDO_MAPPING = 0x1A00;

/// @}

/// @name Mailbox Protocol Types
/// @{

/// CoE (CANopen over EtherCAT) protocol bit.
constexpr uint8_t MBX_PROTO_COE = 0x04;

/// @}

/// @name CoE Service Types
/// @{

constexpr uint8_t COE_SDO_REQUEST = 0x02;
constexpr uint8_t COE_SDO_RESPONSE = 0x03;
constexpr uint8_t COE_SDO_INFO = 0x08;

/// @}

/// @name SDO Command Specifiers
/// @{

constexpr uint8_t SDO_CCS_DOWNLOAD_INIT = 0x01;
constexpr uint8_t SDO_CCS_DOWNLOAD_SEG = 0x00;
constexpr uint8_t SDO_CCS_UPLOAD_INIT = 0x02;
constexpr uint8_t SDO_CCS_UPLOAD_SEG = 0x03;
constexpr uint8_t SDO_CCS_ABORT = 0x04;

constexpr uint8_t SDO_SCS_DOWNLOAD_INIT = 0x03;
constexpr uint8_t SDO_SCS_DOWNLOAD_SEG = 0x01;
constexpr uint8_t SDO_SCS_UPLOAD_INIT = 0x02;
constexpr uint8_t SDO_SCS_UPLOAD_SEG = 0x00;

/// @}

/// @name SDO Abort Codes
/// @{

constexpr uint32_t SDO_ABORT_TOGGLE_BIT = 0x05030000;
constexpr uint32_t SDO_ABORT_TIMEOUT = 0x05040000;
constexpr uint32_t SDO_ABORT_INVALID_CS = 0x05040001;
constexpr uint32_t SDO_ABORT_OUT_OF_MEMORY = 0x05040005;
constexpr uint32_t SDO_ABORT_UNSUPPORTED_ACCESS = 0x06010000;
constexpr uint32_t SDO_ABORT_WRITE_ONLY = 0x06010001;
constexpr uint32_t SDO_ABORT_READ_ONLY = 0x06010002;
constexpr uint32_t SDO_ABORT_OBJECT_NOT_FOUND = 0x06020000;
constexpr uint32_t SDO_ABORT_MAPPING_FAILED = 0x06040041;
constexpr uint32_t SDO_ABORT_PDO_LENGTH = 0x06040042;
constexpr uint32_t SDO_ABORT_PARAM_INCOMPATIBLE = 0x06040043;
constexpr uint32_t SDO_ABORT_INTERNAL_ERROR = 0x06040047;
constexpr uint32_t SDO_ABORT_SUBINDEX_NOT_FOUND = 0x06090011;
constexpr uint32_t SDO_ABORT_VALUE_RANGE = 0x06090030;
constexpr uint32_t SDO_ABORT_VALUE_TOO_HIGH = 0x06090031;
constexpr uint32_t SDO_ABORT_VALUE_TOO_LOW = 0x06090032;
constexpr uint32_t SDO_ABORT_GENERAL_ERROR = 0x08000000;

/// @}

/// @brief Represents a PDO mapping entry (index:subindex:bitlength).
struct PDOMappingEntry {
    uint16_t index;
    uint8_t sub_index;
    uint8_t bit_length;

    /// @brief Returns the 32-bit packed mapping value.
    [[nodiscard]] uint32_t packed() const {
        return (static_cast<uint32_t>(this->index) << 16) |
               (static_cast<uint32_t>(this->sub_index) << 8) |
               static_cast<uint32_t>(this->bit_length);
    }
};

/// @brief Represents a PDO configuration.
struct PDOConfig {
    uint16_t index;
    std::vector<PDOMappingEntry> entries;

    /// @brief Returns the total bit size of all entries.
    [[nodiscard]] size_t total_bits() const {
        size_t bits = 0;
        for (const auto& e : this->entries) bits += e.bit_length;
        return bits;
    }

    /// @brief Returns the total byte size (rounded up).
    [[nodiscard]] size_t total_bytes() const {
        return (this->total_bits() + 7) / 8;
    }
};

/// @brief Object value types supported in the object dictionary.
using ObjectValue = std::variant<
    uint8_t,
    uint16_t,
    uint32_t,
    int8_t,
    int16_t,
    int32_t,
    std::string,
    std::vector<uint8_t>
>;

/// @brief Represents an object dictionary entry.
struct ObjectEntry {
    uint16_t index;
    uint8_t sub_index;
    std::string name;
    ObjectValue value;
    bool readable;
    bool writable;
};

/// @brief CoE Object Dictionary implementation.
class ObjectDictionary {
public:
    ObjectDictionary() = default;

    /// @brief Sets the device identity information.
    void set_identity(
        uint32_t vendor_id,
        uint32_t product_code,
        uint32_t revision,
        uint32_t serial
    ) {
        this->vendor_id = vendor_id;
        this->product_code = product_code;
        this->revision = revision;
        this->serial = serial;
    }

    /// @brief Sets the device name.
    void set_device_name(std::string name) { this->device_name = std::move(name); }

    /// @brief Sets the hardware version string.
    void set_hw_version(std::string version) { this->hw_version = std::move(version); }

    /// @brief Sets the software version string.
    void set_sw_version(std::string version) { this->sw_version = std::move(version); }

    /// @brief Adds a TxPDO (slave to master) configuration.
    void add_tx_pdo(PDOConfig pdo) { this->tx_pdos.push_back(std::move(pdo)); }

    /// @brief Adds an RxPDO (master to slave) configuration.
    void add_rx_pdo(PDOConfig pdo) { this->rx_pdos.push_back(std::move(pdo)); }

    /// @brief Reads an object by index and subindex.
    /// @return The data read, or nullopt if not found.
    [[nodiscard]] std::optional<std::vector<uint8_t>>
    read(uint16_t index, uint8_t sub_index) const {
        std::vector<uint8_t> result;
        if (index == OD_DEVICE_TYPE && sub_index == 0) {
            result.resize(4);
            const uint32_t device_type = 0x00001389;
            std::memcpy(result.data(), &device_type, 4);
            return result;
        }
        if (index == OD_DEVICE_NAME && sub_index == 0) {
            result.assign(this->device_name.begin(), this->device_name.end());
            return result;
        }
        if (index == OD_HW_VERSION && sub_index == 0) {
            result.assign(this->hw_version.begin(), this->hw_version.end());
            return result;
        }
        if (index == OD_SW_VERSION && sub_index == 0) {
            result.assign(this->sw_version.begin(), this->sw_version.end());
            return result;
        }
        if (index == OD_IDENTITY) return this->read_identity(sub_index);
        if (index == OD_SM_COM_TYPE) return this->read_sm_com_type(sub_index);
        if (index == OD_RXPDO_ASSIGN) return this->read_pdo_assign(sub_index, false);
        if (index == OD_TXPDO_ASSIGN) return this->read_pdo_assign(sub_index, true);
        if (index >= OD_RXPDO_MAPPING && index < OD_RXPDO_MAPPING + 0x100)
            return this->read_pdo_mapping(index, sub_index, false);
        if (index >= OD_TXPDO_MAPPING && index < OD_TXPDO_MAPPING + 0x100)
            return this->read_pdo_mapping(index, sub_index, true);
        auto it = this->custom_objects.find(make_key(index, sub_index));
        if (it != this->custom_objects.end())
            return this->value_to_bytes(it->second.value);
        return std::nullopt;
    }

    /// @brief Writes an object by index and subindex.
    /// @return true if write succeeded.
    bool write(uint16_t index, uint8_t sub_index, std::span<const uint8_t> data) {
        auto it = this->custom_objects.find(make_key(index, sub_index));
        if (it != this->custom_objects.end() && it->second.writable) {
            it->second.value = std::vector<uint8_t>(data.begin(), data.end());
            return true;
        }
        return false;
    }

    /// @brief Adds a custom object entry.
    void add_object(ObjectEntry entry) {
        this->custom_objects[make_key(entry.index, entry.sub_index)] = std::move(entry);
    }

    /// @brief Returns total input (TxPDO) size in bytes.
    [[nodiscard]] size_t total_tx_pdo_bytes() const {
        size_t total = 0;
        for (const auto& pdo : this->tx_pdos) total += pdo.total_bytes();
        return total;
    }

    /// @brief Returns total output (RxPDO) size in bytes.
    [[nodiscard]] size_t total_rx_pdo_bytes() const {
        size_t total = 0;
        for (const auto& pdo : this->rx_pdos) total += pdo.total_bytes();
        return total;
    }

    /// @brief Returns the TxPDO configurations.
    [[nodiscard]] const std::vector<PDOConfig>& tx_pdo_configs() const {
        return this->tx_pdos;
    }

    /// @brief Returns the RxPDO configurations.
    [[nodiscard]] const std::vector<PDOConfig>& rx_pdo_configs() const {
        return this->rx_pdos;
    }

private:
    uint32_t vendor_id = 0;
    uint32_t product_code = 0;
    uint32_t revision = 0;
    uint32_t serial = 0;
    std::string device_name = "Virtual Test Slave";
    std::string hw_version = "1.0";
    std::string sw_version = "1.0";
    std::vector<PDOConfig> tx_pdos;
    std::vector<PDOConfig> rx_pdos;
    std::unordered_map<uint32_t, ObjectEntry> custom_objects;

    static uint32_t make_key(uint16_t index, uint8_t sub_index) {
        return (static_cast<uint32_t>(index) << 8) | sub_index;
    }

    [[nodiscard]] std::optional<std::vector<uint8_t>>
    read_identity(uint8_t sub_index) const {
        std::vector<uint8_t> result;
        if (sub_index == 0) {
            result.push_back(4);
            return result;
        }
        result.resize(4);
        uint32_t value = 0;
        switch (sub_index) {
            case 1: value = this->vendor_id; break;
            case 2: value = this->product_code; break;
            case 3: value = this->revision; break;
            case 4: value = this->serial; break;
            default: return std::nullopt;
        }
        std::memcpy(result.data(), &value, 4);
        return result;
    }

    [[nodiscard]] std::optional<std::vector<uint8_t>>
    read_sm_com_type(uint8_t sub_index) const {
        std::vector<uint8_t> result;
        if (sub_index == 0) {
            result.push_back(4);
            return result;
        }
        if (sub_index > 4) return std::nullopt;
        result.push_back(sub_index);
        return result;
    }

    [[nodiscard]] std::optional<std::vector<uint8_t>>
    read_pdo_assign(uint8_t sub_index, bool is_tx) const {
        const auto& pdos = is_tx ? this->tx_pdos : this->rx_pdos;
        std::vector<uint8_t> result;
        if (sub_index == 0) {
            result.push_back(static_cast<uint8_t>(pdos.size()));
            return result;
        }
        if (sub_index > pdos.size()) return std::nullopt;
        result.resize(2);
        const uint16_t pdo_index = pdos[sub_index - 1].index;
        std::memcpy(result.data(), &pdo_index, 2);
        return result;
    }

    [[nodiscard]] std::optional<std::vector<uint8_t>>
    read_pdo_mapping(uint16_t index, uint8_t sub_index, bool is_tx) const {
        const auto& pdos = is_tx ? this->tx_pdos : this->rx_pdos;
        const uint16_t base = is_tx ? OD_TXPDO_MAPPING : OD_RXPDO_MAPPING;
        const size_t pdo_idx = index - base;
        if (pdo_idx >= pdos.size()) return std::nullopt;
        const auto& pdo = pdos[pdo_idx];
        std::vector<uint8_t> result;
        if (sub_index == 0) {
            result.push_back(static_cast<uint8_t>(pdo.entries.size()));
            return result;
        }
        if (sub_index > pdo.entries.size()) return std::nullopt;
        result.resize(4);
        const uint32_t mapping = pdo.entries[sub_index - 1].packed();
        std::memcpy(result.data(), &mapping, 4);
        return result;
    }

    [[nodiscard]] std::vector<uint8_t> value_to_bytes(const ObjectValue& val) const {
        std::vector<uint8_t> result;
        std::visit([&result](const auto& v) {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, std::string>) {
                result.assign(v.begin(), v.end());
            } else if constexpr (std::is_same_v<T, std::vector<uint8_t>>) {
                result = v;
            } else {
                result.resize(sizeof(T));
                std::memcpy(result.data(), &v, sizeof(T));
            }
        }, val);
        return result;
    }
};

}
