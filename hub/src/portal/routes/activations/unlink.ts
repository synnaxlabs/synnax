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
import { handle } from "@/portal/respond";
import { unlink } from "@/server/license/desktop";

/** POST unlinks a Desktop machine: its seat, its renewal secret, and its license. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    await activationFor(portal, session, key);
    await unlink(portal.store, {
      activationKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    return new Response(null, { status: 204 });
  });
