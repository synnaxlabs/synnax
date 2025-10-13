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
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>

static void handler_TheAnswerChanged(
    UA_Client *client,
    UA_UInt32 subId,
    void *subContext,
    UA_UInt32 monId,
    void *monContext,
    UA_DataValue *value
) {
    if (value->hasValue && UA_Variant_isScalar(&value->value) &&
        value->value.type == &UA_TYPES[UA_TYPES_INT32]) {
        UA_Int32 newValue = *(UA_Int32 *) value->value.data;
        printf("The Answer has changed! New value: %i\n", newValue);
    } else {
        printf("The Answer has changed, but the new value is not an Int32.\n");
    }
}

static void handler_TheAnswer3Changed(
    UA_Client *client,
    UA_UInt32 subId,
    void *subContext,
    UA_UInt32 monId,
    void *monContext,
    UA_DataValue *value
) {
    if (value->hasValue && UA_Variant_isScalar(&value->value) &&
        value->value.type == &UA_TYPES[UA_TYPES_BYTE]) {
        UA_Byte newValue = *(UA_Byte *) value->value.data;
        printf("The Answer 3 has changed! New value: %u\n", newValue);
    } else {
        printf("The Answer 3 has changed, but the new value is not a Byte.\n");
    }
}

static UA_StatusCode node_iter(
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

static volatile bool running = true;

void stopHandler(int signum) {
    running = false;
}

int main(int argc, char *argv[]) {
    signal(SIGINT, stopHandler); // Set up signal handler for Ctrl+C

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
    retval = UA_Client_connect(client, "opc.tcp://localhost:4840");
    if (retval != UA_STATUSCODE_GOOD) {
        printf("Could not connect\n");
        UA_Client_delete(client);
        return EXIT_SUCCESS;
    }

    /* Create a subscription */
    UA_CreateSubscriptionRequest request = UA_CreateSubscriptionRequest_default();
    UA_CreateSubscriptionResponse
        response = UA_Client_Subscriptions_create(client, request, NULL, NULL, NULL);

    UA_UInt32 subId = response.subscriptionId;
    if (response.responseHeader.serviceResult == UA_STATUSCODE_GOOD)
        printf("Create subscription succeeded, id %u\n", subId);

    /* Monitor "the.answer" */
    UA_MonitoredItemCreateRequest monRequest = UA_MonitoredItemCreateRequest_default(
        UA_NODEID_STRING(1, "the.answer")
    );

    UA_MonitoredItemCreateResult
        monResponse = UA_Client_MonitoredItems_createDataChange(
            client,
            response.subscriptionId,
            UA_TIMESTAMPSTORETURN_BOTH,
            monRequest,
            NULL,
            handler_TheAnswerChanged,
            NULL
        );

    if (monResponse.statusCode == UA_STATUSCODE_GOOD)
        printf("Monitoring 'the.answer', id %u\n", monResponse.monitoredItemId);

    /* Monitor "the.answer3" */
    UA_MonitoredItemCreateRequest monRequest3 = UA_MonitoredItemCreateRequest_default(
        UA_NODEID_STRING(1, "the.answer3")
    );

    UA_MonitoredItemCreateResult
        monResponse3 = UA_Client_MonitoredItems_createDataChange(
            client,
            response.subscriptionId,
            UA_TIMESTAMPSTORETURN_BOTH,
            monRequest3,
            NULL,
            handler_TheAnswer3Changed,
            NULL
        );
    if (monResponse3.statusCode == UA_STATUSCODE_GOOD)
        printf("Monitoring 'the.answer3', id %u\n", monResponse3.monitoredItemId);

    /* Run in a loop until Ctrl+C */
    while (running) {
        UA_Client_run_iterate(client, 1000); // Wait for 1000 ms for incoming messages
    }

    /* Clean up */
    if (UA_Client_Subscriptions_deleteSingle(client, subId) == UA_STATUSCODE_GOOD)
        printf("Subscription removed\n");
    UA_Client_disconnect(client);
    UA_Client_delete(client);
    return EXIT_SUCCESS;
}
