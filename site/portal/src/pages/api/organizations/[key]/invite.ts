// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { teamFor } from "@/access";
import { form, handle } from "@/respond";
import { invite } from "@/server/member";

/**
 * POST emails `email` an invitation to join the team as `role`. The link in the email
 * lands on the portal's sign-up page. Team admins and staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    const team = await teamFor(portal, session, context.params.key ?? "");
    const body = await form(context);
    await invite(portal.store, portal.directory, {
      team,
      actor: session.userID,
      now: portal.now(),
      email: body.email ?? "",
      role: body.role ?? "",
      redirectURL: new URL("/sign-up", context.url.origin).href,
    });
    return new Response(null, { status: 204 });
  });
