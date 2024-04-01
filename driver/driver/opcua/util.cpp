#include <map>
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/opcua/opcua.h"
#include "driver/driver/opcua/util.h"

/// @brief maps OPCUA data types to their corresponding Synnax types.
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

std::pair<UA_Client*, bool> opcua::connect(
    opcua::ConnectionConfig& cfg,
    synnax::Task &task,
    std::shared_ptr<task::Context> ctx
) {
    UA_Client* client = UA_Client_new();
    UA_ClientConfig_setDefault(UA_Client_getConfig(client));
    UA_StatusCode status = UA_Client_connect(client, cfg.endpoint.c_str());
    if (cfg.username.empty() && cfg.password.empty()) {
        status = UA_Client_connect(client, cfg.endpoint.c_str());
    } else {
        status = UA_Client_connectUsername(
            client,
            cfg.endpoint.c_str(),
            cfg.username.c_str(),
            cfg.password.c_str()
        );
    }
    // UA_Client_disconnect(client);
    // UA_Client_delete(client);
    if (status != UA_STATUSCODE_GOOD) {
        ctx->setState({
            .task = task.key,
            .type = "failed",
            .details = json {
                {"message", "Failed to connect to the OPC UA server."}
            }
        });
        return {nullptr, false};
    }
    return {client, true};
}


