// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <string>

#include "gtest/gtest.h"
#include <fcntl.h>
#include <sys/ioctl.h>
#include <unistd.h>

#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
/// @returns true when the terminal at path refuses programs other than its holder. A
/// program without root cannot open it at all, and root can read its exclusive flag.
bool exclusive(const std::string &path) {
    const int fd = ::open(path.c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK);
    if (fd < 0) return errno == EBUSY;
    int flag = 0;
    const bool read = ::ioctl(fd, TIOCGEXCL, &flag) == 0;
    ::close(fd);
    return read && flag != 0;
}
}

/// @brief it should keep other programs from opening the port while it is open.
TEST_F(PtyTest, KeepsOtherProgramsOutWhileOpen) {
    const auto port = ASSERT_NIL_P(Port::open(this->props()));
    EXPECT_TRUE(exclusive(this->path()));
}

/// @brief it should refuse a port that another program holds with TIOCEXCL alone and
/// leave that program's flag set.
TEST_F(PtyTest, RefusesAPortThatAnotherProgramHoldsExclusively) {
    const int other = ::open(this->path().c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK);
    ASSERT_GE(other, 0);
    x::defer::defer close_other([other] { ::close(other); });
    ASSERT_EQ(::ioctl(other, TIOCEXCL), 0);
    ASSERT_OCCURRED_AS_P(Port::open(this->props()), transport::UNREACHABLE_ERROR);
    int flag = 0;
    ASSERT_EQ(::ioctl(other, TIOCGEXCL, &flag), 0);
    EXPECT_NE(flag, 0);
}
}
