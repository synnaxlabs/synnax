// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* This work is licensed under a Creative Commons CCZero 1.0 Universal License.
 * See http://creativecommons.org/publicdomain/zero/1.0/ for more information. */

#include <vector>

#include <errno.h>
#include <open62541/client_config_default.h>
#include <open62541/client_highlevel.h>
#include <open62541/plugin/log_stdout.h>
#include <open62541/plugin/securitypolicy.h>
#include <open62541/server.h>
#include <open62541/server_config_default.h>
#include <open62541/types.h>
#include <stdio.h>
#include <stdlib.h>

/* sleep_ms */
#ifdef _WIN32
#include <synchapi.h>
#define sleep_ms(ms) Sleep(ms)
#else

#include <unistd.h>

#define sleep_ms(ms) usleep(ms * 1000)
#endif

/* loadFile parses the certificate file.
 *
 * @param  path               specifies the file name given in argv[]
 * @return Returns the file content after parsing */
static UA_INLINE UA_ByteString

load_file(const char *const path) {
    UA_ByteString fileContents = UA_STRING_NULL;

    /* Open the file */
    FILE *fp = fopen(path, "rb");
    if (!fp) {
        errno = 0; /* We read errno also from the tcp layer... */
        return fileContents;
    }

    /* Get the file length, allocate the data and read */
    fseek(fp, 0, SEEK_END);
    fileContents.length = (size_t) ftell(fp);
    fileContents.data = (UA_Byte *) UA_malloc(fileContents.length * sizeof(UA_Byte));
    if (fileContents.data) {
        fseek(fp, 0, SEEK_SET);
        size_t
            read = fread(fileContents.data, sizeof(UA_Byte), fileContents.length, fp);
        if (read != fileContents.length) UA_ByteString_clear(&fileContents);
    } else {
        fileContents.length = 0;
    }
    fclose(fp);

    return fileContents;
}

#define MIN_ARGS 4

int main(int argc, char *argv[]) {
    if (argc < MIN_ARGS) {
        UA_LOG_FATAL(
            UA_Log_Stdout,
            UA_LOGCATEGORY_USERLAND,
            "Arguments are missing. The required arguments are "
            "<opc.tcp://host:port> "
            "<client-certificate.der> <client-private-key.der> "
            "[<trustlist1.crl>, ...]"
        );
        return EXIT_SUCCESS;
    }

    const char *endpointUrl = argv[1];

    /* Load certificate and private key */
    UA_ByteString certificate = load_file(argv[2]);
    UA_ByteString privateKey = load_file(argv[3]);

    /* Load the trustList. Load revocationList is not supported now */
    size_t trustListSize = 0;
    if (argc > MIN_ARGS) trustListSize = (size_t) argc - MIN_ARGS;
    std::vector<UA_ByteString> trustList(trustListSize + 1);
    for (size_t trustListCount = 0; trustListCount < trustListSize; trustListCount++)
        trustList[trustListCount] = load_file(argv[trustListCount + 4]);

    UA_ByteString *revocationList = NULL;
    size_t revocationListSize = 0;

    UA_Client *client = UA_Client_new();
    UA_ClientConfig *cc = UA_Client_getConfig(client);
    cc->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    UA_String_clear(&cc->clientDescription.applicationUri);
    cc->clientDescription.applicationUri = UA_STRING_ALLOC(
        "urn:open62541.server.application"
    );
    UA_StatusCode retval = UA_ClientConfig_setDefaultEncryption(
        cc,
        certificate,
        privateKey,
        trustList.data(),
        trustListSize,
        revocationList,
        revocationListSize
    );
    if (retval != UA_STATUSCODE_GOOD) {
        UA_LOG_FATAL(
            UA_Log_Stdout,
            UA_LOGCATEGORY_USERLAND,
            "Failed to set encryption."
        );
        UA_Client_delete(client);
        return EXIT_FAILURE;
    }

    UA_ByteString_clear(&certificate);
    UA_ByteString_clear(&privateKey);
    for (size_t deleteCount = 0; deleteCount < trustListSize; deleteCount++) {
        UA_ByteString_clear(&trustList[deleteCount]);
    }

    /* Secure client connect */
    cc->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT; /* require encryption */
    retval = UA_Client_connect(client, endpointUrl);
    if (retval != UA_STATUSCODE_GOOD) {
        UA_Client_delete(client);
        return EXIT_FAILURE;
    }

    UA_Variant value;
    UA_Variant_init(&value);

    /* NodeId of the variable holding the current time */
    const UA_NodeId nodeId = UA_NODEID_NUMERIC(
        0,
        UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME
    );
    retval = UA_Client_readValueAttribute(client, nodeId, &value);

    if (retval == UA_STATUSCODE_GOOD &&
        UA_Variant_hasScalarType(&value, &UA_TYPES[UA_TYPES_DATETIME])) {
        UA_DateTime raw_date = *(UA_DateTime *) value.data;
        UA_DateTimeStruct dts = UA_DateTime_toStruct(raw_date);
        UA_LOG_INFO(
            UA_Log_Stdout,
            UA_LOGCATEGORY_USERLAND,
            "date is: %u-%u-%u %u:%u:%u.%03u\n",
            dts.day,
            dts.month,
            dts.year,
            dts.hour,
            dts.min,
            dts.sec,
            dts.milliSec
        );
    }

    /* Clean up */
    UA_Variant_clear(&value);
    UA_Client_delete(client);
    return retval == UA_STATUSCODE_GOOD ? EXIT_SUCCESS : EXIT_FAILURE;
}
