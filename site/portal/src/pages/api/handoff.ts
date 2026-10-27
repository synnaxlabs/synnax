// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { preflight, readSecret, withCORS } from "@/bearer";
import { handle } from "@/respond";
import { take } from "@/server/license/handoff";

export const OPTIONS = preflight;

/**
 * POST answers `{ key }` with the license key held for the handoff code that is the
 * bearer token, and stops holding it. Answers 204 while the portal holds no key for
 * the code. No portal session is involved.
 */
export const POST: APIRoute = async (context) =>
  withCORS(
    await handle(async () => {
      const { portal } = context.locals;
      const key = await take(portal.store, {
        code: readSecret(context.request),
        now: portal.now(),
      });
      return key == null ? new Response(null, { status: 204 }) : Response.json({ key });
    }),
  );
