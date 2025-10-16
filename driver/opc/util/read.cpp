// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "glog/logging.h"
#include "mbedtls/error.h"
#include "mbedtls/x509_crt.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"
#include "open62541/common.h"

#include "x/cpp/telem/series.h"
#include "x/cpp/xerrors/errors.h"

#include "driver/opc/util/util.h"

namespace util {
std::pair<telem::Series, xerrors::Error>
simple_read(std::shared_ptr<UA_Client> client, const std::string &node_id) {
    // Parse the node ID string directly
    auto [ua_node_id, parse_err] = parse_node_id(node_id);
    if (parse_err) return {telem::Series(0), parse_err};

    // Read the value from the node
    UA_Variant value;
    UA_Variant_init(&value);

    UA_StatusCode status = UA_Client_readValueAttribute(
        client.get(),
        ua_node_id,
        &value
    );

    // Clean up the node ID
    UA_NodeId_clear(&ua_node_id);

    if (status != UA_STATUSCODE_GOOD) {
        return {telem::Series(0), util::parse_error(status)};
    }

    // Convert the value to a telemetry series
    telem::DataType data_type = util::ua_to_data_type(value.type);
    telem::Series series(data_type, 1);

    // Write the value to the series
    auto [count, write_err] = util::write_to_series(series, value);

    // Clean up the variant
    UA_Variant_clear(&value);

    if (write_err) { return {telem::Series(0), write_err}; }

    return {std::move(series), xerrors::NIL};
}
}
