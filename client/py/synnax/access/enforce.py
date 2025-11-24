#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

from typing import TYPE_CHECKING

from synnax.ontology.payload import ID

if TYPE_CHECKING:
    from synnax.access.payload import Policy

# Action constants
ALL_ACTION = "*"
CREATE_ACTION = "create"
DELETE_ACTION = "delete"
RETRIEVE_ACTION = "retrieve"
UPDATE_ACTION = "update"


class Request:
    """Request represents an access control request to check if a subject
    can perform an action on one or more objects.
    """

    subject: ID
    actions: list[str]
    objects: list[ID]

    def __init__(
        self,
        subject: ID,
        actions: str | list[str],
        objects: ID | list[ID],
    ):
        self.subject = subject
        self.actions = [actions] if isinstance(actions, str) else actions
        self.objects = [objects] if not isinstance(objects, list) else objects


def allow_request(req: Request, policies: list[Policy]) -> bool:
    """Checks if a request is allowed based on the provided policies.
    This is the client-side equivalent of the Go allowRequest function.

    Args:
        req: The access request to check
        policies: The policies to check against

    Returns:
        True if the request is allowed, False otherwise

    Remarks:
        This function implements the following logic:
        - For each requested object, check if any policy allows the action
        - A policy allows an action if:
          1. The policy's actions include the requested action OR "*" (all actions)
          2. The policy's objects include the requested object, either:
             - Type-level match: policy object has empty key and matching type
             - Instance-level match: policy object has matching type and key
        - ALL requested objects must be allowed for the request to succeed
    """
    for requested_obj in req.objects:
        allowed = False

        for policy in policies:
            # Check if every requested action is allowed by this policy
            action_allowed = (
                all(action in policy.actions for action in req.actions)
                or ALL_ACTION in policy.actions
            )

            if not action_allowed:
                continue

            # Check if any object in the policy matches the requested object
            for policy_obj in policy.objects:
                if policy_obj.key == "":
                    # Type-level match: empty key means the policy applies to all instances of this type
                    if policy_obj.type == requested_obj.type:
                        allowed = True
                        break
                elif (
                    policy_obj.type == requested_obj.type
                    and policy_obj.key == requested_obj.key
                ):
                    # Instance-level match: both type and key must match
                    allowed = True
                    break

            if allowed:
                break

        # If any object is not allowed, the entire request fails
        if not allowed:
            return False

    # All objects are allowed
    return True
