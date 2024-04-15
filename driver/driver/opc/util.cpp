// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/opc/opc.h"
#include "driver/driver/opc/util.h"

/// @brief maps opc data types to their corresponding Synnax types.
std::map<UA_UInt16, synnax::DataType> data_type_map = {
    {UA_NS0ID_BOOLEAN, synnax::UINT8},
    {UA_NS0ID_SBYTE, synnax::INT8},
    {UA_NS0ID_BYTE, synnax::UINT8},
    {UA_NS0ID_INT16, synnax::INT16},
    {UA_NS0ID_UINT16, synnax::UINT16},
    {UA_NS0ID_INT32, synnax::INT32},
    {UA_NS0ID_UINT32, synnax::UINT32},
    {UA_NS0ID_INT64, synnax::INT64},
    {UA_NS0ID_UINT64, synnax::UINT64},
    {UA_NS0ID_FLOAT, synnax::FLOAT32},
    {UA_NS0ID_DOUBLE, synnax::FLOAT64},
    {UA_NS0ID_STRING, synnax::STRING},
    {UA_NS0ID_DATETIME, synnax::TIMESTAMP},
    {UA_NS0ID_GUID, synnax::UINT128},
};

opc::ClientDeleter getDefaultClientDeleter() {
    return [](UA_Client *client) {
        if (client == nullptr) return;
        UA_Client_disconnect(client);
        UA_Client_delete(client);
    };
}

std::pair<std::shared_ptr<UA_Client>, freighter::Error> opc::connect(
    opc::ConnectionConfig &cfg
) {
    auto client = std::shared_ptr<UA_Client>(
        UA_Client_new(), getDefaultClientDeleter());
    UA_ClientConfig_setDefault(UA_Client_getConfig(client.get()));
    UA_StatusCode status = UA_Client_connect(client.get(), cfg.endpoint.c_str());
    if (cfg.username.empty() && cfg.password.empty())
        status = UA_Client_connect(client.get(), cfg.endpoint.c_str());
    else
        status = UA_Client_connectUsername(
            client.get(),
            cfg.endpoint.c_str(),
            cfg.username.c_str(),
            cfg.password.c_str()
        );
    if (status == UA_STATUSCODE_GOOD) return {std::move(client), freighter::NIL};
    const auto status_name = UA_StatusCode_name(status);
    return {
        std::move(client),
        freighter::Error(freighter::TYPE_UNREACHABLE,
                         "Failed to connect: " + std::string(status_name))
    };
}

