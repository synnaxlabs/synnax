// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { CRON_SECRET } from "astro:env/server";

import { open } from "@/portal/portal";
import { sweep } from "@/server/license/expiry";
import { retrieve } from "@/server/organization";
import { emails } from "@/server/session";

/**
 * GET runs the daily expiry sweep. Vercel calls it with the cron secret as a bearer
 * token; anything else is refused.
 */
export const GET: APIRoute = async (context) => {
  if (context.request.headers.get("authorization") !== `Bearer ${CRON_SECRET}`)
    return new Response("Unauthorized", { status: 401 });
  const portal = open(context);
  const sent = await sweep({
    store: portal.store,
    mail: portal.mail,
    recipients: async (key) => {
      const org = await retrieve(portal.store, key);
      return org == null ? [] : await emails(context, org);
    },
    now: portal.now(),
  });
  return Response.json({ sent });
};
