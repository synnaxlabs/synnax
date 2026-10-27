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
import { form, handle } from "@/respond";
import { rename } from "@/server/organization";

/** POST renames the team to `name`. Staff only. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const team = await teamFor(portal, session, context.params.key ?? "");
    const body = await form(context);
    await rename(portal.store, portal.directory, {
      team,
      name: body.name ?? "",
      actor: session.userID,
      now: portal.now(),
    });
    return new Response(null, { status: 204 });
  });
