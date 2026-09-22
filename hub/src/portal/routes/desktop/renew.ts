// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { open } from "@/portal/portal";
import { handle } from "@/portal/respond";
import { unauthorized } from "@/server/errors";
import { renew, resolve } from "@/server/license/desktop";
import { check } from "@/server/ratelimit";

const BEARER = /^Bearer\s+(\S+)$/i;

// The Desktop app calls from its own origin, so the route answers preflights and
// marks every response, refusals included, for any origin. The secret is the guard.
const CORS_HEADERS = {
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "POST, OPTIONS",
  "access-control-allow-headers": "authorization",
};

const withCORS = (res: Response): Response => {
  for (const [name, value] of Object.entries(CORS_HEADERS))
    res.headers.set(name, value);
  return res;
};

export const OPTIONS: APIRoute = () =>
  new Response(null, { status: 204, headers: CORS_HEADERS });

/**
 * POST renews the desktop license of the machine whose renewal secret is the bearer
 * token, and answers `{ token }`. No portal session is involved.
 */
export const POST: APIRoute = async (context) =>
  withCORS(
    await handle(async () => {
      const secret = BEARER.exec(
        context.request.headers.get("authorization") ?? "",
      )?.[1];
      if (secret == null) throw unauthorized();
      const portal = open(context);
      const now = portal.now();
      const machine = await resolve(portal.store, secret);
      await check(portal.store, {
        actor: machine.activation.key,
        organization: machine.license.organization,
        now,
      });
      const { token } = await renew(portal.store, portal.signer, { machine, now });
      return Response.json({ token });
    }),
  );
