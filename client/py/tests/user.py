#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy


def create_user(client: sy.Synnax, username: str, password: str) -> sy.user.User:
    """Creates a user named after the username that can log in with the password."""
    user = client.users.create(name=username)
    client.auth.passwords.create(username, password, subject=user.ontology_id)
    return user
