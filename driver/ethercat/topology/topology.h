// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <unordered_map>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/slave/slave.h"

namespace ethercat::topology {

/// @brief validates configured channels match current bus topology.
/// @param actual_slaves Current slaves from engine->slaves().
/// @param expected Map of device_key to SlaveProperties from config.
/// @return xerrors::NIL if topology matches, TOPOLOGY_MISMATCH otherwise.
[[nodiscard]] inline xerrors::Error validate(
    const std::vector<slave::Properties> &actual_slaves,
    const std::unordered_map<std::string, slave::Properties> &expected
) {
    for (const auto &[device_key, props]: expected) {
        bool found = false;
        for (const auto &slave: actual_slaves) {
            if (slave.position != props.position) continue;
            found = true;
            if (slave.vendor_id != props.vendor_id)
                return xerrors::Error(
                    TOPOLOGY_MISMATCH,
                    "device " + device_key + " at position " +
                        std::to_string(props.position) + ": expected vendor_id 0x" +
                        std::to_string(props.vendor_id) + ", found 0x" +
                        std::to_string(slave.vendor_id)
                );
            if (slave.product_code != props.product_code)
                return xerrors::Error(
                    TOPOLOGY_MISMATCH,
                    "device " + device_key + " at position " +
                        std::to_string(props.position) + ": expected product_code 0x" +
                        std::to_string(props.product_code) + ", found 0x" +
                        std::to_string(slave.product_code)
                );
            break;
        }
        if (!found)
            return xerrors::Error(
                TOPOLOGY_MISMATCH,
                "device " + device_key + " expected at position " +
                    std::to_string(props.position) + " not found on bus"
            );
    }
    return xerrors::NIL;
}

}
