#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes a patched copy of an open62541 source file.

Usage: patch.py <source> <output>

Each patch replaces text that must occur exactly once in the source, so an upgrade that
changes the text fails the build instead of skipping the patch.
"""

import os
import sys

# open62541 drops a Session whose activation fails without closing it on the server
# (open62541/open62541#6562), so a rejected login holds a slot in the server's Session
# table until the Session times out. Synnax clients activate only Sessions they created,
# so the patch closes every one the server still knows.
CLOSE_REJECTED_SESSION = (
    """        /* Activating the Session failed */
        cleanupSession(client);
""",
    """        /* Activating the Session failed */
        if(client->connectStatus == UA_STATUSCODE_GOOD &&
           ar->responseHeader.serviceResult != UA_STATUSCODE_BADSESSIONIDINVALID &&
           ar->responseHeader.serviceResult != UA_STATUSCODE_BADSESSIONCLOSED) {
            UA_CloseSessionRequest request;
            UA_CloseSessionRequest_init(&request);
            request.deleteSubscriptions = true;
            __Client_AsyncService(client, &request,
                                  &UA_TYPES[UA_TYPES_CLOSESESSIONREQUEST], NULL,
                                  &UA_TYPES[UA_TYPES_CLOSESESSIONRESPONSE], NULL,
                                  NULL);
        }
        cleanupSession(client);
""",
)

# open62541 1.5 maps the namespace index of each NodeId that a client sends or receives
# through a table that it makes once, on the first Session of the client. A namespace
# that the server adds later has no index in the table, and the client loses the table
# when it connects again. The Driver reads the namespace array itself and uses the
# server's indexes, so the patch keeps each index as it is.
KEEP_NAMESPACE_INDEXES = (
    """    client->channel.namespaceMapping = nsMapping;
""",
    """    UA_NamespaceMapping_delete(nsMapping);
""",
)

PATCHES = {
    "ua_client_connect.c": [CLOSE_REJECTED_SESSION, KEEP_NAMESPACE_INDEXES],
}

source, output = sys.argv[1:]
with open(source) as f:
    text = f.read()
for old, new in PATCHES[os.path.basename(source)]:
    if text.count(old) != 1:
        sys.exit(f"{source}: the patch text occurs {text.count(old)} times, not once")
    text = text.replace(old, new)
with open(output, "w", newline="\n") as f:
    f.write(text)
