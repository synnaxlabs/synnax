/// std.
#include <map>

/// internal.
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/opcua/opcua.h"
#include "driver/driver/opcua/util.h"

/// @brief maps OPCUA data types to their corresponding Synnax types.
std::map<UA_UInt16, DataType> data_type_map = {
    {UA_NS0ID_BOOLEAN, UINT8},
    {UA_NS0ID_SBYTE, INT8},
    {UA_NS0ID_BYTE, UINT8},
    {UA_NS0ID_INT16, INT16},
    {UA_NS0ID_UINT16, UINT16},
    {UA_NS0ID_INT32, INT32},
    {UA_NS0ID_UINT32, UINT32},
    {UA_NS0ID_INT64, INT64},
    {UA_NS0ID_UINT64, UINT64},
    {UA_NS0ID_FLOAT, FLOAT32},
    {UA_NS0ID_DOUBLE, FLOAT64},
    {UA_NS0ID_STRING, STRING},
    {UA_NS0ID_DATETIME, TIMESTAMP},
    {UA_NS0ID_GUID, UINT128},
};

synnax::Series val_to_series(UA_Variant* val, synnax::DataType dt) {

}

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


