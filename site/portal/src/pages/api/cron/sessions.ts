// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { IDLE_MS } from "@/server/analytics";
import { recap } from "@/server/slack";

/** PERIOD_MS is how often Vercel Cron calls this route. `vercel.json` sets it. */
const PERIOD_MS = 15 * 60 * 1000;

/** Clerk user IDs start with this, and PostHog sees one once the user logs in. */
const CLERK_USER = "user_";

/**
 * GET posts a recap of each Synnax Desktop session that ended in the last period to
 * Slack. A session ends once it has been idle for {@link IDLE_MS}, so the period is
 * that long behind the call. Vercel calls it with the cron secret as a bearer token;
 * anything else is refused.
 */
export const GET: APIRoute = async (context) => {
  const { portal } = context.locals;
  if (context.request.headers.get("authorization") !== `Bearer ${portal.cronSecret}`)
    return new Response("Unauthorized", { status: 401 });
  const slot = Math.floor(portal.now().getTime() / PERIOD_MS) * PERIOD_MS;
  const to = new Date(slot - IDLE_MS);
  const from = new Date(to.getTime() - PERIOD_MS);
  const sessions = await portal.analytics.sessions(from, to);
  for (const session of sessions) {
    const account = session.distinctID.startsWith(CLERK_USER)
      ? await portal.directory.account(session.distinctID)
      : null;
    await portal.slack.post(recap(session, account));
  }
  return Response.json({ sessions: sessions.length });
};
