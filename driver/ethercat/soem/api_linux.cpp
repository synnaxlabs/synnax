// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <system_error>

#include <arpa/inet.h>
#include <linux/if_ether.h>
#include <sys/socket.h>
#include <unistd.h>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
x::errors::Error ProdAPI::check_socket(const char *ifname) {
    const int fd = socket(PF_PACKET, SOCK_RAW, htons(ETH_P_ECAT));
    if (fd >= 0) {
        ::close(fd);
        return x::errors::NIL;
    }
    const std::string iface(ifname);
    if (errno == EPERM)
        return x::errors::Error(
            errors::PERMISSION_DENIED,
            "permission denied opening " + iface +
                ": the Driver needs CAP_NET_RAW. Reinstall the Driver service with "
                "'sudo synnax-driver install', or run the Core or the Driver as root"
        );
    return x::errors::Error(
        errors::INTERFACE_ERROR,
        "failed to open a raw socket on " + iface + ": " +
            std::generic_category().message(errno)
    );
}
}
