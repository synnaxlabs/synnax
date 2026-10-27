// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { desc, eq } from "drizzle-orm";

import { type Portal } from "@/portal";
import { schema } from "@/server/db";
import { ADMIN_ROLE } from "@/server/directory";
import { forbidden, notFound } from "@/server/errors";
import { type Team } from "@/server/member";
import { isMember, retrieve } from "@/server/organization";
import { type Session } from "@/server/session";

export interface LicenseView {
  license: schema.License;
  organization: schema.Organization;
  activations: schema.Activation[];
}

const view = async (
  { store }: Portal,
  session: Session,
  key: string,
): Promise<LicenseView | undefined> => {
  const [found] = await store.query
    .select({ license: schema.license, organization: schema.organization })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .where(eq(schema.license.key, key));
  if (found == null) return undefined;
  if (!session.staff && !isMember(found.organization, session)) return undefined;
  const activations = await store.query
    .select()
    .from(schema.activation)
    .where(eq(schema.activation.license, key))
    .orderBy(desc(schema.activation.lastSeen));
  return { ...found, activations };
};

/**
 * licenseFor loads a license with its ledger for a member of the owning organization.
 * Staff see every license. Throws the same 404 for an unknown key and a non-member, so
 * an outsider cannot tell whether a license exists.
 */
export const licenseFor = async (
  portal: Portal,
  session: Session,
  key: string,
): Promise<LicenseView> => {
  const found = await view(portal, session, key);
  if (found == null) throw notFound("License");
  return found;
};

/** activationFor loads an activation through the same access check as licenseFor. */
export const activationFor = async (
  portal: Portal,
  session: Session,
  key: string,
): Promise<{ activation: schema.Activation } & LicenseView> => {
  const [act] = await portal.store.query
    .select()
    .from(schema.activation)
    .where(eq(schema.activation.key, key));
  const found = act == null ? undefined : await view(portal, session, act.license);
  if (found == null) throw notFound("Activation");
  return { activation: act, ...found };
};

export const requireStaff = (session: Session): void => {
  if (!session.staff) throw forbidden("Staff only");
};

/**
 * teamFor loads a team for a request that changes its members, which a team admin or
 * staff may make. Throws the same 404 for an unknown key and an outsider.
 * @throws {HTTPError} 403 for a member who is not an admin.
 */
export const teamFor = async (
  { store }: Portal,
  session: Session,
  key: string,
): Promise<Team> => {
  const org = await retrieve(store, key);
  if (org?.kind !== "team") throw notFound("Organization");
  if (org.clerkOrgID == null)
    throw new Error(`team ${org.key} has no Clerk organization`);
  const team = { ...org, clerkOrgID: org.clerkOrgID };
  if (session.staff) return team;
  const role = session.teams.find((t) => t.clerkOrgID === org.clerkOrgID)?.role;
  if (role == null) throw notFound("Organization");
  if (role !== ADMIN_ROLE) throw forbidden("Only a team admin can change members");
  return team;
};
