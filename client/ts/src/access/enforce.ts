// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@synnaxlabs/x";

import { type Action } from "@/access/types.gen";
import { type Policy } from "@/access/policy/payload";
import { type ontology } from "@/ontology";

/**
 * Request represents an access control request to check if a subject
 * can perform an action on one or more objects.
 */
export interface Request {
  /** The subject making the request (typically a user) */
  subject: ontology.ID;
  /** The action being requested */
  action: Action;
  /** The objects being accessed */
  objects: ontology.ID | ontology.ID[];
}

/**
 * Checks if a request is allowed based on the provided policies.
 * This is the client-side equivalent of the Go allowRequest function.
 *
 * @param req - The access request to check
 * @param policies - The policies to check against
 * @returns true if the request is allowed, false otherwise
 *
 * @remarks
 * This function implements the following logic:
 * - For each requested object, check if any policy allows the action
 * - A policy allows an action if:
 *   1. The policy's actions include the requested action or "all"
 *   2. The policy's objects include the requested object, either:
 *      - Type-level match: policy object has empty key and matching type
 *      - Instance-level match: policy object has matching type and key
 * - ALL requested objects must be allowed for the request to succeed
 */
export const allowRequest = (req: Request, policies: Policy[]): boolean => {
  const objs = array.toArray(req.objects);
  const { action } = req;
  for (const requestedObj of objs) {
    let allowed = false;

    for (const policy of policies) {
      // Check if every requested action is allowed by this policy
      const actionAllowed = policy.actions.includes(action);
      if (!actionAllowed) continue;

      // Check if any object in the policy matches the requested object
      // Type-level match: empty key means the policy applies to all instances of this type
      for (const policyObj of policy.objects)
        if (policyObj.key === "") {
          if (policyObj.type === requestedObj.type) {
            allowed = true;
            break;
          }
        } else if (
          policyObj.type === requestedObj.type &&
          policyObj.key === requestedObj.key
        ) {
          // Instance-level match: both type and key must match
          allowed = true;
          break;
        }

      if (allowed) break;
    }

    // If any object is not allowed, the entire request fails
    if (!allowed) return false;
  }

  // All objects are allowed
  return true;
};
