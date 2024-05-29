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

void customLogger(
    void *logContext,
    UA_LogLevel level,
    UA_LogCategory category,
    const char *msg,
    va_list args) {

    // Buffer to store the formatted message
    char buffer[1024];
    vsnprintf(buffer, sizeof(buffer), msg, args);

    // Decide on the GLog level based on open62541's log level
    switch (level) {
        case UA_LOGLEVEL_TRACE:
        case UA_LOGLEVEL_DEBUG:
        case UA_LOGLEVEL_INFO:
            VLOG(1) << buffer;
        break;
        case UA_LOGLEVEL_WARNING:
            LOG(WARNING) << buffer;
        break;
        case UA_LOGLEVEL_ERROR:
            LOG(ERROR) << buffer;
        break;
        case UA_LOGLEVEL_FATAL:
            LOG(FATAL) << buffer;
        break;
        default:
            LOG(INFO) << buffer; // Default case falls back to INFO level
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
    auto client_config = UA_Client_getConfig(client.get());
    client_config->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    std::string uri = "http://opcfoundation.org/UA/SecurityPolicy#" + cfg.security_policy_uri;
    client_config->securityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    UA_String_clear(&client_config->clientDescription.applicationUri);
    client_config->clientDescription.applicationUri = UA_STRING_ALLOC("urn:open62541.server.application");
    std::cout << "HERE" << std::endl;

    // UA_ByteString certificate = loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_cert.pem");
    UA_ByteString certificate = convertStringToUAByteString(cfg.certificate);
    UA_ByteString privateKey  = convertStringToUAByteString(cfg.p);
    std::cout << "HERE 2" << std::endl;
    // loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_key.pem");

    // size_t trustListSize = 1;
    // UA_STACKARRAY(UA_ByteString, trustList, trustListSize+1);
    // trustList[0] = loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/server_cert.der");
    // trustList[0] = convertStringToUAByteString(cfg.server_cert);

    UA_StatusCode e_err = UA_ClientConfig_setDefaultEncryption(
        client_config, 
        certificate, 
        privateKey,
        NULL,
        0,
        // trustListSize,
        NULL, 
        0
    );

    std::cout << "HERE 3" << std::endl;

    if(e_err != UA_STATUSCODE_GOOD) {
        LOG(ERROR) << "Failed to configure encryption: " << UA_StatusCode_name(e_err);
        const auto status_name = UA_StatusCode_name(e_err);
        return freighter::Error(freighter::TYPE_UNREACHABLE, "Failed to configure encryption: " + std::string(status_name));
    }

    return freighter::NIL;
}


std::pair<std::shared_ptr<UA_Client>, freighter::Error> opc::connect(
    opc::ConnectionConfig &cfg
) {
    // configure a client
    auto client = std::shared_ptr<UA_Client>(UA_Client_new(), getDefaultClientDeleter());
    UA_ClientConfig *config = UA_Client_getConfig(client.get());
    config->logging->log = customLogger;

    configureEncryption(cfg, client);
    std::cout << "HERE 4" << std::endl;
    UA_StatusCode status;
    try { 
        status = UA_Client_connect(client.get(), cfg.endpoint.c_str());
    } catch (const std::exception &e) {
        std::cout << "Exception: " << e.what() << std::endl;
    }
    std::cout << "HERE 5" << std::endl;
    // else
    //     status = UA_Client_connectUsername(
    //         client.get(),
    //         cfg.endpoint.c_str(),
    //         cfg.username.c_str(),
    //         cfg.password.c_str()
    //     );
    if (status == UA_STATUSCODE_GOOD) return {std::move(client), freighter::NIL};
    const auto status_name = UA_StatusCode_name(status);
    return {
        std::move(client),
        freighter::Error(freighter::TYPE_UNREACHABLE,
                         "Failed to connect: " + std::string(status_name))
    };
}