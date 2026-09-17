// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Portal } from "@/portal/portal";
import { type Organization, type Sender } from "@/server/db/schema";
import { forbidden, notFound } from "@/server/errors";
import { isMember, organizationsFor, pick, retrieve } from "@/server/organization";
import { type Session } from "@/server/session";
import { retrieve as retrieveThread, type Retrieved } from "@/server/support/thread";

export interface SupportView {
  organizations: Organization[];
  organization: Organization;
}

/**
 * supportFor resolves the organization a member is asking support for. `key` picks
 * one of the member's organizations; the personal one is the default.
 */
export const supportFor = async (
  portal: Portal,
  session: Session,
  key: string | null,
): Promise<SupportView> => {
  const organizations = await organizationsFor(portal.store, session);
  const org = pick(organizations, key);
  if (org == null) throw forbidden("You are not a member of that organization");
  return { organizations, organization: org };
};

export interface ThreadView extends Retrieved {
  organization: Organization | null;
  /** sender is how the viewer's replies are attributed: a member writes as the
   * customer, staff reading another organization's thread write as staff. */
  sender: Sender;
}

/**
 * threadFor loads a thread for a member of its organization, or for whoever opened
 * it when it has none. Staff read every thread. Throws a 404 for an unknown key and
 * a 403 for anyone else.
 */
export const threadFor = async (
  portal: Portal,
  session: Session,
  key: string,
): Promise<ThreadView> => {
  const view = await retrieveThread(portal.store, portal.tracker, key);
  if (view == null) throw notFound("Thread");
  const org =
    view.thread.organization == null
      ? null
      : ((await retrieve(portal.store, view.thread.organization)) ?? null);
  const member =
    org == null ? view.thread.createdBy === session.userID : isMember(org, session);
  if (!member && !session.staff)
    throw forbidden("You are not a member of the organization that owns this thread");
  return { ...view, organization: org, sender: member ? "customer" : "staff" };
};
