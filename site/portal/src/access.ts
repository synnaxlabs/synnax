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

const view = async (
  { store }: Portal,
  session: Session,
  key: string,
): Promise<LicenseView | undefined> => {
  const [lic] = await store.query.select().from(license).where(eq(license.key, key));
  if (lic == null) return undefined;
  const organization = await retrieve(store, lic.organization);
  if (organization == null) return undefined;
  if (!session.staff && !isMember(organization, session)) return undefined;
  const activations = await store.query
    .select()
    .from(activation)
    .where(eq(activation.license, key))
    .orderBy(desc(activation.lastSeen));
  return { license: lic, organization, activations };
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
): Promise<{ activation: Activation } & LicenseView> => {
  const [act] = await portal.store.query
    .select()
    .from(activation)
    .where(eq(activation.key, key));
  const found = act == null ? undefined : await view(portal, session, act.license);
  if (found == null) throw notFound("Activation");
  return { activation: act, ...found };
};

export const requireStaff = (session: Session): void => {
  if (!session.staff) throw forbidden("Staff only");
};
