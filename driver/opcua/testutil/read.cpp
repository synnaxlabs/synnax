// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "open62541/client_highlevel.h"
#include "open62541/common.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"

#include "driver/opcua/errors/errors.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/testutil/testutil.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::testutil {
std::pair<::x::telem::Series, x::errors::Error>
simple_read(const std::shared_ptr<UA_Client> &client, const std::string &node_id) {
    // Parse the node ID string - returns RAII-wrapped NodeId
    auto [node_id_wrapper, parse_err] = types::NodeId::parse(node_id);
    if (parse_err) return {::x::telem::Series(0), parse_err};

    // Read the value from the node - RAII wrapper handles cleanup
    types::Variant value;

    // Implicit conversion from NodeId to const UA_NodeId&
    UA_StatusCode status = UA_Client_readValueAttribute(
        client.get(),
        node_id_wrapper, // Implicit conversion
        value.ptr()
    );

    if (status != UA_STATUSCODE_GOOD) {
        return {::x::telem::Series(0), opcua::errors::parse(status)};
    }

    const auto data_type = telem::ua_to_data_type(value.get().type);
    auto [sample, sample_err] = telem::to_sample(value.get(), data_type);
    if (sample_err) return {::x::telem::Series(0), sample_err};
    return {telem::to_series(data_type, {sample}), x::errors::NIL};
}
}
