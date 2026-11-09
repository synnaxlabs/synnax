// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/master/contract.h"
#include "driver/ethercat/testutil/master.h"

namespace driver::ethercat::testutil { namespace {
/// @brief the fake master for the master contract.
class Bus {
    Master m;
    uint16_t slaves = 0;

public:
    Master &master() { return this->m; }

    void add_slave() {
        const auto position = ++this->slaves;
        this->m.add_slave({
            .position = position,
            .name = "Slave " + std::to_string(position),
            .input_pdos = {{
                .pdo_index = 0x1A00,
                .index = 0x6000,
                .sub_index = 0x01,
                .bit_length = 16,
                .is_input = true,
                .data_type = x::telem::INT16_T,
            }},
        });
    }

    void refuse(const uint16_t position) {
        this->m.refuse(position, {.reason = "refused SAFE-OP"});
    }
};
}}

namespace driver::ethercat::master::contract {
INSTANTIATE_TYPED_TEST_SUITE_P(Fake, Contract, testutil::Bus);
}
