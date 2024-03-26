#include <memory>
#include "nlohmann/json.hpp"
#include "scanner.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"
#include "open62541/client_subscriptions.h"
#include "open62541/plugin/log_stdout.h"

using json = nlohmann::json;

using namespace opcua;

void field_err(const std::string &path, const std::string &message, json &err) {
    json field;
    field["path"] = path;
    field["message"] = message;
    err["errors"].push_back(field);
}

json find(json j, const std::string& key, json &err, bool &ok) {
    auto iter = j.find(key);
    if (j.find(key) == j.end()) field_err(key, "missing", err);
    return iter.value();
}

scannerScanCmd::scannerScanCmd(const json &cmd, json& err, bool &ok) {
    auto ep_val = find(cmd, "endpoint", err, ok);
    auto user_val = find(cmd, "username", err, ok);
    auto pwd_val = find(cmd, "password", err, ok);
    if (!ok) {
        return;
    }
    endpoint = ep_val.get<std::string>();
    username = user_val.get<std::string>();
    password = pwd_val.get<std::string>();
}

scanner::scanner(const scannerConfig& config) : client(config.client) {}

void scanner::exec(const std::string type, const json &cmd, json &err, bool &ok) {
    if (type == SCAN_CMD_TYPE) {
        scannerScanCmd scanCmd(cmd, err, ok);
        if (!ok) return;
    }
}

void scanner::scan(const opcua::scannerScanCmd &cmd, json &err) {
    std::unique_ptr<UA_Client, void(*)(UA_Client*)> ua_client(UA_Client_new(), UA_Client_delete);

    UA_ClientConfig_setDefault(UA_Client_getConfig(ua_client.get()));
    UA_StatusCode retval;
    UA_ClientConfig *config = UA_Client_getConfig(ua_client.get());
    if (cmd.username.empty() || cmd.password.empty()) {
        retval = UA_Client_connectUsername(
                ua_client.get(),
                cmd.endpoint.c_str(),
                cmd.username.c_str(),
                cmd.password.c_str()
        );
    } else {
        retval = UA_Client_connect(ua_client.get(), cmd.endpoint.c_str());
    }
    if (retval != UA_STATUSCODE_GOOD) {
        field_err("endpoint", "failed to connect", err);
        return;
    }

    UA_BrowseRequest bReq;
    UA_BrowseRequest_init(&bReq);
    bReq.requestedMaxReferencesPerNode = 0;
    bReq.nodesToBrowse = UA_BrowseDescription_new();
    bReq.nodesToBrowseSize = 1;
    bReq.nodesToBrowse[0].nodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
    bReq.nodesToBrowse[0].resultMask = UA_BROWSERESULTMASK_ALL; // return everything
    UA_BrowseResponse bResp = UA_Client_Service_browse(ua_client.get(), bReq);

    json res;
    for (size_t i = 0; i < bResp.resultsSize; ++i) {
        for (size_t j = 0; j < bResp.results[i].referencesSize; ++j) {
            UA_ReferenceDescription *ref = &(bResp.results[i].references[j]);
            if (ref->nodeId.nodeId.identifierType == UA_NODEIDTYPE_NUMERIC) {
                res["nodes"].push_back(ref->nodeId.nodeId.identifier.numeric);
            }
        }
    }

    UA_BrowseRequest_clear(&bReq);
    UA_BrowseResponse_clear(&bResp);
}