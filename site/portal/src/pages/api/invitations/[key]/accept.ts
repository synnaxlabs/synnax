// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { handle } from "@/respond";
import { accept } from "@/server/invitation";

/**
 * POST puts the logged-in user on the team the invitation is for. The invitation must
 * be to one of the user's verified email addresses.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    await accept(portal.store, portal.directory, {
      key: context.params.key ?? "",
      session: await portal.session(),
      now: portal.now(),
    });
    return new Response(null, { status: 204 });
  });
