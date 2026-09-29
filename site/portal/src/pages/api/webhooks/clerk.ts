// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { verifyWebhook } from "@clerk/astro/webhooks";
import { type APIRoute } from "astro";

import { ensurePersonal, mirrorTeam } from "@/server/organization";

/**
 * POST mirrors Clerk's user and organization creation into the portal: a personal
 * organization per user and a team organization per Clerk organization.
 */
export const POST: APIRoute = async (context) => {
  const { store, webhookSecret } = context.locals.portal;
  let evt: Awaited<ReturnType<typeof verifyWebhook>>;
  try {
    evt = await verifyWebhook(context.request, { signingSecret: webhookSecret });
  } catch (err) {
    console.error("clerk webhook rejected", err);
    return new Response("Bad signature", { status: 400 });
  }
  switch (evt.type) {
    case "user.created": {
      const { id, first_name, last_name, username, email_addresses } = evt.data;
      const name =
        [first_name, last_name].filter((p) => p != null && p !== "").join(" ") ||
        username ||
        email_addresses[0]?.email_address ||
        id;
      await ensurePersonal(store, { userID: id, name });
      break;
    }
    case "organization.created":
    case "organization.updated":
      await mirrorTeam(store, { clerkOrgID: evt.data.id, name: evt.data.name });
      break;
    default:
      break;
  }
  return new Response(null, { status: 204 });
};
