//
// Created by Emiliano Bonilla on 3/29/24.
//

#pragma once

#include "opcua.h"
#include "client/cpp/synnax/synnax.h"

/// external.
#include "open62541/types.h"
#include "open62541/nodeids.h"
#include "open62541/types.h"
#include "open62541/types_generated.h"
#include "open62541/client_config_default.h"
#include "open62541/client.h"
#include "open62541/statuscodes.h"

namespace opcua {
std::pair<UA_Client *, bool> connect(
    opcua::ConnectionConfig& cfg,
    synnax::Task& task,
    std::shared_ptr<task::Context> ctx
);

inline synnax::Series val_to_series(UA_Variant* val, synnax::DataType dt) {
    if (val->type == &UA_TYPES[UA_TYPES_FLOAT]) {
        const auto value = *static_cast<UA_Float*>(val->data);
        if (dt == FLOAT32) return Series(value);
        if (dt == FLOAT64) return Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double*>(val->data);
        // TODO - warn on potential precision drop here
        if (dt == FLOAT32) return Series(static_cast<float>(value));
        if (dt == FLOAT64) return Series(value);
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32*>(val->data);
        if (dt == INT32) return Series(value);
        if (dt == INT64) return Series(static_cast<int64_t>(value));
        if (dt == UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == UINT64) return Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64*>(val->data);
        if (dt == INT32) return Series(static_cast<int32_t>(value));
        if (dt == INT64) return Series(value);
        if (dt == UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == UINT64) return Series(static_cast<uint64_t>(value));
    }
    return Series(1);
}
}
