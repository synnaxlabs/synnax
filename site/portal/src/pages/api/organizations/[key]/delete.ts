// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { requireStaff, teamFor } from "@/access";
import { handle } from "@/respond";
import { badRequest } from "@/server/errors";
import { remove } from "@/server/organization";

/**
 * POST deletes the team in Clerk with its licenses, invitations, and history. Staff
 * only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const team = await teamFor(portal, session, context.params.key ?? "");
    if (team.clerkOrgID === portal.staffOrgID)
      throw badRequest(`${team.name} is the staff team and cannot be deleted`);
    await remove(portal.store, portal.directory, team);
    return new Response(null, { status: 204 });
  });
