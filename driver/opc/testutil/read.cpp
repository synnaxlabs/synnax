// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/errors/errors.h"

#include "driver/opc/errors/errors.h"
#include "driver/opc/telem/telem.h"
#include "driver/opc/testutil/testutil.h"
#include "driver/opc/types/types.h"

namespace driver::opc::testutil {
std::pair<::x::telem::Series, x::errors::Error>
simple_read(std::shared_ptr<UA_Client> client, const std::string &node_id) {
    // Parse the node ID string - returns RAII-wrapped NodeId
    auto [node_id_wrapper, parse_err] = driver::opc::NodeId::parse(node_id);
    if (parse_err) return {::x::telem::Series(0), parse_err};

    // Read the value from the node - RAII wrapper handles cleanup
    driver::opc::Variant value;

    // Implicit conversion from NodeId to const UA_NodeId&
    UA_StatusCode status = UA_Client_readValueAttribute(
        client.get(),
        node_id_wrapper, // Implicit conversion
        value.ptr()
    );

    if (status != UA_STATUSCODE_GOOD) {
        return {::x::telem::Series(0), driver::opc::errors::parse(status)};
    }

    // Convert the value to a telemetry series
    ::x::telem::DataType data_type = driver::opc::telem::ua_to_data_type(value.get().type);
    ::x::telem::Series series(data_type, 1);

    // Write the value to the series
    auto [count, write_err] = driver::opc::telem::write_to_series(series, value.get());

    if (write_err) { return {::x::telem::Series(0), write_err}; }

    return {std::move(series), x::errors::NIL};
}
}
