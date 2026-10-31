// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <functional>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include <fcntl.h>

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/bpf.h"

namespace driver::ethercat::soem {
/// @brief opens BPF devices in order, failing each with the next errno in errnos and
/// opening /dev/null for an errno of 0.
struct FakeOpen {
    std::vector<int> errnos;
    /// @brief the paths opened, in order.
    std::vector<std::string> paths;

    int operator()(const char *path, int) {
        const auto err = this->errnos.at(this->paths.size());
        this->paths.emplace_back(path);
        if (err == 0) return ::open("/dev/null", O_RDWR);
        errno = err;
        return -1;
    }
};

/// @brief it should pass when the first BPF device opens.
TEST(BPFTest, CheckAccessOpensFirstDevice) {
    FakeOpen open{.errnos = {0}};
    ASSERT_NIL(check_bpf_access("en0", std::ref(open)));
    EXPECT_EQ(open.paths, std::vector<std::string>{"/dev/bpf0"});
}

/// @brief it should skip a busy BPF device and pass when the next one opens.
TEST(BPFTest, CheckAccessSkipsBusyDevice) {
    FakeOpen open{.errnos = {EBUSY, 0}};
    ASSERT_NIL(check_bpf_access("en0", std::ref(open)));
    EXPECT_EQ(open.paths, (std::vector<std::string>{"/dev/bpf0", "/dev/bpf1"}));
}

/// @brief it should name the device and the macOS steps that give access to it when the
/// Driver lacks permission.
TEST(BPFTest, CheckAccessReportsPermission) {
    FakeOpen open{.errnos = {EBUSY, EACCES}};
    const auto err = check_bpf_access("en0", std::ref(open));
    ASSERT_OCCURRED_AS(err, errors::PERMISSION_DENIED);
    EXPECT_EQ(
        err.data,
        "The Driver does not have permission to open /dev/bpf1 for en0. To give it "
        "access, install the ChmodBPF package that comes with Wireshark, or give your "
        "user read and write access to /dev/bpf*. Then log out, log in again, and "
        "restart the Driver."
    );
}

/// @brief it should report that every BPF device is in use when the devices run out
/// after busy ones.
TEST(BPFTest, CheckAccessReportsEveryDeviceBusy) {
    FakeOpen open{.errnos = {EBUSY, EBUSY, ENOENT}};
    const auto err = check_bpf_access("en0", std::ref(open));
    ASSERT_OCCURRED_AS(err, errors::INTERFACE_ERROR);
    EXPECT_EQ(
        err.data,
        "Every BPF device is in use, so the Driver cannot open one for en0. Close "
        "other packet capture programs, such as Wireshark, to free one."
    );
}

/// @brief it should name the device and the cause when no BPF device exists.
TEST(BPFTest, CheckAccessReportsMissingDevice) {
    FakeOpen open{.errnos = {ENOENT}};
    const auto err = check_bpf_access("en0", std::ref(open));
    ASSERT_OCCURRED_AS(err, errors::INTERFACE_ERROR);
    EXPECT_EQ(err.data, "failed to open /dev/bpf0 for en0: No such file or directory");
}
}
