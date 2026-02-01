// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/esi/known_devices.h"

#include <cstring>

#include "x/cpp/telem/telem.h"

namespace ethercat::esi {

namespace {

// Include the generated binary blob from the ESI submodule
#include "vendor/esi/esi/generated/registry_blob.inc"

// Binary format structures (must match Go generator)
struct BlobHeader {
    uint32_t magic;
    uint32_t version;
    uint32_t vendor_count;
    uint32_t device_index_count;
    uint32_t device_count;
    uint32_t pdo_count;
    uint32_t string_table_offset;
    uint32_t string_table_size;
};

struct BlobVendor {
    uint32_t vendor_id;
    uint32_t name_offset;
};

struct BlobDeviceIndex {
    uint32_t vendor_id;
    uint32_t product_code;
    uint32_t first_device;
    uint32_t device_count;
};

struct BlobDevice {
    uint32_t revision;
    uint32_t name_offset;
    uint32_t pdo_offset;
    uint16_t input_count;
    uint16_t output_count;
};

struct BlobPDO {
    uint16_t pdo_index;
    uint16_t index;
    uint8_t subindex;
    uint8_t bit_length;
    uint8_t data_type;
    uint8_t padding;
    uint32_t name_offset;
};

inline const BlobHeader* header() {
    return reinterpret_cast<const BlobHeader*>(REGISTRY_BLOB);
}

inline const char* string_at(uint32_t offset) {
    return reinterpret_cast<const char*>(REGISTRY_BLOB + header()->string_table_offset + offset);
}

inline const BlobVendor* vendors() {
    return reinterpret_cast<const BlobVendor*>(REGISTRY_BLOB + sizeof(BlobHeader));
}

inline const BlobDeviceIndex* device_index() {
    return reinterpret_cast<const BlobDeviceIndex*>(
        REGISTRY_BLOB + sizeof(BlobHeader) + header()->vendor_count * sizeof(BlobVendor)
    );
}

inline const BlobDevice* devices() {
    return reinterpret_cast<const BlobDevice*>(
        REGISTRY_BLOB + sizeof(BlobHeader) +
        header()->vendor_count * sizeof(BlobVendor) +
        header()->device_index_count * sizeof(BlobDeviceIndex)
    );
}

inline const BlobPDO* pdos() {
    return reinterpret_cast<const BlobPDO*>(
        REGISTRY_BLOB + sizeof(BlobHeader) +
        header()->vendor_count * sizeof(BlobVendor) +
        header()->device_index_count * sizeof(BlobDeviceIndex) +
        header()->device_count * sizeof(BlobDevice)
    );
}

telem::DataType id_to_data_type(uint8_t id) {
    switch (id) {
        case 1: return telem::UINT8_T;
        case 2: return telem::INT8_T;
        case 3: return telem::INT16_T;
        case 4: return telem::UINT16_T;
        case 5: return telem::INT32_T;
        case 6: return telem::UINT32_T;
        case 7: return telem::INT64_T;
        case 8: return telem::UINT64_T;
        case 9: return telem::FLOAT32_T;
        case 10: return telem::FLOAT64_T;
        default: return telem::UINT8_T;
    }
}

} // anonymous namespace

bool lookup_device_pdos(
    const uint32_t vendor_id,
    const uint32_t product_code,
    const uint32_t revision,
    SlaveInfo &slave
) {
    const auto* idx = device_index();
    const uint32_t idx_count = header()->device_index_count;

    // Binary search for (vendor_id, product_code)
    size_t lo = 0, hi = idx_count;
    while (lo < hi) {
        const size_t mid = lo + (hi - lo) / 2;
        const auto& entry = idx[mid];
        if (entry.vendor_id < vendor_id ||
            (entry.vendor_id == vendor_id && entry.product_code < product_code)) {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }

    if (lo >= idx_count) return false;
    const auto& entry = idx[lo];
    if (entry.vendor_id != vendor_id || entry.product_code != product_code) return false;

    // Search for exact revision match, fallback to first
    const auto* devs = devices();
    const BlobDevice* match = nullptr;
    for (uint32_t i = 0; i < entry.device_count; ++i) {
        const auto& dev = devs[entry.first_device + i];
        if (dev.revision == revision) {
            match = &dev;
            break;
        }
        if (match == nullptr) match = &dev;
    }

    if (match == nullptr) return false;

    const auto* pdo_table = pdos();

    slave.input_pdos.clear();
    slave.input_pdos.reserve(match->input_count);
    for (uint16_t i = 0; i < match->input_count; ++i) {
        const auto& p = pdo_table[match->pdo_offset + i];
        slave.input_pdos.emplace_back(
            p.pdo_index, p.index, p.subindex, p.bit_length,
            true, string_at(p.name_offset), id_to_data_type(p.data_type)
        );
    }

    slave.output_pdos.clear();
    slave.output_pdos.reserve(match->output_count);
    for (uint16_t i = 0; i < match->output_count; ++i) {
        const auto& p = pdo_table[match->pdo_offset + match->input_count + i];
        slave.output_pdos.emplace_back(
            p.pdo_index, p.index, p.subindex, p.bit_length,
            false, string_at(p.name_offset), id_to_data_type(p.data_type)
        );
    }

    slave.pdos_discovered = true;
    return true;
}

std::optional<std::string_view> vendor_name(const uint32_t vendor_id) {
    const auto* vends = vendors();
    const uint32_t count = header()->vendor_count;

    // Binary search for vendor
    size_t lo = 0, hi = count;
    while (lo < hi) {
        const size_t mid = lo + (hi - lo) / 2;
        if (vends[mid].vendor_id < vendor_id) {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }
    if (lo >= count || vends[lo].vendor_id != vendor_id) return std::nullopt;
    return std::string_view(string_at(vends[lo].name_offset));
}

bool is_device_known(const uint32_t vendor_id, const uint32_t product_code) {
    const auto* idx = device_index();
    const uint32_t idx_count = header()->device_index_count;

    // Binary search for (vendor_id, product_code)
    size_t lo = 0, hi = idx_count;
    while (lo < hi) {
        const size_t mid = lo + (hi - lo) / 2;
        const auto& entry = idx[mid];
        if (entry.vendor_id < vendor_id ||
            (entry.vendor_id == vendor_id && entry.product_code < product_code)) {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }
    if (lo >= idx_count) return false;
    const auto& entry = idx[lo];
    return entry.vendor_id == vendor_id && entry.product_code == product_code;
}

}
