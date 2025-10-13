// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <open62541/client_config_default.h>
#include <open62541/client_highlevel.h>
#include <open62541/client_subscriptions.h>
#include <open62541/plugin/log_stdout.h>
#include <stdio.h>
#include <stdlib.h>

[[maybe_unused]] static UA_StatusCode node_iter(
    UA_NodeId childId,
    UA_Boolean isInverse,
    UA_NodeId referenceTypeId,
    void *handle
) {
    if (isInverse) return UA_STATUSCODE_GOOD;
    UA_NodeId *parent = (UA_NodeId *) handle;
    printf(
        "%u, %u --- %u ---> NodeId %u, %u\n",
        parent->namespaceIndex,
        parent->identifier.numeric,
        referenceTypeId.identifier.numeric,
        childId.namespaceIndex,
        childId.identifier.numeric
    );
    return UA_STATUSCODE_GOOD;
}

int main(int argc, char *argv[]) {
    UA_Client *client = UA_Client_new();
    UA_ClientConfig_setDefault(UA_Client_getConfig(client));

    /* Listing endpoints */
    UA_EndpointDescription *endpointArray = NULL;
    size_t endpointArraySize = 0;
    UA_StatusCode retval = UA_Client_getEndpoints(
        client,
        "opc.tcp://localhost:4840",
        &endpointArraySize,
        &endpointArray
    );
    if (retval != UA_STATUSCODE_GOOD) {
        printf("Could not get the endpoints\n");
        UA_Array_delete(
            endpointArray,
            endpointArraySize,
            &UA_TYPES[UA_TYPES_ENDPOINTDESCRIPTION]
        );
        UA_Client_delete(client);
        return EXIT_SUCCESS;
    }
    printf("%i endpoints found\n", (int) endpointArraySize);
    for (size_t i = 0; i < endpointArraySize; i++) {
        printf(
            "URL of endpoint %i is %.*s\n",
            (int) i,
            (int) endpointArray[i].endpointUrl.length,
            endpointArray[i].endpointUrl.data
        );
    }
    UA_Array_delete(
        endpointArray,
        endpointArraySize,
        &UA_TYPES[UA_TYPES_ENDPOINTDESCRIPTION]
    );
    UA_Client_delete(client);

    /* Create a client and connect */
    printf("Creating a client and connecting to the server\n");
    client = UA_Client_new();
    UA_ClientConfig_setDefault(UA_Client_getConfig(client));
    /* Connect to a server */
    /* anonymous connect would be: retval = UA_Client_connect(client,
     * "opc.tcp://localhost:4840"); */
    retval = UA_Client_connect(client, "opc.tcp://localhost:4840");
    if (retval != UA_STATUSCODE_GOOD) {
        printf("Could not connect\n");
        UA_Client_delete(client);
        return EXIT_SUCCESS;
    }

    /* Read attribute */
    UA_Int32 value = 0;
    printf("\nReading the value of node (1, \"the.answer\"):\n");
    UA_Variant *val = UA_Variant_new();
    retval = UA_Client_readValueAttribute(
        client,
        UA_NODEID_STRING(1, const_cast<char*>("the.answer")),
        val
    );
    if (retval == UA_STATUSCODE_GOOD && UA_Variant_isScalar(val) &&
        val->type == &UA_TYPES[UA_TYPES_INT32]) {
        value = *(UA_Int32 *) val->data;
        printf("the value is: %i\n", value);
    }
    UA_Variant_delete(val);

    /* Write node attribute (using the highlevel API) */
    value += 3;
    UA_Variant *myVariant = UA_Variant_new();
    UA_Variant_setScalarCopy(myVariant, &value, &UA_TYPES[UA_TYPES_INT32]);
    UA_Client_writeValueAttribute(client, UA_NODEID_STRING(1, const_cast<char*>("the.answer")), myVariant);
    UA_Variant_delete(myVariant);

    /* Read attribute for "the.answer3" */
    UA_Byte value3 = 0;
    printf("\nReading the value of node (1, \"the.answer3\"):\n");
    UA_Variant *val3 = UA_Variant_new();
    retval = UA_Client_readValueAttribute(
        client,
        UA_NODEID_STRING(1, const_cast<char*>("the.answer3")),
        val3
    );
    if (retval == UA_STATUSCODE_GOOD && UA_Variant_isScalar(val3) &&
        val3->type == &UA_TYPES[UA_TYPES_BYTE]) {
        value3 = *(UA_Byte *) val3->data;
        printf("the value of the.answer3 is: %u\n", value3);
    }
    UA_Variant_delete(val3);

    /* Toggle and write node attribute for "the.answer3" */
    value3 = value3 == 0 ? 1 : 0; // Toggle between 0 and 1
    UA_Variant *myVariant3 = UA_Variant_new();
    UA_Variant_setScalarCopy(myVariant3, &value3, &UA_TYPES[UA_TYPES_BYTE]);
    retval = UA_Client_writeValueAttribute(
        client,
        UA_NODEID_STRING(1, const_cast<char*>("the.answer3")),
        myVariant3
    );
    if (retval == UA_STATUSCODE_GOOD) {
        printf("Successfully wrote %u to the.answer3\n", value3);
    } else {
        printf(
            "Failed to write to the.answer3. Status code %s\n",
            UA_StatusCode_name(retval)
        );
    }
    UA_Variant_delete(myVariant3);

    UA_Client_disconnect(client);
    UA_Client_delete(client);
    return EXIT_SUCCESS;
}
