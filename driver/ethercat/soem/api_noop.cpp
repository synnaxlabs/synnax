// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
x::errors::Error ProdAPI::check_socket(const char *) {
    return x::errors::NIL;
}

bool ProdAPI::dewesoft_filter_bound(const char *) {
    return false;
}

std::optional<std::string> ProdAPI::adapter_name(const char *) {
    return std::nullopt;
}

bool ProdAPI::on_usb(const char *) {
    return false;
}
}
