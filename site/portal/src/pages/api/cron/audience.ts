// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { sync } from "@/server/audience";

/**
 * GET syncs the accounts on the release notes list. Vercel calls it with the cron
 * secret as a bearer token; anything else is refused.
 */
export const GET: APIRoute = async (context) => {
  const { portal } = context.locals;
  if (context.request.headers.get("authorization") !== `Bearer ${portal.cronSecret}`)
    return new Response("Unauthorized", { status: 401 });
  return Response.json(await sync(portal.directory, portal.audience));
};
