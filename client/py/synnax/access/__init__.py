#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient

from synnax.access import policy, role


class Client:
    policies: policy.Client
    roles: role.Client

    def __init__(
        self,
        transport: UnaryClient,
    ):
        self.policies = policy.Client(transport)
        self.roles = role.Client(transport)
