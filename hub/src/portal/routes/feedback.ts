// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { z } from "zod";

import { open } from "@/portal/portal";
import { badRequest, HTTPError, toResponse } from "@/server/errors";
import { ensurePersonal } from "@/server/organization";
import { checkFeedback } from "@/server/ratelimit";
import { record } from "@/server/support/record";

const bodyZ = z.object({
  name: z.string().trim().max(200).default(""),
  email: z.union([z.literal(""), z.email()]).default(""),
  description: z.string().trim().min(1).max(50_000),
  page: z.string().trim().max(500).default("/"),
});

/**
 * POST turns a docs feedback submission into a Plain thread. A signed-in visitor
 * files it under their own customer and personal organization; anyone else under
 * the email they gave, or the anonymous customer when they gave none.
 */
export const POST: APIRoute = async (context) => {
  try {
    const parsed = bodyZ.safeParse(await context.request.json());
    if (!parsed.success) throw badRequest("Enter a message and a valid email");
    const { name, email, description, page } = parsed.data;
    const portal = open(context);
    let session: Awaited<ReturnType<typeof portal.session>> | null = null;
    try {
      session = await portal.session();
    } catch (err) {
      if (!(err instanceof HTTPError) || err.status !== 401)
        throw err instanceof Error ? err : new Error(String(err));
    }
    const actor =
      session?.userID ?? (email === "" ? `ip:${context.clientAddress}` : email);
    await checkFeedback(portal.store, { actor, now: portal.now() });
    let organizationKey: string | undefined;
    if (session != null)
      organizationKey = (
        await ensurePersonal(portal.store, {
          userID: session.userID,
          name: session.name,
        })
      ).key;
    const thread = await portal.support.submitFeedback({
      email: session?.email ?? email,
      name: session?.name ?? name,
      text: description,
      page,
      organizationKey,
    });
    await record(portal.store, {
      kind: "feedback",
      actor,
      organization: organizationKey,
      detail: { thread: thread.id, page },
    });
    return Response.json({ thread: thread.id, ref: thread.ref });
  } catch (err) {
    return toResponse(err);
  }
};
