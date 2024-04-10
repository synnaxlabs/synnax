#pragma once

#include "opcua.h"
#include "client/cpp/synnax/synnax.h"
#include "include/open62541/types.h"
#include "include/open62541/nodeids.h"
#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client.h"
#include "include/open62541/statuscodes.h"

namespace opcua {
std::pair<UA_Client *, bool> connect(
    opcua::ConnectionConfig &cfg,
    synnax::Task &task,
    std::shared_ptr<task::Context> ctx
);

inline synnax::Series val_to_series(UA_Variant *val, synnax::DataType dt) {
    if (val->type == &UA_TYPES[UA_TYPES_FLOAT]) {
        const auto value = *static_cast<UA_Float *>(val->data);
        if (dt == synnax::FLOAT32) return Series(value);
        if (dt == synnax::FLOAT64) return Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double *>(val->data);
        // TODO - warn on potential precision drop here
        if (dt == synnax::FLOAT32) return Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64) return Series(value);
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32 *>(val->data);
        if (dt == synnax::INT32) return Series(value);
        if (dt == synnax::INT64) return Series(static_cast<int64_t>(value));
        if (dt == synnax::UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64) return Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64 *>(val->data);
        if (dt == synnax::INT32) return Series(static_cast<int32_t>(value));
        if (dt == synnax::INT64) return Series(value);
        if (dt == synnax::UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64) return Series(static_cast<uint64_t>(value));
    }
    return Series(1);
}

inline synnax::DataType variant_data_type(UA_Variant &val) {
    if (val.type == &UA_TYPES[UA_TYPES_FLOAT]) return synnax::FLOAT32;
    if (val.type == &UA_TYPES[UA_TYPES_DOUBLE]) return synnax::FLOAT64;
    if (val.type == &UA_TYPES[UA_TYPES_INT16]) return synnax::INT16;
    if (val.type == &UA_TYPES[UA_TYPES_INT32]) return synnax::INT32;
    if (val.type == &UA_TYPES[UA_TYPES_INT64]) return synnax::INT64;
    if (val.type == &UA_TYPES[UA_TYPES_UINT16]) return synnax::UINT16;
    if (val.type == &UA_TYPES[UA_TYPES_UINT32]) return synnax::UINT32;
    if (val.type == &UA_TYPES[UA_TYPES_UINT64]) return synnax::UINT64;
    if (val.type == &UA_TYPES[UA_TYPES_STRING]) return synnax::STRING;
    if (val.type == &UA_TYPES[UA_TYPES_DATETIME]) return synnax::TIMESTAMP;
    if (val.type == &UA_TYPES[UA_TYPES_GUID]) return synnax::UINT128;
    return synnax::DATA_TYPE_UNKNOWN;
}

inline std::pair<std::shared_ptr<UA_Client>, freighter::Error> connect(
    const opcua::ConnectionConfig &cfg
) {
    std::shared_ptr<UA_Client> client(UA_Client_new(), UA_Client_delete);
    UA_ClientConfig_setDefault(UA_Client_getConfig(client.get()));
    UA_StatusCode status;
    if (cfg.username.empty() && cfg.password.empty())
        status = UA_Client_connect(client.get(), cfg.endpoint.c_str());
    else
        status = UA_Client_connectUsername(client.get(), cfg.endpoint.c_str(),
                                           cfg.username.c_str(), cfg.password.c_str());
    if (status != UA_STATUSCODE_GOOD) {
        const auto status_name = UA_StatusCode_name(status);
        return {
            nullptr,
            freighter::Error(freighter::TYPE_UNREACHABLE,
                             "Failed to connect: " + std::string(status_name))
        };
    }
    return {client, freighter::NIL};
}
}
