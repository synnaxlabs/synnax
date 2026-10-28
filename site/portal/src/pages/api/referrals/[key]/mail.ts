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
import { notFound } from "@/server/errors";
import { mailLink, retrieve } from "@/server/referral";

/**
 * POST emails the referral link to the logged-in user, so a user on a phone can open
 * it on their computer.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    const ref = await retrieve(portal.store.query, context.params.key ?? "");
    if (ref == null) throw notFound("Referral");
    await mailLink(portal.mail, ref, session);
    return new Response(null, { status: 204 });
  });
