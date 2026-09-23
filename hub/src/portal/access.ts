// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { desc, eq } from "drizzle-orm";

import { type Portal } from "@/portal/portal";
import {
  type Activation,
  activation,
  type License,
  license,
  type Organization,
} from "@/server/db/schema";
import { forbidden, notFound } from "@/server/errors";
import { isMember, retrieve } from "@/server/organization";
import { type Session } from "@/server/session";

export interface LicenseView {
  license: License;
  organization: Organization;
  activations: Activation[];
}

/**
 * licenseFor loads a license with its ledger for a member of the owning organization.
 * Staff see every license. Throws a 404 for an unknown key and a 403 for a
 * non-member, so an outsider cannot tell the two apart from the page body alone.
 */
export const licenseFor = async (
  { store }: Portal,
  session: Session,
  key: string,
): Promise<LicenseView> => {
  const [lic] = await store.query.select().from(license).where(eq(license.key, key));
  if (lic == null) throw notFound("License");
  const organization = await retrieve(store, lic.organization);
  if (organization == null) throw notFound("Organization");
  if (!session.staff && !isMember(organization, session))
    throw forbidden("You are not a member of the organization that owns this license");
  const activations = await store.query
    .select()
    .from(activation)
    .where(eq(activation.license, key))
    .orderBy(desc(activation.lastSeen));
  return { license: lic, organization, activations };
};

/** activationFor loads an activation through the license access check. */
export const activationFor = async (
  portal: Portal,
  session: Session,
  key: string,
): Promise<{ activation: Activation } & LicenseView> => {
  const [act] = await portal.store.query
    .select()
    .from(activation)
    .where(eq(activation.key, key));
  if (act == null) throw notFound("Activation");
  const view = await licenseFor(portal, session, act.license);
  return { activation: act, ...view };
};

export const requireStaff = (session: Session): void => {
  if (!session.staff) throw forbidden("Staff only");
};
