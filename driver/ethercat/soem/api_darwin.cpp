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
#include <system_error>

#include <fcntl.h>
#include <unistd.h>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
x::errors::Error ProdAPI::check_socket(const char *ifname) {
    const std::string iface(ifname);
    // A BPF device serves one process at a time, so a busy one is skipped.
    for (int i = 0;; ++i) {
        const auto path = "/dev/bpf" + std::to_string(i);
        const int fd = ::open(path.c_str(), O_RDWR);
        if (fd >= 0) {
            ::close(fd);
            return x::errors::NIL;
        }
        if (errno == EBUSY) continue;
        if (errno == EACCES || errno == EPERM)
            return x::errors::Error(
                errors::PERMISSION_DENIED,
                "The Driver does not have permission to open " + path + " for " +
                    iface +
                    ". To give it access, install the ChmodBPF package that comes "
                    "with Wireshark, or give your user read and write access to "
                    "/dev/bpf*. Then log out, log in again, and restart the Driver."
            );
        return x::errors::Error(
            errors::INTERFACE_ERROR,
            "failed to open " + path + " for " + iface + ": " +
                std::generic_category().message(errno)
        );
    }
}
}
