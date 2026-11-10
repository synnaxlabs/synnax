// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { requireStaff } from "@/access";
import { handle } from "@/respond";
import { badRequest, notFound } from "@/server/errors";
import { remove } from "@/server/user";

/**
 * POST deletes a user from Clerk with their personal organization, Desktop licenses,
 * and history. Staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const userID = context.params.id ?? "";
    const account = await portal.directory.account(userID);
    if (account == null) throw notFound("User");
    const teams = await portal.directory.memberships(userID);
    if (teams.some((t) => t.clerkOrgID === portal.staffOrgID))
      throw badRequest(`${account.name} is on the staff team and cannot be deleted`);
    await remove(portal.store, portal.directory, userID);
    return new Response(null, { status: 204 });
  });
