// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// internal
#include "driver/rack/rack.h"
#include "x/cpp/xenv/xenv.h"

const std::string ENV_PREFIX = "SYNNAX_DRIVER_";


xerrors::Error rack::Config::load_env() {
    this->connection.host = xenv::load(
        ENV_PREFIX + "HOST",
        this->connection.host
    );
    this->connection.port = xenv::load(
        ENV_PREFIX + "PORT",
        this->connection.port
    );
    this->connection.username = xenv::load(
        ENV_PREFIX + "USERNAME",
        this->connection.username
    );
    this->connection.password = xenv::load(
        ENV_PREFIX + "PASSWORD",
        this->connection.password
    );
    this->connection.ca_cert_file = xenv::load(
        ENV_PREFIX + "CA_CERT_FILE",
        this->connection.ca_cert_file
    );
    this->connection.client_cert_file = xenv::load(
        ENV_PREFIX + "CLIENT_CERT_FILE",
        this->connection.client_cert_file
    );
    this->connection.client_key_file = xenv::load(
        ENV_PREFIX + "CLIENT_KEY_FILE",
        this->connection.client_key_file
    );
    return xerrors::NIL;
}
