#include <open62541/client_config_default.h>
#include <open62541/client_highlevel.h>
// #include <open62541/plugin/log_stdout.h>
// #include <open62541/plugin/securitypolicy.h>
#include <open62541/server.h>
#include <open62541/server_config_default.h>

#include <stdlib.h>

// #include "common.h"

#define MIN_ARGS 4

int main(int argc, char* argv[]) {
    if(argc < MIN_ARGS) {
        // UA_LOG_FATAL(UA_Log_Stdout, UA_LOGCATEGORY_USERLAND,
        //              "Arguments are missing. The required arguments are "
        //              "<opc.tcp://host:port> "
        //              "<client-certificate.der> <client-private-key.der> "
        //              "[<trustlist1.crl>, ...]");
        return EXIT_SUCCESS;
    }

    const char *endpointUrl = argv[1];

    /* Load certificate and private key */
    // UA_ByteString certificate = loadFile(argv[2]);
    // UA_ByteString privateKey  = loadFile(argv[3]);

      UA_ByteString certificate = UA_STRING_ALLOC("-----BEGIN CERTIFICATE-----\
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

    UA_ByteString privateKey = UA_STRING_ALLOC("-----BEGIN PRIVATE KEY-----\
    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\
    DtU=\
    -----END PRIVATE KEY-----");

    /* Load the trustList. Load revocationList is not supported now */
    size_t trustListSize = 0;
    UA_ByteString *trustList = NULL;
    // if(argc > MIN_ARGS)
    //     trustListSize = (size_t)argc-MIN_ARGS;
    // UA_STACKARRAY(UA_ByteString, trustList, trustListSize+1);
    // for(size_t trustListCount = 0; trustListCount < trustListSize; trustListCount++)
    //     trustList[trustListCount] = loadFile(argv[trustListCount+4]);

    UA_ByteString *revocationList = NULL;
    size_t revocationListSize = 0;

    UA_Client *client = UA_Client_new();
    UA_ClientConfig *cc = UA_Client_getConfig(client);
    cc->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    UA_String_clear(&cc->clientDescription.applicationUri);
    cc->clientDescription.applicationUri = UA_STRING_ALLOC("urn:open62541.server.application");
    UA_StatusCode retval = UA_ClientConfig_setDefaultEncryption(cc, certificate, privateKey,
                                         trustList, trustListSize,
                                         revocationList, revocationListSize);
    if(retval != UA_STATUSCODE_GOOD) {
        // UA_LOG_FATAL(UA_Log_Stdout, UA_LOGCATEGORY_USERLAND,
        //             "Failed to set encryption." );
        UA_Client_delete(client);
        return EXIT_FAILURE;
    }

    UA_ByteString_clear(&certificate);
    UA_ByteString_clear(&privateKey);
    for(size_t deleteCount = 0; deleteCount < trustListSize; deleteCount++) {
        UA_ByteString_clear(&trustList[deleteCount]);
    }

    /* Secure client connect */
    cc->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT; /* require encryption */
    retval = UA_Client_connect(client, endpointUrl);
    if(retval != UA_STATUSCODE_GOOD) {
        UA_Client_delete(client);
        return EXIT_FAILURE;
    }

    // UA_Variant value;
    // UA_Variant_init(&value);

    /* NodeId of the variable holding the current time */
    // const UA_NodeId nodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME);
    // retval = UA_Client_readValueAttribute(client, nodeId, &value);

    // if(retval == UA_STATUSCODE_GOOD &&
    //    UA_Variant_hasScalarType(&value, &UA_TYPES[UA_TYPES_DATETIME])) {
    //     UA_DateTime raw_date  = *(UA_DateTime *) value.data;
    //     UA_DateTimeStruct dts = UA_DateTime_toStruct(raw_date);
    //     UA_LOG_INFO(UA_Log_Stdout, UA_LOGCATEGORY_USERLAND, "date is: %u-%u-%u %u:%u:%u.%03u\n",
    //                 dts.day, dts.month, dts.year, dts.hour, dts.min, dts.sec, dts.milliSec);
    // }

    /* Clean up */
    // UA_Variant_clear(&value);
    UA_Client_delete(client);
    return retval == UA_STATUSCODE_GOOD ? EXIT_SUCCESS : EXIT_FAILURE;
}