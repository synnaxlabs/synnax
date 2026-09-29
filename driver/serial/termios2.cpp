// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>

#include <asm/termbits.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#include <unistd.h>

namespace driver::serial {
/// @returns the output speed of the terminal at path, read with termios2 because
/// <termios.h> reports only the speeds in its table.
std::uint32_t output_speed(const std::string &path) {
    const int fd = ::open(path.c_str(), O_RDWR | O_NOCTTY | O_NONBLOCK);
    termios2 tio{};
    ::ioctl(fd, TCGETS2, &tio);
    ::close(fd);
    return tio.c_ospeed;
}
}
