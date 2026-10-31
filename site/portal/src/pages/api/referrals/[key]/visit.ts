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
import { isInternal } from "@/server/directory";
import { referrerOf, visit } from "@/server/referral";
import { announce, claimed } from "@/server/slack";

/**
 * POST records that the visitor opened the referral link, and claims it for a
 * logged-in visitor. A visitor who is not logged in may call it.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.viewer();
    const ref = await visit(portal.store, {
      key: context.params.key ?? "",
      session,
      now: portal.now(),
    });
    if (ref != null && session != null && !isInternal(session.email))
      await announce(portal.slack, async () =>
        claimed(session, ref, await referrerOf(portal.directory, ref)),
      );
    return new Response(null, { status: 204 });
  });
