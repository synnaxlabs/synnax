#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.access.policy.client import PolicyClient
from synnax.access.role.client import RoleClient


class Client:
    policies: PolicyClient
    roles: RoleClient

    def __init__(
        self,
        policies: PolicyClient,
        roles: RoleClient,
    ):
        self.policies = policies
        self.roles = roles


__all__ = [
    "Client",
    "PolicyClient",
    "RoleClient",
]
