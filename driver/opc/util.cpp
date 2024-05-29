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

<<<<<<< Updated upstream
    // UA_ByteString certificate = loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_cert.pem");
    UA_ByteString certificate = convertStringToUAByteString(cfg.certificate);
    UA_ByteString privateKey  = convertStringToUAByteString(cfg.p);
    // loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_key.pem");

    size_t trustListSize = 1;
    UA_STACKARRAY(UA_ByteString, trustList, trustListSize+1);
    // trustList[0] = loadFile("/Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/server_cert.der");
    trustList[0] = convertStringToUAByteString(cfg.server_cert);

    UA_StatusCode e_err = UA_ClientConfig_setDefaultEncryption(
        client_config, 
        certificate, 
        privateKey,
        trustList,
        trustListSize,
        NULL, 
        0
    );
=======
    LOG(INFO) << "Configuring encryption for client: " << cfg.endpoint;
    LOG(INFO) << "Security policy: " << cfg.security_policy_uri;
    std::string uri;
    if(cfg.security_policy_uri.empty()) return freighter::NIL;
    else{
        // c* that concatenates the security policy with the uri
        uri = "http://opcfoundation.org/UA/SecurityPolicy#"+ cfg.security_policy_uri;
        client_config->securityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    }

    LOG(INFO) << "security policy uri: " << uri;

    // print key and certificate
    LOG(INFO) << "Certificate: " << cfg.certificate;
    LOG(INFO) << "Private key: " << cfg.p;

    // auto cert = stringToUAByteString(cfg.certificate);
    // auto p = stringToUAByteString(cfg.p);

    auto cert = UA_STRING_ALLOC("-----BEGIN CERTIFICATE-----\
    MIIFHjCCBAagAwIBAgIUSHiDuM+ntc8w2q8WV0B9jZgeeGswDQYJKoZIhvcNAQEL\
    BQAwgYQxRDBCBgNVBAMMO3VybjpFbWlsaWFub3MtTWFjQm9vay1Qcm8tMi5sb2Nh\
    bDpmb29iYXI6bXlzZWxmc2lnbmVkY2xpZW50MQswCQYDVQQGEwJDTjEPMA0GA1UE\
    CAwGQVN0YXRlMQwwCgYDVQQHDANGb28xEDAOBgNVBAoMB0JhciBMdGQwHhcNMjQw\
    NTI4MjMxMDE1WhcNMjUwNTI4MjMxMDE1WjCBhDFEMEIGA1UEAww7dXJuOkVtaWxp\
    YW5vcy1NYWNCb29rLVByby0yLmxvY2FsOmZvb2JhcjpteXNlbGZzaWduZWRjbGll\
    bnQxCzAJBgNVBAYTAkNOMQ8wDQYDVQQIDAZBU3RhdGUxDDAKBgNVBAcMA0ZvbzEQ\
    MA4GA1UECgwHQmFyIEx0ZDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB\
    AOO2YUG3dooZ6CmXX6z9oXy0q7MLby3sIIf/mPvqIwxPmdP7Wi4Ta9ze5cX+Vfrv\
    5rTgRV0hCsChVFJGRd++I1j4qpNe8emM8rwWWd9diUwy1dz4VdJz0Qfmi0UxUIht\
    za4XTAenZLev3JWZxf44fnFOkQtHRF/MEfCkO35gDtba7f1TSopmbNDsYxq/voEf\
    ETrc4DoMw5rrmUwK/Mmzmvmr5MJ7ghKjs6gYbocAyenzh4hD5aJyluYaV7TuQgGd\
    F4DvQOEn1GdKCbTZuVGwq0Gua9Zbd7lDe74+gP+TduLHonv6Tarko29rlvu4haH/\
    6hv/+qhrRyFn2RGPGP+7/T8CAwEAAaOCAYQwggGAMB0GA1UdDgQWBBSAKEsiy8aL\
    fCQbRle/agQ0bwmgzTCBxAYDVR0jBIG8MIG5gBSAKEsiy8aLfCQbRle/agQ0bwmg\
    zaGBiqSBhzCBhDFEMEIGA1UEAww7dXJuOkVtaWxpYW5vcy1NYWNCb29rLVByby0y\
    LmxvY2FsOmZvb2JhcjpteXNlbGZzaWduZWRjbGllbnQxCzAJBgNVBAYTAkNOMQ8w\
    DQYDVQQIDAZBU3RhdGUxDDAKBgNVBAcMA0ZvbzEQMA4GA1UECgwHQmFyIEx0ZIIU\
    SHiDuM+ntc8w2q8WV0B9jZgeeGswZQYDVR0RBF4wXIY7dXJuOkVtaWxpYW5vcy1N\
    YWNCb29rLVByby0yLmxvY2FsOmZvb2JhcjpteXNlbGZzaWduZWRjbGllbnSCHUVt\
    aWxpYW5vcy1NYWNCb29rLVByby0yLmxvY2FsMA8GA1UdEwQIMAYBAf8CAQAwCwYD\
    VR0PBAQDAgL0MBMGA1UdJQQMMAoGCCsGAQUFBwMCMA0GCSqGSIb3DQEBCwUAA4IB\
    AQCBhKsnh4a0+/AkFk3Wbof0fCz1kwfWhsm+vrOoDj5io78n3xU6fUUahhjYEiAI\
    HwRrSq2v/QLeqGm0cuCr27bWLbvuftxibLmfQRga2eSe0dtxotYNOU1zeysmeHGL\
    GGf+IDMrzbh5fJBYQ+qqW7yr93ymtUFglpqAoeD57teoDGWDWrVYjQy4KStc8gpF\
    s1AVZPDbF+64B1ToBrCweF+5sDHQUwm2qpKNgKfoenufVUYtIeZXXOqgzDk8+rJs\
    qsD3wahFk+WN+UCmi2HfamTFAUaQXex/8+Ae8a8jOU25WPCAv0Ey3fMaI1t4RtDL\
    kkQkfv++PNu5IWhQzNCx0Bw9-----END CERTIFICATE-----");

    auto p = UA_STRING_ALLOC("-----BEGIN PRIVATE KEY-----\
    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\
    DtU=\
    -----END PRIVATE KEY-----");

    std::vector<UA_ByteString> trusted_certs;
    size_t num_trusted_certs = cfg.trusted_certificates.size();
    for(auto &trusted_cert : cfg.trusted_certificates) {
        trusted_certs.push_back(stringToUAByteString(trusted_cert));
    }

    LOG(INFO) << "Setting up encryption for client: " << cfg.endpoint;
    UA_StatusCode e_err = UA_ClientConfig_setDefaultEncryption(client_config, cert, p, NULL, 0, NULL, 0);
>>>>>>> Stashed changes

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

<<<<<<< Updated upstream
=======
    // setup encryption support if applicable
>>>>>>> Stashed changes
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
    return {
        std::move(client),
        freighter::Error(freighter::TYPE_UNREACHABLE,
                         "Failed to connect: " + std::string(status_name))
    };
}
