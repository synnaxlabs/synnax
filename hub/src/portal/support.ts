// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";

import { type Portal } from "@/portal/portal";
import { type Organization, organization } from "@/server/db/schema";
import { forbidden, notFound } from "@/server/errors";
import {
  ensurePersonal,
  isMember,
  listForMember,
  retrieve,
} from "@/server/organization";
import { type Session } from "@/server/session";
import { type ThreadDetail } from "@/server/support/support";

export interface SupportView {
  organizations: Organization[];
  organization: Organization;
  /** customerID is the member's Plain customer id inside the organization's tenant. */
  customerID: string;
}

/**
 * supportFor resolves the organization a member is asking support for, mirroring it
 * and the member into Plain on the way. `key` picks one of the member's
 * organizations; the personal one is the default.
 */
export const supportFor = async (
  portal: Portal,
  session: Session,
  key: string | null,
): Promise<SupportView> => {
  await ensurePersonal(portal.store, { userID: session.userID, name: session.name });
  const organizations = await listForMember(portal.store, session);
  const org = key == null ? organizations[0] : organizations.find((o) => o.key === key);
  if (org == null) throw forbidden("You are not a member of that organization");
  const customerID = await mirror(portal, session, org);
  return { organizations, organization: org, customerID };
};

export interface ThreadView {
  thread: ThreadDetail;
  organization: Organization;
  /** customerID is set when the viewer may reply, null for staff reading along. */
  customerID: string | null;
}

/**
 * threadFor loads a thread for a member of its organization. Staff read every
 * thread. Throws a 404 for an unknown thread and a 403 for a non-member.
 */
export const threadFor = async (
  portal: Portal,
  session: Session,
  id: string,
): Promise<ThreadView> => {
  const thread = await portal.support.retrieveThread(id);
  if (thread == null || thread.tenantExternalID == null) throw notFound("Thread");
  const org = await retrieve(portal.store, thread.tenantExternalID);
  if (org == null) throw notFound("Thread");
  const member = isMember(org, session);
  if (!member && !session.staff)
    throw forbidden("You are not a member of the organization that owns this thread");
  const customerID = member ? await mirror(portal, session, org) : null;
  return { thread, organization: org, customerID };
};

const mirror = async (
  { store, support }: Portal,
  session: Session,
  org: Organization,
): Promise<string> => {
  if (org.plainTenantID == null) {
    const plainTenantID = await support.ensureTenant(org.key, org.name);
    await store.query
      .update(organization)
      .set({ plainTenantID })
      .where(eq(organization.key, org.key));
    org.plainTenantID = plainTenantID;
  }
  return await support.ensureMember(
    { userID: session.userID, email: session.email, name: session.name },
    org.key,
  );
};
