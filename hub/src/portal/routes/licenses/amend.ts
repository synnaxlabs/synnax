// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { licenseFor, requireStaff } from "@/portal/access";
import { open } from "@/portal/portal";
import { form, handle } from "@/portal/respond";
import { badRequest } from "@/server/errors";
import { amend } from "@/server/license/issue";

/** POST changes the terms of a license already issued. Staff only. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    requireStaff(session);
    await licenseFor(portal, session, key);
    const body = await form(context);
    const term = body.term === "perpetual" ? "perpetual" : "subscription";
    const nodes = Number(body.nodes);
    if (!Number.isInteger(nodes) || nodes < 1)
      throw badRequest("Nodes must be a whole number of at least 1");
    const channels = Number(body.channels || "0");
    if (!Number.isInteger(channels) || channels < 0)
      throw badRequest("Channels must be a whole number, 0 for unlimited");
    await amend(portal.store, {
      licenseKey: key,
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
    return new Response(null, { status: 204 });
  });
