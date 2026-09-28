// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clerkClient } from "@clerk/astro/server";
import { type APIContext } from "astro";

import { type Store } from "@/server/db/db";
import { type Organization } from "@/server/db/schema";
import { mirrorTeam } from "@/server/organization";

/** Listed is one organization in the Clerk dashboard, keyed by its Clerk id. */
export interface Listed {
  clerkOrgID: string;
  name: string;
}

const PAGE = 100;

/** listTeams returns every organization in Clerk, newest first. Staff only. */
export const listTeams = async (context: APIContext): Promise<Listed[]> => {
  const clerk = clerkClient(context);
  const all: Listed[] = [];
  for (let offset = 0; ; offset += PAGE) {
    const page = await clerk.organizations.getOrganizationList({ limit: PAGE, offset });
    all.push(...page.data.map((o) => ({ clerkOrgID: o.id, name: o.name })));
    if (page.data.length < PAGE) return all;
  }
};

/** MAX_USERS is how many ids Clerk resolves in one call. */
const MAX_USERS = 100;

/**
 * namesFor resolves Clerk user ids to the name each one reads as. An id Clerk does not
 * know is left out.
 */
export const namesFor = async (
  context: APIContext,
  userIDs: string[],
): Promise<Record<string, string>> => {
  if (userIDs.length === 0) return {};
  const { data } = await clerkClient(context).users.getUserList({
    userId: userIDs.slice(0, MAX_USERS),
    limit: MAX_USERS,
  });
  return Object.fromEntries(
    data.map((u) => [u.id, u.fullName ?? u.primaryEmailAddress?.emailAddress ?? u.id]),
  );
};

/**
 * adoptTeam mirrors a Clerk organization into the portal's tables and returns the
 * row, so a license can be issued to an organization the webhook has not delivered.
 */
export const adoptTeam = async (
  context: APIContext,
  store: Store,
  clerkOrgID: string,
): Promise<Organization> => {
  const org = await clerkClient(context).organizations.getOrganization({
    organizationId: clerkOrgID,
  });
  return await mirrorTeam(store, { clerkOrgID: org.id, name: org.name });
};
