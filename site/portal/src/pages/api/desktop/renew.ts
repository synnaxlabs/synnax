// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { preflight, secret, withCORS } from "@/bearer";
import { handle } from "@/respond";
import { renew, resolve } from "@/server/license/desktop";
import { check } from "@/server/ratelimit";

export const OPTIONS = preflight;

/**
 * POST renews the desktop license of the machine whose renewal secret is the bearer
 * token, and answers `{ key }`. No portal session is involved.
 */
export const POST: APIRoute = async (context) =>
  withCORS(
    await handle(async () => {
      const { portal } = context.locals;
      const now = portal.now();
      const machine = await resolve(portal.store, secret(context.request));
      await check(portal.store, {
        actor: machine.activation.key,
        organization: machine.license.organization,
        now,
      });
      const { key } = await renew(portal.store, portal.signer, { machine, now });
      return Response.json({ key });
    }),
  );
