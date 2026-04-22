#!/usr/bin/env python3

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: RBAC roles, users, and role assignments.

Standalone script — uses only synnax + stdlib.
Once committed, this file is never modified.
"""

from setup import log, run

import synnax as sy

SETUP_VERSION = "0.52"


def setup(client: sy.Synnax) -> None:
    log("  [rbac] Creating custom role...")

    CUSTOM_ROLE_NAME = "mig_rbac_role"
    USER_PASSWORD = "mig_rbac_pass123"

    role = client.access.roles.create(
        sy.Role(
            name=CUSTOM_ROLE_NAME,
            description="Custom role for migration testing",
        )
    )

    users_spec = [
        ("mig_rbac_custom", "MigCustom", "RbacUser", CUSTOM_ROLE_NAME),
        ("mig_rbac_operator", "MigOperator", "RbacUser", "Operator"),
        ("mig_rbac_viewer", "MigViewer", "RbacUser", "Viewer"),
    ]

    log("  [rbac] Creating users...")
    for username, first_name, last_name, _ in users_spec:
        client.users.create(
            username=username,
            password=USER_PASSWORD,
            first_name=first_name,
            last_name=last_name,
        )

    log("  [rbac] Assigning roles...")
    internal = client.access.roles.retrieve(internal=True)
    builtin_by_name = {r.name: r for r in internal}

    for username, _, _, role_name in users_spec:
        user = client.users.retrieve(username=username)
        if role_name == CUSTOM_ROLE_NAME:
            role_key = role.key
        else:
            role_key = builtin_by_name[role_name].key
        client.access.roles.assign(user=user.key, role=role_key)


run(setup)
