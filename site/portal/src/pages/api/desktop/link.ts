// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { form, handle } from "@/respond";
import { isInternal } from "@/server/directory";
import { badRequest } from "@/server/errors";
import { link } from "@/server/license/desktop";
import { parse } from "@/server/license/fingerprint";
import { readName } from "@/server/license/machine";
import { ensurePersonal } from "@/server/organization";
import { check } from "@/server/ratelimit";
import { convert } from "@/server/referral";
import { active, announce, firstLogin } from "@/server/slack";
import { arrival } from "@/server/user";

/**
 * POST links the logged-in user's machine: issues a desktop license bound to the
 * posted `fingerprint`, named by `name`, and answers `{ key, secret, activation,
 * email }`. `analytics` is true when the user lets the app send usage data.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    const body = await form(context);
    let fingerprint: string[];
    try {
      fingerprint = parse(body.fingerprint ?? "");
    } catch (err) {
      throw badRequest((err as Error).message);
    }
    const machineName = readName(body.name);
    const now = portal.now();
    const org = await ensurePersonal(portal.store, {
      userID: session.userID,
      name: session.name,
    });
    await check(portal.store, {
      actor: session.userID,
      organization: org.key,
      now,
    });
    const linked = await link(portal.store, portal.signer, {
      userID: session.userID,
      userName: session.name,
      fingerprint,
      machineName,
      now,
    });
    const referrer =
      linked.previous == null
        ? await convert(portal.store, portal.directory, portal.analytics, {
            userID: session.userID,
            shared: body.analytics === "true",
            now,
          })
        : undefined;
    if (!isInternal(session.email))
      await announce(portal.slack, async () => {
        const { previous } = linked;
        if (previous == null) return firstLogin(session, machineName, referrer);
        const arrived = await arrival(portal.store, portal.directory, {
          userID: session.userID,
          machine: machineName,
          previous,
          now,
        });
        return arrived == null ? null : active(arrived);
      });
    return Response.json({
      key: linked.key,
      secret: linked.secret,
      activation: linked.activation.key,
      email: session.email,
      user: session.userID,
    });
  });
