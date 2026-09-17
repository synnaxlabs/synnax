// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clerkClient } from "@clerk/astro/server";
import { type APIRoute } from "astro";
import { PLAIN_SIGNING_SECRET } from "astro:env/server";
import { and, count, desc, eq, inArray, isNull } from "drizzle-orm";

import { open } from "@/portal/portal";
import { activation, license, organization } from "@/server/db/schema";
import { listForMember } from "@/server/organization";
import { memberships } from "@/server/session";
import {
  build,
  requestZ,
  SIGNATURE_HEADER,
  verify,
  type View,
} from "@/server/support/card";

const RECENT_ACTIVATIONS = 10;

/**
 * POST serves Plain's customer cards: the organizations, licenses, and recent
 * activations of the customer an agent is looking at. Plain signs the body with the
 * workspace's request signing secret; anything else is refused.
 */
export const POST: APIRoute = async (context) => {
  const body = await context.request.text();
  if (
    !verify(PLAIN_SIGNING_SECRET, body, context.request.headers.get(SIGNATURE_HEADER))
  )
    return new Response("Bad signature", { status: 403 });
  const parsed = requestZ.safeParse(JSON.parse(body));
  if (!parsed.success) return new Response("Bad request", { status: 400 });
  const { cardKeys, customer } = parsed.data;
  const portal = open(context);
  const userID = customer.externalId ?? (await userIDByEmail(context, customer.email));
  const view: View = {
    organizations: [],
    licenses: [],
    activations: [],
    site: context.url.origin,
    now: portal.now(),
  };
  if (userID != null) {
    const clerkOrgIDs = (await memberships(context, userID)).map(
      (m) => m.organization.id,
    );
    view.organizations = await listForMember(portal.store, { userID, clerkOrgIDs });
  }
  if (view.organizations.length > 0) {
    const keys = view.organizations.map((o) => o.key);
    const rows = await portal.store.query
      .select({
        license,
        seats: count(activation.key),
        organizationName: organization.name,
      })
      .from(license)
      .innerJoin(organization, eq(organization.key, license.organization))
      .leftJoin(
        activation,
        and(eq(activation.license, license.key), isNull(activation.releasedAt)),
      )
      .where(inArray(license.organization, keys))
      .groupBy(license.key, organization.name)
      .orderBy(desc(license.issuedAt));
    view.licenses = rows.map((r) => ({
      ...r.license,
      seats: r.seats,
      organizationName: r.organizationName,
    }));
    if (rows.length > 0) {
      const labels = new Map(
        rows.map((r) => [r.license.key, r.license.label || r.license.key]),
      );
      const acts = await portal.store.query
        .select()
        .from(activation)
        .where(inArray(activation.license, [...labels.keys()]))
        .orderBy(desc(activation.lastSeen))
        .limit(RECENT_ACTIVATIONS);
      view.activations = acts.map((a) => ({
        ...a,
        label: labels.get(a.license) ?? "",
      }));
    }
  }
  return Response.json({ cards: build(cardKeys, view) });
};

const userIDByEmail = async (
  context: Parameters<typeof clerkClient>[0],
  email: string,
): Promise<string | null> => {
  if (email === "") return null;
  const users = await clerkClient(context).users.getUserList({
    emailAddress: [email],
    limit: 1,
  });
  return users.data[0]?.id ?? null;
};
