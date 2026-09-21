// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { requireStaff } from "@/portal/access";
import { open } from "@/portal/portal";
import { form, handle } from "@/portal/respond";
import { adoptTeam } from "@/server/directory";
import { badRequest } from "@/server/errors";
import { issue } from "@/server/license/issue";

/**
 * POST issues a license to the Clerk organization in `organization` and answers
 * `{ key }`. Staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const portal = open(context);
    const session = await portal.session();
    requireStaff(session);
    const body = await form(context);
    const term = body.term === "perpetual" ? "perpetual" : "subscription";
    const nodes = Number(body.nodes);
    if (!Number.isInteger(nodes) || nodes < 1)
      throw badRequest("Nodes must be a whole number of at least 1");
    const channels = Number(body.channels || "0");
    if (!Number.isInteger(channels) || channels < 0)
      throw badRequest("Channels must be a whole number, 0 for unlimited");
    if (!body.organization) throw badRequest("Choose an organization");
    const team = await adoptTeam(context, portal.store, body.organization);
    const { key } = await issue(portal.store, {
      organization: team.key,
      edition: "enterprise",
      term,
      nodes,
      channels,
      label: (body.label ?? "").trim(),
      expiresAt:
        term === "subscription" && body.expiresAt
          ? new Date(`${body.expiresAt}T00:00:00Z`)
          : undefined,
      maxVersion: body.maxVersion?.trim() || undefined,
      actor: session.userID,
      now: portal.now(),
    });
    return Response.json({ key });
  });
