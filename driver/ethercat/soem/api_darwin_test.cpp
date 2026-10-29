// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include <unistd.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
/// @brief it should pass when the Driver can open a BPF device, and otherwise name
/// the device and the macOS steps that give access to it.
TEST(SOEMAPITest, CheckSocketReportsBPFPermission) {
    ProdAPI api;
    const auto err = api.check_socket("en0");
    if (::access("/dev/bpf0", R_OK | W_OK) == 0) {
        ASSERT_NIL(err);
        return;
    }
    ASSERT_OCCURRED_AS(err, errors::PERMISSION_DENIED);
    EXPECT_EQ(
        err.data,
        "The Driver does not have permission to open /dev/bpf0 for en0. To give it "
        "access, install the ChmodBPF package that comes with Wireshark, or give your "
        "user read and write access to /dev/bpf*. Then log out, log in again, and "
        "restart the Driver."
    );
}
}
