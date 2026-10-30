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

import { isInternal } from "@/server/directory";
import { ensurePersonal, mirrorTeams } from "@/server/organization";
import { joined } from "@/server/slack";
import { forget } from "@/server/user";

/**
 * POST mirrors Clerk's user and organization creation into the portal: a personal
 * organization per user and a team organization per Clerk organization. A new user
 * outside Synnax Labs is also posted to Slack, and a deleted user's personal
 * organization is deleted.
 */
export const POST: APIRoute = async (context) => {
  const { store, slack, directory, webhookSecret } = context.locals.portal;
  let evt: Awaited<ReturnType<typeof verifyWebhook>>;
  try {
    evt = await verifyWebhook(context.request, { signingSecret: webhookSecret });
  } catch (err) {
    console.error("clerk webhook rejected", err);
    return new Response("Bad signature", { status: 400 });
  }
  switch (evt.type) {
    case "user.created": {
      const {
        id,
        first_name,
        last_name,
        username,
        email_addresses,
        primary_email_address_id,
      } = evt.data;
      const email =
        email_addresses.find((e) => e.id === primary_email_address_id)?.email_address ??
        email_addresses[0]?.email_address ??
        "";
      const name =
        [first_name, last_name].filter((p) => p != null && p !== "").join(" ") ||
        username ||
        email ||
        id;
      await ensurePersonal(store, { userID: id, name });
      if (!isInternal(email)) await slack.post(joined({ userID: id, name, email }));
      break;
    }
    case "user.deleted": {
      const { id } = evt.data;
      if (id == null) throw new Error("clerk sent user.deleted without a user id");
      await forget(store, id);
      break;
    }
    case "organization.created":
    case "organization.updated": {
      // An event can arrive late, even after its team is deleted, so mirror the team as
      // Clerk holds it now.
      const team = await directory.team(evt.data.id);
      if (team != null) await mirrorTeams(store.query, [team]);
      break;
    }
    default:
      break;
  }
  return new Response(null, { status: 204 });
};
