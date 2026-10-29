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
import { isInternal } from "@/server/directory";
import { renew, resolve } from "@/server/license/desktop";
import { check } from "@/server/ratelimit";
import { active, announce } from "@/server/slack";
import { arrival } from "@/server/user";

export const OPTIONS = preflight;

/**
 * POST renews the desktop license of the machine whose renewal secret is the bearer
 * token, and answers `{ key }`. No portal session is involved. The first Desktop use of
 * a day is posted to Slack, unless the user is from Synnax Labs.
 */
export const POST: APIRoute = async (context) =>
  withCORS(
    await handle(async () => {
      const { portal } = context.locals;
      const now = portal.now();
      const machine = await resolve(portal.store, readSecret(context.request));
      await check(portal.store, {
        actor: machine.activation.key,
        organization: machine.license.organization,
        now,
      });
      const { key, owner, previous } = await renew(portal.store, portal.signer, {
        machine,
        now,
      });
      await announce(portal.slack, async () => {
        const arrived = await arrival(portal.store, portal.directory, {
          userID: owner,
          machine: machine.activation.name ?? "Unnamed machine",
          previous,
          now,
        });
        return arrived == null || isInternal(arrived.account.email)
          ? null
          : active(arrived);
      });
      return Response.json({ key });
    }),
  );
