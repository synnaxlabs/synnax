#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: RBAC roles, users, and role assignments."""

import synnax as sy

SETUP_VERSION = "0.52"

CUSTOM_ROLE_NAME = "mig_rbac_role"
CUSTOM_ROLE_DESCRIPTION = "Custom role for migration testing"
PASSWORD = "mig_rbac_pass123"

USERS_SPEC: list[tuple[str, str, str, str]] = [
    ("mig_rbac_custom", "MigCustom", "RbacUser", CUSTOM_ROLE_NAME),
    ("mig_rbac_operator", "MigOperator", "RbacUser", "Operator"),
    ("mig_rbac_viewer", "MigViewer", "RbacUser", "Viewer"),
]

if __name__ == "__main__":
    from setup import log, run

    def setup(client: sy.Synnax) -> None:
        log("  [rbac] Creating custom role...")

        role = client.access.roles.create(
            sy.Role(
                name=CUSTOM_ROLE_NAME,
                description=CUSTOM_ROLE_DESCRIPTION,
            )
        )

        log("  [rbac] Creating users...")
        for username, first_name, last_name, _ in USERS_SPEC:
            client.users.create(
                username=username,
                password=PASSWORD,
                first_name=first_name,
                last_name=last_name,
            )

        log("  [rbac] Assigning roles...")
        internal = client.access.roles.retrieve(internal=True)
        builtin_by_name = {r.name: r for r in internal}

        for username, _, _, role_name in USERS_SPEC:
            user = client.users.retrieve(username=username)
            if role_name == CUSTOM_ROLE_NAME:
                role_key = role.key
            else:
                role_key = builtin_by_name[role_name].key
            client.access.roles.assign(user=user.key, role=role_key)

    run(setup)
