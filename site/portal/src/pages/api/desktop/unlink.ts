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
import { resolve, unlink } from "@/server/license/desktop";

export const OPTIONS = preflight;

/**
 * POST logs out the machine whose renewal secret is the bearer token, so it leaves its
 * account and the secret stops working. No portal session is involved.
 */
export const POST: APIRoute = async (context) =>
  withCORS(
    await handle(async () => {
      const { portal } = context.locals;
      const machine = await resolve(portal.store, readSecret(context.request));
      await unlink(portal.store, {
        activationKey: machine.activation.key,
        actor: machine.activation.key,
        now: portal.now(),
      });
      return new Response(null, { status: 204 });
    }),
  );
