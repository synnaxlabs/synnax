// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include <errno.h>
#include <open62541/client_highlevel.h>
#include <open62541/plugin/accesscontrol_default.h>
#include <open62541/plugin/create_certificate.h>
#include <open62541/plugin/log_stdout.h>
#include <open62541/plugin/securitypolicy.h>
#include <open62541/server.h>
#include <open62541/server_config_default.h>
#include <open62541/types.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
// #include <unistd.h> // For getcwd
#include <iostream>

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
        // exit with errno
        errno = 1; /* We read errno also from the tcp layer... */
        exit(errno);
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

UA_Boolean running = true;

static void stopHandler(int sig) {
    UA_LOG_INFO(UA_Log_Stdout, UA_LOGCATEGORY_USERLAND, "received ctrl-c");
    running = false;
}

static UA_Boolean allowAddNode(
    UA_Server *server,
    UA_AccessControl *ac,
    const UA_NodeId *sessionId,
    void *sessionContext,
    const UA_AddNodesItem *item
) {
    printf("Called allowAddNode\n");
    return UA_TRUE;
}

static UA_Boolean allowAddReference(
    UA_Server *server,
    UA_AccessControl *ac,
    const UA_NodeId *sessionId,
    void *sessionContext,
    const UA_AddReferencesItem *item
) {
    printf("Called allowAddReference\n");
    return UA_TRUE;
}

static UA_Boolean allowDeleteNode(
    UA_Server *server,
    UA_AccessControl *ac,
    const UA_NodeId *sessionId,
    void *sessionContext,
    const UA_DeleteNodesItem *item
) {
    printf("Called allowDeleteNode\n");
    return UA_FALSE; // Do not allow deletion from client
}

static UA_Boolean allowDeleteReference(
    UA_Server *server,
    UA_AccessControl *ac,
    const UA_NodeId *sessionId,
    void *sessionContext,
    const UA_DeleteReferencesItem *item
) {
    printf("Called allowDeleteReference\n");
    return UA_TRUE;
}

static UA_UsernamePasswordLogin userNamePW[2] = {
    {UA_STRING_STATIC("peter"), UA_STRING_STATIC("peter123")},
    {UA_STRING_STATIC("paula"), UA_STRING_STATIC("paula123")}
};

static void setCustomAccessControl(UA_ServerConfig *config) {
    /* Use the default AccessControl plugin as the starting point */
    UA_Boolean allowAnonymous = false;
    UA_String encryptionPolicy = UA_STRING_STATIC(
        "http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256"
    );
    // config->securityPolicies[config->securityPoliciesSize-1].policyUri;
    config->accessControl.clear(&config->accessControl);
    UA_AccessControl_default(config, allowAnonymous, &encryptionPolicy, 2, userNamePW);

    /* Override accessControl functions for nodeManagement */
    config->accessControl.allowAddNode = allowAddNode;
    config->accessControl.allowAddReference = allowAddReference;
    config->accessControl.allowDeleteNode = allowDeleteNode;
    config->accessControl.allowDeleteReference = allowDeleteReference;
}

int main(int argc, char *argv[]) {
    signal(SIGINT, stopHandler);
    signal(SIGTERM, stopHandler);
    UA_ByteString certificate = UA_BYTESTRING_NULL;
    UA_ByteString privateKey = UA_BYTESTRING_NULL;
    if (argc >= 3) {
        /* Load certificate and private key */
        certificate = load_file(argv[1]);
        privateKey = load_file(argv[2]);
    }
    /* Load the trustlist */
    size_t trustListSize = 0;
    if (argc > 3) trustListSize = (size_t) argc - 3;
    std::vector<UA_ByteString> trustList(trustListSize + 1);
    for (size_t i = 0; i < trustListSize; i++)
        trustList[i] = load_file(argv[i + 3]);

    /* Loading of an issuer list, not used in this application */
    size_t issuerListSize = 0;
    UA_ByteString *issuerList = NULL;

    /* Loading of a revocation list currently unsupported */
    UA_ByteString *revocationList = NULL;
    size_t revocationListSize = 0;

    UA_Server *server = UA_Server_new();
    UA_ServerConfig *config = UA_Server_getConfig(server);
    // config->allowNonePolicyPassword = true;

    UA_StatusCode retval = UA_ServerConfig_setDefaultWithSecurityPolicies(
        config,
        4841,
        &certificate,
        &privateKey,
        trustList.data(),
        trustListSize,
        issuerList,
        issuerListSize,
        revocationList,
        revocationListSize
    );
    if (retval != UA_STATUSCODE_GOOD) {
        UA_LOG_ERROR(
            UA_Log_Stdout,
            UA_LOGCATEGORY_SERVER,
            "Error setting up the server with security policies"
        );
    }
    UA_VariableAttributes attr = UA_VariableAttributes_default;
    UA_Int32 myInteger = 42;
    UA_Variant_setScalarCopy(&attr.value, &myInteger, &UA_TYPES[UA_TYPES_INT32]);
    attr.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
    attr.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
    UA_NodeId myIntegerNodeId = UA_NODEID_STRING_ALLOC(1, "the.answer");
    UA_QualifiedName myIntegerName = UA_QUALIFIEDNAME_ALLOC(1, "the answer");
    UA_NodeId parentNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
    UA_NodeId parentReferenceNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_ORGANIZES);
    UA_Server_addVariableNode(
        server,
        myIntegerNodeId,
        parentNodeId,
        parentReferenceNodeId,
        myIntegerName,
        UA_NODEID_NULL,
        attr,
        NULL,
        NULL
    );

    // // add another variable node to the adresspace
    UA_VariableAttributes attr2 = UA_VariableAttributes_default;
    UA_Double myDouble = 3.14;
    UA_Variant_setScalarCopy(&attr2.value, &myDouble, &UA_TYPES[UA_TYPES_DOUBLE]);
    attr2.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 2");
    attr2.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 2");
    attr2.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;
    UA_NodeId myDoubleNodeId = UA_NODEID_STRING_ALLOC(1, "the.answer2");
    UA_QualifiedName myDoubleName = UA_QUALIFIEDNAME_ALLOC(1, "the answer 2");
    UA_Server_addVariableNode(
        server,
        myDoubleNodeId,
        parentNodeId,
        parentReferenceNodeId,
        myDoubleName,
        UA_NODEID_NULL,
        attr2,
        NULL,
        NULL
    );

    // add a uint8 variable node to the adresspace
    UA_VariableAttributes attr3 = UA_VariableAttributes_default;
    UA_Byte myUInt8 = 0;
    UA_Variant_setScalarCopy(&attr3.value, &myUInt8, &UA_TYPES[UA_TYPES_BYTE]);
    attr3.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 3");
    attr3.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 3");
    attr3.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;
    UA_NodeId myUInt8NodeId = UA_NODEID_STRING_ALLOC(1, "the.answer3");
    UA_QualifiedName myUInt8Name = UA_QUALIFIEDNAME_ALLOC(1, "the answer 3");
    UA_Server_addVariableNode(
        server,
        myUInt8NodeId,
        parentNodeId,
        parentReferenceNodeId,
        myUInt8Name,
        UA_NODEID_NULL,
        attr3,
        NULL,
        NULL
    );

    setCustomAccessControl(config);
    UA_ByteString_clear(&certificate);
    UA_ByteString_clear(&privateKey);
    for (size_t i = 0; i < trustListSize; i++)
        UA_ByteString_clear(&trustList[i]);
    if (retval != UA_STATUSCODE_GOOD) goto cleanup;

    if (!running) goto cleanup; /* received ctrl-c already */

    // add a variable node to the adresspace

    /* allocations on the heap need to be freed */
    UA_VariableAttributes_clear(&attr);
    UA_NodeId_clear(&myIntegerNodeId);
    UA_QualifiedName_clear(&myIntegerName);

    retval = UA_Server_run(server, &running);

cleanup:
    UA_Server_delete(server);
    return retval == UA_STATUSCODE_GOOD ? EXIT_SUCCESS : EXIT_FAILURE;
}
