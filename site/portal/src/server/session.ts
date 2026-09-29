// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ADMIN_ROLE, type Directory, type Team } from "@/server/directory";
import { unauthorized } from "@/server/errors";
import { type Membership } from "@/server/organization";

/** Session is what the portal knows about the logged-in user for one request. */
export interface Session extends Membership {
  email: string;
  name: string;
  teams: Team[];
  /** staff is true for admins of the Synnax Labs team organization. */
  staff: boolean;
}

/**
 * resolve reads the logged-in user and their team memberships. Throws a 401 when
 * nobody is logged in.
 */
export const resolve = async (
  directory: Directory,
  userID: string | null,
  staffOrgID: string,
): Promise<Session> => {
  if (userID == null) throw unauthorized();
  const [person, teams] = await Promise.all([
    directory.person(userID),
    directory.memberships(userID),
  ]);
  return {
    userID,
    ...person,
    teams,
    clerkOrgIDs: teams.map((t) => t.clerkOrgID),
    staff: teams.some((t) => t.clerkOrgID === staffOrgID && t.role === ADMIN_ROLE),
  };
};

/**
 * emails returns the addresses to notify for an organization: its owner for a
 * personal organization, its admins for a team.
 */
export const emails = async (
  directory: Directory,
  org: { kind: string; ownerUserID: string | null; clerkOrgID: string | null },
): Promise<string[]> => {
  if (org.kind === "personal") {
    if (org.ownerUserID == null) return [];
    const { email } = await directory.person(org.ownerUserID);
    return email === "" ? [] : [email];
  }
  if (org.clerkOrgID == null) return [];
  return await directory.admins(org.clerkOrgID);
};
