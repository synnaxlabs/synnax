// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { form, handle } from "@/respond";
import { ADMIN_ROLE, MEMBER_ROLE } from "@/server/directory";
import { badRequest, forbidden } from "@/server/errors";

/**
 * POST emails `email` an invitation to join the Clerk organization in `organization` as
 * `role`. The link in the email lands on the portal's sign-up page. Team admins only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    const body = await form(context);
    const team = session.teams.find((t) => t.clerkOrgID === body.organization);
    if (team?.role !== ADMIN_ROLE) throw forbidden("Only a team admin can invite");
    const email = (body.email ?? "").trim();
    if (email === "") throw badRequest("Enter an email address");
    if (body.role !== ADMIN_ROLE && body.role !== MEMBER_ROLE)
      throw badRequest("Choose a role");
    await portal.directory.invite({
      clerkOrgID: team.clerkOrgID,
      email,
      role: body.role,
      inviterUserID: session.userID,
      redirectURL: new URL("/sign-up", context.url.origin).href,
    });
    return new Response(null, { status: 204 });
  });
