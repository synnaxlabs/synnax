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
import { mailLink } from "@/server/referral";

/**
 * POST emails the referral link to the logged-in user, so a user on a phone can open
 * it on their computer. A link is emailed at most once every few minutes.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    await mailLink(portal.store, portal.mail, {
      key: context.params.key ?? "",
      session: await portal.session(),
      now: portal.now(),
    });
    return new Response(null, { status: 204 });
  });
