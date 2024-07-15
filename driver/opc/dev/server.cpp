// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* This work is licensed under a Creative Commons CCZero 1.0 Universal License.
 * See http://creativecommons.org/publicdomain/zero/1.0/ for more information. */

#include <open62541/server.h>
#include <open62541/server_config_default.h>

/* Build Instructions (Linux)
 * - g++ server.cpp -lopen62541 -o server */

using namespace std;

int main() {
    UA_Server *server = UA_Server_new();
    UA_ServerConfig_setDefault(UA_Server_getConfig(server));


    // add a variable node to the adresspace
    UA_VariableAttributes attr = UA_VariableAttributes_default;
    UA_Int32 myInteger = 42;
    UA_Variant_setScalarCopy(&attr.value, &myInteger, &UA_TYPES[UA_TYPES_INT32]);
    attr.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
    attr.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer");
    UA_NodeId myIntegerNodeId = UA_NODEID_STRING_ALLOC(1, "the.answer");
    UA_QualifiedName myIntegerName = UA_QUALIFIEDNAME_ALLOC(1, "the answer");
    UA_NodeId parentNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
    UA_NodeId parentReferenceNodeId = UA_NODEID_NUMERIC(0, UA_NS0ID_ORGANIZES);
    UA_Server_addVariableNode(server, myIntegerNodeId, parentNodeId,
                              parentReferenceNodeId, myIntegerName,
                              UA_NODEID_NULL, attr, NULL, NULL);

    // // add another variable node to the adresspace
    UA_VariableAttributes attr2 = UA_VariableAttributes_default;
    UA_Double myDouble = 3.14;
    UA_Variant_setScalarCopy(&attr2.value, &myDouble, &UA_TYPES[UA_TYPES_DOUBLE]);
    attr2.description = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 2");
    attr2.displayName = UA_LOCALIZEDTEXT_ALLOC("en-US", "the answer 2");
    attr2.accessLevel = UA_ACCESSLEVELMASK_READ | UA_ACCESSLEVELMASK_WRITE;
    UA_NodeId myDoubleNodeId = UA_NODEID_STRING_ALLOC(1, "the.answer2");
    UA_QualifiedName myDoubleName = UA_QUALIFIEDNAME_ALLOC(1, "the answer 2");
    UA_Server_addVariableNode(server, myDoubleNodeId, parentNodeId,
                              parentReferenceNodeId, myDoubleName,
                              UA_NODEID_NULL, attr2, NULL, NULL);



    /* allocations on the heap need to be freed */
    UA_VariableAttributes_clear(&attr);
    UA_NodeId_clear(&myIntegerNodeId);
    UA_QualifiedName_clear(&myIntegerName);

    UA_StatusCode retval = UA_Server_runUntilInterrupt(server);

    UA_Server_delete(server);
    return retval == UA_STATUSCODE_GOOD ? EXIT_SUCCESS : EXIT_FAILURE;
}