// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include "client/cpp/synnax.h"
#include "driver/opc/opc.h"
#include "driver/opc/util.h"

#include "include/open62541/plugin/log_stdout.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client_highlevel.h"


#include "glog/logging.h"

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

void customLogger(void *logContext, UA_LogLevel level, UA_LogCategory category, const char *msg, va_list args) {
    std::string prefix = *static_cast<std::string*>(logContext);

    // Buffer to store the formatted message
    char buffer[1024];
    vsnprintf(buffer, sizeof(buffer), msg, args);

    // Decide on the GLog level based on open62541's log level
    switch (level) {
        case UA_LOGLEVEL_TRACE:
        case UA_LOGLEVEL_DEBUG:
        case UA_LOGLEVEL_INFO:
            VLOG(1) << prefix << buffer;
        break;
        case UA_LOGLEVEL_WARNING:
            LOG(WARNING) << prefix << buffer;
        break;
        case UA_LOGLEVEL_ERROR:
            LOG(ERROR) << prefix << buffer;
        break;
        case UA_LOGLEVEL_FATAL:
            LOG(FATAL) << prefix << buffer;
        break;
        default:
            LOG(INFO) << prefix << buffer; // Default case falls back to INFO level
    }
}


UA_ByteString loadFile(const char *const path) {
    UA_ByteString fileContents = UA_STRING_NULL;

    /* Open the file */
    FILE *fp = fopen(path, "rb");
    if(!fp) {
        errno = 0; /* We read errno also from the tcp layer... */
        return fileContents;
    }

    /* Get the file length, allocate the data and read */
    fseek(fp, 0, SEEK_END);
    fileContents.length = (size_t)ftell(fp);
    fileContents.data = (UA_Byte *)UA_malloc(fileContents.length * sizeof(UA_Byte));
    if(fileContents.data) {
        fseek(fp, 0, SEEK_SET);
        size_t read = fread(fileContents.data, sizeof(UA_Byte), fileContents.length, fp);
        if(read != fileContents.length)
            UA_ByteString_clear(&fileContents);
    } else {
        fileContents.length = 0;
    }
    fclose(fp);

    return fileContents;
}

UA_ByteString convertStringToUAByteString(const std::string &certString) {
    UA_ByteString byteString;
    byteString.length = certString.size();
    byteString.data = (UA_Byte*)UA_malloc(byteString.length * sizeof(UA_Byte));
    if(byteString.data) {
        memcpy(byteString.data, certString.data(), byteString.length);
    }
    return byteString;
}


freighter::Error configureEncryption(opc::ConnectionConfig &cfg, std::shared_ptr<UA_Client> client) {
    // std::cout << cfg.security_policy_uri << std::endl;
    // std::cout << cfg.certificate << std::endl;
    // std::cout << cfg.p << std::endl;
    // std::cout << cfg.server_cert << std::endl;
    
    if(cfg.security_policy_uri =="None") {
        LOG(ERROR) << "[opc.scanner] Missing encryption configuration";
        return freighter::NIL;
    }
    auto client_config = UA_Client_getConfig(client.get());
    client_config->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    
    std::string uri = "http://opcfoundation.org/UA/SecurityPolicy#" + cfg.security_policy_uri;
    client_config->securityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    UA_String_clear(&client_config->clientDescription.applicationUri);
    client_config->clientDescription.applicationUri = UA_STRING_ALLOC("urn:open62541.server.application");

    // TODO: change to not take path as this won't work when certs/keys are not stored on local machine
    UA_ByteString certificate = loadFile(cfg.certificate.c_str());
    UA_ByteString privateKey  = loadFile(cfg.p.c_str());

    size_t trustListSize = 0;
    UA_STACKARRAY(UA_ByteString, trustList, trustListSize+1);
    if (!cfg.server_cert.empty()) 
        trustList[0] = loadFile(cfg.server_cert.c_str());
    UA_StatusCode e_err = UA_ClientConfig_setDefaultEncryption(
        client_config, 
        certificate, 
        privateKey,
        trustList,
        trustListSize,
        NULL, 
        0
    );

    if(e_err != UA_STATUSCODE_GOOD) {
        LOG(ERROR) << "[opc.scanner] Failed to configure encryption: " << UA_StatusCode_name(e_err);
        const auto status_name = UA_StatusCode_name(e_err);
        return freighter::Error(freighter::TYPE_UNREACHABLE, "Failed to configure encryption: " + std::string(status_name));
    }

    return freighter::NIL;
}


std::pair<std::shared_ptr<UA_Client>, freighter::Error> opc::connect(
    opc::ConnectionConfig &cfg,
    std::string log_prefix
) {
    // configure a client
    auto client = std::shared_ptr<UA_Client>(UA_Client_new(), getDefaultClientDeleter());
    UA_ClientConfig *config = UA_Client_getConfig(client.get());
    config->logging->log = customLogger;
    config->logging->context = &log_prefix;

    LOG(INFO) << "[opc.scanner] Cconfiguring encryption";
    configureEncryption(cfg, client);
    UA_StatusCode status;
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
    LOG(ERROR) << "[opc.scanner] Failed to connect: " << std::string(status_name);
    return {
        std::move(client),
        freighter::Error(freighter::TYPE_UNREACHABLE, "Failed to connect: " + std::string(status_name))
    };
}