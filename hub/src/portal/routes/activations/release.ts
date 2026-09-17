// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { activationFor } from "@/portal/access";
import { open } from "@/portal/portal";
import { handle, redirect, wantsHTML } from "@/portal/respond";
import { release } from "@/server/license/activate";

/** POST frees the seat a machine holds. */
export const POST: APIRoute = async (context) => {
  const key = context.params.key ?? "";
  return await handle(context, "/licenses", async () => {
    const portal = open(context);
    const session = await portal.session();
    const { license } = await activationFor(portal, session, key);
    await release(portal.store, {
      activationKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    if (wantsHTML(context)) return redirect(context, `/licenses/${license.key}`);
    return new Response(null, { status: 204 });
  });
};
